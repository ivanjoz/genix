// The Documents page's rules: the project's own documents and its stories' files, grouped by story.
// Pure: no Svelte, no fetch.
import type { ISprint, IStory } from '../project.ts';

// IDocument mirrors backend/projects/types Document: a project document (StoryID 0) or the copy of
// a file attached to a story or to one of its comments.
export interface IDocument {
  ProjectID: number
  ID: number
  StoryID?: number
  Key: string
  ActivityID?: number
  Name: string
  ContentType: string
  Size: number
  UploadedBy?: number
  UploadedAt?: number
  ss: number
  upd?: number
}

export interface IStoryDocuments {
  story: IStory
  // The story's last sprint, or "Backlog" when it was never planned.
  sprintLabel: string
  documents: IDocument[]
}

const newestUploadFirst = (left: IDocument, right: IDocument) => (right.UploadedAt || 0) - (left.UploadedAt || 0)

// projectDocuments are the documents added on this page (no story), newest first.
export const projectDocuments = (documents: IDocument[]): IDocument[] =>
  documents.filter((document) => !document.StoryID && document.ss !== 0).sort(newestUploadFirst)

// storyDocumentGroups is one group per story holding files, the newest story first. The files of a
// deleted story are not shown: the story is gone from the backlog.
export const storyDocumentGroups = (documents: IDocument[], storiesByID: Map<number, IStory>, sprintsByID: Map<number, ISprint>): IStoryDocuments[] => {
  const documentsByStoryID = new Map<number, IDocument[]>()
  for (const document of documents) {
    if (!document.StoryID || document.ss === 0) { continue }
    documentsByStoryID.set(document.StoryID, [...(documentsByStoryID.get(document.StoryID) || []), document])
  }
  const groups: IStoryDocuments[] = []
  for (const [storyID, storyDocuments] of documentsByStoryID) {
    const story = storiesByID.get(storyID)
    if (!story || story.ss === 0) { continue }
    const lastSprint = story.LastSprintID ? sprintsByID.get(story.LastSprintID) : undefined
    groups.push({ story, sprintLabel: lastSprint?.Name || 'Backlog|Backlog', documents: storyDocuments.sort(newestUploadFirst) })
  }
  // Within a project the packed story IDs follow the story numbers.
  return groups.sort((left, right) => right.story.ID - left.story.ID)
}
