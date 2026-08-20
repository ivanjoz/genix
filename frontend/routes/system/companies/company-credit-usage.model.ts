import type { ICompany } from './empresas.svelte';

export type CompanyCreditMetric = 'CPU' | 'Inference';

// One (company, day) aggregate row exactly as the backend sends it. `upd` is the row's time frame,
// which is what lets the delta cache ask only for today plus any day that has since started.
export interface ICompanyCreditDay {
  ID: number;
  CompanyID: number;
  Day: number;
  CPU: number;
  Inference: number;
  upd: number;
  ss: number;
}

export interface ICompanyCreditRoute {
  RouteID: number;
  Route?: string;
  CPU: number;
  Inference: number;
}

// The company's own total for one day, with the per-route split. Separate collection from the user
// rows because they come from separate tables.
export interface ICompanyCreditCompanyDay {
  ID: number;
  Day: number;
  CPU: number;
  Inference: number;
  Routes?: ICompanyCreditRoute[];
  upd: number;
  ss: number;
}

// One user's total for one day. No route split: that lives on the company collection.
export interface ICompanyCreditUserDay {
  ID: number;
  UserID: number;
  Day: number;
  CPU: number;
  Inference: number;
  upd: number;
  ss: number;
}

export interface ICreditRouteName {
  ID: number;
  Route: string;
  upd?: number;
  ss?: number;
}

export interface ICompanyUserLabel {
  ID: number;
  User: string;
  FirstName: string;
  LastName: string;
}

export const COMPANY_CREDIT_WINDOW_DAYS = 30;
// User 1 is the company administrator: the login is fixed at sign-up and never editable.
export const COMPANY_ADMINISTRATOR_USER_ID = 1;
// The by-IDs endpoint packs the company into the requested id, because the versioned by-IDs path
// always resolves its partition from the caller's own token and this panel reads other tenants.
export const COMPANY_USER_LABEL_ID_FACTOR = 100_000;

export const packCompanyUserLabelID = (companyID: number, userID: number): number => {
  return companyID * COMPANY_USER_LABEL_ID_FACTOR + userID;
};

// The route catalog is an independent cold collection shared by both services: a client that
// already holds it never receives it again, so whichever service loads first fills it for both.
const routeNamesByID = new Map<number, string>();

export const mergeCreditRouteNames = (routes: ICreditRouteName[] | undefined): void => {
  for (const route of routes || []) {
    if (route.ID > 0 && route.Route) routeNamesByID.set(route.ID, route.Route);
  }
};

export const getCreditRouteName = (routeID: number): string => {
  return routeNamesByID.get(routeID) || `API.${routeID || 'UNKNOWN'}`;
};

export interface ICreditUsageDay {
  Day: number;
  CPU: number;
  Inference: number;
}

// One company's entitlement and what is left of it, as the report sends it. Remaining rather than
// used, because that is what the limiter answers with when it refuses a charge, and it arrives
// already computed: recomputing it here would be a second implementation of the same subtraction.
export interface ICompanyCreditBudgetMeter {
  ID: number;
  DailyCPU: number;
  DailyInference: number;
  DailyRemainingCPU: number;
  DailyRemainingInference: number;
  MonthlyCPUCeiling: number;
  MonthlyInferenceCeiling: number;
  RemainingCPU: number;
  RemainingInference: number;
  ExtraCPU: number;
  DayExtraCPUUsed: number;
  ExtraRemainingCPU: number;
  IsCurrentMonth: boolean;
  upd: number;
  ss: number;
}

export interface ICompanyCreditSummary {
  CompanyID: number;
  Company: string;
  Status: number;
  CPU: number;
  Inference: number;
  TodayCPU: number;
  TodayInference: number;
  ActiveDays: number;
  Days: ICreditUsageDay[];
}

export interface ICompanyCreditSummaryRanked extends ICompanyCreditSummary {
  Rank: number;
}

export interface ICompanyCreditUserSummary {
  UserID: number;
  CPU: number;
  Inference: number;
  Days: ICreditUsageDay[];
}

const metricValue = (value: unknown): number => {
  const numericValue = Number(value);
  return Number.isFinite(numericValue) ? Math.max(0, numericValue) : 0;
};

const MILLISECONDS_PER_DAY = 86_400_000;

