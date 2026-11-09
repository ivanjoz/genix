import { GetHandler } from '#libs/ui-runtime.svelte.ts';
import { uploadAttachment } from '../story/story.svelte.ts';
import type { IDocument } from './documents.ts';

const DOCUMENTS_ROUTE = 'projects-documents'

// One project's documents (GET.projects-documents?project-id=), delta-cached per project: the
// project's own documents and the copies of its stories' files.
export class ProjectDocumentsService extends GetHandler<IDocument> {
  useCache = { min: 1, ver: 2 }
  projectID: number

  handler(documents: IDocument[]): void {
    this.records = (documents || []).filter((document) => document.ss !== 0)
  }

  // Both writes bring back the delta of the project's documents (GetHandler.writeAndSync), which
  // handler() turns into the list: a deleted document arrives as a removal.

  // The file goes straight to storage, then becomes a project document.
  async addDocument(file: File) {
    const attachment = await uploadAttachment(this.projectID, file)
    await this.writeAndSync(DOCUMENTS_ROUTE, { ...attachment, ProjectID: this.projectID, StoryID: 0, ss: 1 })
  }

  // The backend deletes the file too.
  async deleteDocument(document: IDocument) {
    await this.writeAndSync(DOCUMENTS_ROUTE, { ...document, ss: 0 })
  }

  constructor(projectID: number) {
    super()
    this.projectID = projectID
    this.route = `${DOCUMENTS_ROUTE}?project-id=${projectID}`
  }
}
