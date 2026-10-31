import { describe, expect, test } from 'bun:test';
import { STORY_STATE_FINISHED, STORY_STATE_IN_DEVELOPMENT, type IProject, type ISprint, type IStory } from '../../project.ts';
import {
  ASSIGNED_STORIES_DAYS, WEEKLY_CAPACITY_HOURS, assignedStoryRows, defaultFirstWeekStartDay, personGantt, personWeeks, recentAssignments, storyIDsByProject,
  weekMeterScale, weekStartDay, type IAssignedStoryRow, type IStoryAssignee,
} from './by-person.ts';

// Monday 2026-10-05 as a UnixDay; a SUnixTime at noon UTC of a UnixDay.
const MONDAY = Date.UTC(2026, 9, 5) / 86_400_000
const sUnixTimeAt = (unixDay: number) => (unixDay * 86_400 + 43_200 - 1e9) / 2

const makeRow = (storyID: number, projectID: number, estimatedHours: number): IAssignedStoryRow => ({
  story: { ID: storyID, ProjectID: projectID, Title: `Story ${storyID}`, EstimatedHours: estimatedHours, State: 1, ss: 1 } as IStory,
  project: { ID: projectID, Name: `Project ${projectID}`, Color: 'blue' } as IProject,
  assignedAt: 0,
})

const makeSprint = (sprintID: number, projectID: number, startDay: number, endDay: number, stories: ISprint['Stories']): ISprint =>
  ({ ID: sprintID, ProjectID: projectID, Name: `Sprint ${sprintID}`, StartDate: startDay, EndDate: endDay, State: 2, Stories: stories, ss: 1 })

describe('weekStartDay', () => {
  test('is the Monday of the week', () => {
    expect([MONDAY, MONDAY + 3, MONDAY + 6, MONDAY + 7].map(weekStartDay)).toEqual([MONDAY, MONDAY, MONDAY, MONDAY + 7])
  })
})

