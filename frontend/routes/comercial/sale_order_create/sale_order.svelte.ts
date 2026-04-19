import { POST } from '$libs/http.svelte';
import { type IProducto } from '$routes/negocio/productos/productos.svelte';
import { type IProductoStock, type IProductStockDetail } from '$routes/logistica/products-stock/stock-movement';
import { Loading, Notify } from '$libs/helpers';

export interface ProductoVenta {
  key: string
  cant: number
  producto: IProducto
  presentationID: number
  presentationName: string
  displayName: string
  searchText: string
  isSubUnidad?: boolean
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
  cantidad: number
  isSubUnidad?: boolean
  producto?: IProducto // Helper reference
}

export interface ISaleOrder {
  ID: number
  EmpresaID: number
  WarehouseID: number
  LastPaymentCajaID: number
  ClientID: number
  TotalAmount: number
  TaxAmount: number
  DebtAmount: number
  ActionsIncluded: number[]
  DetailProductsIDs: number[]
  DetailPrices: number[]
  DetailQuantities: number[]
	DetailProductSkus: string[]
  DetailProductPresentations: number[]
  ClientInfo?: {
    Name: string
    RegistryNumber: string
  }
  
  // UI Helpers (not sent or ignored by backend if not in struct)
  montoRecibido: number
  montoVuelto: number
}

export class SaleOrderState {
  // State
  productosStock = $state([] as IProductoStock[])
  form = $state({
    ID: 0, EmpresaID: 0, WarehouseID: 0, LastPaymentCajaID: 1, ClientID: 0, // Default payment caja; UI can overwrite.
    TotalAmount: 0, TaxAmount: 0, DebtAmount: 0,
    ActionsIncluded: [2, 3],
    DetailProductsIDs: [], DetailPrices: [], DetailQuantities: [],
    DetailProductSkus: [], DetailProductPresentations: [],
    montoRecibido: 0, montoVuelto: 0
  } as ISaleOrder)
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
  addProducto(e: ProductoVenta, cant: number, serialNumber?: string) {
    const ventaCant = this.ventaProductosMap.get(e.key)?.cantidad || 0
    const stock = e.cant - ventaCant

    if(stock < cant){
      this.ventaErrorMessage = `No hay suficiente stock de "${e.displayName}" para agregar ${cant} unidades.`
      return
    }

    const ventaProducto = this.ventaProductos.find(x => x.key === e.key)
    if(ventaProducto){
      ventaProducto.cantidad += cant
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
        productoID: e.producto.ID,
        presentationID: e.presentationID,
        presentationName: e.presentationName,
        displayName: e.displayName,
        serialNumbers: new Map(serialNumber ? [[serialNumber,1]] : []),
        isSubUnidad: e.isSubUnidad || false,
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
        let precio = producto.PrecioFinal

        if (vp.isSubUnidad && producto.SbnPreciFinal) {
           precio = producto.SbnPreciFinal
        }

        total += precio * vp.cantidad
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

  async postSaleOrder() {
    if (this.ventaProductos.length === 0) {
      Notify.failure("El carrito está vacío.")
      return false
    }

    if (this.form.WarehouseID === 0) {	
      Notify.failure("Seleccione un almacén.")
      return false
    }

    Loading.standard("Procesando venta...")

    // Prepare detail slices
    this.form.DetailProductsIDs = []
    this.form.DetailPrices = []
    this.form.DetailQuantities = []
		this.form.DetailProductSkus = []
    this.form.DetailProductPresentations = []

    // Flatten cart into order details, keeping the backend's legacy field name for serial numbers.
    for (const vp of this.ventaProductos) {
      let precio = vp.producto?.PrecioFinal || 0
      if (vp.isSubUnidad && vp.producto?.SbnPreciFinal) {
        precio = vp.producto.SbnPreciFinal
      }

      let totalSerialQty = 0
      if (vp.serialNumbers && vp.serialNumbers.size > 0) {
        // Serialized lines travel independently so backend can deduct the matching detail row.
        for (const [serialNumber, qty] of vp.serialNumbers.entries()) {
          this.form.DetailProductsIDs.push(vp.productoID)
          this.form.DetailPrices.push(precio)
          this.form.DetailQuantities.push(qty)
          this.form.DetailProductSkus.push(serialNumber)
          this.form.DetailProductPresentations.push(vp.presentationID)
          totalSerialQty += qty
        }
      }

      // Remaining quantity belongs to the generic stock bucket with no serial number.
      const remainingQty = vp.cantidad - totalSerialQty
      if (remainingQty > 0) {
        this.form.DetailProductsIDs.push(vp.productoID)
        this.form.DetailPrices.push(precio)
        this.form.DetailQuantities.push(remainingQty)
        this.form.DetailProductSkus.push("")
        this.form.DetailProductPresentations.push(vp.presentationID)
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

      const res = await POST({
        route: "sale-order",
        data: this.form,
        successMessage: "Venta registrada con éxito"
      })

      if (res) {
        // Reset state
        this.ventaProductos = []
        this.recalcTotales()
        this.form.ClientID = 0
        this.form.ClientInfo = undefined
        this.form.montoRecibido = 0
        this.form.montoVuelto = 0
        return true
      }
    } catch (error) {
      console.error("Error posting sale order:", error)
    } finally {
      Loading.remove()
    }

    return false
  }
}
