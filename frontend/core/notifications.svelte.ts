// Reactive store + public API for the header notifications & process layer.
//
// Two item families share one store:
//   - Notifications: static messages with a `type` (1 info | 2 warn | 3 error).
//   - Processes: long-running tasks with a live `status` (1 in progress | 2 done
//     | 0 canceled), updated over time via updateProcess().
//
// IDs are returned SYNCHRONOUSLY so callers can reference an item the instant
// they create it (e.g. ImageUploader keeps the process id to report progress).
// The id comes from an in-memory counter seeded once from the unix time; the
// IndexedDB write happens afterwards, fire-and-forget.

import { SvelteMap } from 'svelte/reactivity'
import {
  loadNotifications, putNotification, patchNotification,
  NOTIFICATION_KIND, PROCESS_KIND,
  NOTIFICATION_TYPE_INFO, PROCESS_STATUS_CANCELED, PROCESS_STATUS_IN_PROGRESS, PROCESS_STATUS_DONE,
  type NotificationRow, type NotificationType, type ProcessStatus,
} from './notifications.idb'

// Reactive map keyed by id; the UI derives sorted lists / in-progress counts from it.
export const notifications = $state(new SvelteMap<number, NotificationRow>())

// Counter seeded once at module load to "last 6 digits of unix-seconds × 1000",
// then incremented per item. This gives short, monotonic, collision-free ids
// within a session without any async round-trip.
let idCounter = (Math.floor(Date.now() / 1000) % 1_000_000) * 1000

// Returns the next id synchronously.
const nextID = (): number => ++idCounter

// Hydrate persisted rows into the reactive map (once, on first import). Also lifts
// the counter above any persisted id so a reload within the same second can't reuse one.
let hasHydrated = false
export const hydrateNotifications = async (): Promise<void> => {
  if (hasHydrated) return
  hasHydrated = true
  const rows = await loadNotifications()
  let maxID = idCounter
  for (const row of rows) {
    // A process still "in progress" after a reload can never resume (its producer
    // is gone), so mark it canceled — otherwise the loading ring spins forever.
    if (row.kind === PROCESS_KIND && row.status === PROCESS_STATUS_IN_PROGRESS) {
      row.status = PROCESS_STATUS_CANCELED
      patchNotification(row.id, { status: PROCESS_STATUS_CANCELED })
    }
    notifications.set(row.id, row)
    if (row.id > maxID) maxID = row.id
  }
  idCounter = maxID
}

// Add a static notification. Defaults to type 1 (info). Returns its id synchronously.
export const addNotification = (
  name: string, text: string, type: NotificationType = NOTIFICATION_TYPE_INFO,
): number => {
  const now = Date.now()
  const row: NotificationRow = {
    id: nextID(), kind: NOTIFICATION_KIND, name, text,
    type, status: PROCESS_STATUS_DONE, createdAt: now, updatedAt: now,
  }
  notifications.set(row.id, row)
  putNotification($state.snapshot(row) as NotificationRow)
  return row.id
}

// Add a long-running process. Status must be 1 (in progress) or 2 (done) — never 0.
// Returns its id synchronously so the caller can drive it via updateProcess().
export const addProcess = (
  name: string, text: string, status: 1 | 2 = PROCESS_STATUS_IN_PROGRESS,
): number => {
  const now = Date.now()
  const row: NotificationRow = {
    id: nextID(), kind: PROCESS_KIND, name, text,
    type: NOTIFICATION_TYPE_INFO, status, createdAt: now, updatedAt: now,
  }
  notifications.set(row.id, row)
  putNotification($state.snapshot(row) as NotificationRow)
  return row.id
}

// Update a process in place. Empty-string name/text keeps the previous value;
// an omitted status keeps the previous status. No-op (warns) for unknown ids.
export const updateProcess = (
  id: number, name?: string, text?: string, status?: ProcessStatus,
): void => {
  const row = notifications.get(id)
  if (!row) {
    console.warn('[notifications] updateProcess: unknown id', id)
    return
  }
  // Build a new row and re-set it: SvelteMap only tracks set/delete, NOT in-place
  // mutation of a stored value's fields — without the set, derived readers (the
  // in-progress count driving the spinning border/badge) would never recompute.
  const updated: NotificationRow = {
    ...row,
    name: name || row.name,
    text: text || row.text,
    status: status !== undefined ? status : row.status,
    updatedAt: Date.now(),
  }
  notifications.set(id, updated)
  patchNotification(id, {
    name: updated.name, text: updated.text, status: updated.status, updatedAt: updated.updatedAt,
  })
}

// Count of processes currently in progress — drives the header button's loading ring + badge.
export const inProgressProcessCount = (): number => {
  let count = 0
  for (const row of notifications.values()) {
    if (row.kind === PROCESS_KIND && row.status === PROCESS_STATUS_IN_PROGRESS) count++
  }
  return count
}
