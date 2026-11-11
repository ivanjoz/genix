import { GET } from '#libs/ui-runtime.svelte.ts'
import { TASK_ACTION } from './crm-task.ts'
import type { CrmOpportunitiesService } from './crm-opportunity.svelte.ts'
import { CrmCatalogService } from './crm-pipeline.svelte.ts'

// Mirrors backend/crm/types/task.go. The constants are in crm-task.ts.
export { TASK_ASSIGN, TASK_KIND, TASK_STATUS } from './crm-task.ts'

export interface ICrmTask {
  ID: number
  OpportunityID: number
  /** The stage that generated it; for an approval, the move's target. */
  PipelineID: number
  StageID: number
  Kind: number
  TemplateID: number
  Title: string
  AssignedUserID: number
  /** UnixDay */
  DueDate: number
  TaskStatus: number
  ResultNote: string
  ResultRefID: number
  Created: number
  DoneTime: number
  DoneBy: number
  /** The requester of an approval. */
  CreatedBy: number
  /** What the card shows of the opportunity, attached by the server on read. */
  OpportunityTitle: string
  ClientName: string
  OpportunityAmount: number
  ss: number
  upd: number
}

/**
 * The tasks of ONE user, open or closed: each combination is its own delta list, so a user syncs only
 * their open tasks and a closed one leaves the list as a removed ID. A team leader may read their
 * members' lists (the server checks it). The list is short-lived on purpose: it is an inbox.
 */
export class CrmTasksService extends CrmCatalogService<ICrmTask> {
  // ver 2: tasks now carry their opportunity's title, client and amount; older caches lack them.
  useCache = { min: 0.2, ver: 2 }
  // Never fetches from the constructor: the base constructor runs before this class's `useCache`
  // is set, so it would read with the base version, and two reads with different versions reset
  // each other's cache (full download every time). The caller calls fetch().
  constructor(userID: number, isClosed = false) { super(`crm-tasks?user-id=${userID}&closed=${isClosed ? 1 : 0}`) }
}

/** The assignee drops a template task. */
export const dismissCrmTask = (tasks: CrmTasksService, taskID: number, note = ''): Promise<ICrmTask> =>
  tasks.writeAndSync('crm-task', { ID: taskID, Action: TASK_ACTION.DISMISS, Note: note })

/** The requester withdraws their approval request. The list refreshed is the board's: the card is released. */
export const cancelCrmApprovalRequest = (opportunities: CrmOpportunitiesService, taskID: number): Promise<ICrmTask> =>
  opportunities.writeAndSync('crm-task', { ID: taskID, Action: TASK_ACTION.CANCEL_REQUEST })

/** The approver approves (and the held move runs) or rejects. */
export const decideCrmApproval = (tasks: CrmTasksService, taskID: number, approve: boolean, note: string): Promise<ICrmTask> =>
  tasks.writeAndSync('crm-task-decide', { ID: taskID, Approve: approve, Note: note })

/** The approval an opportunity waits on (none or one), for anyone who can see its pipeline. */
export const getCrmOpportunityApproval = async (opportunityID: number): Promise<ICrmTask | undefined> =>
  ((await GET({ route: `crm-opportunity-approval?opportunity-id=${opportunityID}` })) as ICrmTask[] | null)?.[0]
