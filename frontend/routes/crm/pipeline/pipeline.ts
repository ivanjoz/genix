import { FIELD_TYPE, type IFormField } from '#components/form/fields-form.ts'
import type { ICrmActivity, ICrmActivityType } from '#services/crm/crm-activity.svelte.ts'
import { SYSTEM_ACTIVITY_TYPES, type IActivityTypeOption } from '#services/crm/crm-activity-types.ts'
import type { ICrmCustomFieldValue, ICrmOpportunity, ICrmOpportunityStageLog } from '#services/crm/crm-opportunity.svelte.ts'
import type { ICrmCustomField, ICrmPipeline, ICrmPipelineField, ICrmPipelineStage } from '#services/crm/crm-pipeline.svelte.ts'
import {
  OUTCOME_LOST, OUTCOME_WON, STAGE_LOG_KIND_CLOSED, STAGE_LOG_KIND_CREATED, STAGE_LOG_KIND_EDITED,
} from '#services/crm/crm-stage-log.ts'

// Levels of backend/crm/types/pipeline.go. Not imported from the service: that module pulls in the
// Svelte runtime, and this file stays testable without it.
const LEVEL_EDIT = 2
const LEVEL_FULL = 3

// Mirrors the backend rules (backend/crm/types/pipeline_rules.go) only to disable dragging and
// buttons. The backend check is the one that counts.

export const memberLevel = (pipeline: ICrmPipeline | undefined, userID: number) =>
  pipeline?.Members.find((member) => member.UserID === userID)?.Level ?? 0

export const canCreateOpportunity = (level: number) => level >= LEVEL_EDIT
export const canMoveOpportunity = (level: number) => level >= LEVEL_EDIT
export const canChangeOwner = (level: number) => level >= LEVEL_FULL
export const canEditOpportunity = (level: number, opportunity: ICrmOpportunity, userID: number) =>
  level >= LEVEL_FULL
  || (level === LEVEL_EDIT && (opportunity.CreatedBy === userID || opportunity.OwnerUserID === userID))

/** The pipelines a user sees: only the ones they are a member of. */
export const visiblePipelines = (pipelines: ICrmPipeline[], userID: number) =>
  pipelines.filter((pipeline) => memberLevel(pipeline, userID) > 0).sort((left, right) => left.SortOrder - right.SortOrder)

export const activeStagesOf = (pipeline: ICrmPipeline | undefined): ICrmPipelineStage[] =>
  (pipeline?.Stages ?? []).filter((stage) => stage.Status !== 0)

/** SUnixTime: seconds since 2001-09-09 halved, the backend's datetime format. */
export const sUnixTimeOf = (unixMillis: number) => Math.floor((unixMillis / 1000 - 1e9) / 2)

/** True when the card has sat in a stage with a stale threshold for longer than that many days. */
export const isStale = (opportunity: ICrmOpportunity, stage: ICrmPipelineStage | undefined, nowSUnixTime: number) => {
  if (!stage || stage.StaleDays <= 0 || !opportunity.StageEnteredTime) return false
  return (nowSUnixTime - opportunity.StageEnteredTime) * 2 / 86400 > stage.StaleDays
}

export interface IColumnTotals {
  count: number
  amount: number
  /** Amount × probability, the column's forecast. */
  weightedAmount: number
}

export const columnTotals = (opportunities: ICrmOpportunity[]): IColumnTotals => ({
  count: opportunities.length,
  amount: opportunities.reduce((sum, opportunity) => sum + opportunity.Amount, 0),
  weightedAmount: Math.round(opportunities.reduce((sum, opportunity) => sum + opportunity.Amount * opportunity.Probability / 100, 0)),
})

/** Required fields of the target pipeline the opportunity has no value for: a move needs them first. */
export const missingRequiredFieldIDs = (values: ICrmCustomFieldValue[], targetFields: ICrmPipelineField[]) =>
  targetFields
    .filter((field) => field.Required && !values.some((value) => value.FieldID === field.FieldID))
    .map((field) => field.FieldID)

// ---- custom fields <-> FieldsForm -------------------------------------------------------------

const fieldKey = (fieldID: number) => `f${fieldID}`

export interface IFieldOptionsByType {
  /** Options for the user field type, as { id, name }. */
  users: { id: number, name: string }[]
}

/** Converts the pipeline's picked fields and the catalog into the generic form's field list. */
export const buildOpportunityFormFields = (
  pipeline: ICrmPipeline, catalog: Map<number, ICrmCustomField>, options: IFieldOptionsByType,
): IFormField[] =>
  pipeline.Fields.flatMap((pipelineField) => {
    const customField = catalog.get(pipelineField.FieldID)
    if (!customField) return []
    // The catalog's type numbers are the form's: both follow backend/crm/types/custom_field.go.
    const formType = customField.Type
    const choices = formType === FIELD_TYPE.search
      ? options.users
      : customField.Options.filter((option) => option.Status !== 0).map((option) => ({ id: option.ID, name: option.Name }))
    return [{ key: fieldKey(customField.ID), label: customField.Name, type: formType, options: choices, required: pipelineField.Required }]
  })

