// Pure rules of the web store checkout: no Svelte, no fetch. The backend re-checks every one of
// them (it never trusts this copy); these exist so the shopper sees the problem before submitting.
// See docs/WEB_CHECKOUT_PLAN.md.

import { unpackQuantityLine } from '#core/quantity.ts'

// The [ecommerce] company flags (backend/company_flags.toml).
export const FLAG_CONFIRM_EMAIL_ONLINE_PAYMENT = 10
export const FLAG_REQUIRE_CONTACT_NUMBER = 11
export const FLAG_RECEIPT_AUTOMATIC = 12
export const FLAG_CASH_ON_DELIVERY = 13
export const FLAG_CONFIRM_EMAIL_DEFERRED_PAYMENT = 14

// SaleOrder.DeliveryType.
export const DELIVERY_HOME = 1
export const DELIVERY_PICKUP = 2

// The payment methods the checkout accepts (backend types.WebPayment*).
export const PAYMENT_CARD = 1
export const PAYMENT_YAPE = 2
export const PAYMENT_DEFERRED = 3

// Sent instead of a Culqi token by "Simular Pago (dev mode)": a `go run . dev` backend settles
// company 1's order without charging. Mirrors backend webDevSimulatedPaymentToken.
export const DEV_SIMULATED_PAYMENT_TOKEN = "dev_simulated_payment"

// A boleta from S/ 700 must name its buyer (SUNAT), so the DNI becomes required. In cents.
export const IDENTIFIED_RECEIPT_FROM_CENTS = 70000

// "Nueva dirección" in the address selector.
export const NEW_ADDRESS_ID = 0

export interface IWebCustomerAddress {
  ID: number
  CityID: number
  Address: string
  Reference: string
  ss?: number
}

export interface IWebCustomer {
  ID: number
  Email: string
  FirstName: string
  LastName: string
  Phone: string
  DocumentNumber: string
  EmailConfirmed: boolean
}

// ICustomerDraft is what the shopper types: shared by Regístrate, Mis Datos and the cart's
// Datos Envío, so one form fills the other.
export interface ICustomerDraft {
  Email: string
  FirstName: string
  LastName: string
  Phone: string
  DocumentNumber: string
}

export interface ICheckoutCartLine {
  productID: number
  quantity: number
  unitPriceCents: number
}

export interface IWebCheckoutRequest {
  Lines: { ProductID: number, PresentationID: number, Quantity: number }[]
  DeliveryType: number
  PickupSiteID: number
  PaymentMethod: number
  CulqiToken: string
  AddressID: number
  Shipping: {
    FirstName: string, LastName: string, Email: string, Phone: string, DocumentNumber: string,
    CityID: number, Address: string, Reference: string
  }
}

export const hasCompanyFlag = (flags: number[] | undefined, flagID: number): boolean => {
  return (flags || []).includes(flagID)
}

export const isOnlinePayment = (paymentMethod: number): boolean => {
  return paymentMethod === PAYMENT_CARD || paymentMethod === PAYMENT_YAPE
}

export const cartTotalCents = (lines: ICheckoutCartLine[]): number => {
  return lines.reduce((total, line) => total + line.unitPriceCents * line.quantity, 0)
}

export const formatSoles = (cents: number): string => {
  return `S/ ${(cents / 100).toFixed(2)}`
}

// requiresDocumentNumber: the DNI is asked only when the store issues receipts automatically and
// the total (shipping included) reaches the identified-boleta threshold.
export const requiresDocumentNumber = (flags: number[] | undefined, totalCents: number): boolean => {
  return hasCompanyFlag(flags, FLAG_RECEIPT_AUTOMATIC) && totalCents >= IDENTIFIED_RECEIPT_FROM_CENTS
}

