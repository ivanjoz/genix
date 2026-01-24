import { GetHandler, POST } from "$lib/http"

export interface IUsuario {
  id: number
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
  //Extra
  accesosIDs: number[]
}

export interface IPerfil {
  id: number
  nombre: string
  descripcion?: string
  accesos: number[]
  modulosIDs: number[]
  accesosMap: Map<number, number[]>
  ss: number
  upd: number
}

export class UsuariosService extends GetHandler {
  route = "usuarios"
  useCache = { min: 5, ver: 1 }

  usuarios: IUsuario[] = $state([])
  usuariosMap: Map<number, IUsuario> = $state(new Map())

  handler(response: IUsuario[]) {
    this.usuarios = response || []
    this.usuariosMap = new Map(this.usuarios.map(x => [x.id, x]))
  }

  constructor() {
    super()
    this.fetch()
  }

  updateUsuario(usuario: IUsuario) {
    const existing = this.usuarios.find(x => x.id === usuario.id)
    if (existing) {
      Object.assign(existing, usuario)
    } else {
      this.usuarios.unshift(usuario)
    }
    this.usuariosMap.set(usuario.id, usuario)
  }

  removeUsuario(id: number) {
    this.usuarios = this.usuarios.filter(x => x.id !== id)
    this.usuariosMap.delete(id)
  }
}

export class PerfilesService extends GetHandler {
  route = "perfiles"
  useCache = { min: 5, ver: 1 }

  perfiles: IPerfil[] = $state([])

  handler(response: IPerfil[]) {
    const perfiles = (response || []).filter(x => x.ss > 0)
    for (const pr of perfiles) {
      pr.accesos = pr.accesos || []
      pr.accesosMap = pr.accesosMap || new Map()

      for (const e of pr.accesos) {
        const id = Math.floor(e / 10)
        const nivel = e - (id * 10)
        pr.accesosMap.has(id)
          ? pr.accesosMap.get(id)!.push(nivel)
          : pr.accesosMap.set(id, [nivel])
      }
    }
    this.perfiles = perfiles
  }

  constructor() {
    super()
    this.fetch()
  }
}

export const postUsuario = (data: IUsuario) => {
  return POST({
    data,
    route: "usuarios",
    refreshRoutes: ["usuarios"]
  })
}

export const postUsuarioPropio = (data: IUsuario) => {
  return POST({
    data,
    route: "usuario-propio",
    refreshRoutes: ["usuarios"]
  })
}



