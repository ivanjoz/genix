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
  id: number
  nombre: string
  descripcion?: string
  accesos: number[]
  modulosIDs: number[]
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
  { id: 4, name: "Eliminar", short: "ELIM",
    icon: "icon-trash", color: "#ff5b5b" },
  { id: 7, name: "Todo", short: "EDITAR",
    icon: "icon-shield", color: "#af12eb", color2: "#d35eff" },
]

export class PerfilesService extends GetHandler {
  route = "perfiles"
  useCache = { min: 5, ver: 1 }

  perfiles: IPerfil[] = $state([])
  perfilesMap: Map<number, IPerfil> = $state(new Map())

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
    this.perfilesMap = new Map(perfiles.map(x => [x.id, x]))
  }

  constructor() {
    super()
    this.fetch()
  }

  updatePerfil(perfil: IPerfil) {
    const existing = this.perfiles.find(x => x.id === perfil.id)
    if (existing) {
      Object.assign(existing, perfil)
    } else {
      this.perfiles.unshift(perfil)
    }
    this.perfilesMap.set(perfil.id, perfil)
  }

  removePerfil(id: number) {
    this.perfiles = this.perfiles.filter(x => x.id !== id)
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
