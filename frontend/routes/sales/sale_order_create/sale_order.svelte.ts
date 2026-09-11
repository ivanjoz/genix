import { POST } from '$libs/ui-runtime.svelte';
import { type IProduct } from '$services/production/products.svelte';
import { type IProductStock, type IProductStockDetail } from '$routes/logistics/products-stock/stock-movement';
import { type IClientProvider } from '$services/crm/client-provider.svelte';
import { type IInvoiceSeries } from '$routes/company/configuration/invoice-series';
import { tr } from '$core/store.svelte';
import { Loading, Notify } from '$libs/helpers';
import { validateCustomerIdentity } from './sale_order';
import {
  type Quantity, addQuantity, formatQuantity, packQuantityLine, quantityAmount, totalSubUnits,
} from '$core/quantity';

export interface ProductoVenta {
  key: string
  // Stock on hand for this row, as the {units, sub} pair. There is no separate sub-unit
  // row: a product with a sub-unit is one row that can be sold either way.
  available: Quantity
  subDivisor: number
  producto: IProduct
  presentationID: number
  presentationName: string
  displayName: string
  searchText: string
  serialNumbers?: IProductStockDetail[]
}

export interface SkuCant {
  sku: string, cant: number
}

export interface VentaProducto {
  key: string
  productoID: number
  presentationID: number
  presentationName: string
  displayName: string
  serialNumbers?: Map<string,number>
  lote?: string
  cantidad: Quantity
  subDivisor: number
  producto?: IProduct // Helper reference
}

//STRUCT:comercial.SaleOrder
export interface ISaleOrder {
  Date: number
  WarehouseID: number
  ID: number
  DetailProductsIDs: number[]
  DetailPrices: number[]
  // Packed: units * 1000 + sub. The server re-resolves the divisor and both prices from the
  // catalog, so what travels here is the quantity, not the pricing.
  DetailQuantities: number[]
  DetailSubDivisor: number[]
  DetailProductSkus: string[]
  DetailProductLotIDs: number[]
  DetailProductPresentations: number[]
  TotalAmount: number
  TaxAmount: number
  DebtAmount: number
  ClientID: number
  Created: number
  upd: number
  upc: number
  UpdatedBy: number
  ss: number
  LastPaymentCajaID: number
  ActionsIncluded: number[]
  LastPaymentTime: number
  LastPaymentUser: number
  DeliveryTime: number
  DeliveryUser: number
  PaymentDueDate: number
  // The invoicing series the sale will be issued under. The backend folds it into the last two
  // digits of the sale id, so it is only read on creation. 0 = the till named no series.
  IssueSeriesID: number
  ClientInfo: any
  /* extra fields */
  CompanyID: number
  Name: string
  RegistryNumber: string
  // UI Helpers (not sent or ignored by backend if not in struct)
  montoRecibido: number
  montoVuelto: number
}

// Backend `ActionsIncluded` codes: the payment action posts a cash-bank movement, the delivery action moves stock.
export const SALE_ACTION_PAYMENT = 2
export const SALE_ACTION_DELIVERY = 3

// What a successful submit hands back. The cart travels with the sale because the local
// ticket history is built from both, and postSaleOrder empties the cart on its way out.
export interface PostedSaleOrder {
  sale: ISaleOrder
  soldProducts: VentaProducto[]
}

export class SaleOrderState {
  // State
  productosStock = $state([] as IProductStock[])
  form = $state<ISaleOrder>({
    ID: 0, CompanyID: 0, WarehouseID: 0, LastPaymentCajaID: 0, ClientID: 0, // No payment caja until one is loaded/picked.
    Date: 0, TotalAmount: 0, TaxAmount: 0, DebtAmount: 0,
    ActionsIncluded: [SALE_ACTION_PAYMENT, SALE_ACTION_DELIVERY],
    DetailProductsIDs: [], DetailPrices: [], DetailQuantities: [], DetailSubDivisor: [],
    DetailProductSkus: [], DetailProductLotIDs: [], DetailProductPresentations: [],
    Created: 0, upd: 0, upc: 0, UpdatedBy: 0, ss: 0,
    LastPaymentTime: 0, LastPaymentUser: 0, DeliveryTime: 0, DeliveryUser: 0,
    PaymentDueDate: 0, IssueSeriesID: 0, ClientInfo: undefined, Name: "", RegistryNumber: "",
    montoRecibido: 0, montoVuelto: 0
  })
  filterText = $state("")
  ventaErrorMessage = $state("")
  filterSerialNumber = $state("")

