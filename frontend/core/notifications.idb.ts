// Local persistence for the header notifications & process layer. Backed by
// Dexie/IndexedDB so the user keeps seeing the last notifications/processes even
// after a full reload. There is no server-side counterpart — this is a UI-only
// store scoped per company+env (a different company or environment shows its own
// list, matching how the delta cache and agent chat are scoped).

import Dexie from 'dexie'
import { Env } from '$core/env'

const LOG_PREFIX = '[notifications:idb]'
const NOTIFICATIONS_DB_PREFIX = 'notifications'
const NOTIFICATIONS_DB_VERSION = 1
// Hard cap on persisted rows: keep the newest MAX_ROWS, drop the rest on hydrate.
export const MAX_NOTIFICATION_ROWS = 500

// kind discriminates the two item families that share this store.
export const NOTIFICATION_KIND = 1 // static message carrying a `type`
export const PROCESS_KIND = 2      // long-running task carrying a live `status`

// Notification `type`: severity of a static message.
export const NOTIFICATION_TYPE_INFO = 1
export const NOTIFICATION_TYPE_WARN = 2
export const NOTIFICATION_TYPE_ERROR = 3

// Process `status`: 0 canceled, 1 in progress, 2 done. A process can never be
// created with status 0 — that value is only ever set later by the producer.
export const PROCESS_STATUS_CANCELED = 0
export const PROCESS_STATUS_IN_PROGRESS = 1
export const PROCESS_STATUS_DONE = 2

export type NotificationKind = typeof NOTIFICATION_KIND | typeof PROCESS_KIND
export type NotificationType = 1 | 2 | 3
export type ProcessStatus = 0 | 1 | 2

// One persisted item. `id` is the synchronous counter id we own (see
// notifications.svelte.ts) — NOT a Dexie autoincrement — so the caller can use
// the returned id immediately without awaiting the write. `type` is meaningful
// only for notifications, `status` only for processes.
export interface NotificationRow {
  id: number
  kind: NotificationKind
  name: string
  text: string
  type: NotificationType
  status: ProcessStatus
  createdAt: number
  updatedAt: number
}

class NotificationsDatabase extends Dexie {
  notifications!: Dexie.Table<NotificationRow, number>

  constructor(databaseName: string) {
    super(databaseName)
    // We supply `id` ourselves, so the primary key is plain `id` (no `++`).
    // `createdAt` is indexed to slice/prune the newest rows cheaply.
    this.version(NOTIFICATIONS_DB_VERSION).stores({
      notifications: 'id, createdAt',
    })
  }
}

const databasesByName = new Map<string, NotificationsDatabase>()

const makeDatabaseName = (companyID: number, env: string): string => {
  return `${companyID || 0}_${NOTIFICATIONS_DB_PREFIX}_${env || '000000'}`
}

const getDatabase = (): NotificationsDatabase => {
  const name = makeDatabaseName(Env.getCompanyID(), Env.enviroment || 'main')
  const existing = databasesByName.get(name)
  if (existing) return existing
  const created = new NotificationsDatabase(name)
  databasesByName.set(name, created)
  return created
}

// Load every persisted row newest-first. The store also prunes to MAX_ROWS here
// so the table never grows unbounded across sessions.
export const loadNotifications = async (): Promise<NotificationRow[]> => {
  try {
    const rows = await getDatabase().notifications.orderBy('createdAt').reverse().toArray()
    // Prune the overflow (rows past MAX_ROWS, i.e. the oldest) in the background.
    if (rows.length > MAX_NOTIFICATION_ROWS) {
      const overflowIDs = rows.slice(MAX_NOTIFICATION_ROWS).map((row) => row.id)
      getDatabase().notifications.bulkDelete(overflowIDs).catch((error) => {
        console.warn(`${LOG_PREFIX} prune failed`, error)
      })
      return rows.slice(0, MAX_NOTIFICATION_ROWS)
    }
    return rows
  } catch (error) {
    console.warn(`${LOG_PREFIX} load failed`, error)
    return []
  }
}

// Fire-and-forget insert/replace — callers never await it (the id is already known).
export const putNotification = (row: NotificationRow): void => {
  getDatabase().notifications.put(row).catch((error) => {
    console.warn(`${LOG_PREFIX} put failed`, error)
  })
}

// Fire-and-forget partial update of an existing row.
export const patchNotification = (id: number, patch: Partial<NotificationRow>): void => {
  if (!id) return
  getDatabase().notifications.update(id, patch).catch((error) => {
    console.warn(`${LOG_PREFIX} patch failed`, error)
  })
}
