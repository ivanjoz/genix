// The Sprints page's rules: the board's columns and cards, a card move, the progress of a sprint,
// the carry-over of a closing sprint and the planner's groups. Pure: no Svelte, no fetch.
import type { IDropPosition } from '#libs/drag-drop.ts';
import {
  SPRINT_STATE_ACTIVE, SPRINT_STATE_CLOSED, SPRINT_STATE_PLANNED, STATUS_ACTIVE, STORY_STATE_FINISHED, STORY_STATE_IN_DEVELOPMENT,
  STORY_STATE_IN_REVIEW, STORY_STATE_PENDING, STORY_STATE_STOPPED, isSprintOpen, rankBetween, type IProjectMember, type ISprint, type ISprintStory,
  type IStory,
} from '../project.ts';

export const SPRINT_STATES: Record<number, { name: string, badgeCss: string }> = {
  [SPRINT_STATE_PLANNED]: { name: 'Planned|Planificado', badgeCss: 'bg-blue-bg-strong text-blue-fg' },
  [SPRINT_STATE_ACTIVE]: { name: 'Active|Activo', badgeCss: 'bg-green-bg-strong text-green-fg' },
  [SPRINT_STATE_CLOSED]: { name: 'Closed|Cerrado', badgeCss: 'bg-surface-strong text-fg-soft' },
}

// The board has 4 columns; the third stacks Blocked (on top, red) and In review (below) as two
// separate containers, since cards pass through them quickly.
export const BOARD_COLUMNS: { id: number, name: string, states: number[] }[] = [
  { id: 1, name: 'Pending|Pendiente', states: [STORY_STATE_PENDING] },
  { id: 2, name: 'In development|En desarrollo', states: [STORY_STATE_IN_DEVELOPMENT] },
  { id: 3, name: 'Blocked / In review|Bloqueado / En revisión', states: [STORY_STATE_STOPPED, STORY_STATE_IN_REVIEW] },
  { id: 4, name: 'Finished|Finalizado', states: [STORY_STATE_FINISHED] },
]

// A sprint's default length when it has no predecessor to follow.
export const DEFAULT_SPRINT_DAYS = 14

export interface ISprintCard {
  sprintStory: ISprintStory
  story: IStory
}

// The API omits zero values: a rank of 0 arrives missing.
const cardRank = (sprintStory: ISprintStory) => sprintStory.Rank || 0

// sprintCards pairs the sprint's stories with their backlog record, by rank. A story deleted from
// the backlog no longer shows.
export const sprintCards = (sprint: ISprint, storiesByID: Map<number, IStory>): ISprintCard[] =>
  [...(sprint.Stories || [])]
    .sort((left, right) => cardRank(left) - cardRank(right) || left.StoryID - right.StoryID)
    .flatMap((sprintStory) => {
      const story = storiesByID.get(sprintStory.StoryID)
      return story && story.ss !== 0 ? [{ sprintStory, story }] : []
    })

export const cardsInState = (cards: ISprintCard[], state: number) => cards.filter((card) => card.sprintStory.State === state)

// isNotCompleted: the card of a closed sprint that ended unfinished.
export const isNotCompleted = (sprint: ISprint, sprintStory: ISprintStory) =>
  sprint.State === SPRINT_STATE_CLOSED && sprintStory.State !== STORY_STATE_FINISHED

export interface ISprintProgress {
  storiesCount: number
  finishedCount: number
  committedHours: number
  finishedHours: number
}

export const sprintProgress = (cards: ISprintCard[]): ISprintProgress => {
  const finishedCards = cardsInState(cards, STORY_STATE_FINISHED)
  const sumHours = (summedCards: ISprintCard[]) => summedCards.reduce((total, card) => total + (card.story.EstimatedHours || 0), 0)
  return {
    storiesCount: cards.length, finishedCount: finishedCards.length, committedHours: sumHours(cards), finishedHours: sumHours(finishedCards),
  }
}

// sprintAssigneeIDs: who holds cards in the sprint, by name, with the unassigned (0) last. It feeds
// the board's people filter and its "By assignee" lanes.
export const sprintAssigneeIDs = (cards: ISprintCard[], userName: (userID: number) => string): number[] =>
  [...new Set(cards.map((card) => card.story.AssigneeID || 0))].sort((left, right) => (left === 0 ? 1 : 0) - (right === 0 ? 1 : 0)
    || userName(left).localeCompare(userName(right)))

