import { GetHandler } from '$libs/http.svelte';

export interface IActionRegistered {
  ID: number
  Name: string
}

export interface ICronActionParams {
  Param1?: number
  Param2?: number
  Param3?: number
  Param4?: number
  Param5?: string
  Param6?: string
}

export interface ICronActionScheduled {
  ID: number
  UnixMinutesFrame: number
  CompanyID: number
  ActionID: number
  Updated: number
  Status: number
  InvocationCount: number
  Params?: ICronActionParams
}

export interface ICronActionsScheduledResponse {
  actionsScheduled: ICronActionScheduled[]
  actionsRegistered: IActionRegistered[]
}

export interface ICronActionTableRow extends ICronActionScheduled {
  ActionName: string
}

export class CronActionsService extends GetHandler {
  // The backend currently uses `actionsScheduled` as the delta query key.
  route = 'cron-actions-scheduled?actionsScheduled=0'
  useCache = { min: 0.2, ver: 1 }

  response: ICronActionsScheduledResponse = $state({
    actionsScheduled: [],
    actionsRegistered: [],
  })
  rows: ICronActionTableRow[] = $state([])

  handler(result: ICronActionsScheduledResponse): void {
    this.response = {
      actionsScheduled: result?.actionsScheduled || [],
      actionsRegistered: result?.actionsRegistered || [],
    }
    this.rows = buildCronActionRows(this.response)
  }

  constructor() {
    super()
    this.fetch()
  }
}

export const buildCronActionRows = (
  response: ICronActionsScheduledResponse
): ICronActionTableRow[] => {
  const registeredActionsMap = new Map(
    (response.actionsRegistered || []).map((registeredAction) => [registeredAction.ID, registeredAction.Name])
  )

  // Enrich each scheduled row once so the table stays simple and avoids repeated lookups.
  return (response.actionsScheduled || [])
    .map((scheduledAction) => ({
      ...scheduledAction,
      ActionName: registeredActionsMap.get(scheduledAction.ActionID) || 'No registrada',
    }))
    .sort((leftAction, rightAction) => rightAction.UnixMinutesFrame - leftAction.UnixMinutesFrame)
}
