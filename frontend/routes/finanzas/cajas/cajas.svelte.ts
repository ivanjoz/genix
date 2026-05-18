import { GetHandler, POST, GET } from '$libs/http.svelte';
import { formatTime } from '$libs/helpers';
import { Notify } from '$libs/helpers';

export interface ICashBank {
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
  Cajas: ICashBank[]
  CajasMap: Map<number, ICashBank>
}

export interface ICashBankMovement {
  CajaID: number
  CajaRefID: number
  VentaID: number
  DocumentoID: number
  Tipo: number
  Monto: number
  SaldoFinal: number
  Created: number
  CreatedBy: number
}

export interface ICashReconciliation {
  ID: number
  Tipo: number
  CajaID: number
  SaldoSistema: number
  SaldoDiferencia: number
  SaldoReal: number
  Created: number
  CreatedBy: number
  _error?: string
}

export class CajasService extends GetHandler {
  route = "cajas"
  useCache = { min: 1, ver: 1 }

  Cajas: ICashBank[] = $state([])
  CajasMap: Map<number, ICashBank> = $state(new Map())

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

export const postCaja = (data: ICashBank) => {
  return POST({
    data,
    route: "cajas",
    refreshRoutes: ["cajas"]
  })
}

export interface IGetCajaMovimientos {
  CajaID: number
  dateInicio?: number
  dateFin?: number
  lastRegistros?: number
}

export interface ICajaMovimientosResult {
  movimientos: ICashBankMovement[]
}

export const getCajaMovimientos = async (args: IGetCajaMovimientos): Promise<ICashBankMovement[]> => {
  let route = `caja-movimientos?caja-id=${args.CajaID}`

  if ((!args.dateInicio || !args.dateFin) && !args.lastRegistros) {
    throw ("No se encontró una date de inicio o fin.")
  }

  if (args.dateInicio && args.dateFin) {
    route += `&date-inicio=${args.dateInicio}`
    route += `&date-fin=${args.dateFin}`
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

  return result.movimientos || []
}

export const postCajaMovimiento = (data: ICashBankMovement) => {
  return POST({
    data,
    route: "caja-movimiento",
    refreshRoutes: ["cajas"]
  })
}

export const postCajaCuadre = (data: ICashReconciliation) => {
  return POST({
    data,
    route: "caja-cuadre",
    refreshRoutes: ["cajas"]
  })
}

export interface ICajaCuadresResult {
  cuadres: ICashReconciliation[]
}

export const getCajaCuadres = async (args: IGetCajaMovimientos): Promise<ICashReconciliation[]> => {
  let route = `caja-cuadres?caja-id=${args.CajaID}`

  if ((!args.dateInicio || !args.dateFin) && !args.lastRegistros) {
    throw ("No se encontró una date de inicio o fin.")
  }

  if (args.dateInicio && args.dateFin) {
    route += `&date-hora-inicio=${args.dateInicio * 24 * 60 * 60}`
    route += `&date-hora-fin=${(args.dateFin + 1) * 24 * 60 * 60}`
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

  return result.cuadres || []
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
  { id: 6, name: "Pago Proveedor", group: 2, isNegative: true },
	{ id: 7, name: "Cobro", group: 2 },
  { id: 8, name: "Cobro (Venta)", group: 2 }
]
