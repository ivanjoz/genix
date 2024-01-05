import { GetSignal, POST, makeGETFetchHandler } from "~/shared/http"
import { arrayToMapN, arrayToMapS } from "~/shared/main"

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

export const useSedesAlmacenesAPI = (): GetSignal<IAlmacenes> => {
  return  makeGETFetchHandler(
    { route: "sedes-almacenes", emptyValue: [],
      errorMessage: 'Hubo un error al obtener las sedes / almacenes.',
      cacheSyncTime: 1, mergeRequest: true,
      useIndexDBCache: 'sedes_almacenes',
    },
    (result_) => {
      const result = result_ as IAlmacenes
      console.log("result almacenes::", result)
      result.SedesMap = arrayToMapN(result.Sedes||[],'ID')
      result.AlmacenesMap = arrayToMapN(result.Almacenes||[],'ID')
      return result
    }
  )
}

export const postSede = (data: ISede) => {
  return POST({
    data,
    route: "sedes",
    refreshIndexDBCache: "sedes_almacenes"
  })
}

export const postAlmacen = (data: IAlmacen) => {
  return POST({
    data,
    route: "almacenes",
    refreshIndexDBCache: "sedes_almacenes"
  })
}

export interface IPaisCiudad {
  PaisID: number
  ID: string
  Nombre: string
  PadreID: string
  Jerarquia: number
  Departamento: IPaisCiudad
  Provincia: IPaisCiudad
  upd: number
  _nombre?: string
}

export interface PaisCiudadResult {
  ciudades: IPaisCiudad[]
  distritos: IPaisCiudad[]
  ciudadesMap: Map<string,IPaisCiudad>
}

export const usePaisCiudadesAPI = (): GetSignal<PaisCiudadResult> => {
  return makeGETFetchHandler(
    { route: `pais-ciudades?pais-id=604`, emptyValue: [],
      errorMessage: 'Hubo un error al obtener los paises y ciudades',
      cacheSyncTime: 600, mergeRequest: true,
      useIndexDBCache: 'pais_ciudades',
    },
    ({ Records }) => {
      const ciudades = Records?.filter(x => !x._IS_META) as IPaisCiudad[]
      const distritos: IPaisCiudad[] = []
      const ciudadesMap = arrayToMapS(ciudades,'ID')

      for(let e of ciudades){
        const padre = ciudadesMap.get(e.PadreID)
        if(e.Jerarquia === 3){ distritos.push(e) }
        if(padre){
          if(padre.Jerarquia === 2){ e.Provincia = padre }
          else if(padre.Jerarquia === 1){ e.Departamento = padre }
          if(padre.PadreID && ciudadesMap.has(padre.PadreID)){
            const padre2 = ciudadesMap.get(padre.PadreID)
            if(padre2.Jerarquia === 2){ e.Provincia = padre2 }
            else if(padre2.Jerarquia === 1){ e.Departamento = padre2 }
          }
        }
      }

      return { ciudades, ciudadesMap, distritos }
    }
  )
}