describe('personWeeks', () => {
  // Each story's week, as [startDay, story IDs] of the weeks holding any.
  const storiesByWeek = (weeks: ReturnType<typeof personWeeks>) => weeks
    .map((week) => [week.startDay, week.projects.flatMap((project) => project.sprints.flatMap((weekSprint) => weekSprint.stories.map((entry) => entry.row.story.ID)))])
    .filter(([, storyIDs]) => (storyIDs as number[]).length > 0)

  test('a finished story sits only in the week it was finished', () => {
    const rows = [makeRow(1, 5, 20)]
    const threeWeekSprint = makeSprint(9, 5, MONDAY, MONDAY + 18, [{ StoryID: 1, State: STORY_STATE_FINISHED, Rank: 1, AddedAt: 0, DoneAt: sUnixTimeAt(MONDAY + 9) }])
    const weeks = personWeeks(rows, new Map([[5, [threeWeekSprint]]]), MONDAY, MONDAY + 16)
    expect(weeks.map((week) => [week.startDay, week.assignedHours, week.finishedHours]))
      .toEqual([[MONDAY, 0, 0], [MONDAY + 7, 20, 20], [MONDAY + 14, 0, 0], [MONDAY + 21, 0, 0]])
  })

  test('unfinished stories fill the weeks of their sprint up to the capacity, the last week takes the rest', () => {
    const rows = [makeRow(1, 5, 24), makeRow(2, 5, 24), makeRow(3, 5, 8), makeRow(4, 5, 30), makeRow(5, 5, 30)]
    const twoWeekSprint = makeSprint(9, 5, MONDAY, MONDAY + 11, [
      { StoryID: 1, State: STORY_STATE_IN_DEVELOPMENT, Rank: 1, AddedAt: 0 },
      { StoryID: 2, State: 1, Rank: 2, AddedAt: 0 },
      { StoryID: 3, State: 1, Rank: 3, AddedAt: 0 },
      { StoryID: 4, State: 1, Rank: 4, AddedAt: 0 },
      { StoryID: 5, State: 1, Rank: 5, AddedAt: 0 },
    ])
    const weeks = personWeeks(rows, new Map([[5, [twoWeekSprint]]]), MONDAY, MONDAY + 1)
    expect(storiesByWeek(weeks)).toEqual([[MONDAY, [1, 3]], [MONDAY + 7, [2, 4, 5]]])
    expect(weeks.map((week) => week.assignedHours)).toEqual([32, 84, 0, 0])
  })

  test('unfinished work is never placed before the current week; an overdue sprint keeps it in its last week', () => {
    const rows = [makeRow(1, 5, 4), makeRow(2, 6, 4)]
    const sprintsByProject = new Map([
      [5, [makeSprint(1, 5, MONDAY, MONDAY + 18, [{ StoryID: 1, State: 1, Rank: 1, AddedAt: 0 }])]],
      [6, [makeSprint(2, 6, MONDAY, MONDAY + 4, [{ StoryID: 2, State: 1, Rank: 1, AddedAt: 0 }])]],
    ])
    expect(storiesByWeek(personWeeks(rows, sprintsByProject, MONDAY, MONDAY + 8))).toEqual([[MONDAY, [2]], [MONDAY + 7, [1]]])
  })

  test('the capacity is shared across projects and counts what was already finished that week', () => {
    const rows = [makeRow(1, 5, 30), makeRow(2, 6, 20)]
    const sprintsByProject = new Map([
      [5, [makeSprint(1, 5, MONDAY, MONDAY + 4, [{ StoryID: 1, State: STORY_STATE_FINISHED, Rank: 1, AddedAt: 0, DoneAt: sUnixTimeAt(MONDAY + 1) }])]],
      [6, [makeSprint(2, 6, MONDAY, MONDAY + 11, [{ StoryID: 2, State: 1, Rank: 1, AddedAt: 0 }])]],
    ])
    expect(storiesByWeek(personWeeks(rows, sprintsByProject, MONDAY, MONDAY + 2))).toEqual([[MONDAY, [1]], [MONDAY + 7, [2]]])
  })

  test('a carried-over story sits once, under its latest sprint', () => {
    const rows = [makeRow(1, 5, 10)]
    const sprints = [
      makeSprint(1, 5, MONDAY, MONDAY + 4, [{ StoryID: 1, State: 1, Rank: 1, AddedAt: 0, CarriedToSprintID: 2 }]),
      makeSprint(2, 5, MONDAY + 7, MONDAY + 11, [{ StoryID: 1, State: STORY_STATE_IN_DEVELOPMENT, Rank: 1, AddedAt: 0 }]),
    ]
    const weeks = personWeeks(rows, new Map([[5, sprints]]), MONDAY, MONDAY + 2)
    expect(storiesByWeek(weeks)).toEqual([[MONDAY + 7, [1]]])
    expect(weeks[1].projects[0].sprints.map((weekSprint) => weekSprint.sprint.ID)).toEqual([2])
  })

  test('groups each week by project, then sprint, with the project hours and the stories by number', () => {
    const rows = [makeRow(3, 6, 4), makeRow(2, 5, 6), makeRow(1, 5, 2), makeRow(4, 5, 1)]
    const sprintsByProject = new Map([
      [5, [
        makeSprint(1, 5, MONDAY, MONDAY + 4, [{ StoryID: 2, State: 1, Rank: 1, AddedAt: 0 }, { StoryID: 1, State: 1, Rank: 2, AddedAt: 0 }]),
        makeSprint(7, 5, MONDAY + 2, MONDAY + 9, [{ StoryID: 4, State: 1, Rank: 1, AddedAt: 0 }]),
      ]],
      [6, [makeSprint(2, 6, MONDAY, MONDAY + 4, [{ StoryID: 3, State: 1, Rank: 1, AddedAt: 0 }])]],
    ])
    const [week] = personWeeks(rows, sprintsByProject, MONDAY, MONDAY)
    expect(week.projects.map((project) => [
      project.projectID, project.assignedHours,
      project.sprints.map((weekSprint) => [weekSprint.sprint.ID, weekSprint.stories.map((entry) => entry.row.story.ID)]),
    ])).toEqual([[5, 9, [[1, [1, 2]], [7, [4]]]], [6, 4, [[2, [3]]]]])
  })

  test('opens on the week before the current one', () => {
    expect(defaultFirstWeekStartDay(MONDAY + 3)).toBe(MONDAY - 7)
  })

  test('the meter scale is at least the weekly capacity', () => {
    expect(weekMeterScale([])).toBe(WEEKLY_CAPACITY_HOURS)
    expect(weekMeterScale([{ startDay: MONDAY, assignedHours: 52, finishedHours: 10, projects: [] }])).toBe(52)
  })
})

