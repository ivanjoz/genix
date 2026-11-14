import { CURRENCY_PEN, normalizeCurrency } from '#routes/finance/exchange-rate/exchange-rate.ts'
import type { ICashBank } from '#routes/finance/cash-banks/cajas.svelte.ts'
import { LINE_STATUS, type ITableOrder } from '../dining.ts'

// Charging a table's bill into a sale, or annulling it (docs/DINING_PLAN.md §4). Pure: the backend
// rules the cashier sees before sending, tested without a component or a request.

// "Atención de Mesas" and its sub-access "Cobrar mesa" (backend access.toml).
export const TABLE_SERVICE_ACCESS_ID = 61
export const CHARGE_TABLE_SUB_ACCESS_ID = 2

// How the bill names its buyer: none, an existing client, or one registered now (as at the till).
export const CLIENT_MODE = { NONE: 0, SELECT: 1, REGISTER: 2 } as const

export interface ITableChargeForm {
  IssueSeriesID: number
  ClientMode: number
  ClientID: number
  ClientInfo: { Name: string, RegistryNumber: string, IdentityDocType: number }
  // 0 leaves the bill owed by the client.
  CashBankID: number
}

export const newTableChargeForm = (): ITableChargeForm => ({
  IssueSeriesID: 0, ClientMode: CLIENT_MODE.NONE, ClientID: 0,
  ClientInfo: { Name: '', RegistryNumber: '', IdentityDocType: 0 }, CashBankID: 0,
})

// Why the bill cannot be charged yet, '' when it can: every live line must be served or cancelled
// (backend ChargeSaleLines), and something must have been served.
export function chargeBlocker(order: ITableOrder): string {
  if (order.Lines.some((line) => line.Status !== LINE_STATUS.SERVED && line.Status !== LINE_STATUS.CANCELLED)) {
    return 'Some products are not served yet: serve or cancel them first.|Hay productos sin servir: márquelos servidos o anúlelos antes de cobrar.'
  }
  if (!order.Lines.some((line) => line.Status === LINE_STATUS.SERVED)) {
    return 'The table has no served products to charge.|La mesa no tiene productos servidos que cobrar.'
  }
  return ''
}

// A table bill is in soles, so it is collected into an active cash-bank in soles.
export const chargeCashBankOptions = (cashBanks: ICashBank[]): ICashBank[] =>
  cashBanks.filter((cashBank) => (cashBank.ss || 0) > 0 && normalizeCurrency(cashBank.CurrencyType) === CURRENCY_PEN)

export interface ITableChargeRequest {
  OrderID: number
  IssueSeriesID: number
  ClientID: number
  ClientInfo?: { Name: string, RegistryNumber: string, IdentityDocType: number }
  CashBankID: number
}

// The charge request: only the buyer the chosen mode names travels.
export function buildChargeRequest(order: ITableOrder, form: ITableChargeForm): { request?: ITableChargeRequest, error?: string } {
  const blocker = chargeBlocker(order)
  if (blocker) return { error: blocker }

  // A cleared selector leaves its field empty: it travels as 0.
  const request: ITableChargeRequest = {
    OrderID: order.ID, IssueSeriesID: form.IssueSeriesID || 0, ClientID: 0, CashBankID: form.CashBankID || 0,
  }
  if (form.ClientMode === CLIENT_MODE.SELECT) {
    if (!form.ClientID) return { error: 'Select the client.|Seleccione el cliente.' }
    request.ClientID = form.ClientID
  } else if (form.ClientMode === CLIENT_MODE.REGISTER) {
    const clientName = form.ClientInfo.Name.trim()
    if (!clientName) return { error: "Type the client's name.|Escriba el nombre del cliente." }
    request.ClientInfo = { ...form.ClientInfo, Name: clientName, RegistryNumber: form.ClientInfo.RegistryNumber.trim() }
  }
  return { request }
}

export function buildTableAnnulRequest(order: ITableOrder, reason: string): { request?: { OrderID: number, Reason: string }, error?: string } {
  const trimmedReason = reason.trim()
  if (!trimmedReason) return { error: 'A reason is required to annul.|Se requiere un motivo para anular.' }
  return { request: { OrderID: order.ID, Reason: trimmedReason } }
}
