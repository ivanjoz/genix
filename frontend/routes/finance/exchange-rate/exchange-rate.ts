/**
 * Exchange rate business rules. One stored record per month, the day being the position
 * inside the rate arrays — index 0 is day 1 — and the page reading them as a continuous
 * calendar: one section per month, one row per week of that month.
 *
 * A day the company has not loaded falls back to the BCRP interbank rate, which is published
 * data and not a stored value: it fills the cell so the calendar is readable, and it is the
 * company's own rate — and only that — that ever gets saved.
 */

import type { IExchangeRateDay } from '@ivanjoz/public-business-data'

/** A rate travels and is stored as rate × 1000 (3 decimals as an integer): 3.752 → 3752. */
export const EXCHANGE_RATE_SCALE = 1000
/** Ceiling a rate may reach (1000.000). Mirrors the backend: above it, it is a typo. */
export const EXCHANGE_RATE_MAX = 1000 * EXCHANGE_RATE_SCALE
/** The longest month, which is how many rates a stored row can carry. */
export const DAYS_PER_MONTH_MAX = 31
/** How far back the calendar runs: the current month and the five years before it. */
export const MONTHS_OFFERED = 60

/** Bilingual month names (EN|ES) — pass through tr(). Index 0 is January. */
export const MONTH_NAMES = [
	'January|Enero', 'February|Febrero', 'March|Marzo', 'April|Abril',
	'May|Mayo', 'June|Junio', 'July|Julio', 'August|Agosto',
	'September|Septiembre', 'October|Octubre', 'November|Noviembre', 'December|Diciembre',
]

/** Bilingual weekday names (EN|ES), Monday first — the order of the calendar columns. */
export const WEEKDAY_NAMES = [
	'Monday|Lunes', 'Tuesday|Martes', 'Wednesday|Miércoles', 'Thursday|Jueves',
	'Friday|Viernes', 'Saturday|Sábado', 'Sunday|Domingo',
]

/** Which of the two published rates a cell edits. */
export type RateKind = 'Buy' | 'Sell'

//STRUCT:finance.ExchangeRate
export interface IExchangeRate {
  ID: number
  DetailBuyRate: number[]
  DetailSellRate: number[]
  ss: number
  upd: number
  upv: number
  UpdatedBy: number
  Created: number
  /* extra fields */
	CompanyID?: number
}

/** The part of a stored record these rules read — the rest of the row is server bookkeeping. */
export type IStoredMonthRates =
	Pick<IExchangeRate, 'ID'> & Partial<Pick<IExchangeRate, 'DetailBuyRate' | 'DetailSellRate'>>

/**
 * One month being edited. Buy and Sell always hold 31 slots so a cell can be written by day
 * index without growing the array; the empty tail is trimmed again on save.
 */
export interface IMonthDraft {
	/** The month in YYMM form: 2601 is January 2026. */
	ID: number
	Buy: number[]
	Sell: number[]
	/** True once the user typed into this month and the change has not been saved yet. */
	isDirty: boolean
}

/**
 * One row of the calendar: either the full-width month separator or a week of it. A week carries
 * the day number per weekday, Monday first, and a 0 where the week falls outside the month —
 * the days around the 1st and the 31st belong to the neighbouring month's own section.
 */
export interface ICalendarRow {
	IsMonthHeader: boolean
	/** The month in YYMM form this row belongs to. */
	MonthKey: number
	Days: number[]
}

/** January 2026 → 2601. */
export function toMonthKey(year: number, monthIndex: number): number {
	return (year - 2000) * 100 + monthIndex + 1
}

/** 2601 → 2026. */
export function yearOfMonthKey(monthKey: number): number {
	return 2000 + Math.floor(monthKey / 100)
}

/** 2601 → 0 (January). */
export function monthIndexOfMonthKey(monthKey: number): number {
	return (monthKey % 100) - 1
}

/** Days the month actually has — day 0 of the next month is the last day of this one. */
export function daysInMonth(year: number, monthIndex: number): number {
	return new Date(year, monthIndex + 1, 0).getDate()
}

/** The months the calendar shows, newest first: the given month and the ones before it. */
export function calendarMonthKeys(fromDate: Date, monthsCount: number): number[] {
	const monthKeys: number[] = []
	for (let monthsBack = 0; monthsBack < monthsCount; monthsBack++) {
		const month = new Date(fromDate.getFullYear(), fromDate.getMonth() - monthsBack, 1)
		monthKeys.push(toMonthKey(month.getFullYear(), month.getMonth()))
	}
	return monthKeys
}

/**
 * The span the calendar covers, as the `YYYY-MM-DD` strings the published series is asked for:
 * from the 1st of its oldest month to the last day of its newest one. Asking for exactly what is
 * on screen is what keeps the download to the year files those months fall in.
 */
