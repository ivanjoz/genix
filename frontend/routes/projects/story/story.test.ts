import { describe, expect, test } from 'bun:test';
import { STORY_STATE_IN_DEVELOPMENT, STORY_STATE_STOPPED } from '../project.ts';
import {
  ACTIVITY_KIND_ASSIGNEE_CHANGE, ACTIVITY_KIND_CARRIED_OVER, ACTIVITY_KIND_COMMENT, ACTIVITY_KIND_STATE_CHANGE,
  activityDescription, isStoppedWithReason, makeEmptyStory,
  makeStoryDetailForm, sortActivityNewestFirst, validateStory, type IStoryActivity,
} from './story.ts';
import { MAX_ATTACHMENTS, MAX_ATTACHMENT_BYTES, validateAttachmentFile } from '#domain/attachments/attachments.ts';
import { formatFileSize } from '#libs/helpers.ts';

const makeActivity = (overrides: Partial<IStoryActivity>): IStoryActivity => ({
  ProjectID: 1, StoryID: 1, ID: 1, Kind: ACTIVITY_KIND_COMMENT, AuthorID: 1, Created: 100, ...overrides,
})
const userName = (userID: number) => `User ${userID}`
const sprintName = (sprintID: number) => `Sprint ${sprintID}`

describe('validateStory', () => {
  test('a story needs a title', () => {
    expect(validateStory({ ...makeEmptyStory(1), Title: '  ' })).not.toBe('')
    expect(validateStory({ ...makeEmptyStory(1), Title: 'Login' })).toBe('')
  })
  test('estimated hours go in steps of 0.5', () => {
    expect(validateStory({ ...makeEmptyStory(1), Title: 'Login', EstimatedHours: 1.5 })).toBe('')
    expect(validateStory({ ...makeEmptyStory(1), Title: 'Login', EstimatedHours: 1.25 })).not.toBe('')
    expect(validateStory({ ...makeEmptyStory(1), Title: 'Login', EstimatedHours: -1 })).not.toBe('')
  })
})

describe('attachments', () => {
  test('a file up to 4 MB, up to 30 files', () => {
    expect(validateAttachmentFile({ size: MAX_ATTACHMENT_BYTES }, 0)).toBe('')
    expect(validateAttachmentFile({ size: MAX_ATTACHMENT_BYTES + 1 }, 0)).not.toBe('')
    expect(validateAttachmentFile({ size: 0 }, 0)).not.toBe('')
    expect(validateAttachmentFile({ size: 10 }, MAX_ATTACHMENTS)).not.toBe('')
  })
  test('formatFileSize', () => {
    expect(formatFileSize(500)).toBe('500 B')
    expect(formatFileSize(2048)).toBe('2 KB')
    expect(formatFileSize(3.5 * 1024 * 1024)).toBe('3.5 MB')
  })
})

test('makeStoryDetailForm does not share the checklist with the loaded detail', () => {
  const detail = { ProjectID: 1, ID: 1, Description: '', Checklist: [{ Text: 'A', Done: false }], Attachments: [] }
  const detailForm = makeStoryDetailForm(detail)
  detailForm.Checklist[0].Done = true
  expect(detail.Checklist[0].Done).toBe(false)
})

describe('activity', () => {
  test('a state change names both states and the sprint', () => {
    const stateChange = makeActivity({ Kind: ACTIVITY_KIND_STATE_CHANGE, FromState: STORY_STATE_IN_DEVELOPMENT, ToState: STORY_STATE_STOPPED, SprintID: 2 })
    expect(activityDescription(stateChange, userName, sprintName))
      .toBe('Moved from In development to Blocked in Sprint 2|Pasó de En desarrollo a Bloqueado en Sprint 2')
  })
  test('a carry-over without a target went back to the backlog', () => {
    const carryOver = makeActivity({ Kind: ACTIVITY_KIND_CARRIED_OVER, SprintID: 1 })
    expect(activityDescription(carryOver, userName, sprintName)).toContain('back to the backlog')
  })
  test('an assignee change without an assignee left the story unassigned', () => {
    expect(activityDescription(makeActivity({ Kind: ACTIVITY_KIND_ASSIGNEE_CHANGE }), userName, sprintName)).toBe('Left unassigned|Quedó sin responsable')
  })
  test('a move to Stopped shows its reason', () => {
    expect(isStoppedWithReason(makeActivity({ Kind: ACTIVITY_KIND_STATE_CHANGE, ToState: STORY_STATE_STOPPED, Text: 'Waiting for IT' }))).toBe(true)
    expect(isStoppedWithReason(makeActivity({ Kind: ACTIVITY_KIND_STATE_CHANGE, ToState: STORY_STATE_IN_DEVELOPMENT, Text: 'x' }))).toBe(false)
  })
  test('newest first, ties by ID', () => {
    const sorted = sortActivityNewestFirst([makeActivity({ ID: 1, Created: 5 }), makeActivity({ ID: 3, Created: 9 }), makeActivity({ ID: 2, Created: 9 })])
    expect(sorted.map((activity) => activity.ID)).toEqual([3, 2, 1])
  })
})
