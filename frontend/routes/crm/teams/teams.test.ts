import { describe, expect, it } from 'vitest'
import { makeEmptyTeam, otherTeamOfUser, teamProblem, toggleTeamMember } from './teams.ts'

describe('teams', () => {
  it('clears the leader when the leader leaves', () => {
    const team = { ...makeEmptyTeam(), Name: 'A', MemberUserIDs: [1, 2], LeaderUserID: 2 }
    expect(toggleTeamMember(team, 2, false)).toMatchObject({ MemberUserIDs: [1], LeaderUserID: 0 })
    expect(toggleTeamMember(team, 3, true).MemberUserIDs).toEqual([1, 2, 3])
    expect(toggleTeamMember(team, 3, true).LeaderUserID).toBe(2)
  })

  it('finds the other team of a user but not the edited one', () => {
    const teams = [{ ...makeEmptyTeam(), ID: 1, MemberUserIDs: [5] }, { ...makeEmptyTeam(), ID: 2, MemberUserIDs: [6] }]
    expect(otherTeamOfUser(teams, 5, 2)?.ID).toBe(1)
    expect(otherTeamOfUser(teams, 5, 1)).toBeUndefined()
  })

  it('explains what is missing', () => {
    expect(teamProblem(makeEmptyTeam())).toContain('name')
    expect(teamProblem({ ...makeEmptyTeam(), Name: 'A', MemberUserIDs: [1] })).toContain('leader')
    expect(teamProblem({ ...makeEmptyTeam(), Name: 'A', MemberUserIDs: [1], LeaderUserID: 1 })).toBe('')
  })
})
