import { GetHandler } from '$libs/http.svelte'

export const WEEKS_PER_YEAR = 52

export interface ISalesPlanningWeek {
	Week: number
	Quantity: number
}

//STRUCT:sales.SalesPlanning
export interface ISalesPlanning {
	ID: number
	TempID?: number
	ProductID: number
	BaseQuantity: number
	SeasonalityCurveID: number
	WeeklyQuantity: ISalesPlanningWeek[]
	ss: number
	upd?: number
	UpdatedBy?: number
	Created?: number
}

export interface ISeasonalityCurveWeek {
	Week: number
	/** Multiplier * 1000 (3 decimals as integer): 1.5x => 1500 */
	Percent: number
}

//STRUCT:sales.SeasonalityCurve
export interface ISeasonalityCurve {
	ID: number
	TempID?: number
	Name: string
	Curve: ISeasonalityCurveWeek[]
	ss: number
	upd?: number
	UpdatedBy?: number
	Created?: number
}

export class SalesPlanningService extends GetHandler<ISalesPlanning> {
	route = 'sales-planning'
	useCache = { min: 5, ver: 1 }
	inferRemoveFromStatus = true
	prependOnSave = true

	constructor(init: boolean = false) {
		super()
		if (init) this.fetch()
	}

	handler(result: ISalesPlanning[]): void {
		this.records = []
		this.recordsMap = new Map()
		this.addSavedRecords(...result)
	}
}

export class SeasonalityCurveService extends GetHandler<ISeasonalityCurve> {
	route = 'seasonality-curve'
	useCache = { min: 5, ver: 1 }
	inferRemoveFromStatus = true
	prependOnSave = true

	makeName(record: Partial<ISeasonalityCurve>) {
		return record.Name || ''
	}

	constructor(init: boolean = false) {
		super()
		if (init) this.fetch()
	}

	handler(result: ISeasonalityCurve[]): void {
		this.records = []
		this.recordsMap = new Map()
		this.addSavedRecords(...result)
	}
}

/**
 * Expands a sparse seasonality curve (only some weeks filled) into a full 52-week
 * array of multipliers. Each empty week inherits the most recent earlier filled
 * week's value (forward-fill). Weeks before the first filled week default to 1.0.
 * Returned values are real multipliers (Percent / 1000).
 */
/** Bilingual 3-letter month labels (EN|ES) for the tr() helper. */
const MONTH_LABELS = [
	'JAN|ENE', 'FEB|FEB', 'MAR|MAR', 'APR|ABR', 'MAY|MAY', 'JUN|JUN',
	'JUL|JUL', 'AUG|AGO', 'SEP|SEP', 'OCT|OCT', 'NOV|NOV', 'DEC|DIC',
]

export interface IWeekMonthGroup {
	/** 0-based month index. */
	month: number
	/** Bilingual 3-letter label, e.g. "FEB|FEB" — pass through tr(). */
	monthLabel: string
	/** Day of month the first week of this group starts on. */
	startDay: number
	/** Week numbers (1..52) whose start date falls in this month. */
	weeks: number[]
}

/**
 * Groups the 52 weeks of the year by the calendar month their start date falls in.
 * Week N starts on day (N-1)*7 of the year. Uses a representative non-leap year
 * since the plan is year-agnostic (a recurring yearly curve).
 */
export function weekMonthGroups(): IWeekMonthGroup[] {
	const YEAR = 2025
	const groups: IWeekMonthGroup[] = []
	for (let week = 1; week <= WEEKS_PER_YEAR; week++) {
		const date = new Date(YEAR, 0, 1 + (week - 1) * 7)
		const month = date.getMonth()
		const last = groups[groups.length - 1]
		if (last && last.month === month) {
			last.weeks.push(week)
		} else {
			groups.push({ month, monthLabel: MONTH_LABELS[month], startDay: date.getDate(), weeks: [week] })
		}
	}
	return groups
}

export function resolveCurve(curve: ISeasonalityCurveWeek[]): number[] {
	const byWeek = new Map<number, number>()
	for (const w of curve || []) {
		if (w.Week >= 1 && w.Week <= WEEKS_PER_YEAR) byWeek.set(w.Week, w.Percent)
	}
	const resolved: number[] = []
	let lastPercent = 1000 // default multiplier 1.0 until the first filled week
	for (let week = 1; week <= WEEKS_PER_YEAR; week++) {
		if (byWeek.has(week)) lastPercent = byWeek.get(week) as number
		resolved.push(lastPercent / 1000)
	}
	return resolved
}
