import { GET } from '$libs/http.svelte';
import { Notify } from '$libs/helpers';

export interface IUsuario {
  id: number
  usuario: string
}

interface IQueryAlmacenMovimientos {
  almacenID: number
  fechaInicio: number
  fechaFin: number
}

export interface IAlmacenMovimiento {
  ID: string
  SKU: string
  Lote: string
  AlmacenID: number
  AlmacenOrigenID: number
  DocumentID: number
  ProductoID: number
  Cantidad: number
  AlmacenCantidad: number
  AlmacenOrigenCantidad: number
  SubCantidad: number
  Tipo: number
  Created: number
  CreatedBy: number
}

export interface IProducto {
  ID: number
  Nombre: string
}

export interface IAlmacenMovimientosResult {
  Movimientos: IAlmacenMovimiento[]
  Usuarios: IUsuario[]
  Productos: IProducto[]
}

export const movimientoTipos = [
  { id: 1, name: 'Entrada Manual' }, 
  { id: 2, name: 'Salida Manual' }, 
]

export const queryAlmacenMovimientos = async (args: IQueryAlmacenMovimientos): Promise<IAlmacenMovimientosResult> => {
  let route = `almacen-movimientos?almacen-id=${args.almacenID}`
  
  if (!args.fechaInicio || !args.fechaFin) {
    throw "No se encontró una fecha de inicio o fin."
  }

  route += `&fecha-inicio=${args.fechaInicio}&fecha-fin=${args.fechaFin}`
  let result: IAlmacenMovimientosResult

  try {
    result = await GET({ 
      route,
      errorMessage: 'Hubo un error al obtener los movimientos del almacén',
    })
  } catch (error) {
    Notify.failure(error as string)
    throw error
  }

  console.log("almacén movimientos:", result)
  
  result.Usuarios = result.Usuarios || []
  result.Productos = result.Productos || []
  result.Movimientos = result.Movimientos || []
  result.Movimientos.sort((a, b) => b.Created - a.Created)

  return result
}
