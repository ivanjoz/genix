import { describe, expect, test } from 'bun:test';
import {
  SPRINT_STATE_ACTIVE, SPRINT_STATE_CLOSED, STORY_STATE_FINISHED, STORY_STATE_PENDING,
  type IEpic, type ISprint, type IStory,
} from '../project.ts';
import { makeEmptyStory } from '../story/story.ts';
import {
  NONE_FILTER, applyBacklogRank, backlogRankRequest, filterBacklogStories, formatHours, groupBacklog, hasActiveFilters,
  isStoryPlannable, makeBacklogFilters, type IBacklogFilters,
} from './backlog.ts';

const makeStory = (overrides: Partial<IStory>): IStory => ({ ...makeEmptyStory(1), ...overrides })
const makeEpic = (ID: number, Name: string): IEpic => ({ ProjectID: 1, ID, Name, Color: 'blue', ss: 1 })
const makeSprint = (ID: number, State: number): ISprint => ({ ProjectID: 1, ID, Name: `Sprint ${ID}`, StartDate: 0, EndDate: 0, State, ss: 1 })

const stories = [
  makeStory({ ID: 1, Title: 'Login con SSO', BacklogRank: 3, EpicID: 10, EstimatedHours: 4 }),
  makeStory({ ID: 2, Title: 'Reporte de riego', BacklogRank: 1, EpicID: 10, EstimatedHours: 2.5, AssigneeID: 7 }),
  makeStory({ ID: 3, Title: 'Corregir fecha', BacklogRank: 2, Type: 2, State: STORY_STATE_FINISHED }),
  makeStory({ ID: 4, Title: 'Exportar Excel', BacklogRank: 4, EpicID: 99, State: STORY_STATE_PENDING, LastSprintID: 1 }),
]
const sprintsByID = new Map([[1, makeSprint(1, SPRINT_STATE_ACTIVE)], [2, makeSprint(2, SPRINT_STATE_CLOSED)]])

describe('filterBacklogStories', () => {
  test('no filter keeps every story', () => {
    expect(filterBacklogStories(stories, makeBacklogFilters(), sprintsByID)).toHaveLength(4)
    expect(hasActiveFilters(makeBacklogFilters())).toBe(false)
  })
  test('a cleared select (null) is no filter', () => {
    const clearedFilters = { ...makeBacklogFilters(), typeID: null, epicID: null } as unknown as IBacklogFilters
    expect(filterBacklogStories(stories, clearedFilters, sprintsByID)).toHaveLength(4)
    expect(hasActiveFilters(clearedFilters)).toBe(false)
  })
  test('the text matches the code, the title and the summary', () => {
    const filters = { ...makeBacklogFilters(), text: 'h03' }
    expect(filterBacklogStories(stories, filters, sprintsByID).map((story) => story.ID)).toEqual([3])
    expect(filterBacklogStories(stories, { ...filters, text: 'RIEGO' }, sprintsByID).map((story) => story.ID)).toEqual([2])
  })
  test('NONE_FILTER picks the stories without an epic or without an assignee', () => {
    const withoutEpic = filterBacklogStories(stories, { ...makeBacklogFilters(), epicID: NONE_FILTER }, sprintsByID)
    expect(withoutEpic.map((story) => story.ID)).toEqual([3])
    const unassigned = filterBacklogStories(stories, { ...makeBacklogFilters(), assigneeID: NONE_FILTER }, sprintsByID)
    expect(unassigned.map((story) => story.ID)).toEqual([1, 3, 4])
  })
  test('the "Not planned" state matches stories the API sent without a State', () => {
    const unplannedStory = { ...makeStory({ ID: 5 }), State: undefined } as unknown as IStory
    const filters = { ...makeBacklogFilters(), state: NONE_FILTER }
    expect(filterBacklogStories([unplannedStory], filters, sprintsByID)).toHaveLength(1)
  })
  test('to plan only drops finished stories and stories in an open sprint', () => {
    const filters = { ...makeBacklogFilters(), onlyPlannable: true }
    expect(filterBacklogStories(stories, filters, sprintsByID).map((story) => story.ID)).toEqual([1, 2])
  })
})

describe('isStoryPlannable', () => {
  test('a story whose last sprint closed can be planned again', () => {
    expect(isStoryPlannable(makeStory({ State: STORY_STATE_PENDING, LastSprintID: 2 }), sprintsByID)).toBe(true)
    expect(isStoryPlannable(makeStory({ State: STORY_STATE_PENDING, LastSprintID: 1 }), sprintsByID)).toBe(false)
  })
})

describe('groupBacklog', () => {
  const epics = [makeEpic(20, 'Reportes'), makeEpic(10, 'Acceso')]

  test('one group per epic in creation order, "No epic" last, stories by rank', () => {
    const groups = groupBacklog(stories, epics, true)
    expect(groups.map((group) => group.epic?.ID ?? 0)).toEqual([10, 20, 0])
    expect(groups[0].stories.map((story) => story.ID)).toEqual([2, 1])
    expect(groups[0].estimatedHours).toBe(6.5)
  })
  test('stories of an unknown epic fall into "No epic"', () => {
    const groups = groupBacklog(stories, epics, true)
    expect(groups[2].stories.map((story) => story.ID)).toEqual([3, 4])
  })
  test('empty epics hide while filtering', () => {
    expect(groupBacklog(stories, epics, false).map((group) => group.epic?.ID ?? 0)).toEqual([10, 0])
  })
})

describe('backlogRankRequest', () => {
  test('a missing neighbour is 0', () => {
    expect(backlogRankRequest(1, 4, 10, { previousID: '2', nextID: '' })).toEqual({
      ProjectID: 1, StoryID: 4, PreviousStoryID: 2, NextStoryID: 0, EpicID: 10,
    })
  })
})

describe('applyBacklogRank', () => {
  const rankedStories = [makeStory({ ID: 1, BacklogRank: 1 }), makeStory({ ID: 2, BacklogRank: 2 }), makeStory({ ID: 3, BacklogRank: 3, EpicID: 5 })]
  test('between two neighbours takes the midpoint and the drop group epic', () => {
    const movedStory = applyBacklogRank(rankedStories[2], rankedStories, backlogRankRequest(1, 3, 0, { previousID: '1', nextID: '2' }))
    expect(movedStory.BacklogRank).toBe(1.5)
    expect(movedStory.EpicID).toBe(0)
  })
  test('the moved story is never its own neighbour', () => {
    expect(applyBacklogRank(rankedStories[0], rankedStories, backlogRankRequest(1, 1, 0, { previousID: '1', nextID: '' })).BacklogRank).toBe(1)
  })
})

test('formatHours shows one decimal only when needed', () => {
  expect(formatHours(6)).toBe('6 h')
  expect(formatHours(6.5)).toBe('6.5 h')
})
