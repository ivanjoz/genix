// Reactive store + public API for the header notifications & process layer.
//
// Two item families share one store:
//   - Notifications: static messages sent via sendUserNotification(), decorated
//     with one of the fixed color schemes and icons.
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
  PROCESS_STATUS_CANCELED, PROCESS_STATUS_IN_PROGRESS, PROCESS_STATUS_DONE,
  type NotificationRow, type NotificationColor, type NotificationIcon, type ProcessStatus,
} from './notifications.idb'
import { DEFAULT_ICON_BY_COLOR, RECENT_NOTIFICATIONS_WINDOW_MS } from './notifications'

// Reactive map keyed by id; the UI derives sorted lists / in-progress counts from it.
export const notifications = $state(new SvelteMap<number, NotificationRow>())

export const NOTIFICATIONS_TAB_MESSAGES = 1
export const NOTIFICATIONS_TAB_INFORMATION = 2

// Header layer state, owned here so sendUserNotification can pop the layer open.
// recentSinceTime > 0 means the layer was popped by a message and lists only the
// rows created since then; 0 means the full list.
export const notificationsLayer = $state({
  isOpen: false,
  recentSinceTime: 0,
  selectedTab: NOTIFICATIONS_TAB_MESSAGES,
})

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

export interface UserNotificationOptions {
  title?: string
  subtitle?: string
  // Color scheme of the card. Defaults to blue.
  color?: NotificationColor
  // Defaults to the icon matching the color (blue info, green success, yellow warning, red error).
  icon?: NotificationIcon
}

// Send a message to the user: it is persisted, and the header layer pops open
// showing the messages of the last few seconds until the user clicks outside.
// `message` may span several lines. Returns its id synchronously.
export const sendUserNotification = (message: string, options: UserNotificationOptions = {}): number => {
  const now = Date.now()
  const color = options.color || 'blue'
  const row: NotificationRow = {
    id: nextID(), kind: NOTIFICATION_KIND,
    title: options.title || '', subtitle: options.subtitle || '', message,
    color, icon: options.icon || DEFAULT_ICON_BY_COLOR[color],
    status: PROCESS_STATUS_DONE, createdAt: now, updatedAt: now,
  }
  notifications.set(row.id, row)
  putNotification($state.snapshot(row) as NotificationRow)

  // An already-open layer keeps its mode: a popup keeps its window start (so the
  // list only grows while it stays open), and a full list stays full.
  if (!notificationsLayer.isOpen) {
    notificationsLayer.recentSinceTime = now - RECENT_NOTIFICATIONS_WINDOW_MS
    notificationsLayer.selectedTab = NOTIFICATIONS_TAB_MESSAGES
    notificationsLayer.isOpen = true
  }
  return row.id
}

// Add a long-running process. Status must be 1 (in progress) or 2 (done) — never 0.
// Returns its id synchronously so the caller can drive it via updateProcess().
export const addProcess = (
  name: string, text: string, status: 1 | 2 = PROCESS_STATUS_IN_PROGRESS,
): number => {
  const now = Date.now()
  const row: NotificationRow = {
    id: nextID(), kind: PROCESS_KIND, title: name, subtitle: '', message: text,
    color: 'blue', icon: 'info', status, createdAt: now, updatedAt: now,
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
    title: name || row.title,
    message: text || row.message,
    status: status !== undefined ? status : row.status,
    updatedAt: Date.now(),
  }
  notifications.set(id, updated)
  patchNotification(id, {
    title: updated.title, message: updated.message, status: updated.status, updatedAt: updated.updatedAt,
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
