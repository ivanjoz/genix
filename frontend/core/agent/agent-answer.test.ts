import { describe, expect, it } from 'vitest'
import { formatAgentCell, inlineBarPercents, planAgentChart } from './agent-answer'

describe('agent answer', () => {
  it('formats cells by their column format', () => {
    expect(formatAgentCell('2026-09-28', 'date')).toBe('28/09/2026')
    expect(formatAgentCell(1234.5, 'money')).toBe('1,234.50')
    expect(formatAgentCell(1234, 'number')).toBe('1,234')
    expect(formatAgentCell(0.12345, 'number')).toBe('0.1235')
    expect(formatAgentCell(null, 'money')).toBe('—')
  })

  it('sizes inline bars against the largest value', () => {
    expect(inlineBarPercents({ Label: 'Monto', Format: 'money', Values: [50, -25, null] })).toEqual([100, 50, 0])
  })

  it('splits a chart into its scales, with lines on a long date axis and on y2', () => {
    const days = Array.from({ length: 40 }, (_, index) => `2026-09-${String((index % 28) + 1).padStart(2, '0')}`)
    const plans = planAgentChart({
      Columns: [
        { Label: 'Fecha', Axis: 'x', Format: 'date', Values: days },
        { Label: 'Monto', Axis: 'y', Format: 'money', Values: days.map(() => 10) },
        { Label: 'Cantidad', Axis: 'y2', Values: days.map(() => null) },
      ],
    })
    expect(plans.map((plan) => plan.type)).toEqual(['line', 'line'])
    expect(plans[0].labels[0]).toBe('01/09/2026')
    expect(plans[0].format).toBe('money')
    expect(plans[1].series[0].color).not.toBe(plans[0].series[0].color)
  })

  it('keeps a ranking of records as bars', () => {
    const [plan] = planAgentChart({
      Columns: [
        { Label: 'Producto', Axis: 'x', Values: ['Pan', 'Leche'] },
        { Label: 'Monto', Axis: 'y', Format: 'money', Values: [20, 10] },
      ],
    })
    expect(plan.type).toBe('bar')
    expect(plan.labels).toEqual(['Pan', 'Leche'])
  })
})