// cardsOfAssignees keeps the cards of the chosen people (0 = unassigned); choosing nobody shows all.
export const cardsOfAssignees = (cards: ISprintCard[], assigneeIDs: number[]) =>
  assigneeIDs.length === 0 ? cards : cards.filter((card) => assigneeIDs.includes(card.story.AssigneeID || 0))

// sprintStateShares: the percent of the committed hours each state holds, in the given order. A
// sprint without hours shares nothing (all 0).
export const sprintStateShares = (cards: ISprintCard[], states: number[]) => {
  const committedHours = sprintProgress(cards).committedHours
  return states.map((state) => {
    const stateHours = sprintProgress(cardsInState(cards, state)).committedHours
    return { state, hours: stateHours, percent: committedHours ? stateHours / committedHours * 100 : 0 }
  })
}

export interface ISprintBurndown {
  days: number[]
  /** From the hours committed on the first day down to 0 on the last. */
  ideal: number[]
  /** Hours left at the end of each day; null for the days still to come. */
  remaining: (number | null)[]
}

/**
 * The burndown of a sprint in estimated hours. A story counts from the day it was added (from the
 * first day when it was planned before) until the day it was finished; a reopened one has no
 * DoneAt, so it counts again. A story added mid-sprint raises the line on that day: the ideal line
 * keeps the first day's commitment, so the added scope shows. `dayOfSUnixTime` turns AddedAt /
 * DoneAt into the business day.
 */
export const sprintBurndown = (
  sprint: ISprint, cards: ISprintCard[], today: number, dayOfSUnixTime: (sUnixTime: number) => number,
): ISprintBurndown => {
  const days = Array.from({ length: Math.max(1, sprint.EndDate - sprint.StartDate + 1) }, (_, dayIndex) => sprint.StartDate + dayIndex)
  const lastShownDay = sprint.State === SPRINT_STATE_CLOSED ? sprint.EndDate : Math.min(today, sprint.EndDate)
  // A sprint planned on past dates gets its stories after its end: they count on its last day.
  const addedDay = (card: ISprintCard) => Math.min(sprint.EndDate, Math.max(sprint.StartDate, dayOfSUnixTime(card.sprintStory.AddedAt || 0)))
  const finishedDay = (card: ISprintCard) => card.sprintStory.State === STORY_STATE_FINISHED && card.sprintStory.DoneAt
    ? dayOfSUnixTime(card.sprintStory.DoneAt) : Infinity
  const hoursLeftOn = (day: number) => cards.reduce((total, card) =>
    addedDay(card) <= day && finishedDay(card) > day ? total + (card.story.EstimatedHours || 0) : total, 0)

  const committedHours = cards.reduce((total, card) => addedDay(card) === sprint.StartDate ? total + (card.story.EstimatedHours || 0) : total, 0)
  const lastDayIndex = Math.max(1, days.length - 1)
  return {
    days,
    ideal: days.map((_, dayIndex) => committedHours * (1 - dayIndex / lastDayIndex)),
    remaining: days.map((day) => day <= lastShownDay ? hoursLeftOn(day) : null),
  }
}

// sprintDaysLeft counts today and the end day: an active sprint ending today has 1 day left.
export const sprintDaysLeft = (sprint: ISprint, today: number) => Math.max(0, sprint.EndDate - today + 1)

// sortSprints: the active sprint first, then the planned ones by start, then the closed ones, the
// latest first.
export const sortSprints = (sprints: ISprint[]): ISprint[] => {
  const stateOrder = { [SPRINT_STATE_ACTIVE]: 0, [SPRINT_STATE_PLANNED]: 1, [SPRINT_STATE_CLOSED]: 2 } as Record<number, number>
  return sprints.filter((sprint) => sprint.ss !== 0).sort((left, right) => {
    const byState = (stateOrder[left.State] ?? 3) - (stateOrder[right.State] ?? 3)
    if (byState !== 0) { return byState }
    const byStart = left.State === SPRINT_STATE_CLOSED ? right.StartDate - left.StartDate : left.StartDate - right.StartDate
    return byStart || left.ID - right.ID
  })
}

export interface ISprintMoveRequest {
  ProjectID: number
  SprintID: number
  StoryID: number
  State: number
  PreviousStoryID: number
  NextStoryID: number
  Reason: string
}

export const sprintMoveRequest = (sprint: ISprint, storyID: number, state: number, dropPosition: IDropPosition, reason = ''): ISprintMoveRequest => ({
  ProjectID: sprint.ProjectID,
  SprintID: sprint.ID,
  StoryID: storyID,
  State: state,
  PreviousStoryID: Number(dropPosition.previousID) || 0,
  NextStoryID: Number(dropPosition.nextID) || 0,
  Reason: reason,
})

