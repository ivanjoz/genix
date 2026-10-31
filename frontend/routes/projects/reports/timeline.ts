// The Backlog page's Gantt and Calendario tabs: which rows, bars and calendar entries a project's
// backlog becomes. Pure: no Svelte, no fetch. genix-ui shows these labels as given, so the page
// passes its translate function (tr) in.
import { mondayBasedWeekday, type CalendarActivity, type CalendarColor } from '#components/calendar/calendar.ts';
import type { GanttBar, GanttHeaderBand, GanttMarker, GanttRow } from '#components/gantt/gantt.ts';
import { formatTime, todayUnixDay } from '#libs/helpers.ts';
import {
  SPRINT_STATE_ACTIVE, SPRINT_STATE_CLOSED, SPRINT_STATE_PLANNED, STORY_STATE_FINISHED, STORY_STATE_IN_DEVELOPMENT,
  STORY_STATE_IN_REVIEW, STORY_STATE_NOT_PLANNED, STORY_STATE_PENDING, STORY_STATE_STOPPED, storyCode, storyState,
  type IEpic, type IProject, type ISprint, type IStory,
} from '../project.ts';
import { matchesIDFilter, sortByBacklogRank } from '../backlog/backlog.ts';
import { packIntoLanes, sUnixTimeToUnixMillis } from '../backlog/backlog.utils.ts';

// The project's colors (PROJECT_COLORS) and the timeline's share the same palette names.
export const SPRINT_COLORS: Record<number, CalendarColor> = {
  [SPRINT_STATE_PLANNED]: 'blue', [SPRINT_STATE_ACTIVE]: 'green', [SPRINT_STATE_CLOSED]: 'gray',
}

// A story's bar in a sprint takes the color of its state there.
export const STORY_STATE_COLORS: Record<number, CalendarColor> = {
  [STORY_STATE_NOT_PLANNED]: 'gray', [STORY_STATE_PENDING]: 'blue', [STORY_STATE_IN_DEVELOPMENT]: 'purple',
  [STORY_STATE_STOPPED]: 'red', [STORY_STATE_IN_REVIEW]: 'amber', [STORY_STATE_FINISHED]: 'green',
}

export interface IProjectGanttBar extends GanttBar {
  sprintID?: number
  // The story the bar is, in the sprint Gantt, where a row holds several stories.
  story?: IStory
}

export interface IProjectGanttRow extends GanttRow<IProjectGanttBar> {
  story?: IStory
  // The sprint Gantt: the epic's title row, and the first row of each assignee (0 = unassigned).
  isEpicTitle?: boolean
  assigneeID?: number
}

const SPRINTS_ROW_ID = 'sprints'
const NO_EPIC_ROW_ID = 'epic-0'

// sprintFinishedShare: the estimated hours finished over those committed, the bar's progress.
const sprintFinishedShare = (sprint: ISprint, storiesByID: Map<number, IStory>): number => {
  let committedHours = 0
  let finishedHours = 0
  for (const sprintStory of sprint.Stories || []) {
    const estimatedHours = storiesByID.get(sprintStory.StoryID)?.EstimatedHours || 0
    committedHours += estimatedHours
    if (sprintStory.State === STORY_STATE_FINISHED) { finishedHours += estimatedHours }
  }
  return committedHours ? finishedHours / committedHours : 0
}

// storySprintBars: one bar per sprint the story was in, over the sprint's dates, colored by its
// state there. In a closed sprint an unfinished story says "Not completed".
export const storySprintBars =(story: IStory, sprints: ISprint[], translate: (text: string) => string): IProjectGanttBar[] =>
  sprints.flatMap((sprint) => {
    const sprintStory = (sprint.Stories || []).find((candidate) => candidate.StoryID === story.ID)
    if (!sprintStory) { return [] }
    const isNotCompleted = sprint.State === SPRINT_STATE_CLOSED && sprintStory.State !== STORY_STATE_FINISHED
    return [{
      start: sprint.StartDate, end: sprint.EndDate, sprintID: sprint.ID, color: STORY_STATE_COLORS[sprintStory.State] || 'gray',
      label: `${sprint.Name} · ${translate(isNotCompleted ? 'Not completed|No culminada' : storyState(sprintStory.State).name)}`,
    }]
  })

