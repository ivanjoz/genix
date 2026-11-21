// The company's tax setup (regime, renta prepayment, buen contribuyente, NRUS category). Pure: no
// Svelte, no fetch. Mirrors the TaxRegime* / NrusCategory* constants of config/types/empresas.go.

export const TAX_REGIME_GENERAL = 1
export const TAX_REGIME_MYPE = 2
export const TAX_REGIME_SPECIAL = 3
export const TAX_REGIME_NRUS = 4

export const TAX_REGIME_OPTIONS = [
  { id: TAX_REGIME_GENERAL, name: 'General regime|Régimen General' },
  { id: TAX_REGIME_MYPE, name: 'MYPE tax regime|Régimen MYPE Tributario' },
  { id: TAX_REGIME_SPECIAL, name: 'Special income regime|Régimen Especial de Renta' },
  { id: TAX_REGIME_NRUS, name: 'Nuevo RUS|Nuevo RUS' },
]

export const NRUS_CATEGORY_OPTIONS = [
  { id: 1, name: 'Category 1 (S/ 20 per month)|Categoría 1 (S/ 20 al mes)' },
  { id: 2, name: 'Category 2 (S/ 50 per month)|Categoría 2 (S/ 50 al mes)' },
]

/**
 * The renta rate a regime starts from, in basis points: 1.5 % for the General regime and the RER,
 * 1 % for the MYPE regime (its rate while yearly net income stays within 300 UIT), none for NRUS,
 * whose fixed cuota replaces renta. The company edits it when it uses the coefficient method.
 */
export const defaultIncomeTaxPrepaymentRate = (taxRegime: number): number => {
  if (taxRegime === TAX_REGIME_MYPE) return 100
  if (taxRegime === TAX_REGIME_GENERAL || taxRegime === TAX_REGIME_SPECIAL) return 150
  return 0
}