// validateCustomerDraft returns the first problem of the contact, or "" when it is complete.
export const validateCustomerDraft = (
  draft: ICustomerDraft, flags: number[] | undefined, needsDocumentNumber: boolean,
): string => {
  if (!/^[^\s@]+@[^\s@]+\.[^\s@]{2,}$/.test((draft.Email || '').trim())) {
    return 'Ingrese un correo electrónico válido.'
  }
  if ((draft.FirstName || '').trim().length < 2 || (draft.LastName || '').trim().length < 2) {
    return 'Ingrese sus nombres y apellidos.'
  }
  const phone = (draft.Phone || '').trim()
  if (!phone && hasCompanyFlag(flags, FLAG_REQUIRE_CONTACT_NUMBER)) {
    return 'Ingrese su número de celular.'
  }
  if (phone && !/^\+?[0-9 ]{6,15}$/.test(phone)) {
    return 'El número de celular no es válido.'
  }
  const documentNumber = (draft.DocumentNumber || '').trim()
  if (needsDocumentNumber && !documentNumber) {
    return 'Para compras desde S/ 700 ingrese su DNI.'
  }
  if (documentNumber && !/^[0-9]{8}$/.test(documentNumber)) {
    return 'El DNI debe tener 8 dígitos.'
  }
  return ''
}

// validateAddress: a district (6-digit ubigeo) and a street address.
export const validateAddress = (address: Pick<IWebCustomerAddress, 'CityID' | 'Address'>): string => {
  if (!address.CityID || address.CityID < 10000) {
    return 'Seleccione el departamento, la provincia y el distrito.'
  }
  if ((address.Address || '').trim().length < 4) {
    return 'Ingrese la dirección.'
  }
  return ''
}

// validateDelivery checks the chosen destination before the Pago step.
export const validateDelivery = (
  deliveryType: number, pickupSiteID: number, selectedAddressID: number,
  newAddress: Pick<IWebCustomerAddress, 'CityID' | 'Address'>,
): string => {
  if (deliveryType === DELIVERY_PICKUP) {
    return pickupSiteID > 0 ? '' : 'Seleccione la tienda donde recogerá su pedido.'
  }
  if (deliveryType !== DELIVERY_HOME) {
    return 'Elija entrega a domicilio o recojo en tienda.'
  }
  return selectedAddressID === NEW_ADDRESS_ID ? validateAddress(newAddress) : ''
}

// defaultAddressID pre-selects the address of the customer's last order, else the newest saved
// one, else "Nueva dirección" (also the default when nothing is saved).
export const defaultAddressID = (addresses: IWebCustomerAddress[], lastUsedAddressID?: number): number => {
  if (lastUsedAddressID && addresses.some(address => address.ID === lastUsedAddressID)) {
    return lastUsedAddressID
  }
  if (addresses.length === 0) { return NEW_ADDRESS_ID }
  return Math.max(...addresses.map(address => address.ID))
}

// mostExpensiveOrderLines picks the lines a "Mis Pedidos" card lists: the `limit` lines with the
// highest amount (units × unit price), so the card summarizes the order by what weighed most.
export const mostExpensiveOrderLines = <T extends { Price: number, Quantity: number }>(lines: T[], limit: number): T[] => {
  const lineAmount = (line: T) => unpackQuantityLine(line.Quantity).units * line.Price
  return [...lines].sort((a, b) => lineAmount(b) - lineAmount(a)).slice(0, limit)
}

// formatWebOrderNumber is the number the shopper sees ("Pedido 0001"); same as the backend's
// webOrderNumber, which names the email and the purchase-order PDF.
export const formatWebOrderNumber = (onlineOrderID: number): string => {
  return String(onlineOrderID).padStart(4, '0')
}

// The order Status mirrors SaleOrder.Status: 0 annulled, 1 pending, 2 paid, 3 delivered, 4 paid + delivered.
// Mis Pedidos splits it in two: the payment (webOrderPaymentLabel) and the progress badge (webOrderProgress).
export const webOrderPaymentLabel = (status: number): string => {
  return ['', 'Pendiente de pago', 'Pagado', 'Pendiente de pago', 'Pagado'][status] || ''
}