// spanBar is a group's bar (an epic, a project): from its first story's start to its last story's end.
export const spanBar =(bars: GanttBar[], color: CalendarColor, label: string): IProjectGanttBar[] =>
  bars.length === 0 ? [] : [{ start: Math.min(...bars.map((bar) => bar.start)), end: Math.max(...bars.map((bar) => bar.end)), color, label }]

// isStoryInGanttFilters: the Gantt's state select (0 = any, NONE_FILTER = not planned) and its
// people filter (none chosen = everyone, 0 = unassigned).
export const isStoryInGanttFilters = (story: IStory, state: number, assigneeIDs: number[]): boolean =>
  matchesIDFilter(story.State, state) && (assigneeIDs.length === 0 || assigneeIDs.includes(story.AssigneeID || 0))

// storyAssigneeIDs: who holds the live stories, by name, with the unassigned (0) last: the avatars
// of the Gantt's people filter.
export const storyAssigneeIDs = (stories: IStory[], userName: (userID: number) => string): number[] =>
  [...new Set(stories.filter((story) => story.ss !== 0).map((story) => story.AssigneeID || 0))]
    .sort((left, right) => Number(!left) - Number(!right) || userName(left).localeCompare(userName(right)))

// projectGantt: the sprints on top, then each epic (and "No epic") with its planned stories in
// backlog order. Stories never in a sprint have no dates: they are left out, and so are the ones
// isStoryShown filters out (the sprint bars still count every story).
export const projectGantt = (
  project: IProject, epics: IEpic[], stories: IStory[], sprints: ISprint[], translate: (text: string) => string,
  isStoryShown: (story: IStory) => boolean,
): IProjectGanttRow[] => {
  const liveSprints = sprints.filter((sprint) => sprint.ss !== 0).sort((left, right) => left.StartDate - right.StartDate || left.ID - right.ID)
  const liveStories = stories.filter((story) => story.ss !== 0)
  const storiesByID = new Map(liveStories.map((story) => [story.ID, story]))

  const sprintRows: IProjectGanttRow[] = liveSprints.map((sprint) => ({
    id: `sprint-${sprint.ID}`, parentId: SPRINTS_ROW_ID, label: sprint.Name,
    bars: [{
      start: sprint.StartDate, end: sprint.EndDate, sprintID: sprint.ID, color: SPRINT_COLORS[sprint.State] || 'gray',
      label: sprint.Name, progress: sprintFinishedShare(sprint, storiesByID),
    }],
  }))

  const knownEpicIDs = new Set(epics.map((epic) => epic.ID))
  const epicGroups = [
    ...[...epics].sort((left, right) => left.ID - right.ID).map((epic) => ({ rowID: `epic-${epic.ID}`, name: epic.Name, color: epic.Color as CalendarColor, epicID: epic.ID })),
    { rowID: NO_EPIC_ROW_ID, name: translate('No epic|Sin épica'), color: 'gray' as CalendarColor, epicID: 0 },
  ]
  const epicRows: IProjectGanttRow[] = epicGroups.flatMap((epicGroup) => {
    const epicStories = sortByBacklogRank(liveStories.filter((story) => isStoryShown(story) &&
      (epicGroup.epicID ? story.EpicID === epicGroup.epicID : !story.EpicID || !knownEpicIDs.has(story.EpicID))))
    const storyRows: IProjectGanttRow[] = epicStories
      .map((story) => ({ id: `story-${story.ID}`, parentId: epicGroup.rowID, label: `${storyCode(story)} ${story.Title}`, story, bars: storySprintBars(story, liveSprints, translate) }))
      .filter((storyRow) => storyRow.bars.length > 0)
    if (storyRows.length === 0) { return [] }
    return [{ id: epicGroup.rowID, label: epicGroup.name, bars: spanBar(storyRows.flatMap((storyRow) => storyRow.bars), epicGroup.color, epicGroup.name) }, ...storyRows]
  })

  return [
    ...(sprintRows.length > 0 ? [{ id: SPRINTS_ROW_ID, label: 'Sprints', bars: [] }, ...sprintRows] : []),
    ...epicRows,
  ]
}

