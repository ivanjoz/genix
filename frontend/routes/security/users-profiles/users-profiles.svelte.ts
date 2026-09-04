import { GetHandler, POST } from '$libs/ui-runtime.svelte';
import type { IProfile, IUser } from '$core/types/common';
import { unpackSubAccesos, type ISubAccesoOption } from './users-profiles';
export { postUser, postOwnUser } from '$services/services/users.svelte';

export type { IProfile, IUser };
export type { ISubAccesoOption } from './users-profiles';

// Access ids from backend/access.toml. Both accesses resolve to this single route, so
// canAccessRoute only tells us the page is reachable — each tab is gated on its own id.
export const USERS_ACCESS_ID = 2
export const PROFILES_ACCESS_ID = 3

export interface IAccess {
  id: number
  nombre: string
  descripcion?: string
  orden: number
  acciones: number[]
  // The sub-access flags this access offers, "Todos" first. Empty for most accesses.
  subAccesos: ISubAccesoOption[]
  grupo: number
  modulosIDs: number[]
  ss: number
  upd: number
}

export const accesoAcciones = [
  { id: 1, name: "Visualizar", short: "VER",
    icon: "icon-[fa--eye]", color: "#00c07d", color2: "#49c99c" },
  { id: 2, name: "Crear", short: "CREAR",
    icon: "icon-[fa--pencil]", color: "#0080f9" },
  { id: 3, name: "Editar", short: "EDITAR",
    icon: "icon-[fa--pencil]", color: "#0f6bff" },
  { id: 4, name: "Todo", short: "TODO",
    icon: "icon-[fa--shield]", color: "#af12eb", color2: "#d35eff" },
]

export class UsuariosService extends GetHandler {
  route = "users"
  useCache = { min: 0.1, ver: 1 }

  usuarios: IUser[] = $state([])
  usuariosMap: Map<number, IUser> = $state(new Map())

  handler(response: IUser[]) {
    this.usuarios = response || []
    this.usuariosMap = new Map(this.usuarios.map((usuarioRecord) => [usuarioRecord.ID, usuarioRecord]))
  }

  constructor() {
    super()
    this.fetch()
  }

  updateUsuario(usuario: IUser) {
    const existing = this.usuarios.find((usuarioRecord) => usuarioRecord.ID === usuario.ID)
    if (existing) {
      Object.assign(existing, usuario)
    } else {
      this.usuarios.unshift(usuario)
    }
    this.usuariosMap.set(usuario.ID, usuario)
  }

  removeUsuario(id: number) {
    this.usuarios = this.usuarios.filter(x => x.ID !== id)
    this.usuariosMap.delete(id)
  }
}

export class PerfilesService extends GetHandler {
  route = "perfiles"
  // v3: perfiles now carry SubAccesos, so a cached v2 record would edit as if the profile granted
  // none of them and saving it would silently revoke every sub-access it holds.
  useCache = { min: 5, ver: 3 }

  perfiles: IProfile[] = $state([])
  perfilesMap: Map<number, IProfile> = $state(new Map())

  handler(response: IProfile[]) {
    const perfiles = (response || []).filter(x => x.ss > 0)
    for (const pr of perfiles) {
      pr.Accesos = pr.Accesos || []
      pr.accesosMap = pr.accesosMap || new Map()

      for (const encodedAccess of pr.Accesos) {
        const accessID = Math.floor(encodedAccess / 10)
        const accessLevel = encodedAccess - (accessID * 10)
        pr.accesosMap.has(accessID)
          ? pr.accesosMap.get(accessID)!.push(accessLevel)
          : pr.accesosMap.set(accessID, [accessLevel])
      }

      pr.SubAccesos = pr.SubAccesos || []
      pr.subAccesosMap = unpackSubAccesos(pr.SubAccesos)
    }
    this.perfiles = perfiles
    this.perfilesMap = new Map(perfiles.map(profileRecord => [profileRecord.ID, profileRecord]))
  }

  constructor() {
    super()
    this.fetch()
  }

  updatePerfil(perfil: IProfile) {
    const existing = this.perfiles.find(profileRecord => profileRecord.ID === perfil.ID)
    if (existing) {
      Object.assign(existing, perfil)
    } else {
      this.perfiles.unshift(perfil)
    }
    this.perfilesMap.set(perfil.ID, perfil)
  }

  removePerfil(id: number) {
    this.perfiles = this.perfiles.filter(profileRecord => profileRecord.ID !== id)
    this.perfilesMap.delete(id)
  }
}

export const postPerfil = (data: IProfile) => {
  // The two maps are the editable form shape; the backend only takes the packed Accesos and
  // SubAccesos arrays.
  const dataToSend = { ...data }
  delete (dataToSend as any).accesosMap
  delete (dataToSend as any).subAccesosMap

  return POST({
    data: dataToSend,
    route: "perfiles",
    refreshRoutes: ["perfiles"]
  })
}
