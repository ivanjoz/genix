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
  useCache = { min: 5, ver: 8 }

  productos: IProducto[] = $state([])
  productosMap: Map<number,IProducto> = $state(new Map())

  handler(result: IProducto[]): void {
    console.log("productos result::", result)

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

export interface IListaRegistro {
  ID: number
  ListaID: number
  Nombre: string
  Images?: string[]
  Descripcion?: string
  UpdatedBy?: number
  ss: number
  upd: number
}

export interface IListas {
  Records: IListaRegistro[]
  RecordsMap: Map<number,IListaRegistro>
}

export const listasCompartidas = [
  { id: 1, name: "Categoría" },
  { id: 2, name: "Marca" }
]

export class ListasCompartidasService extends GetHandler {
  useCache = { min: 5, ver: 1 }

  Records: IListaRegistro[] = $state([])
  RecordsMap: Map<number,IListaRegistro>= $state(new Map())

  handler(result: IListaRegistro[]): void {
    console.log("listas result::", result)

    this.RecordsMap = new Map(result.map(x => [x.ID,x]))
    this.Records = result

    for(const e of this.Records){
      if(e.ListaID === 1){ // Productos Categorias
        const imagesMap = new Map((e.Images||[]).map(x => (
          [parseInt(x.split("-")[1]),x])))
        
        e.Images = []
        for(const order of [1,2,3]){
          e.Images.push(imagesMap.get(order)||"")
        }
      }
    }
  }

  constructor(ids: number[]){
    super()
    this.route = `listas-compartidas?ids=${ids.join(",")}`

    if (browser) {
      this.fetch()
    }
  }
}