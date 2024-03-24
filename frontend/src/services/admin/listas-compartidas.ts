import { GetSignal, POST, makeGETFetchHandler } from "~/shared/http"
import { arrayToMapN, arrayToMapS } from "~/shared/main"

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
  registros: IListaRegistro[]
  registrosMap: Map<number,IListaRegistro>
}

export const useListasCompartidasAPI = (ids: number[]): GetSignal<IListas> => {
  const pk = ids.sort().join(",")

  return  makeGETFetchHandler(
    { route: "listas-compartidas",
      errorMessage: 'Hubo un error al obtener las listas compartidas.',
      cacheSyncTime: 1, mergeRequest: true,
      partition: { key: '_pk', value: pk, param: 'ids' },
      useIndexDBCache: 'listas_compartidas',
      makeTransform: e => { e._pk = pk }
    },
    (result_) => {
      const result = result_ as IListas
      result.registros = []
      result.registrosMap = arrayToMapN(result.registros, 'ID')
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
