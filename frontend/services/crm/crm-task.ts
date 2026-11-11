// Plain module so pure files can import it without the Svelte runtime. Mirrors backend/crm/types/task.go.

/** CrmTask.Kind. "Send quote" (2) joins with quotes. */
export const TASK_KIND = { APPROVAL: 1, SEND_EMAIL: 3, ACTIVITY: 4 } as const

/** CrmTask.TaskStatus: pending is the only open state. */
export const TASK_STATUS = { PENDING: 1, DONE: 2, DISMISSED: 3, APPROVED: 4, REJECTED: 5, CANCELLED: 6 } as const

/** CrmTaskActionPayload.Action */
export const TASK_ACTION = { DISMISS: 1, CANCEL_REQUEST: 2 } as const

/** CrmStageTaskTemplate.AssignTo */
export const TASK_ASSIGN = { OWNER: 1, USER: 2 } as const
