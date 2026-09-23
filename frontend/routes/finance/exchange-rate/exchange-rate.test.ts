import { describe, expect, it } from 'vitest'
import {
	buildCalendarRows,
	buildMonthDrafts,
	calendarDateRange,
	calendarMonthKeys,
	convertUnitPrice,
	CURRENCY_PEN,
	CURRENCY_USD,
	daysInMonth,
	effectiveExchangeRate,
	dirtyDraftsToRecords,
	groupPublishedRatesByMonth,
	latestSellRate,
	mergeFetchedRates,
	monthIndexOfMonthKey,
	parseRateInput,
	toMonthKey,
	trimTrailingZeros,
	weeksOfMonth,
	yearOfMonthKey,
	type IMonthDraft,
} from './exchange-rate'

describe('month keys', () => {
	it('packs a year and a month into YYMM', () => {
		expect(toMonthKey(2026, 0)).toBe(2601)
		expect(toMonthKey(2026, 11)).toBe(2612)
		expect(toMonthKey(2025, 8)).toBe(2509)
	})

	it('reads the year and the month back out', () => {
		expect(yearOfMonthKey(2601)).toBe(2026)
		expect(monthIndexOfMonthKey(2601)).toBe(0)
		expect(yearOfMonthKey(2212)).toBe(2022)
		expect(monthIndexOfMonthKey(2212)).toBe(11)
	})

	it('knows how long each month is, leap years included', () => {
		expect(daysInMonth(2026, 1)).toBe(28)
		expect(daysInMonth(2024, 1)).toBe(29)
		expect(daysInMonth(2026, 3)).toBe(30)
		expect(daysInMonth(2026, 11)).toBe(31)
	})

	it('walks backwards month by month, crossing the year', () => {
		expect(calendarMonthKeys(new Date(2026, 1, 15), 4)).toEqual([2602, 2601, 2512, 2511])
	})
})

describe('calendar', () => {
	it('splits a month into Monday-first weeks', () => {
		// March 2026 starts on a Sunday and has 31 days, so it spans six week rows.
		expect(weeksOfMonth(2026, 2)).toEqual([
			[0, 0, 0, 0, 0, 0, 1],
			[2, 3, 4, 5, 6, 7, 8],
			[9, 10, 11, 12, 13, 14, 15],
			[16, 17, 18, 19, 20, 21, 22],
			[23, 24, 25, 26, 27, 28, 29],
			[30, 31, 0, 0, 0, 0, 0],
		])
	})

	it('leaves no gap when the month starts on a Monday', () => {
		// June 2026 starts on a Monday and has 30 days.
		const weeks = weeksOfMonth(2026, 5)
		expect(weeks[0]).toEqual([1, 2, 3, 4, 5, 6, 7])
		expect(weeks[weeks.length - 1]).toEqual([29, 30, 0, 0, 0, 0, 0])
	})

	it('lays a month header before each month and one row per week', () => {
		const rows = buildCalendarRows([2603, 2602])

		expect(rows[0]).toEqual({ IsMonthHeader: true, MonthKey: 2603, Days: [] })
		expect(rows.filter((row) => row.IsMonthHeader).map((row) => row.MonthKey)).toEqual([2603, 2602])
		// March 2026 has six week rows, February 2026 has five.
		expect(rows.filter((row) => !row.IsMonthHeader && row.MonthKey === 2603).length).toBe(6)
		expect(rows.filter((row) => !row.IsMonthHeader && row.MonthKey === 2602).length).toBe(5)
	})
})

describe('drafts', () => {
	it('builds one draft per month, seeded with what is stored', () => {
		const drafts = buildMonthDrafts(
			[{ ID: 2601, DetailBuyRate: [3752, 3760], DetailSellRate: [3765] }],
			[2601, 2512],
		)

		expect(drafts.length).toBe(2)
		expect(drafts[0].ID).toBe(2601)
		expect(drafts[0].Buy.length).toBe(31)
		expect(drafts[0].Buy[0]).toBe(3752)
		expect(drafts[0].Buy[1]).toBe(3760)
		expect(drafts[0].Buy[2]).toBe(0)
		expect(drafts[0].Sell[0]).toBe(3765)
		expect(drafts[1].Buy.every((rate) => rate === 0)).toBe(true)
	})

	it('a sync refreshes clean months and leaves an edited one alone', () => {
		const drafts = buildMonthDrafts([], [2601, 2602])
		drafts[0].Buy[0] = 9999
		drafts[0].isDirty = true

		mergeFetchedRates(drafts, [
			{ ID: 2601, DetailBuyRate: [3752] },
			{ ID: 2602, DetailBuyRate: [3800] },
		])

		expect(drafts[0].Buy[0]).toBe(9999)
		expect(drafts[1].Buy[0]).toBe(3800)
	})

	it('sends only the edited months, without their empty tail', () => {
		const drafts: IMonthDraft[] = buildMonthDrafts([], [2601])
		drafts[0].Buy[0] = 3752
		drafts[0].Sell[1] = 3765
		drafts[0].isDirty = true

		const records = dirtyDraftsToRecords(drafts)

		expect(records.length).toBe(1)
		expect(records[0]).toEqual({ ID: 2601, DetailBuyRate: [3752], DetailSellRate: [0, 3765], ss: 1 })
	})
})

