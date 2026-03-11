import { GetHandler } from '$libs/http.svelte';
import type { IUsuario, IPerfil } from '$core/types/common';
export { postUsuario, postUsuarioPropio } from '$services/services/usuarios.svelte';

export type { IUsuario, IPerfil };

export class UsuariosService extends GetHandler {
  route = "usuarios"
  useCache = { min: 0.1, ver: 1 }

  usuarios: IUsuario[] = $state([])
  usuariosMap: Map<number, IUsuario> = $state(new Map())

  handler(response: IUsuario[]) {
    this.usuarios = response || []
    this.usuariosMap = new Map(this.usuarios.map((usuarioRecord) => [usuarioRecord.ID, usuarioRecord]))
  }

  constructor() {
    super()
    this.fetch()
  }

  updateUsuario(usuario: IUsuario) {
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
  // Bump cache version because perfiles now arrive with Go field names instead of lowercase aliases.
  useCache = { min: 5, ver: 2 }

  perfiles: IPerfil[] = $state([])

  handler(response: IPerfil[]) {
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
    }
    this.perfiles = perfiles
  }

  constructor() {
    super()
    this.fetch()
  }
}