// The window is anchored on the newest day the rows actually carry, so a client whose clock is
// wrong still lines its columns up with the server's data. With no rows at all there is nothing to
// anchor on and today is the only sensible answer — that is the case of a company, or a whole
// platform, that has not spent a credit yet.
const windowLastDay = (days: { Day: number }[]): number => {
  const latestDay = days.reduce((latest, day) => Math.max(latest, metricValue(day.Day)), 0);
  return latestDay || Math.floor(Date.now() / MILLISECONDS_PER_DAY);
};

// Zero-fill the fixed range so chart columns never shift when a day has no usage. Rows outside the
// window are ignored: the delta cache keeps everything it was ever sent, including expired days.
const buildDaySeries = (
  rows: { Day: number; CPU: number; Inference: number }[],
  lastDay: number,
): { days: ICreditUsageDay[]; cpu: number; inference: number; activeDays: number } => {
  const firstDay = lastDay - COMPANY_CREDIT_WINDOW_DAYS + 1;
  const days: ICreditUsageDay[] = Array.from({ length: COMPANY_CREDIT_WINDOW_DAYS }, (_, offset) => ({
    Day: firstDay + offset, CPU: 0, Inference: 0,
  }));

  let cpu = 0;
  let inference = 0;
  let activeDays = 0;
  for (const row of rows) {
    const offset = metricValue(row.Day) - firstDay;
    if (offset < 0 || offset >= COMPANY_CREDIT_WINDOW_DAYS) continue;
    const rowCPU = metricValue(row.CPU);
    const rowInference = metricValue(row.Inference);
    days[offset].CPU = rowCPU;
    days[offset].Inference = rowInference;
    cpu += rowCPU;
    inference += rowInference;
    if (rowCPU > 0 || rowInference > 0) activeDays++;
  }
  return { days, cpu, inference, activeDays };
};

/** Group the platform's daily rows into one summary per company, named from the company catalog. */
export const buildCompanyCreditSummaries = (
  rows: ICompanyCreditDay[] | undefined,
  companies: ICompany[] | undefined,
): ICompanyCreditSummary[] => {
  const sourceRows = rows || [];
  const lastDay = windowLastDay(sourceRows);

  const rowsByCompany = new Map<number, ICompanyCreditDay[]>();
  for (const row of sourceRows) {
    const companyID = metricValue(row.CompanyID);
    if (!companyID) continue;
    const companyRows = rowsByCompany.get(companyID);
    if (companyRows) companyRows.push(row);
    else rowsByCompany.set(companyID, [row]);
  }

  // The catalog drives the list, not the usage rows: a company that has never spent a credit is
  // still a company and must keep its card, showing an empty series.
  const summaries: ICompanyCreditSummary[] = [];
  for (const company of companies || []) {
    if (!company.id) continue;
    const { days, cpu, inference, activeDays } = buildDaySeries(rowsByCompany.get(company.id) || [], lastDay);
    rowsByCompany.delete(company.id);
    summaries.push({
      CompanyID: company.id,
      Company: company.Name || `Company #${company.id}`,
      Status: metricValue(company.ss),
      CPU: cpu,
      Inference: inference,
      TodayCPU: days[days.length - 1].CPU,
      TodayInference: days[days.length - 1].Inference,
      ActiveDays: activeDays,
      Days: days,
    });
  }
  // Usage for a company the catalog has not delivered yet still has to be visible, or credits would
  // silently go unaccounted for while the two caches disagree.
  for (const [companyID, companyRows] of rowsByCompany) {
    const { days, cpu, inference, activeDays } = buildDaySeries(companyRows, lastDay);
    summaries.push({
      CompanyID: companyID, Company: `Company #${companyID}`, Status: 0,
      CPU: cpu, Inference: inference, TodayCPU: days[days.length - 1].CPU,
      TodayInference: days[days.length - 1].Inference, ActiveDays: activeDays, Days: days,
    });
  }
  return summaries;
};

/** Group one company's rows per user. */
export const buildCompanyUserSummaries = (
  rows: ICompanyCreditUserDay[] | undefined,
): ICompanyCreditUserSummary[] => {
  const sourceRows = rows || [];
  const lastDay = windowLastDay(sourceRows);

  const rowsByUser = new Map<number, ICompanyCreditUserDay[]>();
  for (const row of sourceRows) {
    const userRows = rowsByUser.get(row.UserID);
    if (userRows) userRows.push(row);
    else rowsByUser.set(row.UserID, [row]);
  }

  return [...rowsByUser]
    .map(([userID, userRows]) => {
      const { days, cpu, inference } = buildDaySeries(userRows, lastDay);
      return { UserID: userID, CPU: cpu, Inference: inference, Days: days };
    })
    .sort((left, right) =>
      right.CPU - left.CPU || right.Inference - left.Inference || left.UserID - right.UserID
    );
};

