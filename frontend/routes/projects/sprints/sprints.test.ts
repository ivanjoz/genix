import { describe, expect, test } from 'bun:test';
import {
  SPRINT_STATE_ACTIVE, SPRINT_STATE_CLOSED, SPRINT_STATE_PLANNED, STORY_STATES, STORY_STATE_FINISHED, STORY_STATE_IN_DEVELOPMENT,
  STORY_STATE_IN_REVIEW, STORY_STATE_PENDING, STORY_STATE_STOPPED, rankBetween, type ISprint, type ISprintStory, type IStory,
} from '../project.ts';
import { makeEmptyStory } from '../story/story.ts';
import {
  PLAN_BY_STATE, applyCardMove, cardsOfAssignees, sprintAssigneeIDs, defaultCarryOvers, endOfColumnPosition, isNotCompleted, makeEmptySprint, needsMoveReason,
  planGroupsByAssignee, planGroupsByState, plannableStories, sortSprints, sprintBurndown, sprintCards, sprintDaysLeft, sprintMoveRequest,
  sprintProgress, sprintSaveRequest, sprintStateShares, validateSprint,
} from './sprints.ts';

const makeStory = (overrides: Partial<IStory>): IStory => ({ ...makeEmptyStory(1), Title: `Story ${overrides.ID}`, ...overrides })
const makeSprintStory = (StoryID: number, State: number, Rank: number): ISprintStory => ({ StoryID, State, Rank, AddedAt: 1 })
const makeSprint = (overrides: Partial<ISprint>): ISprint => ({
  ProjectID: 1, ID: 1, Name: 'Sprint 1', StartDate: 100, EndDate: 113, State: SPRINT_STATE_ACTIVE, Stories: [], ss: 1, ...overrides,
})

const stories = [
  makeStory({ ID: 1, EstimatedHours: 4, AssigneeID: 7 }),
  makeStory({ ID: 2, EstimatedHours: 2.5 }),
  makeStory({ ID: 3, EstimatedHours: 3, State: STORY_STATE_FINISHED }),
  makeStory({ ID: 4, ss: 0 }),
]
const storiesByID = new Map(stories.map((story) => [story.ID, story]))
const activeSprint = makeSprint({
  Stories: [
    makeSprintStory(1, STORY_STATE_PENDING, 2),
    // A rank of 0 arrives without the field.
    { StoryID: 2, State: STORY_STATE_PENDING, AddedAt: 1 } as ISprintStory,
    makeSprintStory(3, STORY_STATE_FINISHED, 1),
    makeSprintStory(4, STORY_STATE_PENDING, 3),
  ],
})

describe('the board', () => {
  test('cards go by rank and skip deleted stories', () => {
    expect(sprintCards(activeSprint, storiesByID).map((card) => card.story.ID)).toEqual([2, 3, 1])
  })
  test('progress counts stories and estimated hours', () => {
    expect(sprintProgress(sprintCards(activeSprint, storiesByID))).toEqual({ storiesCount: 3, finishedCount: 1, committedHours: 9.5, finishedHours: 3 })
  })
  test('assignees go by name with the unassigned last, and filter the cards', () => {
    const cards = sprintCards(activeSprint, storiesByID)
    expect(sprintAssigneeIDs(cards, (userID) => `User ${userID}`)).toEqual([7, 0])
    expect(cardsOfAssignees(cards, [7]).map((card) => card.story.ID)).toEqual([1])
    expect(cardsOfAssignees(cards, [0]).map((card) => card.story.ID)).toEqual([2, 3])
    expect(cardsOfAssignees(cards, [])).toHaveLength(3)
  })
  test('state shares split the committed hours by state', () => {
    // 9.5 committed hours: 3 finished, 6.5 pending, none stopped.
    const shares = sprintStateShares(sprintCards(activeSprint, storiesByID), [STORY_STATE_FINISHED, STORY_STATE_STOPPED])
    expect(shares.map((share) => share.hours)).toEqual([3, 0])
    expect(shares[0].percent).toBeCloseTo(31.58, 2)
    expect(sprintStateShares([], [STORY_STATE_FINISHED])[0].percent).toBe(0)
  })
  test('days left count today and the end day', () => {
    expect(sprintDaysLeft(activeSprint, 113)).toBe(1)
    expect(sprintDaysLeft(activeSprint, 120)).toBe(0)
  })
  test('only the unfinished cards of a closed sprint are not completed', () => {
    const closedSprint = { ...activeSprint, State: SPRINT_STATE_CLOSED }
    expect(isNotCompleted(closedSprint, makeSprintStory(1, STORY_STATE_PENDING, 1))).toBe(true)
    expect(isNotCompleted(closedSprint, makeSprintStory(3, STORY_STATE_FINISHED, 1))).toBe(false)
    expect(isNotCompleted(activeSprint, makeSprintStory(1, STORY_STATE_PENDING, 1))).toBe(false)
  })
})

