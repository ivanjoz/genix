import { describe, expect, it } from 'vitest'
import { clientWeekBarsByClient, isoWeekCode, lastIsoWeeks } from './client-week-sales.ts'

describe('client week sales bars', () => {
  it('codes ISO weeks as YYWW, the Thursday deciding the year', () => {
    expect(isoWeekCode(new Date(2026, 10, 9))).toBe(2646) // Monday
    expect(isoWeekCode(new Date(2026, 10, 15))).toBe(2646) // Sunday
    expect(isoWeekCode(new Date(2027, 0, 1))).toBe(2653) // a Friday still in 2026's last week
    expect(isoWeekCode(new Date(2026, 0, 1))).toBe(2601) // a Thursday opens 2026's first week
  })

  it('lists the last weeks up to the current one, across a year change', () => {
    expect(lastIsoWeeks(new Date(2027, 0, 13), 4)).toEqual([2652, 2653, 2701, 2702])
  })

  it('sums each week and keeps the unpaid part, empty weeks at zero', () => {
    const rows = [
      { ID: 1, Week: 2646, ClientID: 7, Purchases: [1000, 0, 500, 0, 0, 0, 0], Payments: [400, 0, 500, 0, 0, 0, 0], upd: 1 },
      { ID: 2, Week: 2600, ClientID: 7, Purchases: [9, 0, 0, 0, 0, 0, 0], Payments: [0, 0, 0, 0, 0, 0, 0], upd: 1 },
    ]
    const bars = clientWeekBarsByClient(rows, [2645, 2646])
    expect(bars.get(7)).toEqual([
      { week: 2645, purchases: 0, unpaid: 0 },
      { week: 2646, purchases: 1500, unpaid: 600 },
    ])
  })
})
