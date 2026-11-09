import type { ICrmPipeline, ICrmPipelineMember } from '#services/crm/crm-pipeline.svelte.ts'

// Levels of backend/crm/types/pipeline.go; 0 is "not a member" and is never stored.
export const NO_ACCESS_LEVEL = 0

export const LEVEL_OPTIONS = [
  { id: 0, name: 'None|Ninguno' },
  { id: 1, name: 'View|Ver' },
  { id: 2, name: 'Edit|Editar' },
  { id: 3, name: 'Full|Completo' },
]

export const levelOfUser = (members: ICrmPipelineMember[], userID: number) =>
  members.find((member) => member.UserID === userID)?.Level ?? NO_ACCESS_LEVEL

/** Sets one user's level in a member list; level 0 removes the user. */
export const setMemberLevel = (members: ICrmPipelineMember[], userID: number, level: number): ICrmPipelineMember[] => {
  const otherMembers = members.filter((member) => member.UserID !== userID)
  return level === NO_ACCESS_LEVEL ? otherMembers : [...otherMembers, { UserID: userID, Level: level }]
}

const sortedByUser = (members: ICrmPipelineMember[]) => [...members].sort((left, right) => left.UserID - right.UserID)

export const sameMembers = (left: ICrmPipelineMember[], right: ICrmPipelineMember[]) =>
  JSON.stringify(sortedByUser(left)) === JSON.stringify(sortedByUser(right))

/** The pipelines whose edited member list differs from the stored one: the only ones worth a save. */
export const pipelinesWithChangedMembers = (
  pipelines: ICrmPipeline[], draftMembersByPipeline: Record<number, ICrmPipelineMember[]>,
) => pipelines.filter((pipeline) => {
  const draftMembers = draftMembersByPipeline[pipeline.ID]
  return draftMembers !== undefined && !sameMembers(draftMembers, pipeline.Members)
})
