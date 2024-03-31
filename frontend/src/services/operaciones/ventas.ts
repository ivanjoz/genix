import { Notify } from "notiflix"
import { GET, GetSignal, POST, makeGETFetchHandler } from "~/shared/http"
import { arrayToMapN } from "~/shared/main"
import { IAlmacenMovimientosResult } from "./productos"
import { IUsuario } from "../admin/empresas"
import { fechaUnixToSunix } from "~/core/main"

export interface ICaja {
	ID: number
	SedeID: number
	Nombre: string
	Descripcion: string
	MonedaTipo: number
	CuadreFecha: number
	SaldoCurrent: number
	CuadreSaldo: number
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
      for(let e of result.Cajas){
        e.SaldoCurrent = e.SaldoCurrent || 0
      }
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
  CajaID: number
  fechaInicio?: number
  fechaFin?: number
  lastRegistros?: number
}

export interface ICajaMovimiento {
	CajaID: number
  CajaRefID: number
	VentaID: number
  Tipo: number
  Monto: number
  SaldoFinal: number
  Created: number
  CreatedBy: number
  Usuario?: IUsuario
}

export interface ICajaMovimientosResult {
  movimientos: ICajaMovimiento[]
  usuarios: IUsuario[]
}

export const getCajaMovimientos = async (args: IGetCajaMovimientos): Promise<ICajaMovimiento[]> => {
  let route = `caja-movimientos?caja-id=${args.CajaID}`
  
  if((!args.fechaInicio || !args.fechaFin) && !args.lastRegistros){
    throw("No se encontró una fecha de inicio o fin.")
  }
  
  route += `&fecha-hora-inicio=${fechaUnixToSunix(args.fechaInicio)}`
  route += `&fecha-hora-fin=${fechaUnixToSunix(args.fechaFin + 1)}`
  if(args.lastRegistros){
    route += `&last-registros=${args.lastRegistros}`
  }

  let result: ICajaMovimientosResult

  try {
    result = await GET({ 
      route, emptyValue: [],
      errorMessage: 'Hubo un error al obtener los movimientos de la caja.',
    })
  } catch (error) {
    Notify.failure(error as string)
  }

  const usuariosMap = arrayToMapN(result.usuarios, 'id')
  for(let e of result.movimientos){
    e.Usuario = usuariosMap.get(e.CreatedBy)
  }

  return result.movimientos
}

export const postCajaMovimiento = (data: ICajaMovimiento) => {
  return POST({
    data,
    route: "caja-movimiento",
    refreshIndexDBCache: "cajas"
  })
}

export interface ICajaCuadre {
  ID: number
  Tipo: number
	CajaID: number
	SaldoSistema: number
  SaldoDiferencia: number
  SaldoReal: number
  Created: number
  CreatedBy: number
  Usuario?: IUsuario
  _error?: string
}

export const postCajaCuadre = (data: ICajaCuadre) => {
  return POST({
    data,
    route: "caja-cuadre",
    refreshIndexDBCache: "cajas"
  })
}


export interface ICajaCuadresResult {
  cuadres: ICajaCuadre[]
  usuarios: IUsuario[]
}

export const getCajaCuadres = async (args: IGetCajaMovimientos): Promise<ICajaCuadre[]> => {
  let route = `caja-cuadres?caja-id=${args.CajaID}`
  
  if((!args.fechaInicio || !args.fechaFin) && !args.lastRegistros){
    throw("No se encontró una fecha de inicio o fin.")
  }
  
  route += `&fecha-hora-inicio=${args.fechaInicio*24*60*60 + window._zoneOffset}`
  route += `&fecha-hora-fin=${(args.fechaFin+1)*24*60*60 + window._zoneOffset}`
  if(args.lastRegistros){
    route += `&last-registros=${args.lastRegistros}`
  }

  let result: ICajaCuadresResult

  try {
    result = await GET({ 
      route, emptyValue: [],
      errorMessage: 'Hubo un error al obtener los movimientos de la caja.',
    })
  } catch (error) {
    Notify.failure(error as string)
  }

  const usuariosMap = arrayToMapN(result.usuarios, 'id')
  for(let e of result.cuadres){
    e.Usuario = usuariosMap.get(e.CreatedBy)
  }

  return result.cuadres
}