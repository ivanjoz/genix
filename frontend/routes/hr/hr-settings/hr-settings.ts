// Business rules of the HR settings page: which accounts a company concept may post to, and the
// treatment flags it may combine. The backend (hr/concepts.go validateConcept) enforces the same.

import { PCGE_ACCOUNTS, pcgeAccountLabel } from '#core/pcge.ts'
import { PLAME_CODE_OPTIONS } from '#services/hr/plame-codes.ts'
import type { IConcept } from '#services/hr/payroll.svelte.ts'

export const CONCEPT_KIND_EARNING = 1
export const CONCEPT_KIND_DEDUCTION = 2
const INCOME_TAX_ORDINARY = 1

const postableAccountsStartingWith = (prefixes: string[]) => PCGE_ACCOUNTS
  .filter((account) => account.postable && prefixes.some((prefix) => String(account.code).startsWith(prefix)))
  .map((account) => ({ code: account.code, label: pcgeAccountLabel(account.code) }))

/** An earning is a personnel expense (62). */
export const EARNING_ACCOUNT_OPTIONS = postableAccountsStartingWith(['62'])

/** A deduction is owed to someone: taxes and contributions (40), payroll payables (41), other payables (46), or recovers a receivable from the worker (14). */
export const DEDUCTION_ACCOUNT_OPTIONS = postableAccountsStartingWith(['40', '41', '46', '14'])

/** Table 22 codes a concept of this kind may report; the backend rejects any other chapter. */
export const plameCodeOptionsForKind = (kind: number) => PLAME_CODE_OPTIONS
  .filter((option) => option.kind === kind)
  .map((option) => ({ code: option.code, label: `${option.code} · ${option.name}` }))

export const makeEmptyConcept =(): IConcept => ({
  Kind: CONCEPT_KIND_EARNING, IsRemunerative: true, AffectsEssalud: true, AffectsPension: true,
  IncomeTax: INCOME_TAX_ORDINARY, ComputableForBenefits: false, ExpenseAccountCode: 622, ss: 1,
} as IConcept)

/** A non-remunerative earning pays no contributions and is never computable: clear what it can't carry. */
export const normalizeConcept = (concept: IConcept): IConcept => {
  if (concept.Kind === CONCEPT_KIND_DEDUCTION || !concept.IsRemunerative) {
    concept = { ...concept, AffectsEssalud: false, AffectsPension: false, ComputableForBenefits: false }
  }
  if (concept.Kind === CONCEPT_KIND_DEDUCTION) concept = { ...concept, IsRemunerative: false, IncomeTax: 0, ExpenseAccountCode: 0 }
  else concept = { ...concept, PayableAccountCode: 0 }
  return concept
}