  // Cart
  ventaProductos = $state([] as VentaProducto[])

  // Computed
  ventaProductosMap = $derived.by(() => {
    return new Map(this.ventaProductos.map(x => [x.key, x]))
  })

  constructor() {}

  // Methods
  addProducto(e: ProductoVenta, cant: Quantity, serialNumber?: string) {
    const inCart = this.ventaProductosMap.get(e.key)?.cantidad || { units: 0, sub: 0 }
    // Compare in sub-units so a request for whole units is checked against loose stock too:
    // one box on hand covers six candies when the divisor is 6.
    const remaining = totalSubUnits(e.available, e.subDivisor) - totalSubUnits(inCart, e.subDivisor)

    if(remaining < totalSubUnits(cant, e.subDivisor)){
      const requested = formatQuantity(cant, e.subDivisor, e.producto.SbuUnit)
      this.ventaErrorMessage = `No hay suficiente stock de "${e.displayName}" para agregar ${requested}.`
      return
    }

    const ventaProducto = this.ventaProductos.find(x => x.key === e.key)
    if(ventaProducto){
      ventaProducto.cantidad = addQuantity(ventaProducto.cantidad, cant)
      if(serialNumber){
        const currentSerialNumbers = ventaProducto.serialNumbers || new Map<string, number>()
        const serialAdded = currentSerialNumbers.get(serialNumber)

        if(serialAdded){
          // Serialized stock must stay bounded by the detail row quantity.
          const stockCant = e.serialNumbers?.find((detail) => detail.SerialNumber === serialNumber)?.Quantity || 0
          if((serialAdded + 1) > stockCant){
            this.ventaErrorMessage = `La serie ${serialNumber} del producto "${e.displayName}" sólo posee ${stockCant} unidad(es).`
            return
          }
          const newSerialNumbers = new Map(currentSerialNumbers)
          newSerialNumbers.set(serialNumber, serialAdded + 1)
          ventaProducto.serialNumbers = newSerialNumbers
        } else {
          const newSerialNumbers = new Map(currentSerialNumbers)
          newSerialNumbers.set(serialNumber, 1)
          ventaProducto.serialNumbers = newSerialNumbers
        }
      }
      this.ventaProductos = [...this.ventaProductos]
    } else {
      this.ventaProductos.push({
        key: e.key,
        cantidad: cant,
        subDivisor: e.subDivisor,
        productoID: e.producto.ID,
        presentationID: e.presentationID,
        presentationName: e.presentationName,
        displayName: e.displayName,
        serialNumbers: new Map(serialNumber ? [[serialNumber,1]] : []),
        producto: e.producto
      })
    }

    this.recalcTotales()
    this.filterText = ""
    this.ventaErrorMessage = ""
  }

  removeProducto(key: string) {
    this.ventaProductos = this.ventaProductos.filter(x => x.key !== key)
    this.recalcTotales()
  }

  recalcTotales() {
    let total = 0
    for(let vp of this.ventaProductos){
      const producto = vp.producto
      if(producto){
        // Whole units at their price, sub-units at theirs — never prorated.
        total += quantityAmount(vp.cantidad, producto.FinalPrice, producto.SbuFinalPrice || 0)
      }
    }

    this.form.TotalAmount = total
    const subtotal = Math.floor(total / 1.18)
    this.form.TaxAmount = total - subtotal
    this.form.DebtAmount = 0 // Assuming fully paid for now, adjust if UI allows debt
    this.recalcVuelto()
  }

  recalcVuelto() {
    this.form.montoVuelto = (this.form.montoRecibido || 0) - this.form.TotalAmount
  }

