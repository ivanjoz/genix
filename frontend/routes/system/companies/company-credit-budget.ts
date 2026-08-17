import { GET, POST } from '$libs/ui-runtime.svelte';

export type CompanyCreditBudgetOperation = 'set-daily' | 'set-current' | 'increase-current';

export interface ICompanyCreditBudget {
  CompanyID: number;
  DailyCPU: number;
  DailyInference: number;
  BudgetMonthStartDay: number;
  MonthlyCPUCeiling: number;
  MonthlyInferenceCeiling: number;
  Updated: number;
  MonthCPUUsed: number;
  MonthInferenceUsed: number;
  CurrentCPU: number;
  CurrentInference: number;
  IsCurrentMonth: boolean;
}

export const getCompanyCreditBudget = (companyID: number): Promise<ICompanyCreditBudget> => {
  return GET({ route: `company-credit-budget?target-company-id=${companyID}` });
};

export const mutateCompanyCreditBudget = (
  companyID: number,
  operation: CompanyCreditBudgetOperation,
  cpu: number,
  inference: number,
): Promise<ICompanyCreditBudget> => {
  return POST({
    route: 'company-credit-budget',
    data: { CompanyID: companyID, Operation: operation, CPU: cpu, Inference: inference },
    silentError: true,
  });
};
