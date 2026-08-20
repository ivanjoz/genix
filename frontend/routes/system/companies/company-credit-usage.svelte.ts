import { GetHandler } from '$libs/ui-runtime.svelte';
import {
  mergeCreditRouteNames,
  type ICompanyCreditBudgetMeter,
  type ICompanyCreditCompanyDay,
  type ICompanyCreditDay,
  type ICompanyCreditUserDay,
  type ICreditRouteName,
} from './company-credit-usage.model';

/** Every company's daily aggregate. One backend query per day requested, not per tenant. */
export class CompanyCreditReportService extends GetHandler {
  route = 'company-credit-usage-report';
  useCache = { min: 0.2, ver: 1 };
  keyID = 'ID';

  days: ICompanyCreditDay[] = $state([]);
  // The backend resends every budget meter on every response: the remaining figures move with each
  // charge, so a watermark that withheld them would leave a stale meter on the cards.
  budgetMeters: ICompanyCreditBudgetMeter[] = $state([]);

  handler(response: {
    Days?: ICompanyCreditDay[];
    Budgets?: ICompanyCreditBudgetMeter[];
    Routes?: ICreditRouteName[];
  }): void {
    mergeCreditRouteNames(response?.Routes);
    this.days = response?.Days || [];
    this.budgetMeters = response?.Budgets || [];
    console.debug(`[CompanyCreditReportService] merged days=${this.days.length}`
      + ` budgets=${this.budgetMeters.length}`
      + ` lastDay=${this.days.at(-1)?.Day ?? '-'} lastCPU=${this.days.at(-1)?.CPU ?? '-'}`);
  }

  constructor(init = false) {
    super();
    if (init) void this.fetch();
  }
}

/**
 * One company's thirty days for every user, in a single backend query. Delta snapshots are keyed by
 * route + query string, so each company gets its own collection and watermark for free.
 */
export class CompanyCreditUsageService extends GetHandler {
  route = '';
  useCache = { min: 0.2, ver: 1 };
  keyID = 'ID';

  companyDays: ICompanyCreditCompanyDay[] = $state([]);
  userDays: ICompanyCreditUserDay[] = $state([]);

  handler(response: {
    Company?: ICompanyCreditCompanyDay[];
    Users?: ICompanyCreditUserDay[];
    Routes?: ICreditRouteName[];
  }): void {
    mergeCreditRouteNames(response?.Routes);
    this.companyDays = response?.Company || [];
    this.userDays = response?.Users || [];
    console.debug(`[CompanyCreditUsageService] merged companyDays=${this.companyDays.length}`
      + ` userDays=${this.userDays.length}`);
  }

  constructor(companyID: number, init = false) {
    super();
    this.route = `company-credit-usage?target-company-id=${companyID}`;
    if (init) void this.fetch();
  }
}
