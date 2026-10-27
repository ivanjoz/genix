import { describe, expect, it } from 'vitest'
import { saleSummaryRecordsOfFrameFiles } from './sale_orders_charts.ts'

describe('saleSummaryRecordsOfFrameFiles', () => {
	it('makes a record per day file and joins the divisor rows of a product', () => {
		const records = saleSummaryRecordsOfFrameFiles([{
			keys: [20000], snapshot: 1, slot: 1,
			// Product 77 sold in sixths and, after a refinement, in twelfths; product 78 has no divisor.
			rows: [[77, 77, 78], [6, 12, 1]],
			sums: [
				[1700, 650, 2400], // AmountPEN
				[0, 650, 2400], // DebtAmountPEN
				[3, 1, 2], // SoldUnits
				[2, 6, 0], // SoldSub
				[0, 1, 2], // PendingUnits
				[0, 6, 0], // PendingSub
			],
		}, {
			keys: [20001], snapshot: 1, slot: 1, rows: [[78], [1]],
			sums: [[1200], [0], [1], [0], [0], [0]],
		}])

		expect(records).toHaveLength(2)
		expect(records[0]!.Date).toBe(20000)
		expect(records[0]!.ProductIDs).toEqual([77, 78])
		// 3 units + 2/6, then 1 unit + 6/12.
		expect(records[0]!.Quantity[0]).toBeCloseTo(3 + 2 / 6 + 1.5)
		expect(records[0]!.Quantity[1]).toBe(2)
		expect(records[0]!.QuantityPendingDelivery).toEqual([1.5, 2])
		expect(records[0]!.TotalAmount).toEqual([2350, 2400])
		expect(records[0]!.TotalDebtAmount).toEqual([650, 2400])
		expect(records[1]).toEqual({
			Date: 20001, ProductIDs: [78], Quantity: [1], QuantityPendingDelivery: [0], TotalAmount: [1200], TotalDebtAmount: [0],
		})
	})
})
