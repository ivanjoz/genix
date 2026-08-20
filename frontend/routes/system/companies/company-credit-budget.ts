import { GET, POST } from '$libs/ui-runtime.svelte';

export type CompanyCreditBudgetOperation = 'set-daily' | 'set-current' | 'increase-current';

export interface ICompanyCreditBudget {
  CompanyID: number;
  DailyCPU: number;
  DailyInference: number;
  BudgetMonthStartDay: number;
  MonthlyCPUCeiling: number;
  MonthlyInferenceCeiling: number;
  // The figure the last "set current" wrote. Consumed-since-that-grant is LastSetCPU - CurrentCPU.
  LastSetCPU: number;
  LastSetInference: number;
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

export interface ICompanyCreditBudgetOperation {
  Operation: CompanyCreditBudgetOperation;
  CPU: number;
  Inference: number;
}

// The panel saves every edited row in one request; the backend orders the operations itself, so the
// caller only has to send the ones that changed.
export const mutateCompanyCreditBudget = (
  companyID: number,
  operations: ICompanyCreditBudgetOperation[],
): Promise<ICompanyCreditBudget> => {
  return POST({
    route: 'company-credit-budget',
    data: { CompanyID: companyID, Operations: operations },
    silentError: true,
  });
};
