import { Notify } from '$libs/helpers'
import type { IProducto } from '$routes/negocio/productos/productos.svelte'
import type { IProductCard } from './ProductCardSearch.svelte'

export interface PurchaseOrderItem {
  key: string
  productID: number
  presentationID: number
  presentationName: string
  productName: string
  displayName: string
  sku: string
  cantidad: number
  precio: number
  producto: IProducto
}

export interface IPurchaseOrder {
  ID: number
  ProviderID: number
  WarehouseID: number
  TotalAmount: number
  TaxAmount: number
  DetailProductsIDs: number[]
  DetailPrices: number[]
  DetailQuantities: number[]
  DetailProductSkus: string[]
  DetailProductPresentations: number[]
  Notes: string
}

export class PurchaseOrderState {
  form = $state({
    ID: 0,
    ProviderID: 0,
    WarehouseID: 0,
    TotalAmount: 0,
    TaxAmount: 0,
    DetailProductsIDs: [],
    DetailPrices: [],
    DetailQuantities: [],
    DetailProductSkus: [],
    DetailProductPresentations: [],
    Notes: '',
  } as IPurchaseOrder)

  items = $state([] as PurchaseOrderItem[])
  errorMessage = $state('')

  itemsMap = $derived.by(() => new Map(this.items.map((item) => [item.key, item])))
  itemsCantMap = $derived.by(() => new Map(this.items.map((item) => [item.key, item.cantidad])))

  addItem(card: IProductCard, cant: number = 1) {
    if (cant <= 0) { return }

    const existing = this.items.find((item) => item.key === card.key)
    if (existing) {
      existing.cantidad += cant
      this.items = [...this.items]
    } else {
      this.items.push({
        key: card.key,
        productID: card.productID,
        presentationID: card.presentationID,
        presentationName: card.presentationName,
        productName: card.productName,
        displayName: card.displayName,
        sku: card.sku,
        cantidad: cant,
        precio: card.price || 0,
        producto: card.producto,
      })
    }

    this.recalcTotals()
    this.errorMessage = ''
  }

  removeItem(key: string) {
    this.items = this.items.filter((item) => item.key !== key)
    this.recalcTotals()
  }

  updateQuantity(key: string, cantidad: number) {
    const item = this.items.find((i) => i.key === key)
    if (!item) { return }
    if (cantidad <= 0) {
      this.removeItem(key)
      return
    }
    item.cantidad = cantidad
    this.items = [...this.items]
    this.recalcTotals()
  }

  updatePrice(key: string, precio: number) {
    const item = this.items.find((i) => i.key === key)
    if (!item) { return }
    item.precio = Math.max(0, precio)
    this.items = [...this.items]
    this.recalcTotals()
  }

  recalcTotals() {
    let total = 0
    for (const item of this.items) {
      total += (item.precio || 0) * (item.cantidad || 0)
    }
    this.form.TotalAmount = total
    const subtotal = Math.floor(total / 1.18)
    this.form.TaxAmount = total - subtotal
  }

  reset() {
    this.items = []
    this.form.ProviderID = 0
    this.form.Notes = ''
    this.recalcTotals()
  }

  async postPurchaseOrder(): Promise<boolean> {
    if (this.items.length === 0) {
      Notify.failure('Agregue al menos un producto a la orden.')
      return false
    }
    if (!this.form.ProviderID) {
      Notify.failure('Seleccione un proveedor.')
      return false
    }

    this.form.DetailProductsIDs = this.items.map((item) => item.productID)
    this.form.DetailPrices = this.items.map((item) => item.precio)
    this.form.DetailQuantities = this.items.map((item) => item.cantidad)
    this.form.DetailProductSkus = this.items.map((item) => item.sku)
    this.form.DetailProductPresentations = this.items.map((item) => item.presentationID)

    console.log('[purchase-order] payload (frontend-only stub)', $state.snapshot(this.form))
    Notify.success('Orden generada (frontend stub).')
    return true
  }
}
