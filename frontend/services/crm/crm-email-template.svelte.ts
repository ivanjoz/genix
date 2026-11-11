import { GET, POST } from '#libs/ui-runtime.svelte.ts'
import type { ICrmActivity } from './crm-activity.svelte.ts'
import { CrmCatalogService } from './crm-pipeline.svelte.ts'

// Mirrors backend/crm/types/email_template.go. The placeholders are in crm-email-template.ts.
export { EMAIL_PLACEHOLDERS } from './crm-email-template.ts'

export interface ICrmEmailTemplate {
  ID: number
  Name: string
  Subject: string
  BodyHTML: string
  ss: number
  upd: number
}

/** A template rendered for one opportunity, for the seller to review before sending. */
export interface ICrmEmailDraft {
  To: string[]
  Subject: string
  HTMLBody: string
}

export interface ICrmOpportunityEmail {
  OpportunityID: number
  TemplateID: number
  To: string[]
  Cc: string[]
  Subject: string
  HTMLBody: string
  /** The "send email" task the email fulfils; 0 = none. */
  TaskID: number
}

export class CrmEmailTemplatesService extends CrmCatalogService<ICrmEmailTemplate> {
  constructor(init = false) { super('crm-email-templates', init) }
}

export const postCrmEmailTemplate = (emailTemplates: CrmEmailTemplatesService, emailTemplate: Partial<ICrmEmailTemplate>): Promise<ICrmEmailTemplate> =>
  emailTemplates.writeAndSync('crm-email-template', emailTemplate)

export const getCrmOpportunityEmailDraft = (opportunityID: number, templateID: number): Promise<ICrmEmailDraft> =>
  GET({ route: `crm-opportunity-email-draft?opportunity-id=${opportunityID}&template-id=${templateID}` })

/** Sends the email and answers the Email activity it left on the opportunity's timeline. */
export const postCrmOpportunityEmail = (email: ICrmOpportunityEmail): Promise<ICrmActivity> =>
  POST({ data: email, route: 'crm-opportunity-email' })