describe('published default rates', () => {
	const publishedDay = (date: string, buyScaled: number, sellScaled: number) => ({
		date,
		unixDay: 0,
		buy: buyScaled / 1000,
		sell: sellScaled / 1000,
		buyScaled,
		sellScaled,
	})

	it('asks for the exact span the calendar shows', () => {
		// Newest first, as calendarMonthKeys emits them: February 2026 back to November 2025.
		expect(calendarDateRange([2602, 2601, 2512, 2511])).toEqual({
			from: '2025-11-01',
			to: '2026-02-28',
		})
	})

	it('indexes each published day by its month and its day number', () => {
		const ratesByMonthKey = groupPublishedRatesByMonth([
			publishedDay('2026-01-02', 3752, 3760),
			publishedDay('2026-01-31', 3770, 3778),
			publishedDay('2026-02-03', 3800, 3810),
		])

		expect([...ratesByMonthKey.keys()]).toEqual([2601, 2602])
		expect(ratesByMonthKey.get(2601)!.Buy[1]).toBe(3752)
		expect(ratesByMonthKey.get(2601)!.Sell[1]).toBe(3760)
		// The 31st lands on the last slot, and the days the source never published stay at 0.
		expect(ratesByMonthKey.get(2601)!.Buy[30]).toBe(3770)
		expect(ratesByMonthKey.get(2601)!.Buy[0]).toBe(0)
		expect(ratesByMonthKey.get(2601)!.Buy.length).toBe(31)
		expect(ratesByMonthKey.get(2602)!.Buy[2]).toBe(3800)
	})

	it('keeps the day the source published it on, not the one a UTC parse would give', () => {
		// `new Date('2026-03-01')` is UTC midnight, which is February 28th in Lima.
		const ratesByMonthKey = groupPublishedRatesByMonth([publishedDay('2026-03-01', 3900, 3910)])

		expect([...ratesByMonthKey.keys()]).toEqual([2603])
		expect(ratesByMonthKey.get(2603)!.Buy[0]).toBe(3900)
	})

	it('has nothing to index when the source returned nothing', () => {
		expect(groupPublishedRatesByMonth([]).size).toBe(0)
	})
})

describe('rate values', () => {
	it('drops the empty tail but keeps the holes inside', () => {
		expect(trimTrailingZeros([3752, 0, 3760, 0, 0])).toEqual([3752, 0, 3760])
		expect(trimTrailingZeros([0, 0])).toEqual([])
	})

	it('stores what was typed with 3 decimals', () => {
		expect(parseRateInput(3.752)).toBe(3752)
		expect(parseRateInput('3.7525')).toBe(3753)
		expect(parseRateInput(4)).toBe(4000)
	})

	it('clears the day when the value cannot be a rate', () => {
		expect(parseRateInput('')).toBe(0)
		expect(parseRateInput('abc')).toBe(0)
		expect(parseRateInput(-3.75)).toBe(0)
		expect(parseRateInput(1000.001)).toBe(0)
	})
})

// 20718 is 2026-09-22 and 20698 is 2026-09-02 as UnixDays.
describe('sale exchange rate', () => {
	it('takes the most recent filled sell day within the window', () => {
		const sellRates = new Array(22).fill(0)
		sellRates[17] = 3740 // Sep 18
		sellRates[19] = 3752 // Sep 20; the 21st and 22nd were not loaded
		expect(latestSellRate([{ ID: 2609, DetailSellRate: sellRates }], new Map(), 20718, 7)).toBe(3752)
	})

	it('crosses into the previous month', () => {
		const augustRates = new Array(31).fill(0)
		augustRates[30] = 3701 // Aug 31
		expect(latestSellRate([{ ID: 2608, DetailSellRate: augustRates }], new Map(), 20698, 7)).toBe(3701)
	})

	it('refuses a rate older than the window', () => {
		const sellRates = new Array(22).fill(0)
		sellRates[13] = 3740 // Sep 14: eight days before the 22nd
		expect(latestSellRate([{ ID: 2609, DetailSellRate: sellRates }], new Map(), 20718, 7)).toBe(0)
	})

	it('falls back to the published BCRP rate day by day', () => {
		const companyRates = new Array(22).fill(0)
		companyRates[19] = 3752 // the company loaded Sep 20
		const publishedSell = new Array(31).fill(0)
		publishedSell[21] = 3371 // Sep 22
		publishedSell[19] = 3360 // Sep 20
		const published = new Map([[2609, { Buy: new Array(31).fill(0), Sell: publishedSell }]])
		const records = [{ ID: 2609, DetailSellRate: companyRates }]

		// Sep 22 has no company rate, so BCRP's is the newest day with any rate.
		expect(latestSellRate(records, published, 20718, 7)).toBe(3371)
		// On a day the company loaded, its rate wins over BCRP's.
		expect(latestSellRate(records, published, 20716, 7)).toBe(3752)
	})

	it('moves the spread in the store\'s favour', () => {
		expect(effectiveExchangeRate(3750, 20, CURRENCY_USD)).toBe(3730)
		expect(effectiveExchangeRate(3750, 20, CURRENCY_PEN)).toBe(3770)
		expect(effectiveExchangeRate(3750, 3750, CURRENCY_USD)).toBe(0)
	})

	it('converts each unit price to the cent, as the backend does', () => {
		expect(convertUnitPrice(1000, CURRENCY_PEN, CURRENCY_USD, 3730)).toBe(268)
		expect(convertUnitPrice(268, CURRENCY_USD, CURRENCY_PEN, 3770)).toBe(1010)
		expect(convertUnitPrice(1000, CURRENCY_USD, CURRENCY_USD, 3730)).toBe(1000)
	})
})
