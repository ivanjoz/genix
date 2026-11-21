import { describe, expect, it } from 'vitest'
import type { ITaxDueCalendar } from '#services/accounting/tax-due-calendar.svelte.ts'
import { formatTaxPeriod, nextTaxPeriod, TAX_DUE_COLUMN_LABELS, taxCalendarFormProblem } from './tax-calendar.ts'

const calendarOf = (period: number, dueDays: number[]) => ({ Period: period, DueDays: dueDays, ss: 1, upd: 0 }) as ITaxDueCalendar

describe('tax calendar', () => {
  it('has one column per RUC digit plus the buenos contribuyentes one', () => {
    expect(TAX_DUE_COLUMN_LABELS).toHaveLength(11)
  })

  it('formats a YYMM period', () => {
    expect(formatTaxPeriod(2601)).toBe('2026-01')
    expect(formatTaxPeriod(2612)).toBe('2026-12')
  })

  it('opens a new period on the month after the newest one, crossing the year', () => {
    expect(nextTaxPeriod([])).toBe(0)
    expect(nextTaxPeriod([calendarOf(2610, []), calendarOf(2611, [])])).toBe(2612)
    expect(nextTaxPeriod([calendarOf(2612, [])])).toBe(2701)
  })

  it('refuses a form with a bad period or a missing date', () => {
    const complete = Array(11).fill(20834)
    expect(taxCalendarFormProblem(calendarOf(2701, complete))).toBe('')
    expect(taxCalendarFormProblem(calendarOf(2713, complete))).not.toBe('')
    expect(taxCalendarFormProblem(calendarOf(2701, [...complete.slice(0, 10), 0]))).not.toBe('')
  })
})
