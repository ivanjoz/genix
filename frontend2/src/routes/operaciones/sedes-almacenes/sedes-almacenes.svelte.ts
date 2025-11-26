import { GetHandler } from "$lib/http"

export interface ISede {
  ID: number,
  Nombre: string
  Direccion: string
  Telefono: string
  Descripcion: string
  CiudadID: string
  Ciudad: string
  ss: number
  upd: number
}

export interface IAlmacenLayoutBloque {
  rw: number
  co: number
  nm: string
}

export interface IAlmacenLayout {
  ID: number
  Name: string
  RowCant: number
  ColCant: number
  Bloques?: IAlmacenLayoutBloque[]
}

export interface IAlmacen {
  ID: number,
  SedeID: number
  Nombre: string
  Descripcion: string
  Layout: IAlmacenLayout[]
  ss: number
  upd: number
}

export interface IAlmacenes {
  Almacenes: IAlmacen[]
  AlmacenesMap: Map<number,IAlmacen>
  Sedes: ISede[]
  SedesMap: Map<number,ISede>
}

export class AlmacenesService extends GetHandler {
  route = "sedes-almacenes"
  useCache = { min: 5, ver: 1 }

  Almacenes: IAlmacen[] = $state([])
  AlmacenesMap: Map<number,IAlmacen> = $state(new Map())
  Sedes: ISede[] = $state([])
  SedesMap: Map<number,ISede> = $state(new Map())

  handler(result: IAlmacenes): void {
    console.log("sedes almacenes::", result)
    this.Almacenes = result.Almacenes || []
    this.Sedes = result.Sedes || []
    this.SedesMap = new Map(this.Sedes.map(e => [e.ID,e]))
    this.AlmacenesMap = new Map(this.Almacenes.map(e => [e.ID,e]))
  }

  constructor(){
    super()
    this.fetch()
  }
}