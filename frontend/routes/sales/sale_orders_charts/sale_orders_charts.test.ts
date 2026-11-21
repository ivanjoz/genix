import { describe, expect, it } from 'vitest'
import { saleSummaryRecordsOfFrameFiles } from './sale_orders_charts.ts'

describe('saleSummaryRecordsOfFrameFiles', () => {
	it('makes a record per day, quantities unscaled from the frame floats', () => {
		const records = saleSummaryRecordsOfFrameFiles([{
			keys: [20000, 1], snapshot: 1, slot: 1,
			// Product 77 sold 3 units and 2 sixths plus 1 unit and 6 twelfths; product 78 two whole units.
			rows: [[77, 78]],
			sums: [
				[2350, 2400], // AmountPEN
				[650, 2400], // DebtAmountPEN
				[4833333, 2000000], // SoldQuantity × 1e6
				[1500000, 2000000], // PendingQuantity × 1e6
			],
		}, {
			keys: [20001, 1], snapshot: 1, slot: 1, rows: [[78]],
			sums: [[1200], [0], [1000000], [0]],
		}])

		expect(records).toHaveLength(2)
		expect(records[0]).toEqual({
			Date: 20000, ProductIDs: [77, 78], Quantity: [4.833333, 2], QuantityPendingDelivery: [1.5, 2],
			TotalAmount: [2350, 2400], TotalDebtAmount: [650, 2400],
		})
		expect(records[1]).toEqual({
			Date: 20001, ProductIDs: [78], Quantity: [1], QuantityPendingDelivery: [0], TotalAmount: [1200], TotalDebtAmount: [0],
		})
	})

	it('adds up the sites of a day per product', () => {
		const records = saleSummaryRecordsOfFrameFiles([{
			keys: [20000, 1], snapshot: 1, slot: 1, rows: [[77, 78]],
			sums: [[1000, 500], [0, 0], [1500000, 1000000], [0, 0]],
		}, {
			keys: [20000, 2], snapshot: 1, slot: 1, rows: [[77, 79]],
			sums: [[300, 900], [300, 0], [333333, 2000000], [333333, 0]],
		}])

		expect(records).toEqual([{
			Date: 20000, ProductIDs: [77, 78, 79], Quantity: [1.833333, 1, 2], QuantityPendingDelivery: [0.333333, 0, 0],
			TotalAmount: [1300, 500, 900], TotalDebtAmount: [300, 0, 0],
		}])
	})
})
