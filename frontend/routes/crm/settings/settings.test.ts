import { describe, expect, it } from 'vitest'
import type { ICrmPipelineStage } from '#services/crm/crm-pipeline.svelte.ts'
import { TASK_ASSIGN, TASK_KIND } from '#services/crm/crm-task.ts'
import {
  buildTaskTemplateRows, changeStageTemplates, makeEmptyStage, makeEmptyTaskTemplate, moveItem, moveTemplateToStage, removeStage, setFieldRequiredFrom, setStageApprover,
  setStageStandardField, setTemplateAssignment, toggleField,
} from './settings.ts'

const stageNamed = (ID: number, Name: string): ICrmPipelineStage => ({ ...makeEmptyStage(), ID, Name })

describe('pipeline stage editing', () => {
  it('moves a stage and ignores moves past the ends', () => {
    const stages = [stageNamed(1, 'A'), stageNamed(2, 'B'), stageNamed(3, 'C')]
    expect(moveItem(stages, 1, -1).map((stage) => stage.Name)).toEqual(['B', 'A', 'C'])
    expect(moveItem(stages, 0, -1)).toBe(stages)
    expect(moveItem(stages, 2, 1)).toBe(stages)
  })

  it('drops a new stage but soft-deletes a stored one', () => {
    const stages = [stageNamed(1, 'A'), stageNamed(0, 'New')]
    expect(removeStage(stages, 1)).toHaveLength(1)
    const afterRemovingStored = removeStage(stages, 0)
    expect(afterRemovingStored).toHaveLength(2)
    expect(afterRemovingStored[0].Status).toBe(0)
  })
})

describe('pipeline custom fields', () => {
  it('toggles a field on and off and requires it from a stage', () => {
    const picked = toggleField([], 5)
    expect(picked).toEqual([{ FieldID: 5, RequiredFromStageID: 0 }])
    expect(setFieldRequiredFrom(picked, 5, 3)[0].RequiredFromStageID).toBe(3)
    expect(setFieldRequiredFrom(setFieldRequiredFrom(picked, 5, 3), 5, 0)[0].RequiredFromStageID).toBe(0)
    expect(toggleField(picked, 5)).toEqual([])
  })

  it('marks standard fields on one stage only, without repeating them', () => {
    const stages = [makeEmptyStage(), makeEmptyStage()]
    const required = setStageStandardField(setStageStandardField(stages, 1, 1, true), 1, 1, true)
    expect(required[0].RequiredStandardFields).toEqual([])
    expect(required[1].RequiredStandardFields).toEqual([1])
    expect(setStageStandardField(required, 1, 1, false)[1].RequiredStandardFields).toEqual([])
  })
})

describe('stage approvals and task templates', () => {
  it('sets the approver of one stage only', () => {
    const stages = [stageNamed(1, 'A'), stageNamed(2, 'B')]
    const approved = setStageApprover(stages, 1, 7)
    expect(approved.map((stage) => stage.ApproverUserID)).toEqual([0, 7])
  })

  it('adds and removes templates of one stage without touching the others', () => {
    const stages = [stageNamed(1, 'A'), stageNamed(2, 'B')]
    const withTemplate = changeStageTemplates(stages, 0, (templates) => [...templates, makeEmptyTaskTemplate()])
    expect(withTemplate[0].TaskTemplates).toHaveLength(1)
    expect(withTemplate[1].TaskTemplates).toHaveLength(0)
    expect(changeStageTemplates(withTemplate, 0, () => [])[0].TaskTemplates).toEqual([])
  })

  it('drops the fixed user when a template goes back to the owner', () => {
    const fixed = setTemplateAssignment(makeEmptyTaskTemplate(), TASK_ASSIGN.USER, 9)
    expect(fixed).toMatchObject({ AssignTo: TASK_ASSIGN.USER, AssignUserID: 9 })
    expect(setTemplateAssignment(fixed, TASK_ASSIGN.OWNER)).toMatchObject({ AssignTo: TASK_ASSIGN.OWNER, AssignUserID: 0 })
  })

  it('flattens the templates of live stages, keeping their positions in the full list', () => {
    const emailTemplate = { ...makeEmptyTaskTemplate(), Kind: TASK_KIND.SEND_EMAIL, EmailTemplateID: 5 }
    const fixedTemplate = setTemplateAssignment(makeEmptyTaskTemplate(), TASK_ASSIGN.USER, 9)
    const stages = [
      { ...stageNamed(1, 'A'), Status: 0, TaskTemplates: [makeEmptyTaskTemplate()] },
      { ...stageNamed(2, 'B'), TaskTemplates: [emailTemplate, fixedTemplate] },
    ]
    const rows = buildTaskTemplateRows(stages)
    expect(rows.map((row) => [row.StageIndex, row.TemplateIndex, row.SourceID, row.AssignUserID])).toEqual([[1, 0, 5, 0], [1, 1, 1, 9]])
  })

  it('moves only an unsaved template to another stage', () => {
    const stages = [{ ...stageNamed(1, 'A'), TaskTemplates: [makeEmptyTaskTemplate()] }, stageNamed(2, 'B')]
    const moved = moveTemplateToStage(stages, 0, 0, 1)
    expect(moved.map((stage) => stage.TaskTemplates.length)).toEqual([0, 1])
    const stored = [{ ...stageNamed(1, 'A'), TaskTemplates: [{ ...makeEmptyTaskTemplate(), ID: 3 }] }, stageNamed(2, 'B')]
    expect(moveTemplateToStage(stored, 0, 0, 1)).toBe(stored)
  })
})