/** Stored values -> the generic form's `values` map. */
export const writeCustomValues = (storedValues: ICrmCustomFieldValue[], catalog: Map<number, ICrmCustomField>) => {
  const formValues: Record<string, any> = {}
  for (const storedValue of storedValues) {
    const fieldType = catalog.get(storedValue.FieldID)?.Type
    formValues[fieldKey(storedValue.FieldID)] =
      fieldType === FIELD_TYPE.text || fieldType === FIELD_TYPE.longText ? storedValue.Text
        : fieldType === FIELD_TYPE.multiSelect ? [...storedValue.OptionIDs]
          : storedValue.Number
  }
  return formValues
}

/** The generic form's `values` map -> values to store; fields left empty are not stored. */
export const readCustomValues = (formValues: Record<string, any>, catalog: Map<number, ICrmCustomField>): ICrmCustomFieldValue[] =>
  [...catalog.values()].flatMap((customField) => {
    const formValue = formValues[fieldKey(customField.ID)]
    const isEmpty = formValue === undefined || formValue === null || formValue === '' || (Array.isArray(formValue) && formValue.length === 0)
    if (isEmpty) return []
    const storedValue: ICrmCustomFieldValue = { FieldID: customField.ID, Number: 0, Text: '', OptionIDs: [] }
    if (customField.Type === FIELD_TYPE.text || customField.Type === FIELD_TYPE.longText) storedValue.Text = String(formValue)
    else if (customField.Type === FIELD_TYPE.multiSelect) storedValue.OptionIDs = formValue.map(Number)
    else storedValue.Number = Number(formValue)
    return [storedValue]
  })

/** The activity types a user picks from: the fixed ones first, then the company's own. */
export const activityTypeOptions = (companyTypes: ICrmActivityType[]): IActivityTypeOption[] => [
  ...SYSTEM_ACTIVITY_TYPES,
  ...companyTypes.map((companyType) => ({ id: companyType.ID, name: companyType.Name, icon: companyType.Icon })),
]

/** The type an activity shows. A company type deleted since keeps its entries, under a generic label. */
export const activityTypeOf = (typeOptions: IActivityTypeOption[], typeID: number): IActivityTypeOption =>
  typeOptions.find((option) => option.id === typeID) ?? { id: typeID, name: 'Activity|Actividad', icon: 'icon-[fa--comment]' }

export type OpportunityTimelineEntry =
  | { kind: 'activity', time: number, activity: ICrmActivity }
  | { kind: 'stage', time: number, stageLog: ICrmOpportunityStageLog }

/** The "Activity" tab: what the reps logged and the stage history, in one list, newest first. */
export const buildOpportunityTimeline = (activities: ICrmActivity[], stageLogs: ICrmOpportunityStageLog[]): OpportunityTimelineEntry[] => [
  ...activities.map((activity) => ({ kind: 'activity' as const, time: activity.Created, activity })),
  ...stageLogs.map((stageLog) => ({ kind: 'stage' as const, time: stageLog.Time, stageLog })),
].sort((left, right) => right.time - left.time)

/** One history line, bilingual ("English|Spanish"), chosen by the row's Kind. */
export const stageLogDescription = (stageLog: ICrmOpportunityStageLog, stageName: (pipelineID: number, stageID: number) => string): string => {
  const fromStage = stageName(stageLog.FromPipelineID, stageLog.FromStageID)
  const toStage = stageName(stageLog.ToPipelineID, stageLog.ToStageID)
  if (stageLog.Kind === STAGE_LOG_KIND_CREATED) return `Created in ${toStage}|Creada en ${toStage}`
  if (stageLog.Kind === STAGE_LOG_KIND_EDITED) {
    const changes: [string, string][] = []
    if (stageLog.FromAmount !== stageLog.ToAmount) changes.push(['amount', 'monto'])
    if (stageLog.FromOwnerUserID !== stageLog.ToOwnerUserID) changes.push(['owner', 'responsable'])
    if (stageLog.FromExpectedCloseDate !== stageLog.ToExpectedCloseDate) changes.push(['expected close date', 'fecha de cierre'])
    return `Changed ${changes.map((change) => change[0]).join(', ')}|Cambió ${changes.map((change) => change[1]).join(', ')}`
  }
  if (stageLog.Kind === STAGE_LOG_KIND_CLOSED) {
    if (stageLog.ToOutcome === OUTCOME_WON) return 'Closed as won|Cerrada como ganada'
    if (stageLog.ToOutcome === OUTCOME_LOST) return 'Closed as lost|Cerrada como perdida'
    return `Reopened in ${toStage}|Reabierta en ${toStage}`
  }
  return `Moved from ${fromStage} to ${toStage}|Pasó de ${fromStage} a ${toStage}`
}