export function calendarDateRange(monthKeys: number[]): { from: string; to: string } {
	const oldestMonthKey = monthKeys[monthKeys.length - 1]
	const newestMonthKey = monthKeys[0]
	const newestYear = yearOfMonthKey(newestMonthKey)
	const newestMonthIndex = monthIndexOfMonthKey(newestMonthKey)

	const pad = (value: number) => String(value).padStart(2, '0')
	return {
		from: `${yearOfMonthKey(oldestMonthKey)}-${pad(monthIndexOfMonthKey(oldestMonthKey) + 1)}-01`,
		to: `${newestYear}-${pad(newestMonthIndex + 1)}-${daysInMonth(newestYear, newestMonthIndex)}`,
	}
}

/** One month of published rates, in the same 31-slot shape a draft holds: 0 where nothing exists. */
export interface IPublishedMonthRates {
	Buy: number[]
	Sell: number[]
}

/**
 * The published series indexed the way the calendar reads it: month key → rates by day. The
 * scaled integers are taken as they come, because x1000 is the form the source publishes and
 * the form the page stores — re-scaling a decimal here is where the rounding errors appear.
 */
export function groupPublishedRatesByMonth(
	publishedDays: IExchangeRateDay[],
): Map<number, IPublishedMonthRates> {
	const ratesByMonthKey = new Map<number, IPublishedMonthRates>()

	for (const publishedDay of publishedDays || []) {
		// The date is split, not parsed: `new Date('2026-09-21')` is UTC midnight, which is the
		// 20th in Lima, so every rate would land on the previous day.
		const [year, month, dayOfMonth] = publishedDay.date.split('-').map(Number)
		if (!year || !month || !dayOfMonth) continue

		const monthKey = toMonthKey(year, month - 1)
		let monthRates = ratesByMonthKey.get(monthKey)
		if (!monthRates) {
			monthRates = {
				Buy: new Array(DAYS_PER_MONTH_MAX).fill(0),
				Sell: new Array(DAYS_PER_MONTH_MAX).fill(0),
			}
			ratesByMonthKey.set(monthKey, monthRates)
		}
		monthRates.Buy[dayOfMonth - 1] = publishedDay.buyScaled
		monthRates.Sell[dayOfMonth - 1] = publishedDay.sellScaled
	}
	return ratesByMonthKey
}

/**
 * The month split into Monday-first weeks. Each week is seven slots holding the day number, or 0
 * where the week runs before the 1st or past the last day.
 */
export function weeksOfMonth(year: number, monthIndex: number): number[][] {
	const lastDay = daysInMonth(year, monthIndex)
	// getDay() counts from Sunday; the calendar starts on Monday.
	let weekdayIndex = (new Date(year, monthIndex, 1).getDay() + 6) % 7

	const weeks: number[][] = []
	let currentWeek = new Array(7).fill(0)

	for (let day = 1; day <= lastDay; day++) {
		currentWeek[weekdayIndex] = day
		weekdayIndex++
		if (weekdayIndex === 7) {
			weeks.push(currentWeek)
			currentWeek = new Array(7).fill(0)
			weekdayIndex = 0
		}
	}
	if (weekdayIndex > 0) weeks.push(currentWeek)

	return weeks
}

/** The calendar as flat rows: a month separator followed by that month's weeks, month after month. */
export function buildCalendarRows(monthKeys: number[]): ICalendarRow[] {
	const rows: ICalendarRow[] = []

	for (const monthKey of monthKeys) {
		rows.push({ IsMonthHeader: true, MonthKey: monthKey, Days: [] })
		const weeks = weeksOfMonth(yearOfMonthKey(monthKey), monthIndexOfMonthKey(monthKey))
		for (const week of weeks) {
			rows.push({ IsMonthHeader: false, MonthKey: monthKey, Days: week })
		}
	}
	return rows
}

/** A rate array padded to 31 slots, so any day can be written without resizing it. */
function padToMonth(rates?: number[]): number[] {
	const padded = new Array(DAYS_PER_MONTH_MAX).fill(0)
	for (let dayIndex = 0; dayIndex < (rates?.length || 0); dayIndex++) {
		padded[dayIndex] = rates?.[dayIndex] || 0
	}
	return padded
}

/** One draft per month of the calendar, seeded with what is stored. */
export function buildMonthDrafts(records: IStoredMonthRates[], monthKeys: number[]): IMonthDraft[] {
	const recordByMonthKey = new Map((records || []).map((record) => [record.ID, record]))

	return monthKeys.map((monthKey) => {
		const stored = recordByMonthKey.get(monthKey)
		return {
			ID: monthKey,
			Buy: padToMonth(stored?.DetailBuyRate),
			Sell: padToMonth(stored?.DetailSellRate),
			isDirty: false,
		}
	})
}

/**
 * Re-seeds the drafts from a fresh delta sync, leaving the months the user is editing alone:
 * a sync landing mid-edit must not erase what was typed and not saved yet.
 */
