import { GetHandler } from "$lib/http";
import { browser } from '$app/environment';

export interface IProductoPropiedad {
  id: number, nm: string, ss: number
}

export interface IProductoPropiedades {
  ID: number, Nombre: string, Options: IProductoPropiedad[], Status: number
}

export interface IProductoImage {
  n: string /* Nombre del imagen */
  d: string /* descripcion de la imagen */
}

export interface IProducto {
  ID: number,
  Nombre: string
  Descripcion: string
  Precio?: number
  Descuento?: number
  PrecioFinal?: number
  ContentHTML?: string
  Propiedades?: IProductoPropiedades[]
  Peso?: number
  Volumen?: number
  SbnCantidad?: number
  SbnUnidad?: string
  SbnPrecio?: number
  SbnDescuento?: number
  SbnPreciFinal?: number
  Images?: IProductoImage[]
  Image?: IProductoImage
  CategoriasIDs?: number[]
  Stock?: {a /* almacen */: number, c /* cantidad */: number}[]
  ss: number
  upd: number
  _stock?: number
  _moneda?: string
}

export interface IProductoResult {
  Records?: IProducto[]
  productos: IProducto[]
  productosMap: Map<number,IProducto>
}

export class ProductosService extends GetHandler {
  route = "productos"
  useCache = { min: 5, ver: 1 }

  productos: IProducto[] = $state([])
  productosMap: Map<number,IProducto> = $state(new Map())

  handler(result: IProducto[]): void {
    this.productos = result
    this.productosMap = new Map(result.map(x => [x.ID, x]))
  }

  constructor(){
    super()
    if (browser) {
      this.fetch()
    }
  }
}