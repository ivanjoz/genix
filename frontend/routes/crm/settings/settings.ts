import type {
  ICrmCustomField, ICrmCustomFieldOption, ICrmPipeline, ICrmPipelineField, ICrmPipelineStage, ICrmStageTaskTemplate,
} from '#services/crm/crm-pipeline.svelte.ts'
import { TASK_ASSIGN, TASK_KIND } from '#services/crm/crm-task.ts'

// Field types of backend/crm/types/custom_field.go, in the order the form offers them.
export const CUSTOM_FIELD_TYPE_OPTIONS = [
  { id: 1, name: 'Text|Texto' },
  { id: 2, name: 'Long text|Texto largo' },
  { id: 3, name: 'Integer|Entero' },
  { id: 4, name: 'Decimal|Decimal' },
  { id: 5, name: 'Money|Monto' },
  { id: 6, name: 'Date|Fecha' },
  { id: 7, name: 'Select|Selección' },
  { id: 8, name: 'Multi-select|Selección múltiple' },
  { id: 9, name: 'Checkbox|Casilla' },
  { id: 10, name: 'User|Usuario' },
]

const CUSTOM_FIELD_TYPE_SELECT = 7
const CUSTOM_FIELD_TYPE_MULTI_SELECT = 8

export const customFieldHasOptions = (fieldType: number) =>
  fieldType === CUSTOM_FIELD_TYPE_SELECT || fieldType === CUSTOM_FIELD_TYPE_MULTI_SELECT

const DEFAULT_STAGE_COLOR = '#6b7280'

export const makeEmptyStage = (): ICrmPipelineStage => ({
  ID: 0, Name: '', Color: DEFAULT_STAGE_COLOR, Probability: 0, StaleDays: 0, Status: 1, ForecastCategory: 1,
  RequiredStandardFields: [], ApproverUserID: 0, TaskTemplates: [],
})

/** ID 0: the server allocates the template's ID inside its stage on save. */
export const makeEmptyTaskTemplate = (): ICrmStageTaskTemplate =>
  ({ ID: 0, Kind: TASK_KIND.ACTIVITY, Title: '', ActivityTypeID: 1, EmailTemplateID: 0, DueDays: 1, AssignTo: TASK_ASSIGN.OWNER, AssignUserID: 0 })

export const makeEmptyPipeline = (): ICrmPipeline => ({
  ID: 0, Name: '', Description: '', Color: '#3b82f6', SortOrder: 0,
  Stages: [{ ...makeEmptyStage(), Name: 'New|Nuevo', Probability: 10 }],
  Fields: [], Members: [], GoalPeriodType: 1, ss: 1, upd: 0,
})

export const makeEmptyCustomField = (): ICrmCustomField => ({ ID: 0, Name: '', Type: 1, Options: [], ss: 1, upd: 0 })

export const makeEmptyOption = (): ICrmCustomFieldOption => ({ ID: 0, Name: '', Color: '', Status: 1 })

/** A new option just disappears; a stored one is soft-deleted because opportunities may hold its ID. */
export const removeOption = (options: ICrmCustomFieldOption[], index: number): ICrmCustomFieldOption[] =>
  options[index].ID === 0
    ? options.filter((_, optionIndex) => optionIndex !== index)
    : options.map((option, optionIndex) => optionIndex === index ? { ...option, Status: 0 } : option)

/** Moves the item at `index` one place up (-1) or down (+1); out-of-range moves leave the list as is. */
export const moveItem = <T>(items: T[], index: number, direction: -1 | 1): T[] => {
  const targetIndex = index + direction
  if (targetIndex < 0 || targetIndex >= items.length) return items
  const reordered = [...items]
  ;[reordered[index], reordered[targetIndex]] = [reordered[targetIndex], reordered[index]]
  return reordered
}

/** Stages the editor shows: a soft-deleted one stays in the record (history keeps its label) but not on screen. */
export const activeStages = (pipeline: ICrmPipeline) => pipeline.Stages.filter((stage) => stage.Status !== 0)

/** Removing a stage that already has an ID is a soft delete the server checks; a new one just disappears. */
export const removeStage = (stages: ICrmPipelineStage[], index: number): ICrmPipelineStage[] =>
  stages[index].ID === 0
    ? stages.filter((_, stageIndex) => stageIndex !== index)
    : stages.map((stage, stageIndex) => stageIndex === index ? { ...stage, Status: 0 } : stage)

/** Toggles a custom field on a pipeline, keeping the display order of the ones already picked. */
export const toggleField = (fields: ICrmPipelineField[], fieldID: number): ICrmPipelineField[] =>
  fields.some((field) => field.FieldID === fieldID)
    ? fields.filter((field) => field.FieldID !== fieldID)
    : [...fields, { FieldID: fieldID, RequiredFromStageID: 0 }]

/** Requires a field from a stage on (0 makes it optional). */
export const setFieldRequiredFrom = (fields: ICrmPipelineField[], fieldID: number, stageID: number): ICrmPipelineField[] =>
  fields.map((field) => field.FieldID === fieldID ? { ...field, RequiredFromStageID: stageID } : field)

/** Marks a standard field (STANDARD_FIELD) as required, or not, by the stage at `stageIndex`. */
export const setStageStandardField = (
  stages: ICrmPipelineStage[], stageIndex: number, standardField: number, isRequired: boolean,
): ICrmPipelineStage[] =>
  stages.map((stage, index) => {
    if (index !== stageIndex) return stage
    const others = (stage.RequiredStandardFields ?? []).filter((candidate) => candidate !== standardField)
    return { ...stage, RequiredStandardFields: isRequired ? [...others, standardField] : others }
  })

/** Sets who approves a forward move into the stage at `stageIndex` (0 = no approval). */
export const setStageApprover = (stages: ICrmPipelineStage[], stageIndex: number, approverUserID: number): ICrmPipelineStage[] =>
  stages.map((stage, index) => index === stageIndex ? { ...stage, ApproverUserID: approverUserID } : stage)

/** Edits the task templates of the stage at `stageIndex`: `change` receives the current list. */
export const changeStageTemplates = (
  stages: ICrmPipelineStage[], stageIndex: number, change: (templates: ICrmStageTaskTemplate[]) => ICrmStageTaskTemplate[],
): ICrmPipelineStage[] =>
  stages.map((stage, index) => index === stageIndex ? { ...stage, TaskTemplates: change(stage.TaskTemplates ?? []) } : stage)

/** A template fixed to a user needs that user; an owner template must not carry a stale one. */
export const setTemplateAssignment = (template: ICrmStageTaskTemplate, assignTo: number, assignUserID = 0): ICrmStageTaskTemplate =>
  ({ ...template, AssignTo: assignTo, AssignUserID: assignTo === TASK_ASSIGN.USER ? assignUserID : 0 })
