// The "By person" report: the stories assigned to one person across every project, as a list, a
// Gantt and a weekly workload. Pure: no Svelte, no fetch.
import type { CalendarColor } from '#components/calendar/calendar.ts';
import type { GanttRow } from '#components/gantt/gantt.ts';
import { todayUnixDay } from '#libs/helpers.ts';
import { sUnixTimeToUnixMillis } from '../../backlog/backlog.utils.ts';
import { spanBar, storySprintBars, type IProjectGanttBar } from '../timeline.ts';
import { STORY_STATE_FINISHED, projectCode, storyCode, type IProject, type ISprint, type ISprintStory, type IStory } from '../../project.ts';

// IStoryAssignee mirrors backend/projects/types StoryAssignee: a story's current assignee. upd is
// when the story was assigned to this person.
export interface IStoryAssignee {
  StoryID: number
  ProjectID: number
  AssigneeID?: number
  ss: number
  upd?: number
}

export interface IAssignedStoryRow {
  story: IStory
  project?: IProject
  assignedAt: number
}

// The report covers what was assigned in this many days.
export const ASSIGNED_STORIES_DAYS = 180

// A SUnixTime counts 2-second ticks: (unix seconds - 1e9) / 2.
const SUNIX_TICKS_PER_DAY = 43_200

// recentAssignments keeps the assignments of the last ASSIGNED_STORIES_DAYS, newest first. A story
// fetched by ID carries its slot's tick as upd, not its own last write, so the assignment's own upd
// is the date the report can trust.
export const recentAssignments = (assignments: IStoryAssignee[], nowUnixMillis: number): IStoryAssignee[] => {
  const cutoffSUnixTime = (nowUnixMillis / 1000 - 1e9) / 2 - ASSIGNED_STORIES_DAYS * SUNIX_TICKS_PER_DAY
  return assignments
    .filter((assignment) => assignment.ss !== 0 && (assignment.upd || 0) >= cutoffSUnixTime)
    .sort((left, right) => (right.upd || 0) - (left.upd || 0))
}

// storyIDsByProject is the by-IDs request of the stories: projects-stories-ids is partitioned by project.
export const storyIDsByProject = (assignments: IStoryAssignee[]): Record<number, number[]> => {
  const idsByProject: Record<number, number[]> = {}
  for (const assignment of assignments) {
    (idsByProject[assignment.ProjectID] ||= []).push(assignment.StoryID)
  }
  return idsByProject
}

// assignedStoryRows joins each assignment with its story and project, in the assignments' order.
// A story deleted (or not readable) since is left out.
export const assignedStoryRows = (
  assignments: IStoryAssignee[], storiesByProject: Map<number, Map<number, IStory>>, projectsByID: Map<number, IProject>,
): IAssignedStoryRow[] => {
  const rows: IAssignedStoryRow[] = []
  for (const assignment of assignments) {
    const story = storiesByProject.get(assignment.ProjectID)?.get(assignment.StoryID)
    if (!story || story.ss === 0) { continue }
    rows.push({ story, project: projectsByID.get(assignment.ProjectID), assignedAt: assignment.upd || 0 })
  }
  return rows
}

// projectSprintsInOrder: a project's live sprints, oldest first, so a later sprint's state wins.
const projectSprintsInOrder = (sprintsByProject: Map<number, ISprint[]>, projectID: number): ISprint[] =>
  (sprintsByProject.get(projectID) || []).filter((sprint) => sprint.ss !== 0).sort((left, right) => left.StartDate - right.StartDate || left.ID - right.ID)

const projectLabel = (row: IAssignedStoryRow) => row.project ? `${projectCode(row.project)} · ${row.project.Name}` : `#${row.story.ProjectID}`

export interface IPersonGanttRow extends GanttRow<IProjectGanttBar> {
  projectID: number
}

export interface IPersonGantt {
  rows: IPersonGanttRow[]
  // Stories never in a sprint have no dates: they are left out and counted.
  unplannedCount: number
}