// applyCardMove is the move as the board shows it before the server answers: the same state and
// rank the backend will write.
export const applyCardMove = (sprint: ISprint, moveRequest: ISprintMoveRequest): ISprint => {
  const sprintStories = sprint.Stories || []
  const neighbourRank = (storyID: number) => {
    const neighbour = storyID !== moveRequest.StoryID ? sprintStories.find((sprintStory) => sprintStory.StoryID === storyID) : undefined
    return neighbour ? cardRank(neighbour) : null
  }
  const movedRank = rankBetween(neighbourRank(moveRequest.PreviousStoryID), neighbourRank(moveRequest.NextStoryID))
  return {
    ...sprint,
    Stories: sprintStories.map((sprintStory) => sprintStory.StoryID !== moveRequest.StoryID ? sprintStory
      : { ...sprintStory, State: moveRequest.State, Rank: movedRank }),
  }
}

// endOfColumnPosition: a card moved without dragging (the mobile menu) goes last in its new column.
export const endOfColumnPosition = (cards: ISprintCard[], state: number, movedStoryID: number): IDropPosition => {
  const lastCard = cardsInState(cards, state).filter((card) => card.story.ID !== movedStoryID).at(-1)
  return { previousID: lastCard ? String(lastCard.story.ID) : '', nextID: '' }
}

// needsMoveReason: moving a card to Stopped asks why, unless it was already stopped.
export const needsMoveReason = (sprintStory: ISprintStory, toState: number) =>
  toState === STORY_STATE_STOPPED && sprintStory.State !== STORY_STATE_STOPPED

export interface ICarryOver {
  StoryID: number
  // 0 = back to the backlog.
  ToSprintID: number
}

// The planned sprints that can take the unfinished stories of a closing sprint, the earliest first.
export const plannedSprints = (sprints: ISprint[]) => sortSprints(sprints).filter((sprint) => sprint.State === SPRINT_STATE_PLANNED)

// defaultCarryOvers sends every unfinished story to the next planned sprint, or back to the
// backlog when there is none.
export const defaultCarryOvers = (closingSprint: ISprint, sprints: ISprint[]): ICarryOver[] => {
  const nextSprintID = plannedSprints(sprints)[0]?.ID || 0
  return (closingSprint.Stories || [])
    .filter((sprintStory) => sprintStory.State !== STORY_STATE_FINISHED)
    .map((sprintStory) => ({ StoryID: sprintStory.StoryID, ToSprintID: nextSprintID }))
}

// makeEmptySprint starts the day after the latest sprint ends (today when there is none or it
// ended already) and lasts as long as that sprint did.
export const makeEmptySprint = (projectID: number, sprints: ISprint[], today: number): ISprint => {
  const latestSprint = sprints.filter((sprint) => sprint.ss !== 0).sort((left, right) => right.EndDate - left.EndDate)[0]
  const sprintDays = latestSprint ? latestSprint.EndDate - latestSprint.StartDate + 1 : DEFAULT_SPRINT_DAYS
  const startDate = latestSprint ? Math.max(today, latestSprint.EndDate + 1) : today
  return {
    ProjectID: projectID, ID: 0, Name: '', Goal: '', StartDate: startDate, EndDate: startDate + sprintDays - 1,
    State: SPRINT_STATE_PLANNED, Stories: [], ss: STATUS_ACTIVE,
  }
}

export const validateSprint = (sprint: ISprint): string => {
  if ((sprint.Name || '').trim().length > 80) { return 'The sprint name can have up to 80 characters.|El nombre del sprint puede tener hasta 80 caracteres.' }
  if ((sprint.Goal || '').trim().length > 500) { return 'The sprint goal can have up to 500 characters.|El objetivo del sprint puede tener hasta 500 caracteres.' }
  if (!sprint.StartDate || !sprint.EndDate || sprint.EndDate < sprint.StartDate) {
    return 'The sprint needs a start date and an end date after it.|El sprint necesita una fecha de inicio y una de fin posterior.'
  }
  return ''
}

