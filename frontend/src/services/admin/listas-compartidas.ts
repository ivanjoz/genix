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
  Records: IListaRegistro[]
  RecordsMap: Map<number,IListaRegistro>
}

export const useListasCompartidasAPI = (ids: number[]): GetSignal<IListas> => {
  const pk = ids.sort().join(",")

  return  makeGETFetchHandler(
    { route: "listas-compartidas",
      errorMessage: 'Hubo un error al obtener las listas compartidas.',
      cacheSyncTime: 1, mergeRequest: true,
      partition: { key: 'pk', value: pk, param: 'ids' },
      useIndexDBCache: 'listas_compartidas',
      makeTransform: e => { e.pk = pk }
    },
    (result_) => {
      const result = result_ as IListas
      result.RecordsMap = arrayToMapN(result.Records, 'ID')
      // console.log("listas compartidas API::",result)
      result.Records = result.Records.filter(x => x.ss)
      return result
    }
  )
}

export interface INewIDToID {
  NewID:  number
	TempID: number
}

export const postListaRegistros = (data: IListaRegistro[]) => {
  return POST({
    data,
    route: "listas-compartidas",
    refreshIndexDBCache: "listas_compartidas"
  })
}