export function mergeFetchedRates(drafts: IMonthDraft[], records: IStoredMonthRates[]): void {
	const draftByMonthKey = new Map(drafts.map((draft) => [draft.ID, draft]))

	for (const record of records || []) {
		const draft = draftByMonthKey.get(record.ID)
		if (!draft || draft.isDirty) continue
		draft.Buy = padToMonth(record.DetailBuyRate)
		draft.Sell = padToMonth(record.DetailSellRate)
	}
}

/** Drops the empty tail, so a month loaded up to the 12th stores 12 numbers instead of 31. */
export function trimTrailingZeros(rates: number[]): number[] {
	let lastFilledIndex = rates.length - 1
	while (lastFilledIndex >= 0 && !rates[lastFilledIndex]) lastFilledIndex--
	return rates.slice(0, lastFilledIndex + 1)
}

/** The months the user touched, as records the API accepts. */
export function dirtyDraftsToRecords(drafts: IMonthDraft[]): IExchangeRate[] {
	return drafts
		.filter((draft) => draft.isDirty)
		// Only the month and its rates travel: the audit columns are the server's to write.
		.map((draft) => ({
			ID: draft.ID,
			DetailBuyRate: trimTrailingZeros(draft.Buy),
			DetailSellRate: trimTrailingZeros(draft.Sell),
			ss: 1,
		} as IExchangeRate))
}

/**
 * Turns what was typed into a stored rate (× 1000). Anything that is not a usable rate —
 * empty, negative, not a number, or past the ceiling — clears the day instead of storing junk.
 */
export function parseRateInput(typedValue: string | number): number {
	const rate = typeof typedValue === 'number' ? typedValue : parseFloat(typedValue)
	if (!rate || isNaN(rate) || rate < 0) return 0
	const scaledRate = Math.round(rate * EXCHANGE_RATE_SCALE)
	return scaledRate > EXCHANGE_RATE_MAX ? 0 : scaledRate
}

/** The currency codes a cash-bank, a product and a sale share. Mirrors finance/types. */
export const CURRENCY_PEN = 1
export const CURRENCY_USD = 2

/** How old the latest sell rate may be before a sale that needs it is refused. */
export const EXCHANGE_RATE_MAX_AGE_DAYS = 7

/** A product or sale from before currencies were tracked carries 0, and was in soles. */
export function normalizeCurrency(currencyType: number | undefined): number {
	return currencyType || CURRENCY_PEN
}

/**
 * The sell rate (× 1000) of the most recent day that has one, walking back from `today` (a
 * UnixDay) at most `maxAgeDays`; 0 when the window is empty. A day reads the way the calendar
 * shows it: the company's own rate when it loaded one, the published BCRP rate otherwise. The same
 * rule the backend applies when it creates the sale — the till only previews it.
 */
export function latestSellRate(
	records: IStoredMonthRates[], publishedRatesByMonthKey: Map<number, IPublishedMonthRates>,
	today: number, maxAgeDays: number,
): number {
	const companySellRatesByMonthKey = new Map((records || []).map((record) => [record.ID, record.DetailSellRate || []]))

	for (let day = today; day >= today - maxAgeDays; day--) {
		// A UnixDay is already a local date, so it is read in UTC to not shift it a day back.
		const date = new Date(day * 86_400_000)
		const monthKey = toMonthKey(date.getUTCFullYear(), date.getUTCMonth())
		const dayIndex = date.getUTCDate() - 1
		const sellRate = companySellRatesByMonthKey.get(monthKey)?.[dayIndex]
			|| publishedRatesByMonthKey.get(monthKey)?.Sell[dayIndex] || 0
		if (sellRate > 0) return sellRate
	}
	return 0
}

/**
 * The rate a conversion into `targetCurrency` uses. The spread always favours the store: charging
 * in dollars divides by a lower rate (more dollars), charging in soles multiplies by a higher one
 * (more soles). Both values × 1000; 0 when the spread would leave no positive rate.
 */
export function effectiveExchangeRate(sellRate: number, spread: number, targetCurrency: number): number {
	const effectiveRate = targetCurrency === CURRENCY_USD ? sellRate - spread : sellRate + spread
	return effectiveRate > 0 ? effectiveRate : 0
}

/**
 * One unit price (cents) restated from `priceCurrency` into `saleCurrency`, rounded to the cent
 * per unit so every line is quantity × the printed price and the lines add up to the total.
 */
export function convertUnitPrice(
	price: number, priceCurrency: number, saleCurrency: number, effectiveRate: number,
): number {
	if (priceCurrency === saleCurrency || !price) return price
	if (saleCurrency === CURRENCY_USD) return Math.round((price * EXCHANGE_RATE_SCALE) / effectiveRate)
	return Math.round((price * effectiveRate) / EXCHANGE_RATE_SCALE)
}

/** The symbol an amount is printed with: "US$" for dollars, "S/" for soles. */
export function currencySymbol(currencyType: number | undefined): string {
	return currencyType === CURRENCY_USD ? 'US$' : 'S/'
}