// A team member works a story 6 hours a day, Monday to Saturday.
const WORK_HOURS_PER_DAY = 6
const isSunday = (unixDay: number): boolean => mondayBasedWeekday(unixDay) === 6
const ONE_DAY_BAR_EXTRA_DAYS = 0.6

// workStartBefore is the first day of a story finished on endDay, worked its estimated hours back
// from it: 18 hours finished on the 15th started on the 13th. The end day always counts; Sundays
// before it don't. Without an estimate it takes one day.
export const workStartBefore = (endDay: number, estimatedHours: number): number => {
  let startDay = endDay
  for (let remainingDays = Math.max(1, Math.ceil(estimatedHours / WORK_HOURS_PER_DAY)) - 1; remainingDays > 0;) {
    startDay--
    if (!isSunday(startDay)) { remainingDays-- }
  }
  return startDay
}

// storyWorkSpan: the days a story was worked, from the last day of each of its states (StateDates).
// Finished: from the day it went into development to the day it finished; without a development
// day (or one after the finish), back from the finish by its estimated hours. In development,
// blocked or in review: from the day it went into development (else the day it entered its current
// state) projected to tomorrow. Pending or not planned: no span.
export const storyWorkSpan = (story: IStory, today: number): { start: number, end: number } | null => {
  const dayByState = new Map((story.StateDates || []).map((stateDate) => [stateDate.State, stateDate.Date]))
  const developmentDay = dayByState.get(STORY_STATE_IN_DEVELOPMENT)
  if (story.State === STORY_STATE_FINISHED) {
    const finishedDay = dayByState.get(STORY_STATE_FINISHED)
    if (!finishedDay) { return null }
    if (developmentDay && developmentDay <= finishedDay) { return { start: developmentDay, end: finishedDay } }
    return { start: workStartBefore(finishedDay, story.EstimatedHours || 0), end: finishedDay }
  }
  if (story.State === STORY_STATE_IN_DEVELOPMENT || story.State === STORY_STATE_STOPPED || story.State === STORY_STATE_IN_REVIEW) {
    const startDay = developmentDay || dayByState.get(story.State)
    return startDay ? { start: startDay, end: Math.max(startDay, today + 1) } : null
  }
  return null
}

export interface ISprintGantt {
  rows: IProjectGanttRow[]
  // The sprints, as the header's top row.
  headerBands: GanttHeaderBand[]
}

// sprintGantt: per epic (and "No epic") a title row, then one block of rows per assignee by name
// (unassigned last), the name only on its first row. Each person's stories sit side by side by the
// days they were worked, in as few rows as fit them. Stories with no work span (pending, not
// planned) are left out, and so are the ones isStoryShown filters out.
export const sprintGantt = (
  epics: IEpic[], stories: IStory[], sprints: ISprint[], today: number, translate: (text: string) => string,
  userName: (userID: number) => string, isStoryShown: (story: IStory) => boolean,
): ISprintGantt => {
  const liveStories = stories.filter((story) => story.ss !== 0 && isStoryShown(story))
  const knownEpicIDs = new Set(epics.map((epic) => epic.ID))
  const epicGroups = [
    ...[...epics].sort((left, right) => left.ID - right.ID).map((epic) => ({ name: epic.Name, epicID: epic.ID })),
    { name: translate('No epic|Sin épica'), epicID: 0 },
  ]
  const rows = epicGroups.flatMap((epicGroup) => {
    const storyBars: IProjectGanttBar[] = liveStories
      .filter((story) => epicGroup.epicID ? story.EpicID === epicGroup.epicID : !story.EpicID || !knownEpicIDs.has(story.EpicID))
      .flatMap((story) => {
        const workSpan = storyWorkSpan(story, today)
        if (!workSpan) { return [] }
        // A one-day story is drawn 1.6 days wide, reaching back into the day before, so its title
        // reads; it holds both days of its row.
        const start = workSpan.start === workSpan.end ? workSpan.end - ONE_DAY_BAR_EXTRA_DAYS : workSpan.start
        return [{ start, end: workSpan.end, story, color: STORY_STATE_COLORS[story.State], label: `${storyCode(story)} ${story.Title}` }]
      })
    if (storyBars.length === 0) { return [] }
    const barAssigneeID = (bar: IProjectGanttBar) => bar.story?.AssigneeID || 0
    const assigneeName = (assigneeID: number) => assigneeID ? userName(assigneeID) : translate('Unassigned|Sin responsable')
    const sortedAssigneeIDs = [...new Set(storyBars.map(barAssigneeID))].sort((left, right) =>
      Number(!left) - Number(!right) || assigneeName(left).localeCompare(assigneeName(right)))
    return [
      { id: `epic-${epicGroup.epicID}`, label: epicGroup.name, bars: [], isEpicTitle: true, css: 'border-b-2 border-purple-border' },
      ...sortedAssigneeIDs.flatMap((assigneeID) =>
        // Packed by the whole days each bar touches.
        packIntoLanes(storyBars.filter((bar) => barAssigneeID(bar) === assigneeID).map((bar) => ({ start: Math.floor(bar.start), end: bar.end, bar }))).map((laneSpans, laneIndex): IProjectGanttRow => ({
          id: `epic-${epicGroup.epicID}-user-${assigneeID}-lane-${laneIndex}`, bars: laneSpans.map((laneSpan) => laneSpan.bar),
          ...(laneIndex === 0 ? { label: assigneeName(assigneeID), assigneeID } : { label: '' }),
        }))),
    ]
  })
  const headerBands = sprints
    .filter((sprint) => sprint.ss !== 0)
    .sort((left, right) => left.StartDate - right.StartDate)
    .map((sprint) => ({
      start: sprint.StartDate, end: sprint.EndDate,
      label: `${sprint.Name} (${String(formatTime(sprint.StartDate, 'd/m'))} – ${String(formatTime(sprint.EndDate, 'd/m'))})`,
    }))
  return { rows, headerBands }
}