// plannableStories are what the planner's backlog offers: stories neither finished nor in another
// open sprint, and not in this plan already. Ordered as the backlog.
export const plannableStories = (stories: IStory[], sprints: ISprint[], editedSprintID: number, plannedStoryIDs: Set<number>): IStory[] => {
  const storyIDsInOtherOpenSprints = new Set(sprints
    .filter((sprint) => sprint.ID !== editedSprintID && isSprintOpen(sprint))
    .flatMap((sprint) => (sprint.Stories || []).map((sprintStory) => sprintStory.StoryID)))
  return stories
    .filter((story) => story.ss !== 0 && story.State !== STORY_STATE_FINISHED
      && !storyIDsInOtherOpenSprints.has(story.ID) && !plannedStoryIDs.has(story.ID))
    .sort((left, right) => (left.BacklogRank || 0) - (right.BacklogRank || 0) || left.ID - right.ID)
}

export const PLAN_BY_STATE = 1
export const PLAN_BY_ASSIGNEE = 2

export interface IPlanGroup {
  // A story state, or an assignee's user ID (0 = unassigned).
  id: number
  name: string
  stories: IStory[]
  estimatedHours: number
}

const makePlanGroup = (id: number, name: string, stories: IStory[]): IPlanGroup => ({
  id, name, stories, estimatedHours: stories.reduce((total, story) => total + (story.EstimatedHours || 0), 0),
})

// A state set while planning (a story dropped on a state's group), saved with the plan.
export interface IPlanStateChange {
  State: number
  Reason: string
}

// planStoryState: the story's state in the plan. The one set while planning wins over the sprint's;
// a story just added is Pending.
export const planStoryState = (sprint: ISprint, stateChangeByStoryID: Map<number, IPlanStateChange>, storyID: number) =>
  stateChangeByStoryID.get(storyID)?.State
    || (sprint.Stories || []).find((sprintStory) => sprintStory.StoryID === storyID)?.State
    || STORY_STATE_PENDING

// planGroupsByState groups the plan by the stories' state in the plan. Every state shows, empty or
// not: each is a place to drop a story on.
export const planGroupsByState = (
  plannedStories: IStory[], sprint: ISprint, stateChangeByStoryID: Map<number, IPlanStateChange>, stateNames: Record<number, { name: string }>,
): IPlanGroup[] => {
  const states = [STORY_STATE_PENDING, STORY_STATE_IN_DEVELOPMENT, STORY_STATE_STOPPED, STORY_STATE_IN_REVIEW, STORY_STATE_FINISHED]
  return states.map((state) => makePlanGroup(state, stateNames[state].name,
    plannedStories.filter((story) => planStoryState(sprint, stateChangeByStoryID, story.ID) === state)))
}

// planGroupsByAssignee makes one group per team member, then "Unassigned". assigneeByStoryID holds
// the assignees set while planning, over the stories' own.
export const planGroupsByAssignee = (
  plannedStories: IStory[], members: IProjectMember[], assigneeByStoryID: Map<number, number>, userName: (userID: number) => string,
): IPlanGroup[] => {
  const assigneeOf = (story: IStory) => assigneeByStoryID.get(story.ID) ?? (story.AssigneeID || 0)
  return [
    ...members.map((member) => makePlanGroup(member.UserID, userName(member.UserID),
      plannedStories.filter((story) => assigneeOf(story) === member.UserID))),
    makePlanGroup(0, 'Unassigned|Sin responsable', plannedStories.filter((story) => !assigneeOf(story))),
  ]
}

// sprintSaveRequest is the edit mode's single save: the sprint, its whole plan, and the assignees
// and states that changed while planning (the backend skips a state the story already has).
export const sprintSaveRequest = (
  sprint: ISprint, plannedStoryIDs: number[], assigneeByStoryID: Map<number, number>, stateChangeByStoryID: Map<number, IPlanStateChange>,
  storiesByID: Map<number, IStory>,
) => ({
  Sprint: { ...sprint, Name: (sprint.Name || '').trim(), Goal: (sprint.Goal || '').trim(), Stories: undefined },
  StoryIDs: plannedStoryIDs,
  StoryAssignees: [...assigneeByStoryID]
    .filter(([storyID, assigneeID]) => plannedStoryIDs.includes(storyID) && (storiesByID.get(storyID)?.AssigneeID || 0) !== assigneeID)
    .map(([storyID, assigneeID]) => ({ StoryID: storyID, AssigneeID: assigneeID })),
  StoryStates: [...stateChangeByStoryID]
    .filter(([storyID]) => plannedStoryIDs.includes(storyID))
    .map(([storyID, stateChange]) => ({ StoryID: storyID, State: stateChange.State, Reason: stateChange.Reason })),
})