/** The company's own thirty-day series. */
export const buildCompanyDays = (
  rows: ICompanyCreditCompanyDay[] | undefined,
): ICreditUsageDay[] => {
  return buildDaySeries(rows || [], windowLastDay(rows || [])).days;
};

export interface ICompanyCreditDayDetail {
  Day: number;
  CPU: number;
  Inference: number;
  Routes: Required<ICompanyCreditRoute>[];
}

/**
 * The route breakdown of one day, ranked by what it cost. The route number breaks ties only to keep
 * the order stable, so a list that reshuffles never looks like data changing when nothing did.
 */
export const buildCompanyDayDetail = (
  rows: ICompanyCreditCompanyDay[] | undefined,
  day: number,
): ICompanyCreditDayDetail => {
  const companyRow = (rows || []).find((row) => row.Day === day);
  const routes = (companyRow?.Routes || [])
    .map((route) => ({
      RouteID: metricValue(route.RouteID),
      Route: route.Route || getCreditRouteName(route.RouteID),
      CPU: metricValue(route.CPU),
      Inference: metricValue(route.Inference),
    }))
    .sort((left, right) =>
      right.CPU - left.CPU || right.Inference - left.Inference || left.RouteID - right.RouteID
    );

  return {
    Day: day,
    CPU: metricValue(companyRow?.CPU),
    Inference: metricValue(companyRow?.Inference),
    Routes: routes,
  };
};

export const rankCompanyCreditUsage = (
  companies: ICompanyCreditSummary[],
  metric: CompanyCreditMetric,
  filterText = '',
  masterSearchTextByCompanyID: ReadonlyMap<number, string> = new Map(),
  administratorTextByCompanyID: ReadonlyMap<number, string> = new Map(),
): ICompanyCreditSummaryRanked[] => {
  const secondaryMetric: CompanyCreditMetric = metric === 'CPU' ? 'Inference' : 'CPU';
  const normalizedFilter = filterText.trim().toLowerCase();
  return [...companies]
    .sort((left, right) =>
      right[metric] - left[metric] ||
      right[secondaryMetric] - left[secondaryMetric] ||
      left.CompanyID - right.CompanyID
    )
    .map((company, rankIndex) => ({ ...company, Rank: rankIndex + 1 }))
    .filter((company) => !normalizedFilter ||
      `${company.CompanyID} ${company.Company} ${administratorTextByCompanyID.get(company.CompanyID) || ''} ${masterSearchTextByCompanyID.get(company.CompanyID) || ''}`
        .toLowerCase().includes(normalizedFilter)
    );
};

/**
 * Bar fill for a meter, as a percentage of the allowance it is measured against. An allowance of
 * zero has nothing to be a fraction of, and a remainder above its ceiling can only come from a
 * budget that shrank after the credits were granted: both clamp instead of overflowing the track.
 */
export const creditMeterFillPercent = (remaining: number, total: number): number => {
  const safeRemaining = metricValue(remaining);
  const safeTotal = metricValue(total);
  if (safeTotal <= 0) return 0;
  return Math.min(100, Math.round((safeRemaining / safeTotal) * 10_000) / 100);
};

export const usagePercent = (value: number, total: number): number => {
  const safeValue = metricValue(value);
  const safeTotal = metricValue(total);
  return safeTotal > 0 ? Math.round((safeValue / safeTotal) * 10_000) / 100 : 0;
};

export const splitCompanyCreditRoute = (route: string) => {
  const separatorIndex = route.indexOf('.');
  if (separatorIndex <= 0) return { method: 'API', path: route || 'UNKNOWN' };
  return { method: route.slice(0, separatorIndex), path: route.slice(separatorIndex + 1) };
};

/** Display label for a resolved user record, falling back through the chain the backend used to. */
export const companyUserDisplayName = (
  label: { User?: string; FirstName?: string; LastName?: string } | undefined,
  userID: number,
): string => {
  const fullName = `${label?.FirstName || ''} ${label?.LastName || ''}`.trim();
  return fullName || label?.User?.trim() || `User #${userID}`;
};