// projectMarkers: the project's start and end, when set.
export const projectMarkers = (project: IProject, translate: (text: string) => string): GanttMarker[] => [
  ...(project.StartDate ? [{ day: project.StartDate, label: translate('Project start|Inicio del proyecto'), color: 'blue' as const }] : []),
  ...(project.EndDate ? [{ day: project.EndDate, label: translate('Project end|Fin del proyecto'), color: 'teal' as const }] : []),
]

export interface IProjectCalendarActivity extends CalendarActivity {
  story?: IStory
  sprintID?: number
}

// projectCalendarActivities: the sprints as bars over their dates, the project's start and end,
// and each story on the day it was finished (its latest sprint's DoneAt).
export const projectCalendarActivities = (
  project: IProject, stories: IStory[], sprints: ISprint[], translate: (text: string) => string,
): IProjectCalendarActivity[] => {
  const liveSprints = sprints.filter((sprint) => sprint.ss !== 0)
  const storiesByID = new Map(stories.filter((story) => story.ss !== 0).map((story) => [story.ID, story]))
  const finishedAtByStoryID = new Map<number, number>()
  for (const sprint of liveSprints) {
    for (const sprintStory of sprint.Stories || []) {
      if (sprintStory.State !== STORY_STATE_FINISHED || !sprintStory.DoneAt) { continue }
      finishedAtByStoryID.set(sprintStory.StoryID, Math.max(finishedAtByStoryID.get(sprintStory.StoryID) || 0, sprintStory.DoneAt))
    }
  }
  return [
    ...projectMarkers(project, translate).map((marker) => ({ date: marker.day, title: marker.label || '', color: marker.color, icon: 'icon-[fa--flag-o]' })),
    ...liveSprints.map((sprint) => ({
      date: sprint.StartDate, endDate: sprint.EndDate, title: sprint.Name, sprintID: sprint.ID,
      color: SPRINT_COLORS[sprint.State] || 'gray', icon: 'icon-[fa--columns]',
    })),
    ...[...finishedAtByStoryID].flatMap(([storyID, doneAt]) => {
      const story = storiesByID.get(storyID)
      return story ? [{
        date: todayUnixDay(new Date(sUnixTimeToUnixMillis(doneAt))), title: `${storyCode(story)} ${story.Title}`, story,
        color: 'green' as const, icon: 'icon-[fa--check]',
      }] : []
    }),
  ]
}
