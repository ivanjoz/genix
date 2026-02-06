import { GetHandler } from '$core/http.svelte';
import { type IProducto } from "../productos/productos.svelte"
import { type IProductoStock } from "../productos-stock/productos-stock.svelte"

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

export interface IVenta {
  clienteID: number
  subtotal: number
  igv: number
  total: number
}

export class VentasState {
  // State
  productosStock = $state([] as IProductoStock[])
  form = $state({ total: 0, subtotal: 0, igv: 0 } as IVenta)
  filterText = $state("")
  ventaErrorMessage = $state("")
  filterSku = $state("")

  // Cart
  ventaProductos = $state([] as VentaProducto[])

  // Computed
  ventaProductosMap = $derived.by(() => { // Using $derived.by for complex logic if needed, but Map construction is simple
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
          // Perform immutable update on Map to trigger reactivity
          const newSkus = new Map(currentSkus)
          newSkus.set(sku, skuAdded + 1)
          ventaProducto.skus = newSkus
        } else {
          const newSkus = new Map(currentSkus)
          newSkus.set(sku, 1)
          ventaProducto.skus = newSkus
        }
      }
      this.ventaProductos = [...this.ventaProductos] // trigger update if deep mutation not caught (arrays)
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
        // Adjust price for subunits if needed, based on legacy logic it seems price is total?
        // Legacy: const monto = producto.PrecioFinal * vp.cantidad
        // If subunit, usually price is different, but legacy code uses product.PrecioFinal directly.
        // Assuming ProductoVentaCard logic handles specific subunit price if separate product wasn't created.
        // In legacy: `if(e.isSubUnidad) { ... }`
        // Wait, in legacy, `productosParsed` creates entries.
        // If it is subunit, it has `isSubUnidad: true`.
        // But the `producto` object attached is the SAME parent product.
        // Legacy `recalcVentaTotales`: `const monto = producto.PrecioFinal * vp.cantidad`
        // Wait, if it's a subunit, shouldn't it use `SbnPreciFinal`?
        // Checking legacy `ventas.tsx` line 210: `const monto = producto.PrecioFinal * vp.cantidad`
        // It seems simpler in legacy? Or maybe `ProductoVenta` created `key` S+ID but linked same product.
        // Let's look at legacy lines 120-124:
        // `clone.cant = producto.SbnCantidad` ... `clone.isSubUnidad = true`
        // But `recalcVentaTotales` (line 204) uses `producto.PrecioFinal`.
        // This might be a BUG in legacy or I am misreading.
        // Ah, `SbnPreciFinal` exists in `IProducto` interface in `productos.svelte.ts`.

        if (vp.isSubUnidad && producto.SbnPreciFinal) {
           precio = producto.SbnPreciFinal
        }

        total += precio * vp.cantidad
      }
    }

    this.form.total = total
    this.form.subtotal = Math.floor(total / 1.18)
    this.form.igv = total - this.form.subtotal
  }
}
