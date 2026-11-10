// Constants of backend/crm/types/stage_log.go and opportunity.go, in a plain module so pure
// logic files (pipeline.ts, reports.ts) can import them without pulling in the Svelte runtime.

/** CrmOpportunityStageLog.Kind */
export const STAGE_LOG_KIND_CREATED = 1
export const STAGE_LOG_KIND_MOVED = 2
export const STAGE_LOG_KIND_CLOSED = 3
export const STAGE_LOG_KIND_EDITED = 4

/** CrmOpportunity.Outcome */
export const OUTCOME_OPEN = 0
export const OUTCOME_WON = 1
export const OUTCOME_LOST = 2