describe('moving a card', () => {
  test('rankBetween mirrors the backend', () => {
    expect(rankBetween(1, 2)).toBe(1.5)
    expect(rankBetween(4, null)).toBe(5)
    expect(rankBetween(null, 4)).toBe(3)
    expect(rankBetween(null, null)).toBe(1)
  })
  test('a drop between two cards takes their midpoint and the new state', () => {
    const moveRequest = sprintMoveRequest(activeSprint, 1, STORY_STATE_IN_DEVELOPMENT, { previousID: '3', nextID: '4' })
    const movedCard = applyCardMove(activeSprint, moveRequest).Stories?.find((sprintStory) => sprintStory.StoryID === 1)
    expect(movedCard).toMatchObject({ State: STORY_STATE_IN_DEVELOPMENT, Rank: 2 })
  })
  test('the mobile menu sends the card to the end of its new column', () => {
    const cards = sprintCards(activeSprint, storiesByID)
    expect(endOfColumnPosition(cards, STORY_STATE_PENDING, 2)).toEqual({ previousID: '1', nextID: '' })
    expect(endOfColumnPosition(cards, STORY_STATE_STOPPED, 2)).toEqual({ previousID: '', nextID: '' })
  })
  test('only a move into Stopped asks the reason', () => {
    expect(needsMoveReason(makeSprintStory(1, STORY_STATE_PENDING, 1), STORY_STATE_STOPPED)).toBe(true)
    expect(needsMoveReason(makeSprintStory(1, STORY_STATE_STOPPED, 1), STORY_STATE_STOPPED)).toBe(false)
    expect(needsMoveReason(makeSprintStory(1, STORY_STATE_PENDING, 1), STORY_STATE_IN_DEVELOPMENT)).toBe(false)
  })
})

describe('sprints', () => {
  const plannedSprint = makeSprint({ ID: 2, Name: 'Sprint 2', StartDate: 114, EndDate: 127, State: SPRINT_STATE_PLANNED })
  const closedSprint = makeSprint({ ID: 3, Name: 'Sprint 0', StartDate: 86, EndDate: 99, State: SPRINT_STATE_CLOSED })

  test('active first, then planned, then closed', () => {
    expect(sortSprints([closedSprint, plannedSprint, activeSprint]).map((sprint) => sprint.ID)).toEqual([1, 2, 3])
  })
  test('unfinished stories go to the next planned sprint, or back to the backlog', () => {
    expect(defaultCarryOvers(activeSprint, [activeSprint, plannedSprint])).toEqual([
      { StoryID: 1, ToSprintID: 2 }, { StoryID: 2, ToSprintID: 2 }, { StoryID: 4, ToSprintID: 2 },
    ])
    expect(defaultCarryOvers(activeSprint, [activeSprint]).every((carryOver) => carryOver.ToSprintID === 0)).toBe(true)
  })
  test('a new sprint follows the latest one with its length', () => {
    expect(makeEmptySprint(1, [activeSprint], 105)).toMatchObject({ StartDate: 114, EndDate: 127, State: SPRINT_STATE_PLANNED })
    expect(makeEmptySprint(1, [], 200)).toMatchObject({ StartDate: 200, EndDate: 213 })
    expect(makeEmptySprint(1, [closedSprint], 200)).toMatchObject({ StartDate: 200, EndDate: 213 })
  })
  test('a sprint needs its dates in order', () => {
    expect(validateSprint(makeSprint({}))).toBe('')
    expect(validateSprint(makeSprint({ EndDate: 99 }))).not.toBe('')
  })
})

