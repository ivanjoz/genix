import { GetHandler } from '#libs/ui-runtime.svelte.ts'

// Mirrors backend/crm/types: pipeline.go, custom_field.go, lost_reason.go.

export const PipelineLevel = { VIEW: 1, EDIT: 2, FULL: 3 } as const

export interface ICrmPipelineStage {
  ID: number
  Name: string
  Color: string
  Probability: number
  StaleDays: number
  Status: number
}

export interface ICrmPipelineField {
  FieldID: number
  Required: boolean
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
  useCache = { min: 5, ver: 2 }
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