describe('personGantt', () => {
  test('groups the planned stories under their project and counts the unplanned ones', () => {
    const rows = [makeRow(1, 6, 4), makeRow(2, 5, 4), makeRow(3, 5, 4)]
    const sprintsByProject = new Map([
      [5, [makeSprint(1, 5, MONDAY, MONDAY + 4, [{ StoryID: 2, State: 1, Rank: 1, AddedAt: 0 }])]],
      [6, [makeSprint(2, 6, MONDAY + 7, MONDAY + 11, [{ StoryID: 1, State: 1, Rank: 1, AddedAt: 0 }])]],
    ])
    const gantt = personGantt(rows, sprintsByProject, (text) => text)
    expect(gantt.rows.map((row) => row.id)).toEqual(['project-5', 'story-2', 'project-6', 'story-1'])
    expect(gantt.rows[0].bars[0]).toMatchObject({ start: MONDAY, end: MONDAY + 4 })
    expect(gantt.unplannedCount).toBe(1)
  })
})

// A SUnixTime at a given number of days before now.
const nowUnixMillis = Date.UTC(2026, 9, 11)
const daysAgo = (days: number) => Math.floor((nowUnixMillis / 1000 - 1e9) / 2) - days * 43_200

describe('recentAssignments', () => {
  test('keeps the active ones of the window, newest first', () => {
    const assignments: IStoryAssignee[] = [
      { StoryID: 1, ProjectID: 5, ss: 1, upd: daysAgo(30) },
      { StoryID: 2, ProjectID: 5, ss: 1, upd: daysAgo(ASSIGNED_STORIES_DAYS + 1) },
      { StoryID: 3, ProjectID: 6, ss: 1, upd: daysAgo(2) },
      { StoryID: 4, ProjectID: 6, ss: 0, upd: daysAgo(1) },
    ]
    expect(recentAssignments(assignments, nowUnixMillis).map((assignment) => assignment.StoryID)).toEqual([3, 1])
  })
})

describe('storyIDsByProject', () => {
  test('groups the story IDs by project', () => {
    expect(storyIDsByProject([{ StoryID: 1, ProjectID: 5, ss: 1 }, { StoryID: 3, ProjectID: 6, ss: 1 }, { StoryID: 2, ProjectID: 5, ss: 1 }]))
      .toEqual({ 5: [1, 2], 6: [3] })
  })
})

describe('assignedStoryRows', () => {
  test('joins story and project, skipping deleted or missing stories', () => {
    const story = (ID: number, ss = 1) => ({ ID, ProjectID: 5, ss }) as IStory
    const storiesByProject = new Map([[5, new Map([[1, story(1)], [2, story(2, 0)]])]])
    const projectsByID = new Map([[5, { ID: 5, Name: 'Catalog' } as IProject]])
    const assignments: IStoryAssignee[] = [
      { StoryID: 1, ProjectID: 5, ss: 1, upd: 100 }, { StoryID: 2, ProjectID: 5, ss: 1, upd: 90 }, { StoryID: 3, ProjectID: 5, ss: 1, upd: 80 },
    ]
    const rows = assignedStoryRows(assignments, storiesByProject, projectsByID)
    expect(rows.map((row) => [row.story.ID, row.project?.Name, row.assignedAt])).toEqual([[1, 'Catalog', 100]])
  })
})
