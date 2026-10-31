import { readFileAsBase64 } from '#libs/helpers.ts';
import { GetHandler, POST } from '#libs/ui-runtime.svelte.ts';
import { BACKLOG_ROUTES } from '../project.svelte.ts';
import { makeEmptyStoryDetail, type IAttachment, type IStoryActivity, type IStoryDetail } from './story.ts';

// One story's detail or timeline, delta-cached per story (GET.projects-story-<thing>?project-id=&story-id=):
// a story opened before shows at once, then what changed merges in.
class StoryRecordsService<T extends { ID: number, ss: number }> extends GetHandler<T> {
  useCache = { min: 1, ver: 1 }
  inferRemoveFromStatus = true

  handler(records: T[]): void {
    this.records = []
    this.recordsMap = new Map()
    this.addSavedRecords(...(records || []))
  }

  constructor(routeName: string, projectID: number, storyID: number) {
    super()
    this.route = `${routeName}?project-id=${projectID}&story-id=${storyID}`
  }
}

// A story's detail (GET.projects-story-detail): one record, keyed by StoryID in the cache. It has no
// delta log, so the server answers it whole every time.
class StoryDetailService extends GetHandler<IStoryDetail & { ID: number, ss: number }> {
  useCache = { min: 1, ver: 1 }
  keyID = 'StoryID'

  handler(details: (IStoryDetail & { ID: number, ss: number })[]): void {
    this.records = (details || []).filter((detail) => detail.ss !== 0)
  }

  constructor(projectID: number, storyID: number) {
    super()
    this.route = `projects-story-detail?project-id=${projectID}&story-id=${storyID}`
  }
}

// OpenStory is what the story panel reads when a story opens: its detail and its timeline of
// comments and history.
export class OpenStory {
  #detail = $state<StoryDetailService | null>(null)
  #activity = $state<StoryRecordsService<IStoryActivity & { ss: number }> | null>(null)
  #projectID = 0
  #storyID = 0

  // The cached detail, or an empty one once the server answered that the story has none yet.
  get detail(): IStoryDetail | null {
    if (!this.#detail) { return null }
    return this.#detail.records[0] || (this.#detail.hasFetched ? makeEmptyStoryDetail(this.#projectID, this.#storyID) : null)
  }
  get activity(): IStoryActivity[] { return this.#activity?.records || [] }
  // Something to paint: the cached timeline, or the server's answer (then an empty one is real).
  get isLoaded(): boolean { return !!this.#activity && (this.#activity.records.length > 0 || this.#activity.hasFetched) }

  async load(projectID: number, storyID: number) {
    this.#projectID = projectID
    this.#storyID = storyID
    this.#detail = new StoryDetailService(projectID, storyID)
    this.#activity = new StoryRecordsService<IStoryActivity & { ss: number }>('projects-story-activity', projectID, storyID)
    await Promise.all([this.#detail.fetch(), this.#activity.fetch()])
  }

  reset() {
    this.#detail = null
    this.#activity = null
  }

  async addComment(projectID: number, storyID: number, text: string, attachments: IAttachment[]): Promise<IStoryActivity> {
    const comment = await POST({
      data: { ProjectID: projectID, StoryID: storyID, Text: text, Attachments: attachments }, route: 'projects-story-comments',
      refreshRoutes: BACKLOG_ROUTES,
    }) as IStoryActivity & { ss: number }
    this.#activity?.addSavedRecords(comment)
    return comment
  }
}

// uploadAttachment sends one file, base64, to the backend, which stores it in the project's folder
// and answers the attachment. It becomes one of the story's files when the story or comment listing
// it is saved.
export const uploadAttachment = async (projectID: number, file: File): Promise<IAttachment> => {
  return await POST({
    data: { ProjectID: projectID, Name: file.name, ContentType: file.type, Content: await readFileAsBase64(file) },
    route: 'projects-attachment-upload',
  }) as IAttachment
}
