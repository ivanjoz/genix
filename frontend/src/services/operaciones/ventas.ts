import { Notify } from "notiflix"
import { GET, GetSignal, POST, makeGETFetchHandler } from "~/shared/http"
import { arrayToMapN } from "~/shared/main"
import { IAlmacenMovimientosResult } from "./productos"

export interface ICaja {
	ID: number
	SedeID: number
	Nombre: string
	Descripcion: string
	MonedaTipo: number
	FechaDeCuadre: number
	MontoCurrent: number
	MontoCuadre: number
  Tipo: number
	ss: number
	upd: number
}

export interface ICajaResult {
  Cajas: ICaja[]
  CajasMap: Map<number,ICaja>
}

export const useCajasAPI = (): GetSignal<ICajaResult> => {
  return  makeGETFetchHandler(
    { route: "cajas", emptyValue: [],
      errorMessage: 'Hubo un error al obtener los productos.',
      cacheSyncTime: 1, mergeRequest: true,
      useIndexDBCache: 'cajas',
    },
    (result_) => {
      console.log("result cajas::",result_)
      const result = result_ as ICajaResult
      result.Cajas = result.Cajas || []
      result.CajasMap = arrayToMapN(result.Cajas,'ID')
      return result
    }
  )
}

export const postCaja = (data: ICaja) => {
  return POST({
    data,
    route: "cajas",
    refreshIndexDBCache: "cajas"
  })
}

export interface IGetCajaMovimientos {
  cajaID: number
  fechaInicio?: number
  fechaFin?: number
  lastRegistros?: number
}

export interface ICajaMovimientos {
	CajaID: number
	VentaID: number
  Tipo: number
  Monto: number
  Created: number
  CreatedBy: number
}

export const getCajaMovimientos = async (args: IGetCajaMovimientos): Promise<ICajaMovimientos[]> => {
  let route = `caja-movimientos?almacen-id=${args.cajaID}`
  
  if((!args.fechaInicio || !args.fechaFin) && !args.lastRegistros){
    throw("No se encontró una fecha de inicio o fin.")
  }
  
  route += `&fecha-hora-inicio=${args.fechaInicio*24*60*60 + window._zoneOffset}`
  route += `&fecha-hora-fin=${(args.fechaFin+1)*24*60*60 + window._zoneOffset}`
  if(args.lastRegistros){
    route += `&last-registros=${args.lastRegistros}`
  }

  let result: ICajaMovimientos[]

  try {
    result = await GET({ 
      route, emptyValue: [],
      errorMessage: 'Hubo un error al obtener los movimientos de la caja.',
    })
  } catch (error) {
    Notify.failure(error as string)
  }

  return result
}