import { GetHandler } from '#libs/ui-runtime.svelte.ts'

// Mirrors backend/accounting/types/tax_due_calendar.go. SUNAT's monthly due-date schedule, shared
// by every company of the country; only the SaaS company writes it (System → Calendario Tributario).

export interface ITaxDueCalendar {
  /** YYMM of the tax period declared (not of the month it falls due in). */
  Period: number
  /** 11 UnixDays: RUC last digit 0–9, then buenos contribuyentes / UESP. */
  DueDays: number[]
  ss: number
  upd: number
}

/** The schedule, newest period first (`tax-due-calendar`), keyed by its period. */
export class TaxDueCalendarService extends GetHandler {
  route = 'tax-due-calendar'
  useCache = { min: 60, ver: 1 }
  keyID = 'Period'

  calendars: ITaxDueCalendar[] = $state([])

  handler(result: ITaxDueCalendar[]): void {
    this.calendars = (result || [])
      .filter((calendar) => calendar.ss > 0)
      .sort((left, right) => right.Period - left.Period)
  }

  constructor(init = false) {
    super()
    if (init) this.fetch()
  }
}

// The write brings back the delta of `tax-due-calendar` (GetHandler.writeAndSync).
export const postTaxDueCalendar = (service: TaxDueCalendarService, calendars: ITaxDueCalendar[]): Promise<ITaxDueCalendar[]> =>
  service.writeAndSync('tax-due-calendar', calendars)
