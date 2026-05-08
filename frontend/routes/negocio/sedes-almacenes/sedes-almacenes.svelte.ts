import { GetHandler, POST } from '$libs/http.svelte';

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
  [key: string]: any // For dynamic xy_ properties
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
  useCache = { min: 5, ver: 3 }

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

export const postSede = (data: ISede) => {
  return POST({
    data,
    route: "sedes",
    refreshRoutes: ["sedes-almacenes"]
  })
}

export const postAlmacen = (data: IAlmacen) => {
  return POST({
    data,
    route: "almacenes",
    refreshRoutes: ["sedes-almacenes"]
  })
}

export interface IPaisCiudad {
  PaisID: number
	ID: number
  CiudadID: number
  Nombre: string
  PadreID: number
  Jerarquia: number
  Departamento?: IPaisCiudad
  Provincia?: IPaisCiudad
  upd: number
  _nombre?: string
}

export interface PaisCiudadResult {
  ciudades: IPaisCiudad[]
  distritos: IPaisCiudad[]
  ciudadesMap: Map<number,IPaisCiudad>
}

export class PaisCiudadesService extends GetHandler {
  route = "pais-ciudades?pais-id=604"
  useCache = { min: 600, ver: 1 }

	ciudades: IPaisCiudad[] = $state([]) // Departamentos + Provincias + Distritos
	ciudadesMap: Map<number, IPaisCiudad> = $state(new Map())
	
	distritos: IPaisCiudad[] = $state([])
	provincias: IPaisCiudad[] = $state([])
  departamentos: IPaisCiudad[] = $state([])

  handler(result: IPaisCiudad[]): void {
		const ciudades = result?.filter(x => !(x as any)._IS_META) || []
		const ciudadesMap = new Map(ciudades.map(e => [e.ID, e]))
		
		const distritos: IPaisCiudad[] = []
		const provincias: IPaisCiudad[] = []
		const departamentos: IPaisCiudad[] = []

		console.log("ciudades", ciudades)
		
		for (const e of ciudades) {			
			e.PadreID = parseInt(e.PadreID as unknown as string)
			
      const padre = ciudadesMap.get(e.PadreID)
			if (e.Jerarquia === 3) { distritos.push(e) }
			else if (e.Jerarquia === 2) { provincias.push(e) }
			else if(e.Jerarquia === 1){ departamentos.push(e) }
			
			if (padre) {
        if(padre.Jerarquia === 2){ e.Provincia = padre }
				else if (padre.Jerarquia === 1) {
					e.Departamento = padre
				}
        if(padre.PadreID && ciudadesMap.has(padre.PadreID)){
          const padre2 = ciudadesMap.get(padre.PadreID)
          if(padre2?.Jerarquia === 2){ e.Provincia = padre2 }
          else if(padre2?.Jerarquia === 1){ e.Departamento = padre2 }
        }
      }
    }

    // Build display names
    for(const e of distritos){
      e._nombre = `${e.Departamento?.Nombre||"-"} ► ${e.Provincia?.Nombre||""} ► ${e.Nombre}`
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
