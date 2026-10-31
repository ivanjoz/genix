import { getRecordsByID, type IMinimalRecord } from '@genix/ui';
import { tr } from '#core/store.svelte.ts';
import { GET, GetHandler, POST } from '#libs/ui-runtime.svelte.ts';
import {
  isProjectArchived, isSprintOpen, makeUserLabel, projectCode, sortProjects,
  type IEpic, type IProject, type ISprint, type IStory,
} from './project.ts';
import type { IStoryDetail } from './story/story.ts';
import { applyBacklogRank, type IBacklogRankRequest } from './backlog/backlog.ts';

// Delta-cached list of every project (GET.projects-projects): the Proyectos page and the project
// selectors of Backlog and Sprints.
export class ProjectsService extends GetHandler<IProject> {
  route = 'projects-projects'
  useCache = { min: 1, ver: 1 }
  inferRemoveFromStatus = true

  makeName(project: Partial<IProject>) { return project.Name || '' }

  handler(projects: IProject[]): void {
    this.records = []
    this.recordsMap = new Map()
    this.nameToRecordMap = new Map()
    this.addSavedRecords(...sortProjects(projects || []))
  }

  async save(project: IProject): Promise<IProject> {
    const savedProject = await POST({ data: project, route: this.route, refreshRoutes: [this.route] }) as IProject
    this.addSavedRecords(savedProject)
    this.records = sortProjects(this.records)
    return savedProject
  }

  constructor(init = false) {
    super()
    if (init) { this.fetch() }
  }
}

// The project last opened in Backlog or Sprints, per browser: a page opened without ?project=
// starts on it.
const LAST_PROJECT_STORAGE_KEY = 'projects_last_project_id'
export const lastOpenedProjectID = (): number => Number(localStorage.getItem(LAST_PROJECT_STORAGE_KEY)) || 0
export const rememberOpenedProject = (projectID: number) => { localStorage.setItem(LAST_PROJECT_STORAGE_KEY, String(projectID)) }

// The project picker's options ("P01 · Name"), for the header search box of every project page
// (Backlog, Sprints, Documents and the reports). Deleted projects are left out; archived ones are marked.
export const projectPickerOptions = (projects: IProject[]) => projects.filter((project) => project.ss !== 0).map((project) => ({
  id: project.ID,
  name: `${projectCode(project)} · ${project.Name}${isProjectArchived(project) ? ` (${tr('archived|archivado')})` : ''}`,
}))

// One table of one project, delta-cached per project (GET.projects-<table>?project-id=): it paints
// from the browser cache at once and then merges only what changed on the server.
export class ProjectRecordsService<T extends { ID: number, ss: number }> extends GetHandler<T> {
  useCache = { min: 1, ver: 1 }
  inferRemoveFromStatus = true

  handler(records: T[]): void {
    this.records = []
    this.recordsMap = new Map()
    this.addSavedRecords(...(records || []))
  }

  constructor(routeName: string, projectID: number) {
    super()
    this.route = `${routeName}?project-id=${projectID}`
  }
}

// The cached lists a backlog write can change, the stories' detail and timeline included (prefix
// match), the Documents page's copies of the stories' files and the per-person assignments. Every
// POST of the module marks them, so the next read asks the server even inside the cache's minute.
export const BACKLOG_ROUTES = ['projects-epics', 'projects-stories', 'projects-sprints', 'projects-story-', 'projects-documents', 'projects-assigned-stories']

// ProjectBacklog is one project's backlog: the project (from the cached projects list) and its
// epics, stories and sprints. Backlog and Sprints read it; every save merges what the server answers.
export class ProjectBacklog {
  projectID = $state(0)
  #projects: ProjectsService
  #epics = $state<ProjectRecordsService<IEpic> | null>(null)
  #stories = $state<ProjectRecordsService<IStory> | null>(null)
  #sprints = $state<ProjectRecordsService<ISprint> | null>(null)

