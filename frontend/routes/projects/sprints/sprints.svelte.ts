// The Sprints page's service calls. Each one merges the server's answer into the project's
// ProjectBacklog, which the Backlog page shares.
import { POST } from '#libs/ui-runtime.svelte.ts';
import type { ISprint } from '../project.ts';
import { BACKLOG_ROUTES, type ProjectBacklog } from '../project.svelte.ts';
import { applyCardMove, type ICarryOver, type ISprintMoveRequest } from './sprints.ts';

// saveSprintPlan is the edit mode's save. Planning rewrites the stories' final state and
// assignees on the server, so the whole backlog is read again.
export const saveSprintPlan = async (backlog: ProjectBacklog, saveRequest: object): Promise<ISprint> => {
  const savedSprint = await POST({ data: saveRequest, route: 'projects-sprints', refreshRoutes: BACKLOG_ROUTES }) as ISprint
  backlog.applySprint(savedSprint)
  if (backlog.project) { await backlog.load(backlog.project.ID) }
  return savedSprint
}

export const deleteSprint = async (backlog: ProjectBacklog, sprint: ISprint) => {
  await POST({ data: { Sprint: { ...sprint, ss: 0 } }, route: 'projects-sprints', refreshRoutes: BACKLOG_ROUTES })
  if (backlog.project) { await backlog.load(backlog.project.ID) }
}

export const startSprint = async (backlog: ProjectBacklog, sprint: ISprint) => {
  const savedSprint = await POST({ data: { ProjectID: sprint.ProjectID, SprintID: sprint.ID }, route: 'projects-sprint-start', refreshRoutes: BACKLOG_ROUTES }) as ISprint
  backlog.applySprint(savedSprint)
}

// moveSprintCard shows the move at once; its write goes through the backlog's move queue, which
// reads the board again if the server refuses it.
export const moveSprintCard = (backlog: ProjectBacklog, moveRequest: ISprintMoveRequest) => {
  const movedSprint = backlog.sprints.find((sprint) => sprint.ID === moveRequest.SprintID)
  if (!movedSprint) { return }
  backlog.applySprint(applyCardMove(movedSprint, moveRequest))
  // The story's final state follows its latest sprint: the active one, for a story on its board.
  const movedStory = backlog.stories.find((story) => story.ID === moveRequest.StoryID)
  if (movedStory && movedStory.LastSprintID === moveRequest.SprintID) {
    backlog.applyStory({ ...movedStory, State: moveRequest.State })
  }
  backlog.queueMove(moveRequest.StoryID, async () => {
    const savedSprint = await POST({ data: moveRequest, route: 'projects-sprint-move', refreshRoutes: BACKLOG_ROUTES }) as ISprint
    return () => backlog.applySprint(savedSprint)
  })
}

export const closeSprint = async (backlog: ProjectBacklog, sprint: ISprint, carryOvers: ICarryOver[]) => {
  await POST({ data: { ProjectID: sprint.ProjectID, SprintID: sprint.ID, CarryOvers: carryOvers }, route: 'projects-sprint-close', refreshRoutes: BACKLOG_ROUTES })
  await backlog.load(sprint.ProjectID)
}
