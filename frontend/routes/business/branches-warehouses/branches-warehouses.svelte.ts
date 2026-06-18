import { GetHandler, POST } from '$libs/http.svelte';

export interface ISite {
  ID: number,
  Name: string
  Direccion: string
  Telefono: string
  Description: string
  CityID: number
  Ciudad: string
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
  useCache = { min: 5, ver: 3 }

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

export interface ICityLocation {
  CountryID: number
	ID: number
  Name: string
  ParentID: number
  Hierarchy: number
  Departamento?: ICityLocation
  Provincia?: ICityLocation
  upd: number
  _nombre?: string
}

export interface CountryCityResult {
  ciudades: ICityLocation[]
  distritos: ICityLocation[]
  ciudadesMap: Map<number,ICityLocation>
}

export class CountryCitiesService extends GetHandler {
  route = "country-cities?pais-id=604"
  useCache = { min: 600, ver: 1 }

	ciudades: ICityLocation[] = $state([]) // Departamentos + Provincias + Distritos
	ciudadesMap: Map<number, ICityLocation> = $state(new Map())
	
	distritos: ICityLocation[] = $state([])
	provincias: ICityLocation[] = $state([])
  departamentos: ICityLocation[] = $state([])

  handler(result: ICityLocation[]): void {
		const ciudades = result?.filter(x => !(x as any)._IS_META) || []
		const ciudadesMap = new Map(ciudades.map(e => [e.ID, e]))
		
		const distritos: ICityLocation[] = []
		const provincias: ICityLocation[] = []
		const departamentos: ICityLocation[] = []

		console.log("ciudades", ciudades)
		
		for (const e of ciudades) {			
      const padre = ciudadesMap.get(e.ParentID)
			if (e.Hierarchy === 3) { distritos.push(e) }
			else if (e.Hierarchy === 2) { provincias.push(e) }
			else if(e.Hierarchy === 1){ departamentos.push(e) }

			if (padre) {
        if(padre.Hierarchy === 2){ e.Provincia = padre }
				else if (padre.Hierarchy === 1) {
					e.Departamento = padre
				}
        if(padre.ParentID && ciudadesMap.has(padre.ParentID)){
          const padre2 = ciudadesMap.get(padre.ParentID)
          if(padre2?.Hierarchy === 2){ e.Provincia = padre2 }
          else if(padre2?.Hierarchy === 1){ e.Departamento = padre2 }
        }
      }
    }

    // Build display names
    for(const e of distritos){
      e._nombre = `${e.Departamento?.Name||"-"} ► ${e.Provincia?.Name||""} ► ${e.Name}`
    }

    // console.log("distritos:", distritos)

    this.ciudades = ciudades
		this.distritos = distritos
		this.provincias = provincias
    this.departamentos = departamentos
    this.ciudadesMap = ciudadesMap
  }

  constructor(init?: boolean){
		super()
    if(init){ this.fetch() }
  }
}
