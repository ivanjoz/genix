import { GetHandler, POST } from '$libs/http.svelte';
import { browser } from '$app/environment';
import type { ImageSource } from '$components/ImageUploader.svelte';

export interface IProductoPropiedad {
  id: number, nm: string, ss: number
}

export interface IProductoPropiedades {
  ID: number, Nombre: string, Options: IProductoPropiedad[], Status: number
}

export interface IProductoPresentacion {
  id: number, 
  at: number, /* atributo */
  nm: string, /* nombre */
  cl: string, /* color */
  pc: number, /* precio */
  pd: number, /* precio diferencial */
  ss: number  /* estado */
}

export interface IProductoImage {
  n: string /* Nombre del imagen */
  d: string /* descripcion de la imagen */
}

export interface IProducto {
  ID: number,
  Nombre: string
  Descripcion: string
  Precio: number
  Descuento: number
  MonedaID: number
  PrecioFinal: number
  ContentHTML?: string
  Propiedades: IProductoPropiedades[]
  Presentaciones: IProductoPresentacion[]
  Peso?: number
  Volumen?: number
  SbnCantidad?: number
  SbnUnidad?: string
  SbnPrecio?: number
  SbnDescuento?: number
  SbnPreciFinal?: number
  Images?: IProductoImage[]
  Image?: IProductoImage
  CategoriasIDs: number[]
  AtributosIDs?: number[]
  MarcaID: number
  UnidadID: number
  Stock?: {a /* almacen */: number, c /* cantidad */: number}[]
  Params: number[]
  ss: number
  upd: number
  _stock?: number
  _moneda?: string
  _imageSource?: ImageSource
}

export interface IProductoResult {
  Records?: IProducto[]
  productos: IProducto[]
  productosMap: Map<number,IProducto>
}

export class ProductosService extends GetHandler {
  route = "productos"
  useCache = { min: 5, ver: 9 }

  productos: IProducto[] = $state([])
  productosMap: Map<number,IProducto> = $state(new Map())

  handler(result: IProducto[]): void {
    console.log("productos result::", result)
    for(const e of result){
      e.Image = e.Images?.[0]
      e.CategoriasIDs = e.CategoriasIDs || []
    }

    this.productos = result.filter(x => x.ss)
    this.productosMap = new Map(result.map(x => [x.ID, x]))
  }

  constructor(){
    super()
    this.fetch()
  }
}

export const postProducto = (data: IProducto[]) => {
  return POST({
    data,
    route: "productos",
    refreshRoutes: ["productos"]
  })
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
  route = "listas-compartidas"
  useCache = { min: 5, ver: 6 }

  Records: IListaRegistro[] = $state([])
  RecordsMap: Map<number,IListaRegistro>= $state(new Map())
  ListaRecordsMap: Map<number,IListaRegistro[]>= $state(new Map())

  get(id: number){
    return this.RecordsMap.get(id)
  }

  handler(result: {[k: string]: IListaRegistro[]}): void {
    console.log("result getted::", result)

    this.Records = []
    this.ListaRecordsMap = new Map()

    for(const [key, records] of Object.entries(result)){
      const listaID = parseInt(key.split("_")[1])
      
      for(const e of records){
        if(!e.ss){ continue }
        this.Records.push(e)

        if([1,2].includes(listaID)){ // Productos Categorias
          const imagesMap = new Map((e.Images||[]).filter(x => x).map(x => (
            [parseInt(x.split("-")[1]),x])))
          
          e.Images = []
          for(const order of [1,2,3]){
            e.Images.push(imagesMap.get(order)||"")
          }
        }

        this.ListaRecordsMap.has(e.ListaID)
          ? this.ListaRecordsMap.get(e.ListaID)?.push(e)
          : this.ListaRecordsMap.set(e.ListaID, [e])
      }
    }

    this.RecordsMap = new Map(this.Records.map(x => [x.ID,x]))
  }

  addNew(e: IListaRegistro){
    const records = this.ListaRecordsMap.get(e.ListaID) || []
    records.unshift(e)
    this.ListaRecordsMap.set(e.ListaID, [...records])
    this.ListaRecordsMap = new Map(this.ListaRecordsMap)
    this.RecordsMap.set(e.ID, e)
    this.Records.push(e)
  }

  constructor(ids: number[] = []){
    super()
    if(ids.length > 0){
      this.route = `listas-compartidas?ids=${ids.join(",")}`
    }
    if(ids){
      this.fetch()
    }
  }
}

export interface INewIDToID {
  NewID:  number
	TempID: number
}

export const postListaRegistros = (data: IListaRegistro[]) => {
  return POST({
    data,
    route: "listas-compartidas",
    refreshRoutes: ["listas-compartidas"]
  })
}

export const productoAtributos = [
  { id: 1, name: "Color" },
  { id: 2, name: "Talla" },
  { id: 3, name: "Tamaño" },
  { id: 4, name: "Forma" },
  { id: 5, name: "Presentación" },
]
