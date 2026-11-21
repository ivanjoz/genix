// SUNAT due-date schedule rules for the System page. Pure: no Svelte, no fetch.

import type { ITaxDueCalendar } from '#services/accounting/tax-due-calendar.svelte.ts'

/** RUC last digits 0–9 plus the buenos contribuyentes / UESP column (TaxDueColumns in the backend). */
export const TAX_DUE_COLUMNS = 11

/** Column headers: RUC last digit 0–9, then the buenos contribuyentes / UESP column. */
export const TAX_DUE_COLUMN_LABELS = [
  ...Array.from({ length: 10 }, (_, digit) => `${digit}|${digit}`),
  'Good taxpayers|Buenos contrib.',
]

/** 2601 → "2026-01". */
export const formatTaxPeriod = (period: number): string =>
  `${2000 + Math.floor(period / 100)}-${String(period % 100).padStart(2, '0')}`

/** The period after the newest stored one, so "New period" opens on the next month to transcribe. */
export const nextTaxPeriod = (calendars: ITaxDueCalendar[]): number => {
  const newest = calendars.reduce((max, calendar) => Math.max(max, calendar.Period), 0)
  if (!newest) return 0
  return newest % 100 === 12 ? newest + 100 - 11 : newest + 1
}

/** What the form misses before it can be saved; the backend checks the dates against the period. */
export const taxCalendarFormProblem = (calendar: ITaxDueCalendar): string => {
  const month = calendar.Period % 100
  if (!calendar.Period || month < 1 || month > 12) return 'Enter a YYMM period|Ingrese un periodo AAMM'
  if (calendar.DueDays.length !== TAX_DUE_COLUMNS || calendar.DueDays.some((dueDay) => !dueDay)) {
    return 'Enter every due date|Ingrese todas las fechas de vencimiento'
  }
  return ''
}