export type WebOrderProgressKey = 'annulled' | 'received' | 'preparing' | 'inTransit' | 'delivered'

interface IWebOrderProgressFields { ss: number, FulfillmentStatus: number, DeliveryType: number }

// webOrderProgressSteps are the steps the shopper follows in the order detail; the id is the step
// number webOrderProgressStep returns. The middle step names what the shopper waits for.
export const webOrderProgressSteps = (deliveryType: number): { id: number, key: WebOrderProgressKey, name: string }[] => [
  { id: 0, key: 'received', name: 'Recibido' },
  { id: 1, key: 'preparing', name: 'En Preparación' },
  { id: 2, key: 'inTransit', name: deliveryType === DELIVERY_PICKUP ? 'Para Recoger' : 'En Camino' },
  { id: 3, key: 'delivered', name: 'Entregado' },
]

// webOrderProgressStep: a delivery (Status 3/4) outranks the back office's FulfillmentStatus label
// (0 received, 1 preparing, 2 on the way / ready for pickup).
export const webOrderProgressStep = (order: IWebOrderProgressFields): number => {
  return order.ss === 3 || order.ss === 4 ? 3 : (order.FulfillmentStatus || 0)
}

// webOrderProgress is the badge of a Mis Pedidos card: the current step, or "Anulado".
export const webOrderProgress = (order: IWebOrderProgressFields): { key: WebOrderProgressKey, label: string } => {
  if (order.ss === 0) { return { key: 'annulled', label: 'Anulado' } }
  const currentStep = webOrderProgressSteps(order.DeliveryType)[webOrderProgressStep(order)]
  return { key: currentStep.key, label: currentStep.name }
}

// deferredPaymentLabel names cash on delivery by where the money changes hands.
export const deferredPaymentLabel = (deliveryType: number): string => {
  return deliveryType === DELIVERY_PICKUP ? 'Pago en tienda' : 'Pago contra entrega'
}

// webPaymentMethodLabel names how the shopper chose to pay, in the order detail.
export const webPaymentMethodLabel = (paymentMethod: number, deliveryType: number): string => {
  if (paymentMethod === PAYMENT_CARD) { return 'Tarjeta' }
  if (paymentMethod === PAYMENT_YAPE) { return 'Yape' }
  if (paymentMethod === PAYMENT_DEFERRED) { return deferredPaymentLabel(deliveryType) }
  return ''
}

export const buildWebCheckoutRequest = (args: {
  cartLines: ICheckoutCartLine[]
  draft: ICustomerDraft
  deliveryType: number
  pickupSiteID: number
  selectedAddressID: number
  newAddress: Omit<IWebCustomerAddress, 'ID'>
  paymentMethod: number
  culqiToken: string
}): IWebCheckoutRequest => {
  const isNewAddress = args.deliveryType === DELIVERY_HOME && args.selectedAddressID === NEW_ADDRESS_ID
  return {
    Lines: args.cartLines.map(line => ({ ProductID: line.productID, PresentationID: 0, Quantity: line.quantity })),
    DeliveryType: args.deliveryType,
    PickupSiteID: args.deliveryType === DELIVERY_PICKUP ? args.pickupSiteID : 0,
    PaymentMethod: args.paymentMethod,
    CulqiToken: isOnlinePayment(args.paymentMethod) ? args.culqiToken : '',
    AddressID: args.deliveryType === DELIVERY_HOME ? args.selectedAddressID : 0,
    Shipping: {
      FirstName: args.draft.FirstName.trim(),
      LastName: args.draft.LastName.trim(),
      Email: args.draft.Email.trim(),
      Phone: (args.draft.Phone || '').trim(),
      DocumentNumber: (args.draft.DocumentNumber || '').trim(),
      // A saved address travels by its id; the server copies it into the order.
      CityID: isNewAddress ? args.newAddress.CityID : 0,
      Address: isNewAddress ? args.newAddress.Address.trim() : '',
      Reference: isNewAddress ? (args.newAddress.Reference || '').trim() : '',
    },
  }
}
