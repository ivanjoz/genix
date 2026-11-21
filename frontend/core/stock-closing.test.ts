import { describe, expect, it } from 'vitest'
import { closingStockSeries, isStockNewerThanFrame } from './stock-closing.ts'

const SITE_A = 1
const SITE_B = 2
const siteIDByWarehouseID = new Map([[10, SITE_A], [11, SITE_A], [20, SITE_B]])

describe('closingStockSeries', () => {
	it('rebuilds each day backwards from the current stock of the site', () => {
		const series = closingStockSeries(
			[
				// Site A holds 7 units of product 5 across two warehouses, one of them in lots.
				{ WarehouseID: 10, ProductID: 5, Quantity: 3, SubQuantity: 0 },
				{ WarehouseID: 11, ProductID: 5, Quantity: 0, SubQuantity: 0, DetailQuantity: 4, DetailSubQuantity: 0 },
			],
			siteIDByWarehouseID,
			[
				// Day 100: 10 units in, 2 out. Day 101: 4 out.
				{ keys: [100, SITE_A], snapshot: 1, slot: 1, rows: [[5], [1]], sums: [[10], [0], [-2], [0]] },
				{ keys: [101, SITE_A], snapshot: 1, slot: 1, rows: [[5], [1]], sums: [[0], [0], [-4], [0]] },
			],
			99, 102,
		)
		expect(series).toHaveLength(1)
		expect(series[0].siteID).toBe(SITE_A)
		expect([...series[0].closing]).toEqual([3, 11, 7, 7])
		expect([...series[0].inflow]).toEqual([0, 10, 0, 0])
		expect([...series[0].outflow]).toEqual([0, 2, 4, 0])
	})

	it('converts the sub-units of each row with its own divisor', () => {
		const series = closingStockSeries(
			// Today: 2 units and 6 twelfths, after a refinement from sixths to twelfths.
			[{ WarehouseID: 20, ProductID: 8, Quantity: 2, SubQuantity: 6, SubDivisor: 12 }],
			siteIDByWarehouseID,
			// Yesterday 3 sixths left, recorded before the refinement.
			[{ keys: [200, SITE_B], snapshot: 1, slot: 1, rows: [[8], [6]], sums: [[0], [0], [0], [-3]] }],
			199, 200,
		)
		expect([...series[0].closing]).toEqual([3, 2.5])
		expect([...series[0].outflow]).toEqual([0, 0.5])
	})

	it('keeps a product that moved and has no stock left, and refuses a warehouse without a site', () => {
		const series = closingStockSeries([], siteIDByWarehouseID,
			[{ keys: [300, SITE_B], snapshot: 1, slot: 1, rows: [[9], [1]], sums: [[0], [0], [-5], [0]] }], 299, 300)
		expect([...series[0].closing]).toEqual([5, 0])

		expect(() => closingStockSeries([{ WarehouseID: 99, ProductID: 1, Quantity: 1, SubQuantity: 0 }],
			siteIDByWarehouseID, [], 1, 1)).toThrow('has no site')
	})
})

describe('isStockNewerThanFrame', () => {
	it('asks for the frame again only when a stock row is newer than its read', () => {
		const files = [{ keys: [100, SITE_A], snapshot: 500, slot: 1, rows: [[5], [1]], sums: [[1], [0], [0], [0]] }]
		expect(isStockNewerThanFrame([{ WarehouseID: 10, ProductID: 5, Quantity: 1, SubQuantity: 0, upd: 500 }], 400, files)).toBe(false)
		expect(isStockNewerThanFrame([{ WarehouseID: 10, ProductID: 5, Quantity: 1, SubQuantity: 0, upd: 501 }], 400, files)).toBe(true)
	})
})