describe('the planner', () => {
  test('offers stories neither finished, deleted, in another open sprint nor planned already', () => {
    const backlogStories = [...stories, makeStory({ ID: 5 }), makeStory({ ID: 6 })]
    const otherOpenSprint = makeSprint({ ID: 9, State: SPRINT_STATE_PLANNED, Stories: [makeSprintStory(5, STORY_STATE_PENDING, 1)] })
    expect(plannableStories(backlogStories, [activeSprint, otherOpenSprint], 1, new Set([1])).map((story) => story.ID)).toEqual([2, 6])
  })
  test('by state, a story just added is Pending, a state set while planning wins, and every state shows', () => {
    const stateChanges = new Map([[2, { State: STORY_STATE_STOPPED, Reason: 'Waiting' }]])
    const groups = planGroupsByState([stories[0], stories[1], makeStory({ ID: 8 })], activeSprint, stateChanges, STORY_STATES)
    expect(groups.map((group) => [group.id, group.stories.map((story) => story.ID)])).toEqual([
      [STORY_STATE_PENDING, [1, 8]], [STORY_STATE_IN_DEVELOPMENT, []], [STORY_STATE_STOPPED, [2]], [STORY_STATE_IN_REVIEW, []], [STORY_STATE_FINISHED, []],
    ])
    expect(PLAN_BY_STATE).toBe(1)
  })
  test('by assignee, the planning assignment wins over the story\'s own', () => {
    const groups = planGroupsByAssignee([stories[0], stories[1]], [{ UserID: 7, Role: 2 }], new Map([[1, 0], [2, 7]]), (userID) => `User ${userID}`)
    expect(groups.map((group) => [group.id, group.stories.map((story) => story.ID), group.estimatedHours])).toEqual([[7, [2], 2.5], [0, [1], 4]])
  })
  test('the save sends only the assignees that changed, and the states, of planned stories', () => {
    const stateChanges = new Map([[1, { State: STORY_STATE_STOPPED, Reason: 'Waiting' }], [5, { State: STORY_STATE_IN_REVIEW, Reason: '' }]])
    const saveRequest = sprintSaveRequest(makeSprint({ Name: '  Sprint 1 ' }), [1, 2], new Map([[1, 7], [2, 7], [5, 7]]), stateChanges, storiesByID)
    expect(saveRequest.Sprint.Name).toBe('Sprint 1')
    expect(saveRequest.StoryAssignees).toEqual([{ StoryID: 2, AssigneeID: 7 }])
    expect(saveRequest.StoryStates).toEqual([{ StoryID: 1, State: STORY_STATE_STOPPED, Reason: 'Waiting' }])
  })
})

describe('burndown', () => {
  // In these fixtures a SUnixTime is the day itself.
  const dayOf = (sUnixTime: number) => sUnixTime
  const burndownSprint = makeSprint({
    StartDate: 100, EndDate: 104,
    Stories: [
      // Planned before the start, finished on day 101.
      { StoryID: 1, State: STORY_STATE_FINISHED, Rank: 1, AddedAt: 90, DoneAt: 101 },
      // Added on day 102.
      { StoryID: 2, State: STORY_STATE_PENDING, Rank: 2, AddedAt: 102 },
      // Finished and reopened: no DoneAt, it still counts.
      { StoryID: 3, State: STORY_STATE_IN_DEVELOPMENT, Rank: 3, AddedAt: 90 },
    ],
  })
  const burndownCards = sprintCards(burndownSprint, storiesByID)

  test('a story leaves the day it is finished and joins the day it is added; days to come are empty', () => {
    const burndown = sprintBurndown(burndownSprint, burndownCards, 103, dayOf)
    expect(burndown.days).toEqual([100, 101, 102, 103, 104])
    expect(burndown.remaining).toEqual([7, 3, 5.5, 5.5, null])
  })
  test('the ideal line goes from the first day commitment to 0', () => {
    expect(sprintBurndown(burndownSprint, burndownCards, 103, dayOf).ideal).toEqual([7, 5.25, 3.5, 1.75, 0])
  })
  test('a closed sprint shows every day', () => {
    const closed = sprintBurndown({ ...burndownSprint, State: SPRINT_STATE_CLOSED }, burndownCards, 200, dayOf)
    expect(closed.remaining).toEqual([7, 3, 5.5, 5.5, 5.5])
  })
  test('a story added after the end counts on the last day', () => {
    const lateSprint = makeSprint({ StartDate: 100, EndDate: 102, State: SPRINT_STATE_CLOSED, Stories: [{ StoryID: 1, State: STORY_STATE_PENDING, Rank: 1, AddedAt: 150 }] })
    expect(sprintBurndown(lateSprint, sprintCards(lateSprint, storiesByID), 200, dayOf).remaining).toEqual([0, 0, 4])
  })
})
