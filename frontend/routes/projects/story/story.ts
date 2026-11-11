// A story's form, its detail and its timeline: pure rules shared by the Backlog and the boards.
import {
  STATUS_ACTIVE, STORY_PRIORITY_MEDIUM, STORY_STATES, STORY_STATE_NOT_PLANNED, STORY_STATE_STOPPED, STORY_TYPE_STORY,
  type IStory,
} from '../project.ts';
import type { IAttachment } from '#domain/attachments/attachments.ts';

export interface IChecklistItem {
  Text: string
  Done?: boolean
}

// A story's long content. ID is the story's ID: the by-IDs cache keys every record by ID. Its upd is
// the cache slot's version, not the record's write time.
export interface IStoryDetail {
  ProjectID: number
  ID: number
  Description?: string
  Checklist?: IChecklistItem[]
  Attachments?: IAttachment[]
  ss?: number
  upd?: number
}

export const ACTIVITY_KIND_COMMENT = 1
export const ACTIVITY_KIND_STATE_CHANGE = 2
export const ACTIVITY_KIND_ASSIGNEE_CHANGE = 3
export const ACTIVITY_KIND_ADDED_TO_SPRINT = 4
export const ACTIVITY_KIND_REMOVED_FROM_SPRINT = 5
export const ACTIVITY_KIND_CARRIED_OVER = 6

export interface IStoryActivity {
  ProjectID: number
  StoryID: number
  ID: number
  Kind: number
  Text?: string
  FromState?: number
  ToState?: number
  SprintID?: number
  ToSprintID?: number
  AssigneeID?: number
  Attachments?: IAttachment[]
  AuthorID: number
  Created: number
}

export const makeEmptyStory = (projectID: number, epicID = 0): IStory => ({
  ProjectID: projectID, ID: 0, Type: STORY_TYPE_STORY, Title: '', Summary: '', EpicID: epicID,
  Priority: STORY_PRIORITY_MEDIUM, AssigneeID: 0, EstimatedHours: 0, BacklogRank: 0, State: STORY_STATE_NOT_PLANNED, ss: STATUS_ACTIVE,
})

export const makeEmptyStoryDetail = (projectID: number, storyID = 0): IStoryDetail => ({
  ProjectID: projectID, ID: storyID, Description: '', Checklist: [], Attachments: [],
})

// makeStoryDetailForm clones the detail for its form: the checklist and the files are edited in place.
export const makeStoryDetailForm = (detail: IStoryDetail): IStoryDetail => ({
  ...detail,
  Checklist: (detail.Checklist || []).map((item) => ({ ...item })),
  Attachments: [...(detail.Attachments || [])],
})

// validateStory mirrors backend/projects normalizeStory: the backend checks again.
export const validateStory = (story: IStory): string => {
  const title = (story.Title || '').trim()
  if (!title || title.length > 200) {
    return 'The story needs a title of up to 200 characters.|La historia necesita un título de hasta 200 caracteres.'
  }
  if ((story.Summary || '').trim().length > 500) {
    return 'The short description can have up to 500 characters.|La descripción corta puede tener hasta 500 caracteres.'
  }
  const estimatedHours = Number(story.EstimatedHours || 0)
  if (Number.isNaN(estimatedHours) || estimatedHours < 0 || estimatedHours > 2000 || (estimatedHours * 2) % 1 !== 0) {
    return 'The estimated hours must be between 0 and 2000, in steps of 0.5.|Las horas estimadas deben estar entre 0 y 2000, en pasos de 0.5.'
  }
  return ''
}

const stateName = (state?: number) => STORY_STATES[state || 0]?.name || STORY_STATES[STORY_STATE_NOT_PLANNED].name

// activityDescription is one history line of the timeline, bilingual ("English|Spanish") so the
// page translates it. Comments have no description: their text is shown as written.
export const activityDescription = (
  activity: IStoryActivity, userName: (userID: number) => string, sprintName: (sprintID: number) => string,
): string => {
  const [fromEnglish, fromSpanish] = stateName(activity.FromState).split('|')
  const [toEnglish, toSpanish] = stateName(activity.ToState).split('|')
  switch (activity.Kind) {
    case ACTIVITY_KIND_STATE_CHANGE:
      return `Moved from ${fromEnglish} to ${toEnglish} in ${sprintName(activity.SprintID || 0)}`
        + `|Pasó de ${fromSpanish} a ${toSpanish} en ${sprintName(activity.SprintID || 0)}`
    case ACTIVITY_KIND_ASSIGNEE_CHANGE:
      return activity.AssigneeID
        ? `Assigned to ${userName(activity.AssigneeID)}|Asignada a ${userName(activity.AssigneeID)}`
        : 'Left unassigned|Quedó sin responsable'
    case ACTIVITY_KIND_ADDED_TO_SPRINT:
      return `Added to ${sprintName(activity.SprintID || 0)}|Agregada a ${sprintName(activity.SprintID || 0)}`
    case ACTIVITY_KIND_REMOVED_FROM_SPRINT:
      return `Removed from ${sprintName(activity.SprintID || 0)}|Quitada de ${sprintName(activity.SprintID || 0)}`
    case ACTIVITY_KIND_CARRIED_OVER:
      return activity.ToSprintID
        ? `Not completed in ${sprintName(activity.SprintID || 0)}: carried to ${sprintName(activity.ToSprintID)}`
          + `|No culminada en ${sprintName(activity.SprintID || 0)}: pasó a ${sprintName(activity.ToSprintID)}`
        : `Not completed in ${sprintName(activity.SprintID || 0)}: back to the backlog`
          + `|No culminada en ${sprintName(activity.SprintID || 0)}: volvió al backlog`
    default:
      return ''
  }
}

// isStoppedWithReason: a move to Stopped carries its reason, shown under the history line.
export const isStoppedWithReason = (activity: IStoryActivity) =>
  activity.Kind === ACTIVITY_KIND_STATE_CHANGE && activity.ToState === STORY_STATE_STOPPED && !!activity.Text

// sortActivityNewestFirst: the timeline reads from the latest entry down.
export const sortActivityNewestFirst = (activities: IStoryActivity[]): IStoryActivity[] =>
  [...activities].sort((left, right) => right.Created - left.Created || right.ID - left.ID)
