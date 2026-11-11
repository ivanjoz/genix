import { describe, expect, it } from 'vitest'
import { TASK_KIND, TASK_STATUS } from '#services/crm/crm-task.ts'
import type { ICrmTask } from '#services/crm/crm-task.svelte.ts'
import {
  groupClosedTasksByDay, groupOpenTasks, packDaysIntoRows, relativeDayLabel, sortClosedTasks, taskColumnsForWidth, teamMembersLedBy,
} from './tasks.ts'

const makeTask = (overrides: Partial<ICrmTask>): ICrmTask => ({
  ID: 1, OpportunityID: 1, PipelineID: 1, StageID: 1, Kind: TASK_KIND.ACTIVITY, TemplateID: 1, Title: 'Call', AssignedUserID: 1,
  DueDate: 100, TaskStatus: TASK_STATUS.PENDING, ResultNote: '', ResultRefID: 0, Created: 0, DoneTime: 0, DoneBy: 0, CreatedBy: 0,
  OpportunityTitle: '', ClientName: '', OpportunityAmount: 0, ss: 1, upd: 0,
  ...overrides,
})

describe('groupOpenTasks', () => {
  it('splits approvals apart and groups the rest by due day, oldest day first', () => {
    const { approvals, days } = groupOpenTasks([
      makeTask({ ID: 1, DueDate: 105 }),
      makeTask({ ID: 6, DueDate: 98 }),
      makeTask({ ID: 2, DueDate: 98 }),
      makeTask({ ID: 3, DueDate: 100 }),
      makeTask({ ID: 4, Kind: TASK_KIND.APPROVAL, DueDate: 0 }),
    ], 100)
    expect(approvals.map((task) => task.ID)).toEqual([4])
    expect(days.map((day) => [day.dueDate, day.isOverdue])).toEqual([[98, true], [100, false], [105, false]])
    expect(days[0].tasks.map((task) => task.ID)).toEqual([2, 6])
  })

  it('drops closed tasks and counts a task with no due date as today', () => {
    const { approvals, days } = groupOpenTasks([
      makeTask({ ID: 1, DueDate: 0 }),
      makeTask({ ID: 2, TaskStatus: TASK_STATUS.DONE }),
    ], 100)
    expect(approvals).toEqual([])
    expect(days.map((day) => day.dueDate)).toEqual([100])
    expect(days[0].tasks.map((task) => task.ID)).toEqual([1])
  })
})

describe('packDaysIntoRows', () => {
  const makeDay = (dueDate: number, cardCount: number) =>
    ({ dueDate, isOverdue: false, tasks: Array.from({ length: cardCount }, (_, index) => makeTask({ ID: dueDate * 10 + index })) })
  const rowDates = (rows: ReturnType<typeof packDaysIntoRows>) => rows.map((row) => row.map((day) => day.dueDate))

  it('puts short consecutive days on one row while they fit', () => {
    expect(rowDates(packDaysIntoRows([makeDay(1, 2), makeDay(2, 2), makeDay(3, 1), makeDay(4, 1), makeDay(5, 3)], 4)))
      .toEqual([[1, 2], [3, 4], [5]])
  })

  it('gives a day wider than the row a row of its own', () => {
    expect(rowDates(packDaysIntoRows([makeDay(1, 1), makeDay(2, 6), makeDay(3, 1)], 4))).toEqual([[1], [2], [3]])
  })

  it('stacks every day when only one column fits', () => {
    expect(rowDates(packDaysIntoRows([makeDay(1, 1), makeDay(2, 1)], 1))).toEqual([[1], [2]])
  })
})

describe('taskColumnsForWidth', () => {
  it('fits 300px cards and caps at 4 columns', () => {
    expect(taskColumnsForWidth(200, 8)).toBe(1)
    expect(taskColumnsForWidth(616, 8)).toBe(2)
    expect(taskColumnsForWidth(2400, 8)).toBe(4)
  })
})

describe('groupClosedTasksByDay', () => {
  // SUnixTime of 12:00 UTC on a UnixDay: the same local date in any timezone within ±11h.
  const middayOf = (unixDay: number) => (unixDay * 86400 + 43200 - 1e9) / 2

  it('groups closed tasks by the day they were closed, latest day and latest task first', () => {
    const days = groupClosedTasksByDay([
      makeTask({ ID: 1, TaskStatus: TASK_STATUS.DONE, DoneTime: middayOf(20000) }),
      makeTask({ ID: 2, TaskStatus: TASK_STATUS.DISMISSED, DoneTime: middayOf(20002) }),
      makeTask({ ID: 3, TaskStatus: TASK_STATUS.DONE, DoneTime: middayOf(20000) + 60 }),
      makeTask({ ID: 4 }),
    ])
    expect(days.map((day) => [day.doneDay, day.tasks.map((task) => task.ID)])).toEqual([[20002, [2]], [20000, [3, 1]]])
  })
})

describe('relativeDayLabel', () => {
  it('names the days around today and leaves the rest to their date', () => {
    expect(relativeDayLabel(100, 100)).toBe('Today|Hoy')
    expect(relativeDayLabel(101, 100)).toBe('Tomorrow|Mañana')
    expect(relativeDayLabel(99, 100)).toBe('Yesterday|Ayer')
    expect(relativeDayLabel(90, 100)).toBe('')
  })
})

describe('teamMembersLedBy', () => {
  const teams = [
    { ID: 1, Name: 'A', LeaderUserID: 5, MemberUserIDs: [5, 7, 6], ss: 1, upd: 0 },
    { ID: 2, Name: 'B', LeaderUserID: 9, MemberUserIDs: [9, 8], ss: 1, upd: 0 },
    { ID: 3, Name: 'C', LeaderUserID: 5, MemberUserIDs: [6, 10], ss: 1, upd: 0 },
  ]
  it('lists the members of the teams the user leads, once each and without the leader', () => {
    expect(teamMembersLedBy(5, teams)).toEqual([6, 7, 10])
    expect(teamMembersLedBy(1, teams)).toEqual([])
  })
})

describe('sortClosedTasks', () => {
  it('keeps only closed tasks, latest first', () => {
    const sorted = sortClosedTasks([
      makeTask({ ID: 1, TaskStatus: TASK_STATUS.DONE, DoneTime: 10 }),
      makeTask({ ID: 2, TaskStatus: TASK_STATUS.REJECTED, DoneTime: 20 }),
      makeTask({ ID: 3 }),
    ])
    expect(sorted.map((task) => task.ID)).toEqual([2, 1])
  })
})
