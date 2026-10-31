import { GetHandler, POST } from '#libs/ui-runtime.svelte.ts';
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

  #applySavedDocument(savedDocument: IDocument) {
    const otherDocuments = this.records.filter((document) => document.ID !== savedDocument.ID)
    // The API omits a zero ss: a deleted document arrives without it.
    this.records = savedDocument.ss ? [...otherDocuments, savedDocument] : otherDocuments
  }

  // The file goes straight to storage, then becomes a project document.
  async addDocument(file: File) {
    const attachment = await uploadAttachment(this.projectID, file)
    this.#applySavedDocument(await POST({
      data: { ...attachment, ProjectID: this.projectID, StoryID: 0, ss: 1 }, route: DOCUMENTS_ROUTE, refreshRoutes: [DOCUMENTS_ROUTE],
    }))
  }

  // The backend deletes the file too.
  async deleteDocument(document: IDocument) {
    this.#applySavedDocument(await POST({ data: { ...document, ss: 0 }, route: DOCUMENTS_ROUTE, refreshRoutes: [DOCUMENTS_ROUTE] }))
  }

  constructor(projectID: number) {
    super()
    this.projectID = projectID
    this.route = `${DOCUMENTS_ROUTE}?project-id=${projectID}`
  }
}
