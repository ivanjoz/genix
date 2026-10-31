// The Backlog tab's rules: which stories show, how they group by epic and in which order, and what a
// drop means. Pure: no Svelte, no fetch.
import type { IDropPosition } from '#libs/drag-drop.ts';
import {
  STORY_STATES, STORY_STATE_FINISHED, STORY_TYPES, isSprintOpen, rankBetween, storyCode,
  type IEpic, type IProjectMember, type ISprint, type IStory,
} from '../project.ts';

// No filter is a select left empty: 0, or null once the user clears it (genix-ui's SearchSelect
// saves null). Its placeholder reads "All ...".
// The value for stories without an epic, unassigned or not planned (the zero of each field): negative,
// since 0 is "no filter".
export const NONE_FILTER = -1

export interface IBacklogFilters {
  text: string
  typeID: number
  state: number
  assigneeID: number
  epicID: number
  onlyPlannable: boolean
}

export const makeBacklogFilters = (): IBacklogFilters => ({
  text: '', typeID: 0, state: 0, assigneeID: 0, epicID: 0, onlyPlannable: false,
})

export const hasActiveFilters = (filters: IBacklogFilters) =>
  !!filters.text || !!filters.typeID || !!filters.state || !!filters.assigneeID || !!filters.epicID || filters.onlyPlannable

// The options of the filter selects, shared by the selects and the active filter tags. genix-ui
// shows option names as given, so they come translated. "All" is no option: it is the empty select,
// shown by its placeholder.
export const backlogFilterOptions = (
  epics: IEpic[], members: IProjectMember[], userName: (userID: number) => string, translate: (text: string) => string,
) => ({
  typeOptions: STORY_TYPES.map((type) => ({ id: type.id, name: translate(type.name) })),
  stateOptions: Object.entries(STORY_STATES).map(([state, stateInfo]) => ({ id: Number(state) || NONE_FILTER, name: translate(stateInfo.name) })),
  epicOptions: [{ ID: NONE_FILTER, Name: translate('No epic|Sin épica') }, ...epics.map((epic) => ({ ID: epic.ID, Name: epic.Name }))],
  assigneeOptions: [{ ID: NONE_FILTER, Name: translate('Unassigned|Sin responsable') },
    ...members.map((member) => ({ ID: member.UserID, Name: userName(member.UserID) }))],
})

// isStoryPlannable: a story a sprint can still take, neither finished nor in an open sprint.
export const isStoryPlannable = (story: IStory, sprintsByID: Map<number, ISprint>) => {
  if (story.State === STORY_STATE_FINISHED) { return false }
  const lastSprint = story.LastSprintID ? sprintsByID.get(story.LastSprintID) : undefined
  return !lastSprint || !isSprintOpen(lastSprint)
}

export const matchesIDFilter = (storyValueID: number | undefined, filterID: number) => {
  if (!filterID) { return true }
  if (filterID === NONE_FILTER) { return !storyValueID }
  return storyValueID === filterID
}

export const filterBacklogStories = (
  stories: IStory[], filters: IBacklogFilters, sprintsByID: Map<number, ISprint>,
): IStory[] => {
  const filterText = filters.text.trim().toLowerCase()
  return stories.filter((story) => {
    if (filters.typeID && story.Type !== filters.typeID) { return false }
    // "Not planned" is state 0: its option is NONE_FILTER.
    if (!matchesIDFilter(story.State, filters.state)) { return false }
    if (!matchesIDFilter(story.AssigneeID, filters.assigneeID)) { return false }
    if (!matchesIDFilter(story.EpicID, filters.epicID)) { return false }
    if (filters.onlyPlannable && !isStoryPlannable(story, sprintsByID)) { return false }
    if (!filterText) { return true }
    return [storyCode(story), story.Title, story.Summary].join(' ').toLowerCase().includes(filterText)
  })
}

// sortByBacklogRank: the manual order, ties (a story ranked into an equal slot) by ID, which is number order inside a project. The API
// omits a rank of 0.
export const sortByBacklogRank = (stories: IStory[]): IStory[] =>
  [...stories].sort((left, right) => (left.BacklogRank || 0) - (right.BacklogRank || 0) || left.ID - right.ID)

export interface IBacklogGroup {
  // null is the group of the stories without an epic, always last.
  epic: IEpic | null
  stories: IStory[]
  estimatedHours: number
}

export const sumEstimatedHours = (stories: IStory[]) => stories.reduce((total, story) => total + (story.EstimatedHours || 0), 0)

// groupBacklog makes one group per epic (in creation order) and the "No epic" group last. With
// showEmptyGroups every epic shows, so a story can be dropped into an empty one.
export const groupBacklog = (stories: IStory[], epics: IEpic[], showEmptyGroups: boolean): IBacklogGroup[] => {
  const storiesByEpicID = new Map<number, IStory[]>()
  for (const story of sortByBacklogRank(stories)) {
    const epicID = story.EpicID || 0
    storiesByEpicID.set(epicID, [...(storiesByEpicID.get(epicID) || []), story])
  }
  const epicGroups: IBacklogGroup[] = [...epics].sort((left, right) => left.ID - right.ID)
    .map((epic) => ({ epic, stories: storiesByEpicID.get(epic.ID) || [], estimatedHours: 0 }))
  const knownEpicIDs = new Set(epics.map((epic) => epic.ID))
  // Stories of an epic that is no longer listed fall into "No epic".
  const storiesWithoutEpic = sortByBacklogRank(stories.filter((story) => !story.EpicID || !knownEpicIDs.has(story.EpicID)))
  const groups = [...epicGroups, { epic: null, stories: storiesWithoutEpic, estimatedHours: 0 }]
  return groups
    .map((group) => ({ ...group, estimatedHours: sumEstimatedHours(group.stories) }))
    .filter((group) => showEmptyGroups || group.stories.length > 0)
}

export interface IBacklogRankRequest {
  ProjectID: number
  StoryID: number
  PreviousStoryID: number
  NextStoryID: number
  EpicID: number
}

// backlogRankRequest is what a drop in the backlog sends: the story, its new neighbours (0 = none)
// and the epic group it fell into.
export const backlogRankRequest = (projectID: number, storyID: number, epicID: number, dropPosition: IDropPosition): IBacklogRankRequest => ({
  ProjectID: projectID,
  StoryID: storyID,
  PreviousStoryID: Number(dropPosition.previousID) || 0,
  NextStoryID: Number(dropPosition.nextID) || 0,
  EpicID: epicID,
})

// applyBacklogRank is the drop as the backlog shows it before the server answers: the same rank
// and epic backend/projects PostStoryRank will write.
export const applyBacklogRank = (story: IStory, stories: IStory[], rankRequest: IBacklogRankRequest): IStory => {
  const neighbourRank = (storyID: number) => {
    const neighbour = storyID !== story.ID ? stories.find((candidate) => candidate.ID === storyID) : undefined
    return neighbour ? neighbour.BacklogRank || 0 : null
  }
  return {
    ...story,
    BacklogRank: rankBetween(neighbourRank(rankRequest.PreviousStoryID), neighbourRank(rankRequest.NextStoryID)),
    EpicID: rankRequest.EpicID,
  }
}

export const formatHours = (hours: number) => `${Number.isInteger(hours) ? hours : hours.toFixed(1)} h`
