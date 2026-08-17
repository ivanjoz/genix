import { GET } from '$libs/ui-runtime.svelte';

export interface ICompanyCreditUsageDay {
  Day: number;
  CPU: number;
  Inference: number;
}

export interface ICompanyCreditUsageSummary {
  CompanyID: number;
  Company: string;
  Status: number;
  AdminName: string;
  AdminUser: string;
  CPU: number;
  Inference: number;
  TodayCPU: number;
  TodayInference: number;
  ActiveDays: number;
  Days: ICompanyCreditUsageDay[];
}

export interface ICompanyCreditUsageReport {
  FirstDay: number;
  LastDay: number;
  GeneratedAt: number;
  Companies: ICompanyCreditUsageSummary[];
}

export interface ICompanyCreditUsageRoute {
  RouteID: number;
  Route: string;
  CPU: number;
  Inference: number;
}

export interface ICompanyCreditUsageDetail {
  CompanyID: number;
  Day: number;
  CPU: number;
  Inference: number;
  Routes: ICompanyCreditUsageRoute[];
}

export interface ICompanyCreditUsageUser {
  UserID: number;
  Name: string;
  User: string;
  CPU: number;
  Inference: number;
  Days: ICompanyCreditUsageDay[];
}

export interface ICompanyCreditUsageUsersReport {
  CompanyID: number;
  FirstDay: number;
  LastDay: number;
  Users: ICompanyCreditUsageUser[];
}

// Report snapshots are mutable for today, so explicit refreshes always reach the backend.
export const getCompanyCreditUsageReport = (): Promise<ICompanyCreditUsageReport> => {
  return GET({ route: 'company-credit-usage-report' });
};

export const getCompanyCreditUsageDetail = (
  companyID: number,
  day: number,
): Promise<ICompanyCreditUsageDetail> => {
  return GET({ route: `company-credit-usage-detail?target-company-id=${companyID}&day=${day}` });
};

export const getCompanyCreditUsageUsers = (companyID: number): Promise<ICompanyCreditUsageUsersReport> => {
  return GET({ route: `company-credit-usage-users?target-company-id=${companyID}` });
};
