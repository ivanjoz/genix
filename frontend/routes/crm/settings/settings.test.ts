import { describe, expect, it } from 'vitest'
import type { ICrmPipelineStage } from '#services/crm/crm-pipeline.svelte.ts'
import { makeEmptyStage, moveItem, removeStage, setFieldRequired, toggleField } from './settings.ts'

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
  it('toggles a field on and off and sets it required', () => {
    const picked = toggleField([], 5)
    expect(picked).toEqual([{ FieldID: 5, Required: false }])
    expect(setFieldRequired(picked, 5, true)[0].Required).toBe(true)
    expect(toggleField(picked, 5)).toEqual([])
  })
})
