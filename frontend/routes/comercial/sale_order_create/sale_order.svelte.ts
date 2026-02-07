import { POST } from '$libs/http.svelte';
import { type IProducto } from '$routes/negocio/productos/productos.svelte';
import { type IProductoStock } from '$routes/logistica/productos-stock/productos-stock.svelte';
import { Loading, Notify } from '$libs/helpers';

export interface ProductoVenta {
  key: string
  cant: number
  producto: IProducto
  searchText: string
  isSubUnidad?: boolean
  skus?: IProductoStock[]
}

export interface SkuCant {
  sku: string, cant: number
}

export interface VentaProducto {
  key: string
  productoID: number
  skus?: Map<string,number>
  lote?: string
  cantidad: number
  isSubUnidad?: boolean
  producto?: IProducto // Helper reference
}

export interface ISaleOrder {
  ID: number
  EmpresaID: number
  AlmacenID: number
  CajaID_: number
  TotalAmount: number
  TaxAmount: number
  DebtAmount: number
  ProcessesIncluded_: number[]
  DetailProductsIDs: number[]
  DetailPrices: number[]
  DetailQuantities: number[]
  
  // UI Helpers (not sent or ignored by backend if not in struct)
  montoRecibido: number
  montoVuelto: number
}

export class SaleOrderState {
  // State
  productosStock = $state([] as IProductoStock[])
  form = $state({
    ID: 0, EmpresaID: 0, AlmacenID: 0, CajaID_: 1, // Default CajaID_ to 1 for now or handle in UI
    TotalAmount: 0, TaxAmount: 0, DebtAmount: 0,
    ProcessesIncluded_: [2, 3],
    DetailProductsIDs: [], DetailPrices: [], DetailQuantities: [],
    montoRecibido: 0, montoVuelto: 0
  } as ISaleOrder)
  filterText = $state("")
  ventaErrorMessage = $state("")
  filterSku = $state("")

  // Cart
  ventaProductos = $state([] as VentaProducto[])

  // Computed
  ventaProductosMap = $derived.by(() => {
    return new Map(this.ventaProductos.map(x => [x.key, x]))
  })

  constructor() {}

  // Methods
  addProducto(e: ProductoVenta, cant: number, sku?: string) {
    const ventaCant = this.ventaProductosMap.get(e.key)?.cantidad || 0
    const stock = e.cant - ventaCant

    if(stock < cant){
      this.ventaErrorMessage = `No hay suficiente stock de "${e.producto?.Nombre}" para agregar ${cant} unidades.`
      return
    }

    const ventaProducto = this.ventaProductos.find(x => x.key === e.key)
    if(ventaProducto){
      ventaProducto.cantidad += cant
      if(sku){
        const currentSkus = ventaProducto.skus || new Map<string, number>()
        const skuAdded = currentSkus.get(sku)

        if(skuAdded){
          const stockCant = e.skus?.find(x => x.SKU === sku)?.Cantidad || 0
          if((skuAdded + 1) > stockCant){
            this.ventaErrorMessage = `El SKU ${sku} del producto "${e.producto?.Nombre}" sólo posee ${stockCant} unidad(es).`
            return
          }
          const newSkus = new Map(currentSkus)
          newSkus.set(sku, skuAdded + 1)
          ventaProducto.skus = newSkus
        } else {
          const newSkus = new Map(currentSkus)
          newSkus.set(sku, 1)
          ventaProducto.skus = newSkus
        }
      }
      this.ventaProductos = [...this.ventaProductos]
    } else {
      this.ventaProductos.push({
        key: e.key,
        cantidad: cant,
        productoID: e.producto.ID,
        skus: new Map(sku ? [[sku,1]] : []),
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
      return
    }

    if (this.form.AlmacenID === 0) {	
      Notify.failure("Seleccione un almacén.")
      return
    }

    Loading.standard("Procesando venta...")

    // Prepare detail slices
    this.form.DetailProductsIDs = []
    this.form.DetailPrices = []
    this.form.DetailQuantities = []

    for (const vp of this.ventaProductos) {
      this.form.DetailProductsIDs.push(vp.productoID)
      this.form.DetailQuantities.push(vp.cantidad)
      
      let precio = vp.producto?.PrecioFinal || 0
      if (vp.isSubUnidad && vp.producto?.SbnPreciFinal) {
        precio = vp.producto.SbnPreciFinal
      }
      this.form.DetailPrices.push(precio)
    }

    try {
      const res = await POST({
        route: "sale_order",
        data: this.form,
        successMessage: "Venta registrada con éxito"
      })

      if (res) {
        // Reset state
        this.ventaProductos = []
        this.recalcTotales()
        this.form.montoRecibido = 0
        this.form.montoVuelto = 0
      }
    } catch (error) {
      console.error("Error posting sale order:", error)
    } finally {
      Loading.remove()
    }
  }
}
