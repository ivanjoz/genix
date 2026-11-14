// What a worker's payroll month really cost the company (docs/MANUFACTURING_PLAN.md §9.3): the
// earnings, the employer contributions and the month's own provision lines (gratificación, CTS,
// vacation), minus the part of a benefit paid in the month that was charged to provisions already
// accrued, so nothing is counted twice. The "Horas vs planilla" report and the monthly labor
// adjustment of the accounting both read it, so the page and the entry always agree.

// hr ConceptKind values (backend/hr/types/legal_concepts.go).
const CONCEPT_KIND_EARNING = 1
const CONCEPT_KIND_EMPLOYER = 3
const CONCEPT_KIND_PROVISION = 4

export interface IPayrollCostLine {
  Kind: number
  Amount: number
  ProvisionedAmount?: number
}

export const payrollActualCost = (lines: IPayrollCostLine[] | undefined): number => {
  let actualCost = 0
  for (const line of lines || []) {
    if (line.Kind === CONCEPT_KIND_EARNING || line.Kind === CONCEPT_KIND_EMPLOYER || line.Kind === CONCEPT_KIND_PROVISION) {
      actualCost += line.Amount || 0
    }
    actualCost -= line.ProvisionedAmount || 0
  }
  return actualCost
}
