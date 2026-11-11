import type { IAttachment } from '#domain/attachments/attachments.ts'
import { readFileAsBase64 } from '#libs/helpers.ts'
import { GET, POST } from '#libs/ui-runtime.svelte.ts'
import { CrmCatalogService } from './crm-pipeline.svelte.ts'

// Mirrors backend/crm/types/activity.go. The fixed catalogs are in crm-activity-types.ts.

export interface ICrmActivityType {
  ID: number
  Name: string
  Icon: string
  ss: number
  upd: number
}

export interface ICrmActivity {
  ID: number
  OpportunityID: number
  TypeID: number
  Text: string
  Attachments?: IAttachment[]
  AuthorID: number
  Created: number
  /** The opportunity's pipeline when written, and the YYMM of Created: the report cache-group key. */
  PipelineID: number
  Period: number
}

export class CrmActivityTypesService extends CrmCatalogService<ICrmActivityType> {
  constructor(init = false) { super('crm-activity-types', init) }
}

export const postCrmActivityType = (activityTypes: CrmActivityTypesService, activityType: Partial<ICrmActivityType>): Promise<ICrmActivityType> =>
  activityTypes.writeAndSync('crm-activity-type', activityType)

export const getCrmOpportunityActivity = (opportunityID: number): Promise<ICrmActivity[]> =>
  GET({ route: `crm-opportunity-activity?opportunity-id=${opportunityID}` })

/** With taskID the activity completes that pending activity task in the same transaction. */
export const postCrmOpportunityActivity = (activity: Partial<ICrmActivity>, taskID = 0): Promise<ICrmActivity> =>
  POST({ data: { ...activity, TaskID: taskID }, route: 'crm-opportunity-activity' })

// uploadCrmAttachment sends one file, base64, to the opportunity's folder. It becomes an activity's
// file when the activity listing it is saved.
export const uploadCrmAttachment = async (opportunityID: number, file: File): Promise<IAttachment> =>
  await POST({
    data: { OpportunityID: opportunityID, Name: file.name, ContentType: file.type, Content: await readFileAsBase64(file) },
    route: 'crm-attachment-upload',
  }) as IAttachment
