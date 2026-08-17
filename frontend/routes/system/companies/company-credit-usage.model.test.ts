import { describe, expect, test } from 'bun:test';
import {
  normalizeCompanyCreditDetail,
  normalizeCompanyCreditReport,
  normalizeCompanyCreditUsers,
  rankCompanyCreditUsage,
  splitCompanyCreditRoute,
  usagePercent,
} from './company-credit-usage.model';

describe('company credit usage model', () => {
  const companies = normalizeCompanyCreditReport([
    { CompanyID: 2, Company: 'Beta', Status: 1, AdminName: 'Beatriz Admin', AdminUser: 'admin', CPU: 20, Inference: 2, TodayCPU: 3, TodayInference: 0, ActiveDays: 2, Days: [] },
    { CompanyID: 1, Company: 'Alpha', Status: 1, AdminName: 'Alex Owner', AdminUser: 'admin', CPU: 10, Inference: 9, TodayCPU: 0, TodayInference: 1, ActiveDays: 1, Days: [] },
  ]);

  test('ranks by the selected independent credit pool', () => {
    expect(rankCompanyCreditUsage(companies, 'CPU').map((company) => company.CompanyID)).toEqual([2, 1]);
    expect(rankCompanyCreditUsage(companies, 'Inference').map((company) => company.CompanyID)).toEqual([1, 2]);
  });

  test('preserves global rank while filtering by company name or ID', () => {
    expect(rankCompanyCreditUsage(companies, 'CPU', 'alpha')[0].Rank).toBe(2);
    expect(rankCompanyCreditUsage(companies, 'CPU', '2')[0].Company).toBe('Beta');
  });

  test('filters by administrator and company master fields', () => {
    const masterSearch = new Map([[2, 'Beta Legal 20123456789']]);
    expect(rankCompanyCreditUsage(companies, 'CPU', 'beatriz')[0].CompanyID).toBe(2);
    expect(rankCompanyCreditUsage(companies, 'CPU', '20123456789', masterSearch)[0].CompanyID).toBe(2);
  });

  test('uses safe metrics and zero percentages', () => {
    const normalized = normalizeCompanyCreditReport([{ ...companies[0], CPU: Number.NaN }]);
    expect(normalized[0].CPU).toBe(0);
    expect(usagePercent(3, 0)).toBe(0);
    expect(usagePercent(1, 4)).toBe(25);
  });

  test('normalizes route detail and splits API labels', () => {
    const detail = normalizeCompanyCreditDetail({
      CompanyID: 1, Day: 100, CPU: 2, Inference: 0,
      Routes: [{ RouteID: 9, Route: '', CPU: 2, Inference: Number.NaN }],
    });
    expect(detail.Routes[0].Route).toBe('API.UNKNOWN');
    expect(detail.Routes[0].Inference).toBe(0);
    expect(splitCompanyCreditRoute('GET.sale-summary')).toEqual({ method: 'GET', path: 'sale-summary' });
  });

  test('normalizes per-user daily usage', () => {
    const report = normalizeCompanyCreditUsers({
      CompanyID: 1, FirstDay: 100, LastDay: 129,
      Users: [{
        UserID: 7, Name: '', User: 'ada', CPU: Number.NaN, Inference: 2,
        Days: [{ Day: 100, CPU: 3, Inference: Number.NaN }],
      }],
    });
    expect(report.Users[0].Name).toBe('ada');
    expect(report.Users[0].CPU).toBe(0);
    expect(report.Users[0].Days[0].Inference).toBe(0);
  });
});