// personGantt: one group per project, by name, spanning its stories; under it each story with one
// bar per sprint it was in, as on the Backlog's Gantt.
export const personGantt = (rows: IAssignedStoryRow[], sprintsByProject: Map<number, ISprint[]>, translate: (text: string) => string): IPersonGantt => {
  const storyRowsByProject = new Map<number, { label: string, color: CalendarColor, storyRows: IPersonGanttRow[] }>()
  let unplannedCount = 0
  for (const row of rows) {
    const projectID = row.story.ProjectID
    const bars = storySprintBars(row.story, projectSprintsInOrder(sprintsByProject, projectID), translate)
    if (bars.length === 0) { unplannedCount++; continue }
    if (!storyRowsByProject.has(projectID)) {
      storyRowsByProject.set(projectID, { label: projectLabel(row), color: (row.project?.Color || 'gray') as CalendarColor, storyRows: [] })
    }
    storyRowsByProject.get(projectID)!.storyRows.push({
      id: `story-${row.story.ID}`, parentId: `project-${projectID}`, projectID, label: `${storyCode(row.story)} ${row.story.Title}`, bars,
    })
  }
  const projectGroups = [...storyRowsByProject].sort(([, left], [, right]) => left.label.localeCompare(right.label))
  return {
    rows: projectGroups.flatMap(([projectID, group]) => [
      { id: `project-${projectID}`, projectID, label: group.label, bars: spanBar(group.storyRows.flatMap((storyRow) => storyRow.bars), group.color, group.label) },
      ...group.storyRows,
    ]),
    unplannedCount,
  }
}

// A person's week is measured against this many hours.
export const WEEKLY_CAPACITY_HOURS = 40

// weekStartDay: the Monday of a UnixDay's week. UnixDay 0 (1970-01-01) was a Thursday.
export const weekStartDay = (unixDay: number): number => unixDay - (((unixDay + 3) % 7) + 7) % 7

export interface IWeekStory {
  row: IAssignedStoryRow
  // Its state in the latest sprint holding it.
  state: number
  // The latest sprint holding it.
  sprint: ISprint
}

// A project's stories of one sprint in a week column, under the sprint's name.
export interface IWeekSprint {
  sprint: ISprint
  stories: IWeekStory[]
}

// A week column's stories of one project, under the project's divider.
export interface IWeekProject {
  projectID: number
  label: string
  assignedHours: number
  sprints: IWeekSprint[]
}

export interface IPersonWeek {
  startDay: number
  assignedHours: number
  finishedHours: number
  projects: IWeekProject[]
}

// The Calendar shows this many week columns: by default the previous week, this one and the next two.
export const VISIBLE_WEEKS = 4

// defaultFirstWeekStartDay: the Monday of the week before today's.
export const defaultFirstWeekStartDay = (today: number): number => weekStartDay(today) - 7

const storyHours = (weekStories: IWeekStory[]): number =>
  weekStories.reduce((hours, weekStory) => hours + (weekStory.row.story.EstimatedHours || 0), 0)

// weekProjects: the week's stories grouped by project (by name), then by sprint (oldest first), each
// sprint's stories by number.
const weekProjects = (weekStories: IWeekStory[]): IWeekProject[] => {
  const projectsByID = new Map<number, IWeekProject>()
  for (const weekStory of weekStories) {
    const projectID = weekStory.row.story.ProjectID
    if (!projectsByID.has(projectID)) { projectsByID.set(projectID, { projectID, label: projectLabel(weekStory.row), assignedHours: 0, sprints: [] }) }
    const weekProject = projectsByID.get(projectID)!
    let weekSprint = weekProject.sprints.find((candidate) => candidate.sprint.ID === weekStory.sprint.ID)
    if (!weekSprint) {
      weekSprint = { sprint: weekStory.sprint, stories: [] }
      weekProject.sprints.push(weekSprint)
    }
    weekSprint.stories.push(weekStory)
    weekProject.assignedHours += weekStory.row.story.EstimatedHours || 0
  }
  const projects = [...projectsByID.values()].sort((left, right) => left.label.localeCompare(right.label))
  for (const project of projects) {
    project.sprints.sort((left, right) => left.sprint.StartDate - right.sprint.StartDate || left.sprint.ID - right.sprint.ID)
    for (const weekSprint of project.sprints) { weekSprint.stories.sort((left, right) => left.row.story.ID - right.row.story.ID) }
  }
  return projects
}

