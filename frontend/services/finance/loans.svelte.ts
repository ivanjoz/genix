import { GetHandler } from '#libs/ui-runtime.svelte.ts'

// Mirrors backend/finance/types/loans.go. Loans the company owes, with the lender's installment
// schedule. Read by the loans page and, later, by the cash flow forecast.

export interface ILoanInstallment {
  Number: number
  /** UnixDay. */
  DueDate: number
  /** Cents of the loan's currency. */
  Principal: number
  Interest: number
  Fees: number
  /** Server-authoritative: recomputed from the loan's payments. */
  PaidAmount: number
}

export interface ILoan {
  ID: number
  Name: string
  /** A provider in client_provider. */
  LenderID: number
  CashBankID: number
  CurrencyType: number
  /** Cents. */
  Principal: number
  DisbursementDate: number
  /** TEA in percent × 10000 (12.5 % → 125000); 0 when not entered. */
  AnnualRate: number
  Method: number
  GraceMonths: number
  Installments: ILoanInstallment[]
  Notes: string
  IsDisbursementRegistered: boolean
  /** Request-only: on creation, also write the disbursement into CashBankID. */
  RegisterDisbursement?: boolean
  ss: number
  upd: number
}

export interface ILoanPayment {
  LoanID: number
  InstallmentNumber: number
  CashBankID: number
  /** Positive, cents. */
  Amount: number
  /** UnixDay. */
  Date: number
}

/** Active and paid-off loans (`loans`). */
export class LoansService extends GetHandler<ILoan> {
  route = 'loans'
  useCache = { min: 5, ver: 1 }

  loans: ILoan[] = $state([])

  handler(result: ILoan[]): void {
    this.loans = (result || [])
      .filter((loan) => loan.ss > 0)
      .map((loan) => ({ ...loan, Installments: loan.Installments || [] }))
      .sort((left, right) => left.ss - right.ss || right.DisbursementDate - left.DisbursementDate)
  }

  constructor(init = false) {
    super()
    if (init) this.fetch()
  }
}

// Both writes bring back the delta of `loans` (GetHandler.writeAndSync); a payment also moves a
// cash account, whose page syncs on its own.
export const postLoan = (service: LoansService, loan: ILoan): Promise<ILoan> =>
  service.writeAndSync('loans', loan)

export const postLoanPayment = (service: LoansService, payment: ILoanPayment): Promise<ILoan> =>
  service.writeAndSync('loan-payment', payment)
