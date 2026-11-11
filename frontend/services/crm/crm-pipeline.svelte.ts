import { GetHandler } from '#libs/ui-runtime.svelte.ts'

// Mirrors backend/crm/types: pipeline.go, custom_field.go, lost_reason.go.

export { FORECAST_CATEGORY } from './crm-forecast-category.ts'

export const PipelineLevel = { VIEW: 1, EDIT: 2, FULL: 3 } as const

export interface ICrmPipelineStage {
  ID: number
  Name: string
  Color: string
  Probability: number
  StaleDays: number
  Status: number
  /** FORECAST_CATEGORY: how the forecast counts open opportunities of this stage. */
  ForecastCategory: number
  /** STANDARD_FIELD values an opportunity must have to enter this stage or any later one. */
  RequiredStandardFields: number[]
  /** A forward move into this stage waits for this user's approval; 0 = no approval. */
  ApproverUserID: number
  /** Tasks proposed to the opportunity every time it enters this stage. */
  TaskTemplates: ICrmStageTaskTemplate[]
}

export interface ICrmStageTaskTemplate {
  ID: number
  /** TASK_KIND.ACTIVITY or TASK_KIND.SEND_EMAIL. */
  Kind: number
  Title: string
  /** ACTIVITY: the type to log. SEND_EMAIL: always the Email type (the server sets it). */
  ActivityTypeID: number
  /** SEND_EMAIL: the template the email starts from. */
  EmailTemplateID: number
  /** Days after entering the stage. */
  DueDays: number
  /** TASK_ASSIGN: the opportunity's owner, or the fixed AssignUserID. */
  AssignTo: number
  AssignUserID: number
}

export interface ICrmPipelineField {
  FieldID: number
  /** 0 = optional; otherwise required to enter that stage or any later one (by board position). */
  RequiredFromStageID: number
}

export interface ICrmPipelineMember {
  UserID: number
  Level: number
}

export interface ICrmPipeline {
  ID: number
  Name: string
  Description: string
  Color: string
  SortOrder: number
  Stages: ICrmPipelineStage[]
  Fields: ICrmPipelineField[]
  Members: ICrmPipelineMember[]
  /** GOAL_PERIOD: goals and forecast by month or by week. */
  GoalPeriodType: number
  ss: number
  upd: number
}

export interface ICrmCustomFieldOption {
  ID: number
  Name: string
  Color: string
  Status: number
}

export interface ICrmCustomField {
  ID: number
  Name: string
  Type: number
  Options: ICrmCustomFieldOption[]
  ss: number
  upd: number
}

export interface ICrmLostReason {
  ID: number
  Name: string
  ss: number
  upd: number
}

/** A delta list of one CRM catalog: every route syncs the whole table, active rows only. */
export abstract class CrmCatalogService<T extends { ID: number, ss: number }> extends GetHandler<T> {
  useCache = { min: 5, ver: 3 }
  inferRemoveFromStatus = true

  records: T[] = $state([])
  recordsMap: Map<number, T> = $state(new Map())

  handler(result: T[]): void {
    const activeRecords = (result || []).filter((record) => record.ss > 0)
    this.records = activeRecords
    this.recordsMap = new Map(activeRecords.map((record) => [record.ID, record]))
  }

  // The route is a constructor argument, not a subclass field: a field initializer would run
  // after super() and undo the route the base constructor fetched with.
  constructor(route: string, init = false) {
    super()
    this.route = route
    if (init) this.fetch()
  }
}

export class CrmPipelinesService extends CrmCatalogService<ICrmPipeline> {
  constructor(init = false) { super('crm-pipelines', init) }

  // A stage with no required standard field or task template has none on the wire (a nil slice is
  // sent as null); every reader treats both as lists, so they become lists here, once.
  handler(result: ICrmPipeline[]): void {
    super.handler((result || []).map((pipeline) => ({
      ...pipeline,
      Stages: (pipeline.Stages ?? []).map((stage) => ({
        ...stage,
        RequiredStandardFields: stage.RequiredStandardFields ?? [],
        TaskTemplates: stage.TaskTemplates ?? [],
      })),
    })))
  }
}

export class CrmCustomFieldsService extends CrmCatalogService<ICrmCustomField> {
  constructor(init = false) { super('crm-custom-fields', init) }

  // A field with no options has none on the wire: colbin sends the nil slice as null and the JSON
  // path omits it. Every reader treats Options as a list, so it becomes one here, once.
  handler(result: ICrmCustomField[]): void {
    super.handler((result || []).map((field) => ({ ...field, Options: field.Options ?? [] })))
  }
}

export class CrmLostReasonsService extends CrmCatalogService<ICrmLostReason> {
  constructor(init = false) { super('crm-lost-reasons', init) }
}

// Each write brings back the delta of the list it changes (GetHandler.writeAndSync): no GET follows.
export const postCrmPipeline = (pipelines: CrmPipelinesService, pipeline: Partial<ICrmPipeline>): Promise<ICrmPipeline> =>
  pipelines.writeAndSync('crm-pipeline', pipeline)

/** Replaces who can work in one pipeline; the only writer of Members. */
export const postCrmPipelineMembers = (
  pipelines: CrmPipelinesService, pipelineID: number, members: ICrmPipelineMember[],
): Promise<ICrmPipeline> =>
  pipelines.writeAndSync('crm-pipeline-members', { PipelineID: pipelineID, Members: members })

// A pipeline holds its custom fields, so the pipelines list is marked and read by the page that shows it.
export const postCrmCustomField = (customFields: CrmCustomFieldsService, customField: Partial<ICrmCustomField>): Promise<ICrmCustomField> =>
  customFields.writeAndSync('crm-custom-field', customField, ['crm-pipelines'])

export const postCrmLostReason = (lostReasons: CrmLostReasonsService, lostReason: Partial<ICrmLostReason>): Promise<ICrmLostReason> =>
  lostReasons.writeAndSync('crm-lost-reason', lostReason)
