import { getRecordWithCache, type IMinimalRecord, type IRecordRef } from '@genix/ui';
import { readFileAsBase64 } from '#libs/helpers.ts';
import { GetHandler, POST } from '#libs/ui-runtime.svelte.ts';
import { BACKLOG_ROUTES } from '../project.svelte.ts';
import type { IAttachment } from '#domain/attachments/attachments.ts';
import type { IStoryActivity, IStoryDetail } from './story.ts';

// One story's timeline, delta-cached per story (GET.projects-story-<thing>?project-id=&story-id=):
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

// The stories' details by story ID (GET.projects-story-details-ids), cached per story by genix-ui.
const STORY_DETAILS_ROUTE = 'projects-story-details-ids'

// OpenStory is what the story panel reads when a story opens: its detail and its timeline of
// comments and history.
export class OpenStory {
  // The detail paints from the browser's by-IDs cache, then revalidates: the server answers it again
  // only when its slot moved. Every story has one: it is written with the story.
  #detail = $state<IRecordRef<IStoryDetail & IMinimalRecord> | null>(null)
  #activity = $state<StoryRecordsService<IStoryActivity & { ss: number }> | null>(null)

  get detail(): IStoryDetail | null { return this.#detail?.record || null }
  get activity(): IStoryActivity[] { return this.#activity?.records || [] }
  // Something to paint: the cached timeline, or the server's answer (then an empty one is real).
  get isLoaded(): boolean { return !!this.#activity && (this.#activity.records.length > 0 || this.#activity.hasFetched) }

  async load(projectID: number, storyID: number) {
    this.#detail = getRecordWithCache<IStoryDetail & IMinimalRecord>(STORY_DETAILS_ROUTE, storyID)
    this.#activity = new StoryRecordsService<IStoryActivity & { ss: number }>('projects-story-activity', projectID, storyID)
    await this.#activity.fetch()
  }

  reset() {
    this.#detail = null
    this.#activity = null
  }

  // The timeline comes back with the write (GetHandler.writeAndSync); BACKLOG_ROUTES marks the
  // story's counter and the Documents page's copies of the comment's files.
  async addComment(projectID: number, storyID: number, text: string, attachments: IAttachment[]): Promise<IStoryActivity> {
    if (!this.#activity) { throw new Error('Cannot comment: the story is not open') }
    return await this.#activity.writeAndSync<IStoryActivity>(
      'projects-story-comments', { ProjectID: projectID, StoryID: storyID, Text: text, Attachments: attachments }, BACKLOG_ROUTES,
    )
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
