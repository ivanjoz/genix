import { describe, expect, it } from 'vitest'
import type { ICrmPipeline } from '#services/crm/crm-pipeline.svelte.ts'
import { levelOfUser, pipelinesWithChangedMembers, sameMembers, setMemberLevel } from './pipeline-access.ts'

const pipelineWith = (ID: number, Members: ICrmPipeline['Members']) => ({ ID, Members } as ICrmPipeline)

describe('pipeline member levels', () => {
  it('adds, changes and removes a member', () => {
    const added = setMemberLevel([], 4, 2)
    expect(levelOfUser(added, 4)).toBe(2)
    expect(levelOfUser(setMemberLevel(added, 4, 3), 4)).toBe(3)
    expect(setMemberLevel(added, 4, 0)).toEqual([])
    expect(levelOfUser(added, 9)).toBe(0)
  })

  it('compares member lists regardless of order', () => {
    expect(sameMembers([{ UserID: 1, Level: 1 }, { UserID: 2, Level: 3 }], [{ UserID: 2, Level: 3 }, { UserID: 1, Level: 1 }])).toBe(true)
    expect(sameMembers([{ UserID: 1, Level: 1 }], [{ UserID: 1, Level: 2 }])).toBe(false)
  })

  it('reports only the pipelines whose draft differs', () => {
    const pipelines = [pipelineWith(1, [{ UserID: 1, Level: 1 }]), pipelineWith(2, [])]
    const drafts = { 1: [{ UserID: 1, Level: 1 }], 2: [{ UserID: 5, Level: 2 }] }
    expect(pipelinesWithChangedMembers(pipelines, drafts).map((pipeline) => pipeline.ID)).toEqual([2])
  })
})
