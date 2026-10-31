import { getRecordsByID, type IMinimalRecord } from '@genix/ui';
import { GetHandler } from '#libs/ui-runtime.svelte.ts';
import type { ISprint, IStory } from '../../project.ts';
import { ProjectRecordsService } from '../../project.svelte.ts';
import { recentAssignments, storyIDsByProject, type IStoryAssignee } from './by-person.ts';

// One person's assignments across every project (GET.projects-assigned-stories?user-id=), delta-cached
// per person: the first read lists them, later reads only bring the stories that changed hands.
// The rows are keyed by StoryID (they have no ID), which GetHandler's ID-typed records can't express.
class StoryAssigneesService extends GetHandler {
  useCache = { min: 1, ver: 1 }
  keyID = 'StoryID'
  inferRemoveFromStatus = true

  handler(assignments: IStoryAssignee[]): void {
    this.records = []
    this.recordsMap = new Map()
    this.addSavedRecords(...(assignments || []))
  }

  constructor(assigneeID: number) {
    super()
    this.route = `projects-assigned-stories?user-id=${assigneeID}`
  }
}

// loadProjectSprints reads the sprints of each project, each through the delta cache the Backlog and
// Sprints pages share. The sprints give the stories their dates (Gantt and weeks).
const loadProjectSprints = async (projectIDs: number[]): Promise<Map<number, ISprint[]>> => {
  const sprintServices = projectIDs.map((projectID) => new ProjectRecordsService<ISprint>('projects-sprints', projectID))
  await Promise.all(sprintServices.map((sprintService) => sprintService.fetch()))
  return new Map(projectIDs.map((projectID, projectIndex) => [projectID, sprintServices[projectIndex].records]))
}

// loadStoriesByProject reads the stories by ID from every project in one request (cached per story
// in the browser): stories live in the company's partition, so the IDs go unpartitioned and the
// answer is grouped by each story's project.
const loadStoriesByProject = async (storyIDs: number[]): Promise<Map<number, Map<number, IStory>>> => {
  const storiesByID = await getRecordsByID<IStory & IMinimalRecord>('projects-stories-ids', storyIDs)
  const storiesByProject = new Map<number, Map<number, IStory>>()
  for (const story of storiesByID.values()) {
    if (!storiesByProject.has(story.ProjectID)) { storiesByProject.set(story.ProjectID, new Map()) }
    storiesByProject.get(story.ProjectID)!.set(story.ID, story)
  }
  return storiesByProject
}

// AssignedStories loads the report of one person: the recent assignments, then their stories, and
// those projects' sprints.
export class AssignedStories {
  assigneeID = $state(0)
  assignments = $state<IStoryAssignee[]>([])
  storiesByProject = $state(new Map<number, Map<number, IStory>>())
  sprintsByProject = $state(new Map<number, ISprint[]>())
  isLoaded = $state(false)

  async load(assigneeID: number) {
    this.assigneeID = assigneeID
    this.isLoaded = false
    const assigneesService = new StoryAssigneesService(assigneeID)
    await assigneesService.fetch()
    const assignments = recentAssignments(assigneesService.records as IStoryAssignee[], Date.now())
    const idsByProject = storyIDsByProject(assignments)
    const [storiesByProject, sprintsByProject] = await Promise.all([
      assignments.length > 0 ? loadStoriesByProject(Object.values(idsByProject).flat()) : Promise.resolve(new Map<number, Map<number, IStory>>()),
      loadProjectSprints(Object.keys(idsByProject).map(Number)),
    ])
    // Another person picked meanwhile: that load owns the report.
    if (assigneeID !== this.assigneeID) { return }
    this.assignments = assignments
    this.storiesByProject = storiesByProject
    this.sprintsByProject = sprintsByProject
    this.isLoaded = true
  }
}
