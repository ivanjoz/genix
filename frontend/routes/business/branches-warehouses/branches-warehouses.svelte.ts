import { GetHandler, POST } from '$libs/ui-runtime.svelte';

export interface ISite {
  ID: number,
  Name: string
  Address: string
  Telefono: string
  Description: string
  CityID: number
  City: string
  ss: number
  upd: number
}

export interface IWarehouseLayoutBlock {
  rw: number
  co: number
  nm: string
}

export interface IWarehouseLayout {
  ID: number
  Name: string
  RowCant: number
  ColCant: number
  Bloques?: IWarehouseLayoutBlock[]
  [key: string]: any // For dynamic xy_ properties
}

export interface IWarehouse {
  ID: number,
  SiteID: number
  Name: string
  Description: string
  Layout: IWarehouseLayout[]
  ss: number
  upd: number
}

export interface IWarehouses {
  Almacenes: IWarehouse[]
  AlmacenesMap: Map<number,IWarehouse>
  Sedes: ISite[]
  SedesMap: Map<number,ISite>
}

export class WarehousesService extends GetHandler {
  route = "locations-warehouses"
  useCache = { min: 5, ver: 5 }

  Almacenes: IWarehouse[] = $state([])
  AlmacenesMap: Map<number,IWarehouse> = $state(new Map())
  Sedes: ISite[] = $state([])
  SedesMap: Map<number,ISite> = $state(new Map())

  handler(result: IWarehouses): void {
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

export const postSite = (data: ISite) => {
  return POST({
    data,
    route: "sites",
    refreshRoutes: ["locations-warehouses"]
  })
}

export const postWarehouse = (data: IWarehouse) => {
  return POST({
    data,
    route: "warehouses",
    refreshRoutes: ["locations-warehouses"]
  })
}
