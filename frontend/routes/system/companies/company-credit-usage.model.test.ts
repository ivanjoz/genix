import { describe, expect, test } from 'bun:test';
import type { ICompany } from './empresas.svelte';
import {
  type ICompanyCreditCompanyDay,
  type ICompanyCreditDay,
  type ICompanyCreditUserDay,
  buildCompanyDays,
  buildCompanyCreditSummaries,
  buildCompanyDayDetail,
  buildCompanyUserSummaries,
  companyUserDisplayName,
  creditMeterFillPercent,
  rankCompanyCreditUsage,
  splitCompanyCreditRoute,
  usagePercent,
} from './company-credit-usage.model';

const LAST_DAY = 129;
const FIRST_DAY = LAST_DAY - 29;

const companyRow = (
  companyID: number, day: number, cpu: number, inference: number,
): ICompanyCreditDay => ({
  ID: companyID * 100_000 + day, CompanyID: companyID, Day: day,
  CPU: cpu, Inference: inference, upd: 200_000_000 + day, ss: 1,
});

const userRow = (
  userID: number, day: number, cpu: number, inference: number,
): ICompanyCreditUserDay => ({
  ID: userID * 100_000 + day, UserID: userID, Day: day,
  CPU: cpu, Inference: inference, upd: 200_000_000 + day, ss: 1,
});

const companyDayRow = (
  day: number, cpu: number, inference: number, routes?: ICompanyCreditCompanyDay['Routes'],
): ICompanyCreditCompanyDay => ({
  ID: day, Day: day, CPU: cpu, Inference: inference, Routes: routes,
  upd: 200_000_000 + day, ss: 1,
});

const companies = [
  { id: 1, Name: 'Alpha', ss: 1 },
  { id: 2, Name: 'Beta', LegalName: 'Beta Legal', RUC: '20123456789', ss: 1 },
] as ICompany[];

describe('company credit summaries', () => {
  const rows = [
    companyRow(1, FIRST_DAY, 10, 9), companyRow(1, LAST_DAY, 0, 1),
    companyRow(2, FIRST_DAY, 17, 2), companyRow(2, LAST_DAY, 3, 0),
  ];

  test('groups rows per company, totalling and zero-filling the thirty-day window', () => {
    const summaries = buildCompanyCreditSummaries(rows, companies);
    const alpha = summaries.find((company) => company.CompanyID === 1)!;
    expect(alpha.Company).toBe('Alpha');
    expect(alpha.CPU).toBe(10);
    expect(alpha.Inference).toBe(10);
    expect(alpha.Days).toHaveLength(30);
    expect(alpha.Days[0].Day).toBe(FIRST_DAY);
    expect(alpha.Days[29].Day).toBe(LAST_DAY);
    // A day with no row must still hold a column, or the chart shifts.
    expect(alpha.Days[1]).toEqual({ Day: FIRST_DAY + 1, CPU: 0, Inference: 0 });
  });

  test('reports today from the last column and counts only days with usage', () => {
    const alpha = buildCompanyCreditSummaries(rows, companies).find((c) => c.CompanyID === 1)!;
    expect(alpha.TodayCPU).toBe(0);
    expect(alpha.TodayInference).toBe(1);
    expect(alpha.ActiveDays).toBe(2);
  });

  // The catalog drives the list: a company that has never spent a credit still needs its card.
  test('keeps a company with no usage at all', () => {
    const summaries = buildCompanyCreditSummaries([companyRow(2, LAST_DAY, 5, 0)], companies);
    const alpha = summaries.find((company) => company.CompanyID === 1)!;
    expect(alpha.Company).toBe('Alpha');
    expect(alpha.CPU).toBe(0);
    expect(alpha.Days).toHaveLength(30);
  });

  test('renders every company even when nothing has been spent yet', () => {
    const summaries = buildCompanyCreditSummaries([], companies);
    expect(summaries.map((company) => company.CompanyID)).toEqual([1, 2]);
    expect(summaries[0].Days).toHaveLength(30);
  });

  // Usage for a company the catalog has not delivered yet must not silently disappear.
  test('surfaces usage from a company missing from the catalog', () => {
    const summaries = buildCompanyCreditSummaries([companyRow(9, LAST_DAY, 1, 0)], companies);
    const unknown = summaries.find((company) => company.CompanyID === 9)!;
    expect(unknown.Company).toBe('Company #9');
    expect(unknown.CPU).toBe(1);
  });

  // The delta cache keeps every row it was ever sent, including days that have left the window.
  test('ignores rows older than the window', () => {
    const summaries = buildCompanyCreditSummaries(
      [companyRow(1, FIRST_DAY - 5, 999, 0), companyRow(1, LAST_DAY, 4, 0)], companies,
    );
    expect(summaries[0].CPU).toBe(4);
  });

  test('survives a non-finite metric instead of rendering NaN', () => {
    const summaries = buildCompanyCreditSummaries(
      [companyRow(1, LAST_DAY, Number.NaN, 3)], companies,
    );
    expect(summaries[0].CPU).toBe(0);
    expect(summaries[0].Inference).toBe(3);
  });
});

