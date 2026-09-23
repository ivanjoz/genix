import { describe, expect, it } from 'vitest'
import {
  sortNotificationRows, selectVisibleNotifications, notificationAppearance, MAX_VISIBLE_NOTIFICATIONS,
} from './notifications'
import {
  NOTIFICATION_KIND, PROCESS_KIND, PROCESS_STATUS_IN_PROGRESS, PROCESS_STATUS_DONE, PROCESS_STATUS_CANCELED,
  type NotificationRow,
} from './notifications.idb'

const makeRow = (id: number, createdAt: number, patch: Partial<NotificationRow> = {}): NotificationRow => ({
  id, kind: NOTIFICATION_KIND, title: '', subtitle: '', message: '', color: 'blue', icon: 'info',
  status: PROCESS_STATUS_DONE, createdAt, updatedAt: createdAt, ...patch,
})

describe('sortNotificationRows', () => {
  it('pins running processes above newer messages', () => {
    const running = makeRow(1, 100, { kind: PROCESS_KIND, status: PROCESS_STATUS_IN_PROGRESS })
    const sorted = sortNotificationRows([makeRow(2, 200), running, makeRow(3, 300)])
    expect(sorted.map((row) => row.id)).toEqual([1, 3, 2])
  })

  it('orders rows sent in the same millisecond by id, newest first', () => {
    const sorted = sortNotificationRows([makeRow(1, 100), makeRow(3, 100), makeRow(2, 100)])
    expect(sorted.map((row) => row.id)).toEqual([3, 2, 1])
  })
})

describe('selectVisibleNotifications', () => {
  const rows = sortNotificationRows([makeRow(1, 1000), makeRow(2, 5000), makeRow(3, 9000)])

  it('shows the full list with nothing hidden when not in recent mode', () => {
    const selection = selectVisibleNotifications(rows, 0)
    expect(selection.visibleRows).toHaveLength(3)
    expect(selection.hiddenCount).toBe(0)
  })

  it('shows only rows since the window start and counts the rest as hidden', () => {
    const selection = selectVisibleNotifications(rows, 4000)
    expect(selection.visibleRows.map((row) => row.id)).toEqual([3, 2])
    expect(selection.hiddenCount).toBe(1)
  })

  it('caps the full list, and the hidden count never points past the cap', () => {
    const manyRows = sortNotificationRows(Array.from({ length: 120 }, (_, i) => makeRow(i + 1, i)))
    expect(selectVisibleNotifications(manyRows, 0).visibleRows).toHaveLength(MAX_VISIBLE_NOTIFICATIONS)
    expect(selectVisibleNotifications(manyRows, 119).hiddenCount).toBe(MAX_VISIBLE_NOTIFICATIONS - 1)
  })
})

describe('notificationAppearance', () => {
  it('uses the message color and icon as given', () => {
    expect(notificationAppearance(makeRow(1, 0, { color: 'yellow', icon: 'error' }))).toEqual({ color: 'yellow', icon: 'error' })
  })

  it('derives a process look from its status', () => {
    const process = (status: 0 | 1 | 2) => makeRow(1, 0, { kind: PROCESS_KIND, status })
    expect(notificationAppearance(process(PROCESS_STATUS_IN_PROGRESS)).icon).toBe('spinner')
    expect(notificationAppearance(process(PROCESS_STATUS_DONE)).color).toBe('green')
    expect(notificationAppearance(process(PROCESS_STATUS_CANCELED)).color).toBe('red')
  })
})
