import { todayUnixDay } from '#libs/helpers.ts'
import type { ICrmTeam } from '#services/crm/crm-team.svelte.ts'
import { TASK_KIND, TASK_STATUS } from '#services/crm/crm-task.ts'
import type { ICrmTask } from '#services/crm/crm-task.svelte.ts'

/** The open tasks due on one day: the page draws a day divider over their cards. */
export interface TaskDayGroup {
  /** UnixDay */
  dueDate: number
  isOverdue: boolean
  tasks: ICrmTask[]
}

/**
 * The open tasks of one user split for the inbox. An approval has no due date and someone is waiting
 * on it, so approvals lead on their own; every other task goes under the day it is due, oldest day
 * first, so the overdue days come before today. A task without a due date counts as due today.
 */
export const groupOpenTasks = (tasks: ICrmTask[], today: number): { approvals: ICrmTask[]; days: TaskDayGroup[] } => {
  const approvals: ICrmTask[] = []
  const tasksByDay = new Map<number, ICrmTask[]>()
  for (const task of tasks) {
    if (task.TaskStatus !== TASK_STATUS.PENDING) continue
    if (task.Kind === TASK_KIND.APPROVAL) { approvals.push(task); continue }
    const dueDate = task.DueDate || today
    tasksByDay.set(dueDate, [...(tasksByDay.get(dueDate) ?? []), task])
  }
  const days = [...tasksByDay.entries()]
    .sort(([dayA], [dayB]) => dayA - dayB)
    .map(([dueDate, dayTasks]) => ({ dueDate, isOverdue: dueDate < today, tasks: dayTasks.sort((taskA, taskB) => taskA.ID - taskB.ID) }))
  return { approvals: approvals.sort((taskA, taskB) => taskA.ID - taskB.ID), days }
}

export const MAX_TASK_COLUMNS = 4
const MIN_TASK_CARD_WIDTH = 300

/** How many card columns fit in the list's width: at least 1, at most MAX_TASK_COLUMNS. */
export const taskColumnsForWidth = (listWidth: number, columnGap: number): number =>
  Math.max(1, Math.min(MAX_TASK_COLUMNS, Math.floor((listWidth + columnGap) / (MIN_TASK_CARD_WIDTH + columnGap))))

/**
 * Packs consecutive days into rows of `columns` cards so short days share a row instead of leaving
 * it half empty: with 4 columns, a day of 2 cards and the next day of 2 go side by side. A day that
 * does not fit in the row's free columns starts a new row; a day with more cards than columns takes
 * a row of its own (its cards wrap inside it).
 */
export const packDaysIntoRows = (days: TaskDayGroup[], columns: number): TaskDayGroup[][] => {
  const rows: TaskDayGroup[][] = []
  let usedColumns = columns
  for (const day of days) {
    const dayColumns = Math.min(day.tasks.length, columns)
    if (usedColumns + dayColumns > columns) {
      rows.push([])
      usedColumns = 0
    }
    rows[rows.length - 1].push(day)
    usedColumns += dayColumns
  }
  return rows
}

/** "Today" / "Tomorrow" / "Yesterday" for the days near today; empty for the rest, which show their date. */
export const relativeDayLabel = (dueDate: number, today: number): string =>
  ({ [-1]: 'Yesterday|Ayer', 0: 'Today|Hoy', 1: 'Tomorrow|Mañana' } as Record<number, string>)[dueDate - today] ?? ''

/** The users whose tasks a leader may read: the members of every team they lead, themselves excluded. */
export const teamMembersLedBy = (leaderUserID: number, teams: ICrmTeam[]): number[] => {
  const memberIDs = new Set<number>()
  for (const team of teams) {
    if (team.LeaderUserID !== leaderUserID) continue
    for (const memberID of team.MemberUserIDs ?? []) if (memberID !== leaderUserID) memberIDs.add(memberID)
  }
  return [...memberIDs].sort((memberA, memberB) => memberA - memberB)
}

/** How a closed task ended, as the label of its badge. */
export const TASK_RESULT_LABELS: Record<number, string> = {
  [TASK_STATUS.DONE]: 'Done|Hecha',
  [TASK_STATUS.DISMISSED]: 'Dismissed|Descartada',
  [TASK_STATUS.APPROVED]: 'Approved|Aprobada',
  [TASK_STATUS.REJECTED]: 'Rejected|Rechazada',
  [TASK_STATUS.CANCELLED]: 'Cancelled|Cancelada',
}

/** The colors of each result's badge: green for what went through, red for a refusal, grey for the rest. */
export const TASK_RESULT_BADGE_CSS: Record<number, string> = {
  [TASK_STATUS.DONE]: 'bg-green-bg-strong text-green-fg',
  [TASK_STATUS.APPROVED]: 'bg-green-bg-strong text-green-fg',
  [TASK_STATUS.REJECTED]: 'bg-red-bg-strong text-red-fg',
  [TASK_STATUS.DISMISSED]: 'bg-surface-muted text-fg-soft',
  [TASK_STATUS.CANCELLED]: 'bg-surface-muted text-fg-soft',
}

/** Most recent first: what the user finished last is what they look for. */
export const sortClosedTasks = (tasks: ICrmTask[]): ICrmTask[] =>
  tasks.filter((task) => task.TaskStatus !== TASK_STATUS.PENDING).sort((taskA, taskB) => (taskB.DoneTime - taskA.DoneTime) || (taskB.ID - taskA.ID))

/** The closed tasks of one local calendar day: the page draws them as one block beside the day. */
export interface ClosedTaskDay {
  /** UnixDay */
  doneDay: number
  tasks: ICrmTask[]
}

/** A SUnixTime as the UnixDay of its local calendar date. */
export const localUnixDayOf = (sUnixTime: number): number => todayUnixDay(new Date((1e9 + sUnixTime * 2) * 1000))

/** Groups the closed tasks by the day they were closed, latest day first (each day keeps the latest first). */
export const groupClosedTasksByDay = (tasks: ICrmTask[]): ClosedTaskDay[] => {
  const days: ClosedTaskDay[] = []
  for (const task of sortClosedTasks(tasks)) {
    const doneDay = localUnixDayOf(task.DoneTime)
    const lastDay = days[days.length - 1]
    if (lastDay?.doneDay === doneDay) lastDay.tasks.push(task)
    else days.push({ doneDay, tasks: [task] })
  }
  return days
}