// personWeeks: the person's workload in VISIBLE_WEEKS columns from firstWeekStartDay, oldest first,
// empty weeks included. Each story sits in exactly one week, by its latest sprint (the user's rule):
// - Finished: the week its card reached Finished.
// - Not finished: a week of its sprint, from the current week on (the last one when the sprint is
//   over). The weeks fill in order up to WEEKLY_CAPACITY_HOURS; a story that doesn't fit moves to the
//   next week of its sprint, and the sprint's last week takes whatever is left.
export const personWeeks = (
  rows: IAssignedStoryRow[], sprintsByProject: Map<number, ISprint[]>, firstWeekStartDay: number, today: number,
): IPersonWeek[] => {
  const weekStoriesByStart = new Map<number, IWeekStory[]>()
  const weekHours = (startDay: number) => storyHours(weekStoriesByStart.get(startDay) || [])
  const placeStory = (startDay: number, weekStory: IWeekStory) => {
    if (!weekStoriesByStart.has(startDay)) { weekStoriesByStart.set(startDay, []) }
    weekStoriesByStart.get(startDay)!.push(weekStory)
  }

  const unfinishedStories: { weekStory: IWeekStory, sprintStory: ISprintStory }[] = []
  for (const row of rows) {
    // Sprints come oldest first, so the last one holding the story is where it stands now.
    const latestSprint = projectSprintsInOrder(sprintsByProject, row.story.ProjectID)
      .findLast((sprint) => (sprint.Stories || []).some((candidate) => candidate.StoryID === row.story.ID))
    if (!latestSprint) { continue }
    const sprintStory = latestSprint.Stories!.find((candidate) => candidate.StoryID === row.story.ID)!
    const weekStory: IWeekStory = { row, state: sprintStory.State, sprint: latestSprint }
    if (sprintStory.State !== STORY_STATE_FINISHED) {
      unfinishedStories.push({ weekStory, sprintStory })
      continue
    }
    const finishedDay = sprintStory.DoneAt ? todayUnixDay(new Date(sUnixTimeToUnixMillis(sprintStory.DoneAt))) : latestSprint.EndDate
    placeStory(weekStartDay(finishedDay), weekStory)
  }

  // The sprint that ends first takes the earliest weeks; inside a sprint, the work already started
  // goes first, then the board order.
  unfinishedStories.sort((left, right) =>
    left.weekStory.sprint.EndDate - right.weekStory.sprint.EndDate || left.weekStory.sprint.ID - right.weekStory.sprint.ID
    || Number(!left.sprintStory.StartedAt) - Number(!right.sprintStory.StartedAt) || left.sprintStory.Rank - right.sprintStory.Rank)
  const currentWeekStartDay = weekStartDay(today)
  for (const { weekStory } of unfinishedStories) {
    const lastWeekStartDay = weekStartDay(weekStory.sprint.EndDate)
    let startDay = Math.min(lastWeekStartDay, Math.max(weekStartDay(weekStory.sprint.StartDate), currentWeekStartDay))
    const hours = weekStory.row.story.EstimatedHours || 0
    // An empty week takes the story even when it alone passes the capacity.
    while (startDay < lastWeekStartDay && weekHours(startDay) > 0 && weekHours(startDay) + hours > WEEKLY_CAPACITY_HOURS) { startDay += 7 }
    placeStory(startDay, weekStory)
  }

  return Array.from({ length: VISIBLE_WEEKS }, (_, weekIndex) => {
    const startDay = firstWeekStartDay + weekIndex * 7
    const weekStories = weekStoriesByStart.get(startDay) || []
    return {
      startDay,
      assignedHours: storyHours(weekStories),
      finishedHours: storyHours(weekStories.filter((weekStory) => weekStory.state === STORY_STATE_FINISHED)),
      projects: weekProjects(weekStories),
    }
  })
}

// weekMeterScale: the hours the full width of every week's meter stands for, so weeks compare at a
// glance and the capacity line sits at the same place in all of them.
export const weekMeterScale = (weeks: IPersonWeek[]): number =>
  Math.max(WEEKLY_CAPACITY_HOURS, ...weeks.flatMap((week) => [week.assignedHours, week.finishedHours]))
