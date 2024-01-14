import { GetSignal, POST, makeGETFetchHandler } from "~/shared/http"
import { arrayToMapN, arrayToMapS } from "~/shared/main"

export interface IProductoPropiedad {
  ID: number
  Nombre: string
  Options: string[]
}

export interface IListaRegistro {
  ListaID: number
  ID: string,
  Nombre: string
  ss: number
  upd: number
}

export interface IListasResult {
  Records?: IListaRegistro[]
  registros: IListaRegistro[]
  registrosMap: Map<number,IListaRegistro[]>
}

export const useListasCompartidasAPI = (): GetSignal<IListasResult> => {
  return  makeGETFetchHandler(
    { route: "listas-compartidas", emptyValue: [],
      errorMessage: 'Hubo un error al obtener las listas compartidas.',
      cacheSyncTime: 1, mergeRequest: true,
      useIndexDBCache: 'listas_compartidas',
    },
    (result_) => {
      const result = result_ as IListasResult

      return result
    }
  )
}

export const postListaRegistros = (data: IListaRegistro[]) => {
  return POST({
    data,
    route: "listas-compartidas",
    refreshIndexDBCache: "listas_compartidas"
  })
}
