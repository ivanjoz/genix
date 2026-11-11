import { GetHandler, POST } from '#libs/ui-runtime.svelte.ts';
import type { IProfile, IUser } from '#core/types/common.ts';
import { fromAccesoGrants, type ISubAccesoOption } from './users-profiles';
export { postUser, postOwnUser } from '#services/services/users.svelte.ts';

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
  ss: number
  upd: number
}

export const accesoAcciones = [
  { id: 1, name: "Visualizar", short: "VER",
    icon: "icon-[mdi--eye]", color: "var(--green-solid)", color2: "color-mix(in oklab, var(--green-solid) 80%, var(--surface))" },
  { id: 2, name: "Crear", short: "CREAR",
    icon: "icon-[fa--pencil]", color: "var(--blue-solid)" },
  { id: 3, name: "Editar", short: "EDITAR",
    icon: "icon-[fa--pencil]", color: "var(--blue-solid)" },
  { id: 4, name: "Todo", short: "TODO",
    icon: "icon-[fa--shield]", color: "var(--purple-solid)", color2: "color-mix(in oklab, var(--purple-solid) 80%, var(--surface))" },
]

export class UsuariosService extends GetHandler {
  route = "users"
  // The active users, synced by delta: only changed users travel, a deactivated one arrives as a
  // removed ID. v3: the route moved to the delta protocol, so a v2 cache has no cursor to resume.
  useCache = { min: 0.1, ver: 3 }

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
  // The active profiles, synced by delta: only changed profiles travel, a deactivated one arrives
  // as a removed ID. v5: the route moved to the delta protocol, so a v4 cache has no cursor to resume.
  useCache = { min: 5, ver: 5 }

  perfiles: IProfile[] = $state([])
  perfilesMap: Map<number, IProfile> = $state(new Map())

  handler(response: IProfile[]) {
    const perfiles = response || []
    for (const pr of perfiles) {
      pr.AccesosGrants = pr.AccesosGrants || []
      const { accesosMap, subAccesosMap } = fromAccesoGrants(pr.AccesosGrants)
      pr.accesosMap = accesosMap
      pr.subAccesosMap = subAccesosMap
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
  // The two maps are the editable form shape; the backend only takes the AccesosGrants list, which
  // the caller has already built from them.
  const dataToSend = { ...data }
  delete (dataToSend as any).accesosMap
  delete (dataToSend as any).subAccesosMap

  return POST({
    data: dataToSend,
    route: "perfiles",
    refreshRoutes: ["perfiles"]
  })
}
