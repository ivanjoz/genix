// Pure rules for the header notifications layer: ordering, the "recent" popup
// window and the color/icon a row renders with. No Svelte, no IndexedDB.

import {
  PROCESS_KIND, PROCESS_STATUS_IN_PROGRESS, PROCESS_STATUS_DONE, PROCESS_STATUS_CANCELED,
  type NotificationRow, type NotificationColor, type NotificationIcon,
} from './notifications.idb'

// The full list (opened from the header button) shows at most this many rows.
export const MAX_VISIBLE_NOTIFICATIONS = 80
// A popup opened by sendUserNotification shows the rows created in this window.
export const RECENT_NOTIFICATIONS_WINDOW_MS = 5000

// Icon a message gets when the caller only picks a color.
export const DEFAULT_ICON_BY_COLOR: Record<NotificationColor, NotificationIcon> = {
  blue: 'info', green: 'success', yellow: 'warning', red: 'error',
}

const isRunningProcess = (row: NotificationRow): boolean =>
  row.kind === PROCESS_KIND && row.status === PROCESS_STATUS_IN_PROGRESS

// In-progress processes float to the top, everything else is newest-first. Rows
// sent in the same millisecond fall back to the id, which grows per row.
export const sortNotificationRows = (rows: NotificationRow[]): NotificationRow[] => {
  return [...rows].sort((a, b) => {
    const aActive = isRunningProcess(a) ? 0 : 1
    const bActive = isRunningProcess(b) ? 0 : 1
    if (aActive !== bActive) return aActive - bActive
    return (b.createdAt - a.createdAt) || (b.id - a.id)
  })
}

// Rows the layer renders. recentSinceTime = 0 is the full list (capped); otherwise
// only rows created since that time, plus how many of the full list stay hidden
// (the "See n more messages..." link).
export const selectVisibleNotifications = (
  sortedRows: NotificationRow[], recentSinceTime: number,
): { visibleRows: NotificationRow[], hiddenCount: number } => {
  const cappedRows = sortedRows.slice(0, MAX_VISIBLE_NOTIFICATIONS)
  if (!recentSinceTime) return { visibleRows: cappedRows, hiddenCount: 0 }
  const visibleRows = cappedRows.filter((row) => row.createdAt >= recentSinceTime)
  return { visibleRows, hiddenCount: cappedRows.length - visibleRows.length }
}

// A process has no color of its own: its live status decides it.
export const notificationAppearance = (
  row: NotificationRow,
): { color: NotificationColor, icon: NotificationIcon | 'spinner' } => {
  if (row.kind !== PROCESS_KIND) return { color: row.color, icon: row.icon }
  if (row.status === PROCESS_STATUS_CANCELED) return { color: 'red', icon: 'error' }
  if (row.status === PROCESS_STATUS_DONE) return { color: 'green', icon: 'success' }
  return { color: 'blue', icon: 'spinner' }
}
