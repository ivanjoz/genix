import type {
  ICompanyCreditUsageDetail,
  ICompanyCreditUsageRoute,
  ICompanyCreditUsageSummary,
  ICompanyCreditUsageUsersReport,
} from './company-credit-usage';

export type CompanyCreditMetric = 'CPU' | 'Inference';

export interface ICompanyCreditUsageRanked extends ICompanyCreditUsageSummary {
  Rank: number;
}

const metricValue = (value: unknown): number => {
  const numericValue = Number(value);
  return Number.isFinite(numericValue) ? Math.max(0, numericValue) : 0;
};

// Normalize server metrics at the report boundary so tables and percentages never render NaN.
export const normalizeCompanyCreditReport = (
  companies: ICompanyCreditUsageSummary[] | undefined,
): ICompanyCreditUsageSummary[] => {
  return (companies || []).map((company) => ({
    ...company,
    CompanyID: metricValue(company.CompanyID),
    Company: company.Company || `Company #${company.CompanyID}`,
    Status: metricValue(company.Status),
    AdminName: String(company.AdminName || '').trim(),
    AdminUser: String(company.AdminUser || '').trim(),
    CPU: metricValue(company.CPU),
    Inference: metricValue(company.Inference),
    TodayCPU: metricValue(company.TodayCPU),
    TodayInference: metricValue(company.TodayInference),
    ActiveDays: metricValue(company.ActiveDays),
    Days: (company.Days || []).map((day) => ({
      Day: metricValue(day.Day), CPU: metricValue(day.CPU), Inference: metricValue(day.Inference),
    })),
  }));
};

export const rankCompanyCreditUsage = (
  companies: ICompanyCreditUsageSummary[],
  metric: CompanyCreditMetric,
  filterText = '',
  masterSearchTextByCompanyID: ReadonlyMap<number, string> = new Map(),
): ICompanyCreditUsageRanked[] => {
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
      `${company.CompanyID} ${company.Company} ${company.AdminName} ${company.AdminUser} ${masterSearchTextByCompanyID.get(company.CompanyID) || ''}`
        .toLowerCase().includes(normalizedFilter)
    );
};

export const usagePercent = (value: number, total: number): number => {
  const safeValue = metricValue(value);
  const safeTotal = metricValue(total);
  return safeTotal > 0 ? Math.round((safeValue / safeTotal) * 10_000) / 100 : 0;
};

export const normalizeCompanyCreditDetail = (
  detail: ICompanyCreditUsageDetail,
): ICompanyCreditUsageDetail => ({
  ...detail,
  CompanyID: metricValue(detail.CompanyID),
  Day: metricValue(detail.Day),
  CPU: metricValue(detail.CPU),
  Inference: metricValue(detail.Inference),
  Routes: (detail.Routes || []).map((route): ICompanyCreditUsageRoute => ({
    ...route,
    RouteID: metricValue(route.RouteID),
    Route: route.Route || 'API.UNKNOWN',
    CPU: metricValue(route.CPU),
    Inference: metricValue(route.Inference),
  })),
});

export const normalizeCompanyCreditUsers = (
  report: ICompanyCreditUsageUsersReport,
): ICompanyCreditUsageUsersReport => ({
  ...report,
  CompanyID: metricValue(report.CompanyID),
  FirstDay: metricValue(report.FirstDay),
  LastDay: metricValue(report.LastDay),
  Users: (report.Users || []).map((user) => ({
    ...user,
    UserID: metricValue(user.UserID),
    Name: String(user.Name || user.User || `User #${user.UserID}`).trim(),
    User: String(user.User || '').trim(),
    CPU: metricValue(user.CPU),
    Inference: metricValue(user.Inference),
    Days: (user.Days || []).map((day) => ({
      Day: metricValue(day.Day), CPU: metricValue(day.CPU), Inference: metricValue(day.Inference),
    })),
  })),
});

export const splitCompanyCreditRoute = (route: string) => {
  const separatorIndex = route.indexOf('.');
  if (separatorIndex <= 0) return { method: 'API', path: route || 'UNKNOWN' };
  return { method: route.slice(0, separatorIndex), path: route.slice(separatorIndex + 1) };
};
