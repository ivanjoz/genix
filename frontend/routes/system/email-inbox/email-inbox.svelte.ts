import { GET } from '#libs/ui-runtime.svelte.ts';
import type { ISentEmail } from './email-inbox.ts';

// A page of the company's emails, newest first and without bodies; beforeID pages back from the
// oldest one already shown.
export const loadSentEmails = async (companyID: number, beforeID?: number): Promise<ISentEmail[]> => {
  const route = `sent-emails?target-company-id=${companyID}` + (beforeID ? `&before=${beforeID}` : '')
  return await GET({ route }) as ISentEmail[] || []
}

// One email with its HTML body, for the preview.
export const loadSentEmail = async (companyID: number, sentEmailID: number): Promise<ISentEmail> => {
  return await GET({ route: `sent-email?target-company-id=${companyID}&id=${sentEmailID}` }) as ISentEmail
}
