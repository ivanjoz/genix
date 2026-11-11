// One row of the sent_emails table (backend/core/types/sent_emails.go) that core.SendEmail writes.
// The list arrives without HTMLBody: the preview loads it by ID.
export interface ISentEmail {
  CompanyID: number
  ID: number
  To: string[]
  Cc?: string[]
  Bcc?: string[]
  Subject: string
  HTMLBody?: string
  AttachmentNames?: string[]
  Sender: string
  SendError?: string
  ss: number
  upd: number
}

// The page size of GET.sent-emails: a shorter page means there is nothing older to load.
export const SENT_EMAILS_PAGE_SIZE = 100

// Mirrors the SentEmailStatus* constants of the backend.
export const SENT_EMAIL_STATUS = { STORED: 1, SENT: 2, FAILED: 3 } as const

// How the email reached its recipients, with the text color the inbox shows it in.
export const describeDelivery = (sentEmail: ISentEmail): { text: string, css: string } => {
  if (sentEmail.ss === SENT_EMAIL_STATUS.SENT) {
    return { text: 'Sent|Enviado', css: 'text-green-fg' }
  }
  if (sentEmail.ss === SENT_EMAIL_STATUS.FAILED) {
    return { text: 'Send failed|Falló el envío', css: 'text-red-fg' }
  }
  return { text: 'Only stored, not sent|Solo registrado, no enviado', css: 'text-purple-fg' }
}

// Every recipient of the email, To first, as the list row shows them.
export const listAllRecipients = (sentEmail: ISentEmail): string => {
  return [...(sentEmail.To || []), ...(sentEmail.Cc || []), ...(sentEmail.Bcc || [])].join(', ')
}

// True when the subject or any address holds the filter text, ignoring case.
export const sentEmailMatchesFilter = (sentEmail: ISentEmail, filterText: string): boolean => {
  const normalizedFilter = filterText.trim().toLowerCase()
  if (!normalizedFilter) { return true }
  return [sentEmail.Subject || '', listAllRecipients(sentEmail)].some((text) => text.toLowerCase().includes(normalizedFilter))
}
