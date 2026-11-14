import { describe, expect, it } from 'vitest'
import { LINE_STATUS, type ITableOrder, type ITableOrderLine } from '../dining.ts'
import {
  CLIENT_MODE, buildChargeRequest, buildTableAnnulRequest, chargeBlocker, chargeCashBankOptions, newTableChargeForm,
} from './table-charge.ts'

const orderWith = (...statuses: number[]): ITableOrder => ({
  ID: 9, LayoutID: 1, TableShapeID: 3, WarehouseID: 2, ss: 1, upd: 0,
  Lines: statuses.map((status, index) => ({ LineID: index + 1, Round: 1, ProductID: 7, PresentationID: 0, Quantity: 1000,
    Price: 1500, Status: status } as ITableOrderLine)),
})

describe('chargeBlocker', () => {
  it('lets a bill of served and cancelled lines be charged', () => {
    expect(chargeBlocker(orderWith(LINE_STATUS.SERVED, LINE_STATUS.CANCELLED))).toBe('')
  })

  it('blocks while a line is to confirm, in the kitchen or ready', () => {
    for (const liveStatus of [LINE_STATUS.TO_CONFIRM, LINE_STATUS.ORDERED, LINE_STATUS.READY]) {
      expect(chargeBlocker(orderWith(LINE_STATUS.SERVED, liveStatus))).not.toBe('')
    }
  })

  it('blocks a bill with nothing served', () => {
    expect(chargeBlocker(orderWith(LINE_STATUS.CANCELLED))).not.toBe('')
  })
})

describe('buildChargeRequest', () => {
  const servedOrder = orderWith(LINE_STATUS.SERVED)

  it('sends only the buyer the chosen client mode names', () => {
    const form = { ...newTableChargeForm(), ClientID: 4, ClientInfo: { Name: ' Ana ', RegistryNumber: '', IdentityDocType: 0 } }
    expect(buildChargeRequest(servedOrder, form).request).toEqual({ OrderID: 9, IssueSeriesID: 0, ClientID: 0, CashBankID: 0 })

    const selected = buildChargeRequest(servedOrder, { ...form, ClientMode: CLIENT_MODE.SELECT, CashBankID: 5 }).request
    expect(selected).toEqual({ OrderID: 9, IssueSeriesID: 0, ClientID: 4, CashBankID: 5 })

    const registered = buildChargeRequest(servedOrder, { ...form, ClientMode: CLIENT_MODE.REGISTER }).request
    expect(registered?.ClientID).toBe(0)
    expect(registered?.ClientInfo?.Name).toBe('Ana')
  })

  it('requires the client the mode asks for', () => {
    expect(buildChargeRequest(servedOrder, { ...newTableChargeForm(), ClientMode: CLIENT_MODE.SELECT }).error).toBeTruthy()
    expect(buildChargeRequest(servedOrder, { ...newTableChargeForm(), ClientMode: CLIENT_MODE.REGISTER }).error).toBeTruthy()
  })

  it('sends a cleared selector as 0', () => {
    const form = { ...newTableChargeForm(), CashBankID: undefined as unknown as number, IssueSeriesID: undefined as unknown as number }
    expect(buildChargeRequest(servedOrder, form).request).toMatchObject({ CashBankID: 0, IssueSeriesID: 0 })
  })

  it('refuses a bill that is not ready to charge', () => {
    expect(buildChargeRequest(orderWith(LINE_STATUS.ORDERED), newTableChargeForm()).error).toBeTruthy()
  })
})

describe('buildTableAnnulRequest', () => {
  it('requires a reason and trims it', () => {
    expect(buildTableAnnulRequest(orderWith(), '  ').error).toBeTruthy()
    expect(buildTableAnnulRequest(orderWith(), ' cliente se fue ').request).toEqual({ OrderID: 9, Reason: 'cliente se fue' })
  })
})

describe('chargeCashBankOptions', () => {
  it('offers active cash-banks in soles only', () => {
    const cashBanks = [{ ID: 1, ss: 1, CurrencyType: 1 }, { ID: 2, ss: 1, CurrencyType: 2 }, { ID: 3, ss: 0, CurrencyType: 1 },
      { ID: 4, ss: 1, CurrencyType: 0 }] as Parameters<typeof chargeCashBankOptions>[0]
    expect(chargeCashBankOptions(cashBanks).map((cashBank) => cashBank.ID)).toEqual([1, 4])
  })
})
