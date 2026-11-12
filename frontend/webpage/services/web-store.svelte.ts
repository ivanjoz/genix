// Web store customer session and the public "p-web-*" API (docs/WEB_CHECKOUT_PLAN.md).
// The customer token is NOT the back-office session: it travels in X-Customer-Token and only
// to the web store routes, and lives in localStorage per company.
import { Env, type ICompanyParams } from '#core/env.ts';
import type {
  ICustomerDraft, IWebCheckoutRequest, IWebCustomer, IWebCustomerAddress,
} from '#ecommerce/services/web-checkout.ts';

export interface IWebCustomerSession {
  Token: string
  Customer: IWebCustomer
  HasPassword: boolean
  Addresses: IWebCustomerAddress[]
}

export interface IWebCheckoutIdentifyResult extends Partial<IWebCustomerSession> {
  CodeRequired?: boolean
  Sent?: boolean
  RetryAfterSeconds?: number
}

export interface IWebCheckoutResult {
  // The order number the shopper sees; the sale (and its boleta) exists only once it is paid.
  OnlineSaleOrderID: number
  SaleOrderID: number
  OrderNumber: string
  TotalAmount: number
  ShippingAmount: number
  Status: number
  // 0 none, 1 boleta (paid), 2 Orden de Compra (paid later).
  DocumentType: number
}

export interface IWebOrderLine {
  ProductID: number
  Price: number
  Quantity: number
}

export interface IWebOrder {
  ID: number
  Date: number
  ss: number
  Lines: IWebOrderLine[]
  TotalAmount: number
  DebtAmount: number
  ShippingAmount: number
  DeliveryType: number
  PickupSiteID: number
  Shipping: { CityID: number, Address: string }
  // The sale this order became when it was paid or delivered; 0 before.
  SaleOrderID: number
}

const emptyDraft = (): ICustomerDraft => ({ Email: '', FirstName: '', LastName: '', Phone: '', DocumentNumber: '' })

const sessionStorageKey = () => `genixWebCustomer-${Env.getCompanyID()}`
const draftStorageKey = () => `genixWebCustomerDraft-${Env.getCompanyID()}`

const readStoredJSON = <T>(key: string): T | null => {
  if (typeof localStorage === 'undefined') { return null }
  try { return JSON.parse(localStorage.getItem(key) || 'null') } catch { return null }
}

export const WebStore = $state({
  // Copy of the public company file: Env.empresa is not reactive.
  company: {} as ICompanyParams,
  token: '',
  customer: null as IWebCustomer | null,
  hasPassword: false,
  addresses: [] as IWebCustomerAddress[],
  // What the shopper typed, shared by Regístrate, Mis Datos and the cart's Datos Envío.
  draft: emptyDraft(),
})

export const loadWebStoreCompany = async (): Promise<ICompanyParams> => {
  if (!WebStore.company.id) { WebStore.company = { ...(await Env.loadEmpresaConfig()) } }
  return WebStore.company
}

export const saveCustomerDraft = () => {
  if (typeof localStorage === 'undefined') { return }
  localStorage.setItem(draftStorageKey(), JSON.stringify(WebStore.draft))
}

// The delivery choices of the cart's Datos Envío step, kept per company so a reload does not lose
// what the shopper already picked or typed.
export interface ICheckoutDelivery {
  deliveryType: number
  pickupSiteID: number
  selectedAddressID: number
  newAddress: { CityID: number, Address: string, Reference: string }
}

const deliveryStorageKey = () => `genixWebCheckoutDelivery-${Env.getCompanyID()}`

export const saveCheckoutDelivery = (delivery: ICheckoutDelivery) => {
  if (typeof localStorage === 'undefined') { return }
  localStorage.setItem(deliveryStorageKey(), JSON.stringify(delivery))
}

export const readStoredCheckoutDelivery = (): ICheckoutDelivery | null => {
  return readStoredJSON<ICheckoutDelivery>(deliveryStorageKey())
}

// applyWebCustomerSession stores a session answer and fills the draft from the account, so the
// checkout opens with the customer's data.
export const applyWebCustomerSession = (session: IWebCustomerSession) => {
  WebStore.token = session.Token || WebStore.token
  WebStore.customer = session.Customer
  WebStore.hasPassword = session.HasPassword
  WebStore.addresses = (session.Addresses || []).filter(address => address.ss !== 0)
  const customer = session.Customer
  WebStore.draft = {
    Email: customer.Email, FirstName: customer.FirstName, LastName: customer.LastName,
    Phone: customer.Phone || '', DocumentNumber: customer.DocumentNumber || '',
  }
  saveCustomerDraft()
  if (typeof localStorage !== 'undefined') { localStorage.setItem(sessionStorageKey(), WebStore.token) }
}

export const logoutWebCustomer = () => {
  WebStore.token = ''
  WebStore.customer = null
  WebStore.hasPassword = false
  WebStore.addresses = []
  if (typeof localStorage !== 'undefined') { localStorage.removeItem(sessionStorageKey()) }
}

