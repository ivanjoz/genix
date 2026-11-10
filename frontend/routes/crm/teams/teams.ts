import type { ICrmTeam } from '#services/crm/crm-team.svelte.ts'

export const makeEmptyTeam = (): ICrmTeam => ({ ID: 0, Name: '', LeaderUserID: 0, MemberUserIDs: [], ss: 1, upd: 0 })

/** The team a user belongs to, other than the one being edited: a user can be in only one team. */
export const otherTeamOfUser = (teams: ICrmTeam[], userID: number, editingTeamID: number) =>
  teams.find((team) => team.ID !== editingTeamID && team.MemberUserIDs.includes(userID))

/** Adds or removes a member; the leader is cleared when they leave the team. */
export const toggleTeamMember = (team: ICrmTeam, userID: number, isMember: boolean): ICrmTeam => {
  const memberUserIDs = isMember
    ? [...new Set([...team.MemberUserIDs, userID])]
    : team.MemberUserIDs.filter((memberUserID) => memberUserID !== userID)
  const leaderUserID = memberUserIDs.includes(team.LeaderUserID) ? team.LeaderUserID : 0
  return { ...team, MemberUserIDs: memberUserIDs, LeaderUserID: leaderUserID }
}

/** Why a team cannot be saved yet, or an empty string; the server checks everything again. */
export const teamProblem = (team: ICrmTeam) => {
  if (!team.Name.trim()) return 'The team needs a name.|El equipo necesita un nombre.'
  if (!team.MemberUserIDs.length) return 'Pick at least one member.|Elija al menos un miembro.'
  if (!team.MemberUserIDs.includes(team.LeaderUserID)) return 'Pick a leader among the members.|Elija un líder entre los miembros.'
  return ''
}