  constructor(projects: ProjectsService) {
    this.#projects = projects
  }

  get project(): IProject | null {
    const project = this.#projects.recordsMap.get(this.projectID)
    return project && project.ss !== 0 ? project : null
  }
  get epics(): IEpic[] { return this.#epics?.records || [] }
  get stories(): IStory[] { return this.#stories?.records || [] }
  get sprints(): ISprint[] { return this.#sprints?.records || [] }
  // Something to paint: the cached stories, or the server's answer (then an empty backlog is real).
  get isLoaded(): boolean { return !!this.#stories && (this.#stories.records.length > 0 || this.#stories.hasFetched) }

  // Opening a project paints its cached lists and then syncs them; loading the same project again
  // only asks the server for what changed.
  async load(projectID: number) {
    if (projectID === this.projectID && this.#stories && this.#sprints && this.#epics) {
      await Promise.all([this.#epics.fetchOnline(), this.#stories.fetchOnline(), this.#sprints.fetchOnline()])
      return
    }
    this.projectID = projectID
    this.#epics = new ProjectRecordsService<IEpic>('projects-epics', projectID)
    this.#stories = new ProjectRecordsService<IStory>('projects-stories', projectID)
    this.#sprints = new ProjectRecordsService<ISprint>('projects-sprints', projectID)
    await Promise.all([this.#epics.fetch(), this.#stories.fetch(), this.#sprints.fetch()])
  }

  // The project panel opened from the Proyectos page edits only the epics: the stories and sprints
  // are not read.
  async loadEpics(projectID: number) {
    if (projectID === this.projectID && this.#epics) {
      await this.#epics.fetchOnline()
      return
    }
    this.projectID = projectID
    this.#epics = new ProjectRecordsService<IEpic>('projects-epics', projectID)
    this.#stories = null
    this.#sprints = null
    await this.#epics.fetch()
  }

  async saveEpics(epics: IEpic[]): Promise<IEpic[]> {
    const savedEpics = await POST({ data: epics, route: 'projects-epics', refreshRoutes: BACKLOG_ROUTES }) as IEpic[]
    this.#epics?.addSavedRecords(...savedEpics)
    // A deleted epic leaves its stories without one: the backend rewrote them.
    const deletedEpicIDs = savedEpics.filter((epic) => epic.ss === 0).map((epic) => epic.ID)
    if (deletedEpicIDs.length > 0) {
      this.#stories?.addSavedRecords(...this.stories.filter((story) => deletedEpicIDs.includes(story.EpicID || 0)).map((story) => ({ ...story, EpicID: 0 })))
    }
    return savedEpics
  }

  async saveStory(story: IStory, detail: IStoryDetail): Promise<{ Story: IStory, Detail: IStoryDetail }> {
    const saved = await POST({
      data: { Story: story, Detail: detail }, route: 'projects-stories', refreshRoutes: BACKLOG_ROUTES,
    }) as { Story: IStory, Detail: IStoryDetail }
    this.applyStory(saved.Story)
    return saved
  }

  async deleteStory(story: IStory) {
    const deleted = await POST({ data: { Story: { ...story, ss: 0 }, Detail: {} }, route: 'projects-stories', refreshRoutes: BACKLOG_ROUTES }) as { Story: IStory }
    this.applyStory(deleted.Story)
    // The backend took it off the open sprints.
    this.#sprints?.addSavedRecords(...this.sprints.filter(isSprintOpen)
      .map((sprint) => ({ ...sprint, Stories: (sprint.Stories || []).filter((sprintStory) => sprintStory.StoryID !== story.ID) })))
  }

  // A backlog drop shows at once; its write goes through the move queue.
  rankStory(rankRequest: IBacklogRankRequest) {
    const movedStory = this.stories.find((story) => story.ID === rankRequest.StoryID)
    if (!movedStory) { return }
    this.applyStory(applyBacklogRank(movedStory, this.stories, rankRequest))
    this.queueMove(rankRequest.StoryID, async () => {
      const savedStory = await POST({ data: rankRequest, route: 'projects-stories-rank', refreshRoutes: BACKLOG_ROUTES }) as IStory
      return () => this.applyStory(savedStory)
    })
  }

  // Moves (backlog drops, board cards) are shown before they are written, and written one at a
  // time in the order they were made, so the server ends in the state the user sees. The server's
  // answers are merged only once the last queued move is written: an older answer merged earlier
  // would undo the moves made since. If any write fails (genix-ui notifies it), the lists are read
  // again instead, so they show what is stored.
  #moveQueue: Promise<void> = Promise.resolve()
  #pendingMovesCount = 0
  #savedMoveAnswers: (() => void)[] = []
  #hasFailedMove = false
  // One entry per queued move: the stories that show a spinner until their move is written.
  #movingStoryIDs = $state<number[]>([])

  isStoryMoving(storyID: number): boolean { return this.#movingStoryIDs.includes(storyID) }
  // A sync while moves are written would show the server's state without them.
  get hasPendingMoves(): boolean { return this.#pendingMovesCount > 0 }

  // writeMove sends one move and returns how to merge the server's answer.
  queueMove(storyID: number, writeMove: () => Promise<() => void>) {
    this.#pendingMovesCount++
    this.#movingStoryIDs = [...this.#movingStoryIDs, storyID]
    this.#moveQueue = this.#moveQueue.then(async () => {
      try {
        this.#savedMoveAnswers.push(await writeMove())
      } catch {
        this.#hasFailedMove = true
      }
      this.#pendingMovesCount--
      // A failed reload is notified by genix-ui too; the queue must keep taking moves.
      if (this.#pendingMovesCount === 0) { await this.#settleMoves().catch(() => {}) }
      this.#movingStoryIDs = this.#movingStoryIDs.toSpliced(this.#movingStoryIDs.indexOf(storyID), 1)
    })
  }

  async #settleMoves() {
    const savedMoveAnswers = this.#savedMoveAnswers
    const hasFailedMove = this.#hasFailedMove
    this.#savedMoveAnswers = []
    this.#hasFailedMove = false
    if (hasFailedMove) {
      await this.load(this.projectID)
      return
    }
    for (const applySavedMove of savedMoveAnswers) { applySavedMove() }
  }

  applyStory(savedStory: IStory) {
    this.#stories?.addSavedRecords(savedStory)
  }

  applySprint(savedSprint: ISprint) {
    this.#sprints?.addSavedRecords(savedSprint)
  }
}

export interface ITeamCandidate {
  ID: number
  Name: string
}

interface IUserLabel extends IMinimalRecord {
  Username: string
  FirstName?: string
  LastName?: string
}

// ProjectPeople names the people of the projects: the team picker's candidates (users of the
// module holding Edit) and the names of team members, read by ID.
export class ProjectPeople {
  candidates = $state<ITeamCandidate[]>([])
  userNamesByID = $state(new Map<number, string>())

  async loadCandidates() {
    this.candidates = (await GET({ route: 'projects-team-candidates' }) as ITeamCandidate[]) || []
    const userNamesByID = new Map(this.userNamesByID)
    for (const candidate of this.candidates) { userNamesByID.set(candidate.ID, candidate.Name) }
    this.userNamesByID = userNamesByID
  }

  async rememberUsers(userIDs: number[]) {
    const newUserIDs = [...new Set(userIDs)].filter((userID) => userID > 0 && !this.userNamesByID.has(userID))
    if (newUserIDs.length === 0) { return }
    const users = await getRecordsByID<IUserLabel>('users-ids', newUserIDs)
    const userNamesByID = new Map(this.userNamesByID)
    for (const [userID, user] of users) { userNamesByID.set(userID, makeUserLabel(user)) }
    this.userNamesByID = userNamesByID
  }

  userName(userID: number): string {
    return this.userNamesByID.get(userID) || `#${userID}`
  }
}