// webStoreFetch calls a p-web-* route. Errors arrive as {"error": "..."}; a 401 means the stored
// token expired or was revoked, so the session is dropped.
const webStoreFetch = async <T>(route: string, body?: unknown): Promise<T> => {
  const headers: Record<string, string> = { 'Content-Type': 'application/json' }
  if (WebStore.token) { headers['X-Customer-Token'] = WebStore.token }
  const response = await fetch(Env.makeRoute(route), {
    method: body === undefined ? 'GET' : 'POST',
    headers,
    body: body === undefined ? undefined : JSON.stringify(body),
  })
  if (!response.ok) {
    let message = `Error ${response.status}`
    try { message = (await response.json())?.error || message } catch { /* not JSON */ }
    if (response.status === 401 && WebStore.token) { logoutWebCustomer() }
    throw new Error(message)
  }
  return response.json() as Promise<T>
}

// restoreWebCustomerSession re-reads the stored session on load.
export const restoreWebCustomerSession = async () => {
  const draft = readStoredJSON<ICustomerDraft>(draftStorageKey())
  if (draft) { WebStore.draft = { ...emptyDraft(), ...draft } }
  const token = typeof localStorage !== 'undefined' ? localStorage.getItem(sessionStorageKey()) : ''
  if (!token || WebStore.customer) { return }
  WebStore.token = token
  try {
    applyWebCustomerSession(await webStoreFetch<IWebCustomerSession>('p-web-customer'))
  } catch (error) {
    console.warn('[web-store] stored session rejected:', error)
  }
}

export const requestWebCustomerCode = (email: string) => {
  return webStoreFetch<{ Sent: boolean, RetryAfterSeconds: number }>('p-web-customer-code', { Email: email })
}

// verifyWebCustomerCode logs in (or registers, with the draft as profile) with a mailed code.
export const verifyWebCustomerCode = async (email: string, code: string, addresses: IWebCustomerAddress[] = []) => {
  const draft = WebStore.draft
  const session = await webStoreFetch<IWebCustomerSession>('p-web-customer-verify', {
    Email: email, Code: code, Addresses: addresses,
    Profile: { FirstName: draft.FirstName, LastName: draft.LastName, Phone: draft.Phone, DocumentNumber: draft.DocumentNumber },
  })
  applyWebCustomerSession(session)
  return session
}

export const loginWebCustomer = async (email: string, password: string) => {
  applyWebCustomerSession(await webStoreFetch<IWebCustomerSession>('p-web-customer-login', { Email: email, Password: password }))
}

// identifyWebCheckout: a guest pressed pay. Either a code was mailed (CodeRequired) or the
// session comes back right away.
export const identifyWebCheckout = async (paymentMethod: number, addresses: IWebCustomerAddress[]) => {
  const draft = WebStore.draft
  const result = await webStoreFetch<IWebCheckoutIdentifyResult>('p-web-checkout-identify', {
    Email: draft.Email, PaymentMethod: paymentMethod, Addresses: addresses,
    Profile: { FirstName: draft.FirstName, LastName: draft.LastName, Phone: draft.Phone, DocumentNumber: draft.DocumentNumber },
  })
  if (!result.CodeRequired && result.Token) { applyWebCustomerSession(result as IWebCustomerSession) }
  return result
}

export const saveWebCustomerProfile = async () => {
  const draft = WebStore.draft
  const customer = await webStoreFetch<IWebCustomer>('p-web-customer', {
    FirstName: draft.FirstName, LastName: draft.LastName, Phone: draft.Phone, DocumentNumber: draft.DocumentNumber,
  })
  WebStore.customer = customer
  saveCustomerDraft()
}

export const saveWebCustomerPassword = async (currentPassword: string, newPassword: string) => {
  await webStoreFetch('p-web-customer-password', { CurrentPassword: currentPassword, NewPassword: newPassword })
  WebStore.hasPassword = true
}

// saveWebCustomerAddress creates, edits or deletes (ss 0) one saved address.
export const saveWebCustomerAddress = async (address: IWebCustomerAddress) => {
  const saved = await webStoreFetch<IWebCustomerAddress>('p-web-customer-address', address)
  const others = WebStore.addresses.filter(current => current.ID !== saved.ID)
  WebStore.addresses = saved.ss === 0 ? others : [...others, saved]
  return saved
}

export const getWebCustomerOrders = () => {
  return webStoreFetch<{ Orders: IWebOrder[], ProductNames: Record<number, string> }>('p-web-customer-orders')
}

export const getWebShippingAmount = async (districtID: number): Promise<number> => {
  const result = await webStoreFetch<{ ShippingAmount: number }>(`p-web-shipping-amount?city-id=${districtID}`)
  return result.ShippingAmount
}

export const postWebCheckout = (request: IWebCheckoutRequest) => {
  return webStoreFetch<IWebCheckoutResult>('p-web-checkout', request)
}

// downloadWebOrderPDF fetches the Orden de Compra of an online order with the session header (a
// plain link cannot carry it) and hands it to the browser as a file.
export const downloadWebOrderPDF = async (onlineSaleOrderID: number) => {
  const response = await fetch(Env.makeRoute(`p-web-order-pdf?id=${onlineSaleOrderID}`), {
    headers: { 'X-Customer-Token': WebStore.token },
  })
  if (!response.ok) {
    let message = `Error ${response.status}`
    try { message = (await response.json())?.error || message } catch { /* not JSON */ }
    throw new Error(message)
  }
  const fileURL = URL.createObjectURL(await response.blob())
  const link = document.createElement('a')
  link.href = fileURL
  link.download = `orden-de-compra-${onlineSaleOrderID}.pdf`
  link.click()
  URL.revokeObjectURL(fileURL)
}
