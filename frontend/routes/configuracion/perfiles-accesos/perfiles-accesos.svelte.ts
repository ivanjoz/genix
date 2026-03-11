import { GetHandler, POST } from '$libs/http.svelte';

export interface IAcceso {
  id: number
  nombre: string
  descripcion?: string
  orden: number
  acciones: number[]
  grupo: number
  modulosIDs: number[]
  ss: number
  upd: number
}

export interface IPerfil {
  ID: number
  EmpresaID: number
  Nombre: string
  Descripcion?: string
  Accesos: number[]
  Modulos: number[]
  accesosMap: Map<number, number[]>
  ss: number
  upd: number
  _open?: boolean
}

export const accesoAcciones = [
  { id: 1, name: "Visualizar", short: "VER",
    icon: "icon-eye", color: "#00c07d", color2: "#49c99c" },
  { id: 2, name: "Crear", short: "CREAR",
    icon: "icon-pencil", color: "#0080f9" },
  { id: 3, name: "Editar", short: "EDITAR",
    icon: "icon-pencil", color: "#0f6bff" },
  { id: 4, name: "Todo", short: "TODO",
    icon: "icon-shield", color: "#af12eb", color2: "#d35eff" },
]

export class PerfilesService extends GetHandler {
  route = "perfiles"
  // Bump cache version because perfiles now arrive with Go field names instead of lowercase aliases.
  useCache = { min: 5, ver: 2 }

  perfiles: IPerfil[] = $state([])
  perfilesMap: Map<number, IPerfil> = $state(new Map())

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
    this.perfilesMap = new Map(perfiles.map(profileRecord => [profileRecord.ID, profileRecord]))
  }

  constructor() {
    super()
    this.fetch()
  }

  updatePerfil(perfil: IPerfil) {
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

export const postPerfil = (data: IPerfil) => {
  // Convert accesosMap to accesos array before sending
  const dataToSend = { ...data }
  delete (dataToSend as any).accesosMap
  delete (dataToSend as any)._open

  return POST({
    data: dataToSend,
    route: "perfiles",
    refreshRoutes: ["perfiles"]
  })
}
