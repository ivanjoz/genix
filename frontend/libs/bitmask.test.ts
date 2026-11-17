import { describe, expect, it } from 'vitest'
import { bitmaskAdd, bitmaskHas } from './bitmask.ts'

describe('bitmask', () => {
  it('stores option ID n as bit n-1', () => {
    const mask = bitmaskAdd(bitmaskAdd(0, 1), 3)
    expect(mask).toBe(5)
    expect(bitmaskHas(mask, 1)).toBe(true)
    expect(bitmaskHas(mask, 2)).toBe(false)
    expect(bitmaskHas(mask, 3)).toBe(true)
    expect(bitmaskAdd(mask, 3)).toBe(mask)
  })
})
