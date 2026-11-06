// Mirrors backend/hr/types/employee_records.go: one table for everything that happened to an
// employee on a date or in a month. ConceptID says which (a legal concept, LegalConcept in
// hr-catalogs.ts, or a company concept); each reads only the columns its backend comment names. Money is cents, dates UnixDays (both included), periods YYMM. No Svelte
// here, so pure business logic can import it; the services are in employee-records.svelte.ts.

export interface IEmployeeRecord {
  ID: number
  EmployeeID: number
  /** What the record is: a legal concept (up to LEGAL_CONCEPT_MAX_ID) or a company concept. */
  ConceptID: number
  /** The month the payroll applies it to; derived by the server for time off and advances. */
  Period: number
  /** 2000 + Period / 100: the year list the record is cached in. */
  Year: number
  /** The day it happened (Registros), paid (advance) or granted (loan). */
  Date: number
  FromDate: number
  ToDate: number
  /** Days, hours, or a loan's installments; a time off's calendar days (derived). */
  Quantity: number
  Amount: number
  /** The cash or bank account an advance or a loan left from. */
  CashBankID: number
  /** The loan of an installment. */
  LoanID: number
  /** The closed payslip that settled it. */
  PayslipID: number
  Note: string
  ApprovedBy: number
  ss: number
  upd: number
}

/** One numbering for every concept. */
export const RecordStatus = { Cancelled: 0, Pending: 1, Active: 2, Settled: 3 } as const

/** A record the payroll reads: in force, or already settled by a closed payslip. */
export const isLiveRecord = (record: IEmployeeRecord) => record.ss === RecordStatus.Active || record.ss === RecordStatus.Settled

export const yearOfPeriod = (period: number) => 2000 + Math.floor(period / 100)
