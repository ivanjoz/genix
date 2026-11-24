// The tax groups (backend finance/types TaxGroup*): a sale line's IGV treatment, and in the
// cash-documents frame how the tax comes out of a row's amount. 8 onward are taxes besides the IGV.
export const TAX_GROUP_TAXED_18 = 1
export const TAX_GROUP_TAXED_10 = 2
export const TAX_GROUP_EXONERATED = 3
export const TAX_GROUP_UNAFFECTED = 4
export const TAX_GROUP_EXPORT = 5
export const TAX_GROUP_NO_DOCUMENT = 6
/** Gratuito: a line given away. It moves no cash, but the seller still owes its IGV. */
export const TAX_GROUP_FREE = 7
/** A sale's detracción still to deposit into the Banco de la Nación account (part of its debt). */
export const TAX_GROUP_DETRACTION = 10
/** A sale's retención: withheld by the customer, a credit against the IGV. */
export const TAX_GROUP_RETENTION = 12

/** A cash-documents row of this group is a memo beside its document's total, never part of it. */
export const isMemoTaxGroup = (taxGroup: number) => taxGroup >= TAX_GROUP_FREE

/** The IGV inside a gross amount of a sold treatment; 0 for the groups that charge none. */
export const igvOfGross = (taxGroup: number, gross: number) =>
  taxGroup === TAX_GROUP_TAXED_18 ? gross * 18 / 118 : taxGroup === TAX_GROUP_TAXED_10 ? gross * 10 / 110 : 0

/** Any sold treatment but an export can be given away (SUNAT codes 11, 21, 37): the backend refuses the others. */
export const canBeGivenAway = (taxGroup: number) => taxGroup >= TAX_GROUP_TAXED_18 && taxGroup <= TAX_GROUP_UNAFFECTED
