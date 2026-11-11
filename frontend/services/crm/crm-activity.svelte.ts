import type { IAttachment } from '#domain/attachments/attachments.ts'
import { readFileAsBase64 } from '#libs/helpers.ts'
import { GETWithGroupCache, POST } from '#libs/ui-runtime.svelte.ts'
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

/** The opportunity's timeline entries: one cache group per opportunity, downloaded again only after a new entry. */
export const getCrmOpportunityActivity = async (opportunityID: number): Promise<ICrmActivity[]> => {
  const groups = await GETWithGroupCache<ICrmActivity>('crm-opportunity-activity', { 'opportunity-id': String(opportunityID) })
  return groups.flatMap((group) => group.records || []).filter((activity) => (activity?.ID || 0) > 0)
}

/** An entry written from the opportunity panel. One that completes a task goes through completeCrmActivityTask. */
export const postCrmOpportunityActivity = (activity: Partial<ICrmActivity>): Promise<ICrmActivity> =>
  POST({ data: activity, route: 'crm-opportunity-activity' })

// uploadCrmAttachment sends one file, base64, to the opportunity's folder. It becomes an activity's
// file when the activity listing it is saved.
export const uploadCrmAttachment = async (opportunityID: number, file: File): Promise<IAttachment> =>
  await POST({
    data: { OpportunityID: opportunityID, Name: file.name, ContentType: file.type, Content: await readFileAsBase64(file) },
    route: 'crm-attachment-upload',
  }) as IAttachment