  // allSeries and selectedClient come from the page's services: the sale can only be
  // stamped with a series the company has, and the buyer it will bill is either the
  // picked client row or the details typed at the till.
  async postSaleOrder(allSeries: IInvoiceSeries[], selectedClient?: IClientProvider):
    Promise<PostedSaleOrder | undefined> {

    if (this.ventaProductos.length === 0) {
      Notify.failure("El carrito está vacío.")
      return undefined
    }

    if (this.form.WarehouseID === 0) {
      Notify.failure("Seleccione un almacén.")
      return undefined
    }

    const issueSeries = allSeries.find(series => series.SeriesID === this.form.IssueSeriesID)
    if (issueSeries) {
      // The client row is the base and what was typed at the till overrides it,
      // which is how the backend resolves the buyer too.
      const identityProblem = validateCustomerIdentity(issueSeries.DocType, this.form.TotalAmount,
        this.form.ClientInfo?.Name || selectedClient?.Name || "",
        this.form.ClientInfo?.RegistryNumber || selectedClient?.RegistryNumber || "")

      if (identityProblem) {
        Notify.failure(tr(identityProblem))
        return undefined
      }
    }

    // A payment always books a cash-bank movement, so the "Pagado" action can never travel without a caja.
    if (!this.form.LastPaymentCajaID) {
      this.form.ActionsIncluded = this.form.ActionsIncluded.filter((actionID) => actionID !== SALE_ACTION_PAYMENT)
    }

    // Paid on creation: a due date would contradict it, and the form hides that input, so it must not travel.
    if (this.form.ActionsIncluded.includes(SALE_ACTION_PAYMENT)) {
      this.form.PaymentDueDate = 0
    }

    Loading.standard("Procesando venta...")

    // Prepare detail slices
    this.form.DetailProductsIDs = []
    this.form.DetailPrices = []
    this.form.DetailQuantities = []
    this.form.DetailSubDivisor = []
		this.form.DetailProductSkus = []
    this.form.DetailProductPresentations = []

    const pushLine = (vp: VentaProducto, quantity: Quantity, serialNumber: string) => {
      this.form.DetailProductsIDs.push(vp.productoID)
      // Sent for the operator's reference only; the server overwrites both prices from the
      // catalog before anything is stored.
      this.form.DetailPrices.push(vp.producto?.FinalPrice || 0)
      this.form.DetailQuantities.push(packQuantityLine(quantity, vp.subDivisor))
      this.form.DetailSubDivisor.push(vp.subDivisor)
      this.form.DetailProductSkus.push(serialNumber)
      this.form.DetailProductPresentations.push(vp.presentationID)
    }

    // Flatten cart into order details, keeping the backend's legacy field name for serial numbers.
    for (const vp of this.ventaProductos) {
      let totalSerialQty = 0
      if (vp.serialNumbers && vp.serialNumbers.size > 0) {
        // Serialized lines travel independently so backend can deduct the matching detail row.
        // A serial number is one physical item, so these are always whole units.
        for (const [serialNumber, qty] of vp.serialNumbers.entries()) {
          pushLine(vp, { units: qty, sub: 0 }, serialNumber)
          totalSerialQty += qty
        }
      }

      // Remaining quantity belongs to the generic stock bucket with no serial number.
      const remainingQty: Quantity = { units: vp.cantidad.units - totalSerialQty, sub: vp.cantidad.sub }
      if (remainingQty.units > 0 || remainingQty.sub > 0) {
        pushLine(vp, remainingQty, "")
      }
    }

    try {
      // Send a single client source to the backend: either an existing ID or a new client payload.
      if (this.form.ClientInfo?.Name?.trim()) {
        this.form.ClientInfo = {
          Name: this.form.ClientInfo.Name.trim(),
          RegistryNumber: this.form.ClientInfo.RegistryNumber?.trim() || "",
        }
        this.form.ClientID = 0
      } else {
        this.form.ClientInfo = undefined
      }

      const createdSale = await POST({
        route: "sale-order",
        data: this.form,
        successMessage: "Venta registrada con éxito"
      }) as ISaleOrder | undefined

      if (createdSale) {
        // Handed over before the cart is cleared: the response carries packed detail lines,
        // not the products behind them, so this is the only moment the sale can still be
        // described as what was sold.
        const soldProducts = this.ventaProductos

        // Reset state
        this.ventaProductos = []
        this.recalcTotales()
        this.form.ClientID = 0
        this.form.ClientInfo = undefined
        this.form.montoRecibido = 0
        this.form.montoVuelto = 0
        return { sale: createdSale, soldProducts }
      }
    } catch (error) {
      console.error("Error posting sale order:", error)
    } finally {
      Loading.remove()
    }

    return undefined
  }
}