describe('company ranking', () => {
  const summaries = buildCompanyCreditSummaries([
    companyRow(1, LAST_DAY, 10, 9), companyRow(2, LAST_DAY, 20, 2),
  ], companies);

  test('ranks by the selected independent credit pool', () => {
    expect(rankCompanyCreditUsage(summaries, 'CPU').map((c) => c.CompanyID)).toEqual([2, 1]);
    expect(rankCompanyCreditUsage(summaries, 'Inference').map((c) => c.CompanyID)).toEqual([1, 2]);
  });

  test('preserves global rank while filtering by company name or ID', () => {
    expect(rankCompanyCreditUsage(summaries, 'CPU', 'alpha')[0].Rank).toBe(2);
    expect(rankCompanyCreditUsage(summaries, 'CPU', '2')[0].Company).toBe('Beta');
  });

  test('filters by administrator label and company master fields', () => {
    const masterSearch = new Map([[2, 'Beta Legal 20123456789']]);
    const administrators = new Map([[2, 'Beatriz Admin admin']]);
    expect(rankCompanyCreditUsage(summaries, 'CPU', 'beatriz', new Map(), administrators)[0].CompanyID).toBe(2);
    expect(rankCompanyCreditUsage(summaries, 'CPU', '20123456789', masterSearch)[0].CompanyID).toBe(2);
  });

  test('uses safe percentages', () => {
    expect(usagePercent(3, 0)).toBe(0);
    expect(usagePercent(1, 4)).toBe(25);
  });

  test('fills the credit meter without ever overflowing its track', () => {
    expect(creditMeterFillPercent(2_000, 10_000)).toBe(20);
    // An allowance of zero is an unconfigured budget, not an empty bar out of a division by zero.
    expect(creditMeterFillPercent(500, 0)).toBe(0);
    // A remainder above its ceiling means the budget shrank after the credits were granted.
    expect(creditMeterFillPercent(12_000, 10_000)).toBe(100);
    expect(creditMeterFillPercent(-5, 10_000)).toBe(0);
  });
});

describe('company drill-down', () => {
  const companyRows = [
    companyDayRow(FIRST_DAY, 30, 5, [
      { RouteID: 1, CPU: 10, Inference: 2 },
      { RouteID: 2, CPU: 20, Inference: 3 },
    ]),
    companyDayRow(LAST_DAY, 4, 0),
  ];
  const userRows = [userRow(7, FIRST_DAY, 12, 2), userRow(9, FIRST_DAY, 18, 3)];

  test('builds the company series from the company collection', () => {
    const days = buildCompanyDays(companyRows);
    expect(days).toHaveLength(30);
    expect(days[0].CPU).toBe(30);
    expect(days[29].CPU).toBe(4);
  });

  test('ranks users by cost from the user collection', () => {
    const users = buildCompanyUserSummaries(userRows);
    expect(users.map((user) => user.UserID)).toEqual([9, 7]);
    expect(users[0].CPU).toBe(18);
    expect(users[0].Days).toHaveLength(30);
  });

  test('ranks a day breakdown by CPU and labels routes without a name', () => {
    const detail = buildCompanyDayDetail(companyRows, FIRST_DAY);
    expect(detail.CPU).toBe(30);
    expect(detail.Routes.map((route) => route.RouteID)).toEqual([2, 1]);
    expect(detail.Routes[0].Route).toBe('API.2');
  });

  test('returns an empty breakdown for a day with no company row', () => {
    const detail = buildCompanyDayDetail(companyRows, FIRST_DAY + 3);
    expect(detail.CPU).toBe(0);
    expect(detail.Routes).toEqual([]);
  });
});

describe('labels', () => {
  test('falls back through full name, login and id', () => {
    expect(companyUserDisplayName({ FirstName: ' Ada', LastName: 'Lovelace ' }, 7)).toBe('Ada Lovelace');
    expect(companyUserDisplayName({ User: 'ada' }, 7)).toBe('ada');
    expect(companyUserDisplayName(undefined, 7)).toBe('User #7');
  });

  test('splits API labels', () => {
    expect(splitCompanyCreditRoute('GET.sale-summary')).toEqual({ method: 'GET', path: 'sale-summary' });
    expect(splitCompanyCreditRoute('weird')).toEqual({ method: 'API', path: 'weird' });
  });
});
