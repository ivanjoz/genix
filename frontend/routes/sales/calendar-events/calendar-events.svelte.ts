import { GetHandler, POST } from '#libs/ui-runtime.svelte.ts'

// Mirrors backend sales/types/calendar_event.go.
export interface ICalendarEvent {
  ID: number
  // UnixDay
  Date: number
  EventType: number
  Name: string
  // Request flag on save; on read, set from the list the event came in (the global partition 0).
  IsGlobal?: boolean
  ss: number
  upd?: number
}

export const CALENDAR_EVENT_TYPES = [
  { id: 1, name: 'Holiday|Feriado' },
  { id: 2, name: 'Payday|Día de pago' },
  { id: 3, name: 'Gratificación|Gratificación' },
  { id: 4, name: 'Commercial date|Fecha comercial' },
]

// The super-admin (bootstrap user 1) is the only one who may edit the global holidays.
export const SUPER_ADMIN_USER_ID = 1

// The global Peru holidays plus the company's own events, merged into one list by date. Both lists
// are delta-synced with their own cursor; their IDs can collide, so a row is told apart by IsGlobal.
export class CalendarEventsService extends GetHandler<ICalendarEvent> {
  route = 'calendar-events'
  useCache = { min: 5, ver: 1 }

  handler(result: { GlobalEvents?: ICalendarEvent[], CompanyEvents?: ICalendarEvent[] }): void {
    const globalEvents = (result?.GlobalEvents || []).map((event) => ({ ...event, IsGlobal: true }))
    const companyEvents = (result?.CompanyEvents || []).map((event) => ({ ...event, IsGlobal: false }))
    this.records = [...globalEvents, ...companyEvents].sort((leftEvent, rightEvent) => leftEvent.Date - rightEvent.Date)
  }

  constructor() {
    super()
    this.fetch()
  }
}

// Creates or updates an event; ss = 0 deletes it.
export const postCalendarEvent = (event: ICalendarEvent) => {
  return POST({ route: 'calendar-events', data: event, refreshRoutes: ['calendar-events'] }) as Promise<ICalendarEvent>
}
