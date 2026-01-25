import { GetHandler, POST, GET } from '$core/http'
import { formatTime } from '$core/helpers'
import { Notify } from '$core/lib/helpers';

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
  CajasMap: Map<number, ICaja>
}

export interface IUsuario {
  id: number
  usuario: string
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

export class CajasService extends GetHandler {
  route = "cajas"
  useCache = { min: 1, ver: 1 }

  Cajas: ICaja[] = $state([])
  CajasMap: Map<number, ICaja> = $state(new Map())

  handler(result: ICajaResult): void {
    console.log("result cajas::", result)
    this.Cajas = result.Cajas || []
    for (let e of this.Cajas) {
      e.SaldoCurrent = e.SaldoCurrent || 0
    }
    this.CajasMap = new Map(this.Cajas.map(x => [x.ID, x]))
  }

  constructor() {
    super()
    this.fetch()
  }
}

export const postCaja = (data: ICaja) => {
  return POST({
    data,
    route: "cajas",
    refreshRoutes: ["cajas"]
  })
}

export interface IGetCajaMovimientos {
  CajaID: number
  fechaInicio?: number
  fechaFin?: number
  lastRegistros?: number
}

export interface ICajaMovimientosResult {
  movimientos: ICajaMovimiento[]
  usuarios: IUsuario[]
}

const arrayToMapN = <T extends { [key: string]: any }>(
  arr: T[], key: string
): Map<number, T> => {
  return new Map(arr.map(x => [x[key] as number, x]))
}

export const getCajaMovimientos = async (args: IGetCajaMovimientos): Promise<ICajaMovimiento[]> => {
  let route = `caja-movimientos?caja-id=${args.CajaID}`

  if ((!args.fechaInicio || !args.fechaFin) && !args.lastRegistros) {
    throw ("No se encontró una fecha de inicio o fin.")
  }

  if (args.fechaInicio && args.fechaFin) {
    route += `&fecha-hora-inicio=${args.fechaInicio}`
    route += `&fecha-hora-fin=${args.fechaFin + 1}`
  }
  if (args.lastRegistros) {
    route += `&last-registros=${args.lastRegistros}`
  }

  let result: ICajaMovimientosResult

  try {
    result = await GET({ route })
  } catch (error) {
    console.log("Error:", error)
    Notify.failure(error as string)
    throw error
  }

  const usuariosMap = arrayToMapN(result.usuarios, 'id')
  for (let e of result.movimientos) {
    e.Usuario = usuariosMap.get(e.CreatedBy)
  }

  return result.movimientos
}

export const postCajaMovimiento = (data: ICajaMovimiento) => {
  return POST({
    data,
    route: "caja-movimiento",
    refreshRoutes: ["cajas"]
  })
}

export const postCajaCuadre = (data: ICajaCuadre) => {
  return POST({
    data,
    route: "caja-cuadre",
    refreshRoutes: ["cajas"]
  })
}

export interface ICajaCuadresResult {
  cuadres: ICajaCuadre[]
  usuarios: IUsuario[]
}

export const getCajaCuadres = async (args: IGetCajaMovimientos): Promise<ICajaCuadre[]> => {
  let route = `caja-cuadres?caja-id=${args.CajaID}`

  if ((!args.fechaInicio || !args.fechaFin) && !args.lastRegistros) {
    throw ("No se encontró una fecha de inicio o fin.")
  }

  if (args.fechaInicio && args.fechaFin) {
    route += `&fecha-hora-inicio=${args.fechaInicio * 24 * 60 * 60}`
    route += `&fecha-hora-fin=${(args.fechaFin + 1) * 24 * 60 * 60}`
  }
  if (args.lastRegistros) {
    route += `&last-registros=${args.lastRegistros}`
  }

  let result: ICajaCuadresResult

  try {
    result = await GET({ route })
  } catch (error) {
    console.log("Error:", error)
    Notify.failure(error as string)
    throw error
  }

  console.log("Result Cajas::", result)

  const usuariosMap = arrayToMapN(result.usuarios, 'id')
  for (const e of result.cuadres || []) {
    e.Usuario = usuariosMap.get(e.CreatedBy)
  }

  return result.cuadres
}

// Constantes compartidas
export const cajaTipos = [
  { id: 1, name: "Caja" },
  { id: 2, name: "Cuenta Bancaria" }
]

export const cajaMovimientoTipos = [
  { id: 1, name: "-", group: 1 },
  { id: 2, name: "Cuadre Físico", group: 1 },
  { id: 3, name: "Transferencia", group: 2, isNegative: true },
  { id: 4, name: "Retiro", group: 2, isNegative: true },
  { id: 5, name: "Pérdida", group: 2, isNegative: true },
  { id: 6, name: "Pago", group: 2, isNegative: true },
  { id: 7, name: "Cobro", group: 2 }
]

