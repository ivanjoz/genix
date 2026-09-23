import { getBcrpRateRange } from '@ivanjoz/public-business-data'
import { GetHandler } from '$libs/ui-runtime.svelte'
import {
	calendarDateRange,
	groupPublishedRatesByMonth,
	type IExchangeRate,
	type IPublishedMonthRates,
} from './exchange-rate'

/**
 * One record per month, keyed by its YYMM id, so the delta cache holds at most twelve rows
 * per year and a day edited today re-sends only its month.
 */
export class ExchangeRatesService extends GetHandler<IExchangeRate> {
	route = 'exchange-rates'
	useCache = { min: 5, ver: 1 }
	inferRemoveFromStatus = true

	constructor(init: boolean = false) {
		super()
		if (init) this.fetch()
	}

	handler(result: IExchangeRate[]): void {
		this.records = []
		this.recordsMap = new Map()
		this.addSavedRecords(...(result || []))
	}
}

/**
 * The BCRP interbank series behind the default rates of the calendar. It is public data on
 * another origin — static files, no auth, no backend of ours — so it does not go through
 * GetHandler: the client caches each year in IndexedDB under the hash the source gives it, and
 * a published year never changes, so the download happens once per browser.
 */
export class BcrpDefaultRates {
	ratesByMonthKey = $state(new Map<number, IPublishedMonthRates>())

	async load(monthKeys: number[]): Promise<void> {
		const { from, to } = calendarDateRange(monthKeys)
		try {
			this.ratesByMonthKey = groupPublishedRatesByMonth(await getBcrpRateRange(from, to))
		} catch (error) {
			// The defaults are a convenience: without them the calendar still shows and saves the
			// company's own rates, so a source that is down must not take the page with it.
			console.warn('BCRP default exchange rates unavailable:', error)
		}
	}
}
