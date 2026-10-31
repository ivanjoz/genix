import { describe, expect, test } from 'bun:test';
import {
  SPRINT_STATE_ACTIVE, SPRINT_STATE_CLOSED, STORY_STATE_FINISHED, STORY_STATE_IN_DEVELOPMENT, STORY_STATE_PENDING, STORY_STATE_STOPPED, makeEmptyProject,
  type IEpic, type ISprint, type IStory,
} from '../project.ts';
import { makeEmptyStory } from '../story/story.ts';
import { sUnixTimeToUnixMillis } from '../backlog/backlog.utils.ts';
import { NONE_FILTER } from '../backlog/backlog.ts';
import { isStoryInGanttFilters, projectCalendarActivities, projectGantt, projectMarkers, sprintGantt, storyWorkSpan, workStartBefore } from './timeline.ts';

// Shows the English half, so assertions read plainly.
const translate = (text: string) => text.split('|')[0]

const project = { ...makeEmptyProject(1), ID: 1, Name: 'Fertirriego', StartDate: 20000, EndDate: 20100 }
const makeStory = (overrides: Partial<IStory>): IStory => ({ ...makeEmptyStory(1), Title: `Story ${overrides.ID}`, ...overrides })
const epics: IEpic[] = [{ ProjectID: 1, ID: 10, Name: 'Reportes', Color: 'purple', ss: 1 }]
const stories = [
  makeStory({ ID: 1, EpicID: 10, EstimatedHours: 4 }),
  makeStory({ ID: 2, EstimatedHours: 4 }),
  makeStory({ ID: 3, EpicID: 10 }),
]
const closedSprint: ISprint = {
  ProjectID: 1, ID: 1, Name: 'Sprint 1', StartDate: 20010, EndDate: 20023, State: SPRINT_STATE_CLOSED, ss: 1,
  Stories: [
    { StoryID: 1, State: STORY_STATE_FINISHED, Rank: 1, AddedAt: 1, DoneAt: 395000000 },
    { StoryID: 2, State: STORY_STATE_STOPPED, Rank: 2, AddedAt: 1, CarriedToSprintID: 2 },
  ],
}
const activeSprint: ISprint = {
  ProjectID: 1, ID: 2, Name: 'Sprint 2', StartDate: 20024, EndDate: 20037, State: SPRINT_STATE_ACTIVE, ss: 1,
  Stories: [{ StoryID: 2, State: STORY_STATE_IN_DEVELOPMENT, Rank: 1, AddedAt: 2 }],
}

describe('projectGantt', () => {
  const ganttRows = projectGantt(project, epics, stories, [activeSprint, closedSprint], translate, () => true)

  test('sprints on top by start, then the epics with their planned stories', () => {
    expect(ganttRows.map((row) => [row.id, row.parentId])).toEqual([
      ['sprints', undefined], ['sprint-1', 'sprints'], ['sprint-2', 'sprints'],
      ['epic-10', undefined], ['story-1', 'epic-10'],
      ['epic-0', undefined], ['story-2', 'epic-0'],
    ])
  })
  test('a sprint bar carries its finished share of the estimated hours', () => {
    expect(ganttRows[1].bars[0]).toMatchObject({ start: 20010, end: 20023, color: 'gray', progress: 0.5 })
    expect(ganttRows[2].bars[0]).toMatchObject({ color: 'green', progress: 0 })
  })
  test('a story has one bar per sprint, colored by its state there', () => {
    const storyRow = ganttRows.find((row) => row.id === 'story-2')
    expect(storyRow?.bars.map((bar) => [bar.sprintID, bar.color, bar.label])).toEqual([
      [1, 'red', 'Sprint 1 · Not completed'], [2, 'purple', 'Sprint 2 · In development'],
    ])
  })
  test('an epic bar spans its stories, in the epic color', () => {
    expect(ganttRows.find((row) => row.id === 'epic-10')?.bars).toEqual([{ start: 20010, end: 20023, color: 'purple', label: 'Reportes' }])
    expect(ganttRows.find((row) => row.id === 'epic-0')?.label).toBe('No epic')
  })
})

describe('projectCalendarActivities', () => {
  test('the project dates, the sprints over their days and finished stories on their day', () => {
    const activities = projectCalendarActivities(project, stories, [closedSprint, activeSprint], translate)
    expect(activities.map((activity) => [activity.title, activity.date, activity.endDate])).toEqual([
      ['Project start', 20000, undefined], ['Project end', 20100, undefined],
      ['Sprint 1', 20010, 20023], ['Sprint 2', 20024, 20037],
      // DoneAt 395000000 is unix 1790000000.
      ['H01 Story 1', Math.floor(1790000000 / 86400), undefined],
    ])
  })
  test('without project dates there are no markers', () => {
    expect(projectMarkers({ ...project, StartDate: 0, EndDate: 0 }, translate)).toEqual([])
  })
})

