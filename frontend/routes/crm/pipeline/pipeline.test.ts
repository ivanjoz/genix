import { describe, expect, it } from 'vitest'
import type { ICrmCustomFieldValue, ICrmOpportunity, ICrmOpportunityStageLog } from '#services/crm/crm-opportunity.svelte.ts'
import type { ICrmCustomField, ICrmPipeline } from '#services/crm/crm-pipeline.svelte.ts'
import {
  activityTypeOf, activityTypeOptions, buildOpportunityFormFields, buildOpportunityTimeline, canEditOpportunity, canMoveOpportunity,
  columnTotals, isStale, missingRequiredFieldIDs, readCustomValues, stageLogDescription, visiblePipelines, writeCustomValues,
} from './pipeline.ts'

const opportunity = (overrides: Partial<ICrmOpportunity> = {}) =>
  ({ ID: 1, Amount: 1000, Probability: 50, CreatedBy: 7, OwnerUserID: 7, StageEnteredTime: 0, CustomValues: [], ...overrides }) as ICrmOpportunity

const catalog = new Map<number, ICrmCustomField>([
  [1, { ID: 1, Name: 'Origen', Type: 7, Options: [{ ID: 1, Name: 'Web', Color: '', Status: 1 }, { ID: 2, Name: 'Viejo', Color: '', Status: 0 }], ss: 1, upd: 0 }],
  [2, { ID: 2, Name: 'Nota', Type: 1, Options: [], ss: 1, upd: 0 }],
  [3, { ID: 3, Name: 'Etiquetas', Type: 8, Options: [{ ID: 5, Name: 'A', Color: '', Status: 1 }], ss: 1, upd: 0 }],
])

describe('permission mirror', () => {
  it('lets Edit move anything but edit only their own; Full edits any', () => {
    expect(canMoveOpportunity(1)).toBe(false)
    expect(canMoveOpportunity(2)).toBe(true)
    expect(canEditOpportunity(2, opportunity(), 7)).toBe(true)
    expect(canEditOpportunity(2, opportunity(), 9)).toBe(false)
    expect(canEditOpportunity(3, opportunity(), 9)).toBe(true)
  })

  it('shows only the pipelines the user is a member of', () => {
    const pipelines = [
      { ID: 1, SortOrder: 2, Members: [{ UserID: 7, Level: 1 }] },
      { ID: 2, SortOrder: 1, Members: [{ UserID: 7, Level: 3 }] },
      { ID: 3, SortOrder: 0, Members: [{ UserID: 8, Level: 3 }] },
    ] as ICrmPipeline[]
    expect(visiblePipelines(pipelines, 7).map((pipeline) => pipeline.ID)).toEqual([2, 1])
  })
})

describe('board numbers', () => {
  it('totals and weights a column', () => {
    expect(columnTotals([opportunity(), opportunity({ Amount: 500, Probability: 10 })]))
      .toEqual({ count: 2, amount: 1500, weightedAmount: 550 })
  })

  it('flags a card stale only past the stage threshold', () => {
    const daysInSUnix = (days: number) => days * 86400 / 2
    const stage = { StaleDays: 7 } as any
    const entered = opportunity({ StageEnteredTime: 1000 })
    expect(isStale(entered, stage, 1000 + daysInSUnix(6))).toBe(false)
    expect(isStale(entered, stage, 1000 + daysInSUnix(8))).toBe(true)
    expect(isStale(entered, { StaleDays: 0 } as any, 1000 + daysInSUnix(99))).toBe(false)
  })

  it('lists the required target fields a card lacks', () => {
    const values = [{ FieldID: 1 }] as ICrmCustomFieldValue[]
    expect(missingRequiredFieldIDs(values, [{ FieldID: 1, Required: true }, { FieldID: 2, Required: true }, { FieldID: 3, Required: false }])).toEqual([2])
  })
})

describe('custom field form conversion', () => {
  const pipeline = { Fields: [{ FieldID: 1, Required: true }, { FieldID: 2, Required: false }, { FieldID: 9, Required: false }] } as ICrmPipeline

  it('builds the form fields, hiding deleted options and unknown fields', () => {
    const fields = buildOpportunityFormFields(pipeline, catalog, { users: [] })
    expect(fields.map((field) => field.label)).toEqual(['Origen', 'Nota'])
    expect(fields[0].options).toEqual([{ id: 1, name: 'Web' }])
    expect(fields[0].required).toBe(true)
  })

  it('round-trips stored values through the form map', () => {
    const stored: ICrmCustomFieldValue[] = [
      { FieldID: 1, Number: 1, Text: '', OptionIDs: [] },
      { FieldID: 2, Number: 0, Text: 'hola', OptionIDs: [] },
      { FieldID: 3, Number: 0, Text: '', OptionIDs: [5] },
    ]
    const formValues = writeCustomValues(stored, catalog)
    expect(formValues).toEqual({ f1: 1, f2: 'hola', f3: [5] })
    expect(readCustomValues(formValues, catalog)).toEqual(stored)
  })

  it('does not store empty fields', () => {
    expect(readCustomValues({ f1: null, f2: '', f3: [] }, catalog)).toEqual([])
  })
})

describe('activity timeline', () => {
  const stageLog = (overrides: Partial<ICrmOpportunityStageLog>) =>
    ({ ID: 1, OpportunityID: 1, FromPipelineID: 1, FromStageID: 1, ToPipelineID: 1, ToStageID: 2, Outcome: 0, Time: 0, UserID: 7, ...overrides }) as ICrmOpportunityStageLog
  const stageName = (_pipelineID: number, stageID: number) => `S${stageID}`

  it('offers the system types, then the company ones, and labels a deleted type', () => {
    const options = activityTypeOptions([{ ID: 41, Name: 'Cata', Icon: 'icon-[fa--flag]', ss: 1, upd: 0 }])
    expect(options[0].id).toBe(1)
    expect(options.at(-1)).toEqual({ id: 41, name: 'Cata', icon: 'icon-[fa--flag]' })
    expect(activityTypeOf(options, 99).name).toBe('Activity|Actividad')
  })

  it('merges activities and stage changes, newest first', () => {
    const timeline = buildOpportunityTimeline(
      [{ ID: 1, OpportunityID: 1, TypeID: 2, Text: 'a', AuthorID: 7, Created: 20 }],
      [stageLog({ Time: 10 }), stageLog({ ID: 2, Time: 30 })],
    )
    expect(timeline.map((entry) => entry.time)).toEqual([30, 20, 10])
    expect(timeline[1].kind).toBe('activity')
  })

  it('describes each kind of stage change', () => {
    expect(stageLogDescription(stageLog({ FromPipelineID: 0, FromStageID: 0 }), stageName)).toBe('Created in S2|Creada en S2')
    expect(stageLogDescription(stageLog({}), stageName)).toBe('Moved from S1 to S2|Pasó de S1 a S2')
    expect(stageLogDescription(stageLog({ ToStageID: 1 }), stageName)).toBe('Reopened in S1|Reabierta en S1')
    expect(stageLogDescription(stageLog({ ToStageID: 1, Outcome: 1 }), stageName)).toBe('Closed as won|Cerrada como ganada')
    expect(stageLogDescription(stageLog({ ToStageID: 1, Outcome: 2 }), stageName)).toBe('Closed as lost|Cerrada como perdida')
  })
})
