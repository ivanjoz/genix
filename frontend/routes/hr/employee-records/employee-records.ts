// Pure business logic of the Employee Records page: which of an employee's records are Registros or
// loans, what each concept asks for, and which records can still be edited. No Svelte, no fetch.

import { formatPeriodMonth, periodOfDay } from '#routes/hr/payroll/payroll.ts'
import { RecordStatus, isLiveRecord, type IEmployeeRecord } from '#services/hr/employee-records.ts'
import { ConceptGroup, LEGAL_CONCEPT_MAX_ID, LegalConcept, legalConceptsOfGroup } from '#services/hr/hr-catalogs.ts'
import type { IConcept } from '#services/hr/payroll.svelte.ts'

/** The legal concepts registered by hand on Registros: overtime 25 % / 35 % and absences. */
const ENTRY_LEGAL_CONCEPT_IDS = new Set(legalConceptsOfGroup(ConceptGroup.Entry).map((concept) => concept.id))

/** A Registros record: a company concept, or a legal one registered by hand. */
export const isEntryConcept = (conceptID: number): boolean =>
  conceptID > LEGAL_CONCEPT_MAX_ID || ENTRY_LEGAL_CONCEPT_IDS.has(conceptID)

/** The concepts a Registro can be: the hand-registered legal ones first, then the company's. Legal
 * names are bilingual: the caller translates them. */
export const entryConceptOptions = (companyConcepts: IConcept[]): { ID: number, Name: string }[] => [
  ...legalConceptsOfGroup(ConceptGroup.Entry).map((concept) => ({ ID: concept.id, Name: concept.name })),
  ...companyConcepts.map((concept) => ({ ID: concept.ID, Name: concept.Name })),
]

/** What a Registro of the concept asks for: absence days, overtime hours, or a company concept's amount. */
export const entryInput = (conceptID: number): 'days' | 'hours' | 'amount' | undefined => {
  if (conceptID === LegalConcept.Absence) return 'days'
  if (conceptID === LegalConcept.Overtime25 || conceptID === LegalConcept.Overtime35) return 'hours'
  if (conceptID > LEGAL_CONCEPT_MAX_ID) return 'amount'
  return undefined
}

/** The employee's live Registros, newest payroll month first, then newest date. */
export const employeeEntries = (records: IEmployeeRecord[]): IEmployeeRecord[] => records
  .filter((record) => isEntryConcept(record.ConceptID) && isLiveRecord(record))
  .sort((left, right) => right.Period - left.Period || right.Date - left.Date || right.ID - left.ID)

/** The payroll months a Registro can be paid in: the twelve of its date's year (the server enforces
 * the same year), so a late-reported one can go to a later month. */
export const entryMonthOptions = (entryDate: number): { id: number, name: string }[] => {
  const yearPrefix = Math.floor(periodOfDay(entryDate) / 100) * 100
  return Array.from({ length: 12 }, (_, monthIndex) => {
    const period = yearPrefix + monthIndex + 1
    return { id: period, name: formatPeriodMonth(period) }
  })
}

/** A new Registro of the employee dated today, paid in today's month (the user can change both). */
export const makeEmptyEntry = (employeeID: number, todayUnixDay: number): IEmployeeRecord => ({
  EmployeeID: employeeID, Date: todayUnixDay, Period: periodOfDay(todayUnixDay), ss: RecordStatus.Active,
} as IEmployeeRecord)

/** Settled records were used by a closed payslip: read-only until that payroll is reopened. */
export const isEditableRecord = (record: IEmployeeRecord): boolean => record.ss === RecordStatus.Active

/** The employee's live loans. */
export const employeeLoans = (records: IEmployeeRecord[]): IEmployeeRecord[] =>
  records.filter((record) => record.ConceptID === LegalConcept.Loan && isLiveRecord(record))

/** The employee's live loan installments, oldest month first. */
export const employeeLoanInstallments = (records: IEmployeeRecord[]): IEmployeeRecord[] => records
  .filter((record) => record.ConceptID === LegalConcept.LoanInstallment && isLiveRecord(record))
  .sort((left, right) => left.Period - right.Period)
