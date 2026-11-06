// Business rules of the HR settings page: which accounts a company concept may post to, and the
// treatment flags it may combine. The backend (hr/concepts.go validateConcept) enforces the same.

import { PCGE_ACCOUNTS, pcgeAccountLabel } from '#core/pcge.ts'
import { PLAME_CODE_OPTIONS } from '#services/hr/plame-codes.ts'
import { LEGAL_CONCEPTS, LEGAL_CONCEPT_MAX_ID, type ILegalConcept } from '#services/hr/hr-catalogs.ts'
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
  // Only an earning can be paid on a payslip of its own.
  if (concept.Kind === CONCEPT_KIND_DEDUCTION) concept = { ...concept, IsRemunerative: false, IncomeTax: 0, ExpenseAccountCode: 0, SeparatePayslip: false }
  else concept = { ...concept, PayableAccountCode: 0 }
  return concept
}

/** Legal concepts (IDs up to LEGAL_CONCEPT_MAX_ID) are fixed in code: they are shown, never edited. */
export const isLegalConceptID = (conceptID: number): boolean => conceptID > 0 && conceptID <= LEGAL_CONCEPT_MAX_ID

/** A legal concept in the company concept's shape, so both share one table. Its Name stays bilingual. */
const legalConceptAsConcept = (concept: ILegalConcept): IConcept => ({
  ID: concept.id, Name: concept.name, Kind: concept.kind, PlameCode: concept.plameCode || '',
  IsRemunerative: !!concept.isRemunerative, AffectsEssalud: !!concept.affectsEssalud, AffectsPension: !!concept.affectsPension,
  IncomeTax: concept.incomeTax || 0, ComputableForBenefits: !!concept.computableForBenefits,
  ExpenseAccountCode: concept.expenseAccountCode || 0, PayableAccountCode: concept.payableAccountCode || 0,
  SeparatePayslip: false, ss: 1, upd: 0,
})

/** Where a concept comes from: the legal ones fixed in code, or the company's own. */
export const ConceptOrigin = { Legal: 1, Company: 2 } as const

/** The concepts of the chosen origins (the toggle is required: at least one is always chosen). */
export const conceptsOfOrigins = (concepts: IConcept[], origins: number[]): IConcept[] =>
  concepts.filter((concept) => origins.includes(isLegalConceptID(concept.ID) ? ConceptOrigin.Legal : ConceptOrigin.Company))

/** Every concept of the company in one list, by ID: the legal ones (1–200), then the company's own (201+). */
export const allConcepts = (companyConcepts: IConcept[]): IConcept[] =>
  [...LEGAL_CONCEPTS.map(legalConceptAsConcept), ...companyConcepts].sort((left, right) => left.ID - right.ID)
