import { CrmCatalogService } from './crm-pipeline.svelte.ts'

// Mirrors backend/crm/types/team.go.
export interface ICrmTeam {
  ID: number
  Name: string
  LeaderUserID: number
  MemberUserIDs: number[]
  ss: number
  upd: number
}

export class CrmTeamsService extends CrmCatalogService<ICrmTeam> {
  constructor(init = false) { super('crm-teams', init) }
}

export const postCrmTeam = (teams: CrmTeamsService, team: Partial<ICrmTeam>): Promise<ICrmTeam> =>
  teams.writeAndSync('crm-team', team)
