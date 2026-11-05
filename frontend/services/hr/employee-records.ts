// Mirrors backend/hr/types/employee_records.go: one table for everything that happened to an
// employee on a date or in a month. Type says which; each type reads only the columns its comment
// in the backend names. Money is cents, dates UnixDays (both included), periods YYMM. No Svelte
// here, so pure business logic can import it; the services are in employee-records.svelte.ts.

export interface IEmployeeRecord {
  ID: number
  EmployeeID: number
  Type: number
  /** The month the payroll applies it to; derived by the server for time off and advances. */
  Period: number
  /** 2000 + Period / 100: the year list the record is cached in. */
  Year: number
  /** Paid (advance) or granted (loan). */
  Date: number
  FromDate: number
  ToDate: number
  /** Days, hours, or a loan's installments; a time off's calendar days (derived). */
  Quantity: number
  Amount: number
  ConceptID: number
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

/** Mirrors the backend Record* constants (the catalogs in hr-catalogs.ts carry their names). */
export const RecordType = {
  AbsenceDays: 1, Overtime25: 2, Overtime35: 3, ConceptAmount: 4, ExtraordinaryPayment: 5,
  AdvanceWeekly: 10, AdvanceBiweekly: 11, AdvanceOther: 12,
  Loan: 20, LoanInstallment: 21,
  Vacation: 30, MedicalLeavePaid: 31, MedicalLeaveSubsidized: 32, MaternityLeave: 33, UnpaidLeave: 34,
} as const

/** One numbering for every type. */
export const RecordStatus = { Cancelled: 0, Pending: 1, Active: 2, Settled: 3 } as const

/** A record the payroll reads: in force, or already settled by a closed payslip. */
export const isLiveRecord = (record: IEmployeeRecord) => record.ss === RecordStatus.Active || record.ss === RecordStatus.Settled

export const yearOfPeriod = (period: number) => 2000 + Math.floor(period / 100)
