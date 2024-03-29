import { GetSignal, POST, makeGETFetchHandler } from "~/shared/http"
import { arrayToMapN } from "~/shared/main"

export interface ICaja {
	ID: number
	SedeID: number
	Nombre: string
	Descripcion: string
	MonedaTipo: number
	FechaDeCuadre: number
	MontoCurrent: number
	MontoCuadre: number
	ss: number
	upd: number
}

export interface ICajaResult {
  Records: ICaja[]
  RecordsMap: Map<number,ICaja>
}

export const useCajasAPI = (): GetSignal<ICajaResult> => {
  return  makeGETFetchHandler(
    { route: "cajas", emptyValue: [],
      errorMessage: 'Hubo un error al obtener los productos.',
      cacheSyncTime: 1, mergeRequest: true,
      useIndexDBCache: 'cajas',
    },
    (result_) => {
      const result = result_ as ICajaResult
      result.Records = result.Records || []
      result.RecordsMap = arrayToMapN(result.Records,'ID')
      return result
    }
  )
}

export const postCaja = (data: ICaja) => {
  return POST({
    data,
    route: "caja",
    refreshIndexDBCache: "cajas"
  })
}