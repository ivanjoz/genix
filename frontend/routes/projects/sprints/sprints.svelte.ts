// The Sprints page's service calls. Each write brings back the delta of the project's sprints
// (GetHandler.writeAndSync) into the ProjectBacklog, which the Backlog page shares.
import { POST } from '#libs/ui-runtime.svelte.ts';
import type { ISprint } from '../project.ts';
import { BACKLOG_ROUTES, type ProjectBacklog } from '../project.svelte.ts';
import { applyCardMove, type ICarryOver, type ISprintMoveRequest } from './sprints.ts';

// saveSprintPlan is the edit mode's save. Planning rewrites the stories' final state and
// assignees on the server, so the stories are read again.
export const saveSprintPlan = async (backlog: ProjectBacklog, saveRequest: object): Promise<ISprint> => {
  const savedSprint = await backlog.writeToSprints<ISprint>('projects-sprints', saveRequest)
  await backlog.reloadStoriesAndEpics()
  return savedSprint
}

export const deleteSprint = async (backlog: ProjectBacklog, sprint: ISprint) => {
  await backlog.writeToSprints('projects-sprints', { Sprint: { ...sprint, ss: 0 } })
  await backlog.reloadStoriesAndEpics()
}

export const startSprint = async (backlog: ProjectBacklog, sprint: ISprint) => {
  await backlog.writeToSprints<ISprint>('projects-sprint-start', { ProjectID: sprint.ProjectID, SprintID: sprint.ID })
}

// moveSprintCard shows the move at once; its write goes through the backlog's move queue, which
// reads the board again if the server refuses it. A move is the one write that does not bring back
// a delta: the queue merges the answers only once the last move is written, and a delta applied
// before would undo the moves made since.
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
  await backlog.writeToSprints('projects-sprint-close', { ProjectID: sprint.ProjectID, SprintID: sprint.ID, CarryOvers: carryOvers })
  await backlog.reloadStoriesAndEpics()
}
