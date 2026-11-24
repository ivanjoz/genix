// What a payment of a sale leaves owed, given its detracción. Pure: no Svelte, no fetch.
//
// Mirrors backend sales/types/sale_withholdings.go (ApplyDetractionDeposit), which is the authority:
// the open detracción is collected only into the Banco de la Nación detracciones account, and that
// account collects nothing else of the sale.

// TaxGroupDetraction (backend finance/types), the Type of a sale's detracción entry.
const IMPOSED_TAX_DETRACTION = 10

export interface ISaleImposedTax {
  Type: number
  Amount: number
  Line?: number
  Code?: number
  Rate?: number
  SettledAmount?: number
}

/** What of the sale's detracción the customer has not deposited yet. */
export const openDetractionOf = (imposedTaxes: ISaleImposedTax[] | undefined) =>
  (imposedTaxes || []).filter((imposedTax) => imposedTax.Type === IMPOSED_TAX_DETRACTION)
    .reduce((open, imposedTax) => open + imposedTax.Amount - (imposedTax.SettledAmount || 0), 0)

/**
 * The debt a full payment leaves: into the detracciones account it deposits the open detracción only;
 * into any other account it collects everything but the open detracción.
 */
export function remainingDebtAfterPayment(debtAmount: number, imposedTaxes: ISaleImposedTax[] | undefined, isDetractionsAccount: boolean): number {
  const openDetraction = openDetractionOf(imposedTaxes)
  return isDetractionsAccount ? debtAmount - openDetraction : Math.min(openDetraction, debtAmount)
}
