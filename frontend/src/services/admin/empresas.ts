import { GetSignal, POST, makeGETFetchHandler } from "~/shared/http"
import { arrayToMapN } from "~/shared/main"

interface IEmpresa {
  id: number,
  email: string
  nombre: string
  razonSocial: string
  ruc: number
  telefono: string
  ss: number
  upd: number
}

export const useEmpresasAPI = (): GetSignal<IEmpresa[]> => {
  return  makeGETFetchHandler(
    { route: "empresas", emptyValue: [],
      errorMessage: 'Hubo un error al obtener las empresas.',
      cacheSyncTime: 1, mergeRequest: true,
      useIndexDBCache: 'empresas',
    },
    (result) => {
      console.log("resultado obtenido:: ", result.Records)
      return result.Records
    }
  )
}

export interface IUsuario {
  id: number,
  companyID: number
  nombres: string
  apellidos: string
  email: string
  usuario: string
  documentoNro: string
  cargo: string
  perfilesIDs: number[]
  rolesIDs: number[]
  ss: number
  upd: number
  created: number
  password1: string
  password2: string
}

export interface IUsuarioResult {
  usuarios: IUsuario[]
  usuariosMap: Map<number,IUsuario>
}

export const useUsuariosAPI = (): GetSignal<IUsuarioResult> => {
  return  makeGETFetchHandler(
    { route: "usuarios", emptyValue: {},
      errorMessage: 'Hubo un error al obtener los usuarios.',
      cacheSyncTime: 1, mergeRequest: true,
      useIndexDBCache: 'usuarios',
    },
    (result) => {
      const usuarios = result.Records
      return {
        usuarios,
        usuariosMap: arrayToMapN(usuarios,'id')
      } as IUsuarioResult
    }
  )
}

export const postUsuario = (data: IUsuario) => {
  return POST({
    data,
    route: "usuarios",
    refreshIndexDBCache: "usuarios"
  })
}

export interface IAcceso {
  id: number
  nombre: string
  descripcion?: string
  grupo: number
  orden: number
  modulosIDs: number[]
  acciones: number[]
  ss: number
  upd: number
}

export const useAccesosAPI = (): GetSignal<IAcceso[]> => {
  return makeGETFetchHandler(
    { route: "seguridad-accesos", emptyValue: [],
      errorMessage: 'Hubo un error al obtener los accesos.',
      cacheSyncTime: 1, mergeRequest: true,
      useIndexDBCache: 'seguridad_accesos',
    },
    (result) => {
      const accesos = result.Records
      console.log("accesos obtenidos::", accesos)
      return accesos
    }
  )
}

export const postSeguridadAccesos = (data: IAcceso) => {
  return POST({
    data,
    route: "seguridad-accesos",
    refreshIndexDBCache: "seguridad_accesos"
  })
}


export interface IPefil {
  id: number
  nombre: string
  descripcion?: string
  accesos: number[]
  modulosIDs: number[]
  accesosMap: Map<number,number[]>
  ss: number
  upd: number
  _open?: boolean
}

export interface IPerfiles {
  perfiles: IPefil[]
}

export const usePerfilesAPI = (): GetSignal<IPerfiles> => {
  return  makeGETFetchHandler(
    { route: "perfiles", emptyValue: [],
      errorMessage: 'Hubo un error al obtener los perfiles.',
      cacheSyncTime: 1, mergeRequest: true, useIndexDBCache: 'perfiles',
    },
    (result) => {
      for(let pr of result.Records){
        pr.accesos = pr.accesos || []
        pr.accesosMap = pr.accesosMap || new Map()
        
        for(let e of pr.accesos){
          const id = Math.floor(e / 10)
          const nivel = e - (id * 10)
          pr.accesosMap.has(id) 
            ? pr.accesosMap.get(id).push(nivel)
            : pr.accesosMap.set(id, [nivel])
        }
      }
      
      const perfiles = result.Records.filter(x => x.ss > 0)
      console.log("Perfiles Obtenidos:: ", perfiles)
      return { perfiles }
    }
  )
}

export const postPerfil = (data: IPefil) => {
  return POST({
    data,
    route: "perfiles",
    refreshIndexDBCache: "perfiles"
  })
}