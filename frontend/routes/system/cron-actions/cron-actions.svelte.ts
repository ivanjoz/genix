import { GetHandler } from '$libs/http.svelte';

export interface IActionRegistered {
  ID: number
  Name: string
}

export interface ICronActionParams {
  p1?: number
  p2?: number
  p3?: number
  p4?: number
  p5?: string
  p6?: string
}

export interface ICronActionScheduled {
  ID: number
  UnixMinutesFrame: number
  CompanyID: number
  ActionID: number
  InvocationCount: number
	Params?: ICronActionParams
	upd: number
  ss: number
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
	useCache = { min: 0.25, ver: 4 }
	keysIDs = { "actionsScheduled": ["UnixMinutesFrame", "ID"] }

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
