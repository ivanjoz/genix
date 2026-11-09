// Files attached to an entry (a project story or comment, a CRM activity). Each feature has its own
// upload route and folder; the shape and the limits are shared.

// The backend's cap (MaxAttachmentBytes in backend/projects and backend/crm/types): the file travels
// base64 in the upload request, inside a Lambda's 6 MB payload.
export const MAX_ATTACHMENT_BYTES = 4 * 1024 * 1024
export const MAX_ATTACHMENTS = 30

export interface IAttachment {
  Key: string
  Name: string
  ContentType: string
  Size: number
  UploadedBy?: number
  UploadedAt?: number
}

// validateAttachmentFile is checked before the file is sent.
export const validateAttachmentFile = (file: { size: number }, attachedCount: number): string => {
  if (file.size <= 0 || file.size > MAX_ATTACHMENT_BYTES) {
    return 'A file can weigh up to 4 MB.|Un archivo puede pesar hasta 4 MB.'
  }
  if (attachedCount >= MAX_ATTACHMENTS) {
    return 'Up to 30 files can be attached.|Se pueden adjuntar hasta 30 archivos.'
  }
  return ''
}