describe('times', () => {
  test('a SUnixTime reads back as unix milliseconds', () => {
    expect(sUnixTimeToUnixMillis(395000000)).toBe(1790000000000)
  })
})

// Unix day 20740 is a Wednesday, 20738 a Monday, 20737 a Sunday.
describe('sprint Gantt', () => {
  const today = 20745
  test('workStartBefore counts the end day and skips Sundays', () => {
    expect(workStartBefore(20740, 18)).toBe(20738)
    expect(workStartBefore(20738, 12)).toBe(20736)
    expect(workStartBefore(20740, 0)).toBe(20740)
  })

  test('storyWorkSpan: development to finish, else back by hours, unfinished up to tomorrow', () => {
    const finished = makeStory({ ID: 1, State: STORY_STATE_FINISHED, EstimatedHours: 18, StateDates: [{ State: STORY_STATE_FINISHED, Date: 20740 }] })
    expect(storyWorkSpan(finished, today)).toEqual({ start: 20738, end: 20740 })
    finished.StateDates!.push({ State: STORY_STATE_IN_DEVELOPMENT, Date: 20735 })
    expect(storyWorkSpan(finished, today)).toEqual({ start: 20735, end: 20740 })
    const stopped = makeStory({ ID: 2, State: STORY_STATE_STOPPED, StateDates: [{ State: STORY_STATE_IN_DEVELOPMENT, Date: 20741 }, { State: STORY_STATE_STOPPED, Date: 20743 }] })
    expect(storyWorkSpan(stopped, today)).toEqual({ start: 20741, end: 20746 })
    expect(storyWorkSpan(makeStory({ ID: 3, State: STORY_STATE_PENDING, StateDates: [{ State: STORY_STATE_PENDING, Date: 20740 }] }), today)).toBeNull()
  })

  test('per epic and assignee, stories side by side, a new row only where they overlap', () => {
    const finishedOn = (id: number, assigneeID: number, developmentDay: number, finishedDay: number) => makeStory({
      ID: id, EpicID: 10, AssigneeID: assigneeID, State: STORY_STATE_FINISHED,
      StateDates: [{ State: STORY_STATE_IN_DEVELOPMENT, Date: developmentDay }, { State: STORY_STATE_FINISHED, Date: finishedDay }],
    })
    const sprintStories = [
      finishedOn(1, 7, 20730, 20732), finishedOn(2, 7, 20733, 20735), finishedOn(3, 7, 20734, 20736),
      finishedOn(5, 0, 20730, 20731), finishedOn(6, 8, 20734, 20736), makeStory({ ID: 4 }),
      // One-day stories: drawn from 0.6 days before, holding the day before in their row.
      finishedOn(7, 8, 20733, 20733), finishedOn(8, 8, 20737, 20737),
    ]
    const userNames: Record<number, string> = { 7: 'Zoe', 8: 'Ana' }
    const gantt = sprintGantt(epics, sprintStories, [closedSprint], today, translate, (userID) => userNames[userID], () => true)
    expect(gantt.rows.map((row) => [row.label, row.bars.map((bar) => bar.story?.ID)])).toEqual([
      ['Reportes', []], ['Ana', [7, 6]], ['', [8]], ['Zoe', [1, 2]], ['', [3]], ['Unassigned', [5]],
    ])
    expect(gantt.rows[1].bars[0]).toMatchObject({ start: 20732.4, end: 20733 })
    expect(gantt.headerBands[0].label).toContain('Sprint 1 (')
  })
})

test('isStoryInGanttFilters: a state (NONE_FILTER = not planned) and the chosen people (0 = unassigned)', () => {
  const story = makeStory({ ID: 1, State: STORY_STATE_FINISHED, AssigneeID: 7 })
  expect(isStoryInGanttFilters(story, 0, [])).toBe(true)
  expect(isStoryInGanttFilters(story, STORY_STATE_FINISHED, [7, 0])).toBe(true)
  expect(isStoryInGanttFilters(story, STORY_STATE_PENDING, [])).toBe(false)
  expect(isStoryInGanttFilters(story, 0, [0])).toBe(false)
  expect(isStoryInGanttFilters(makeStory({ ID: 2 }), NONE_FILTER, [0])).toBe(true)
})
