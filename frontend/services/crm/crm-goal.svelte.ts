import { CrmCatalogService } from './crm-pipeline.svelte.ts'

// Mirrors backend/crm/types/goal.go.
export { GOAL_METRIC } from './crm-goal-metric.ts'

export interface ICrmGoal {
  ID: number
  PipelineID: number
  UserID: number
  /** YYMM */
  Period: number
  Metric: number
  Target: number
  ss: number
  upd: number
}

export class CrmGoalsService extends CrmCatalogService<ICrmGoal> {
  constructor(init = false) { super('crm-goals', init) }
}

/** Saves the changed cells of the grid; a Target of 0 clears one. The goals list resyncs after. */
export const postCrmGoals = (goals: CrmGoalsService, changed: Partial<ICrmGoal>[]): Promise<ICrmGoal[]> =>
  goals.writeAndSync('crm-goals', { Goals: changed })
