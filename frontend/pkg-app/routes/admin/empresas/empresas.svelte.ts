import { GetHandler, POST } from '$core/lib/http';

export interface IEmpresaSmtp {
  Host: string
  Port: string
  User: string
  Password: string
  Email: string
}

export interface IEmpresaCulqui {
  RsaKey: string
  RsaKeyID: string
  LlaveLive: string
  LlaveDev: string
  LlavePubLive: string
  LlavePubDev: string
}

export interface IEmpresa {
  id: number
  Email: string
  Nombre: string
  RazonSocial: string
  RUC: string
  Telefono: string
  Representante: string
  Direccion: string
  Ciudad: string
  SmtpConfig: IEmpresaSmtp
  CulquiConfig: IEmpresaCulqui
  ss: number
  upd: number
}

export class EmpresasService extends GetHandler {
  route = "empresas"
  useCache = { min: 5, ver: 1 }

  empresas: IEmpresa[] = $state([])
  empresasMap: Map<number, IEmpresa> = $state(new Map())

  handler(response: any) {
    // Following the original structure: response.Records contains the array
    const empresas = (response?.Records || response || []) as IEmpresa[]
    
    this.empresas = empresas.map(empresa => {
      empresa.SmtpConfig = empresa.SmtpConfig || {} as IEmpresaSmtp
      empresa.CulquiConfig = empresa.CulquiConfig || {} as IEmpresaCulqui
      return empresa
    })
    
    this.empresasMap = new Map(this.empresas.map(x => [x.id, x]))
  }

  constructor() {
    super()
    this.fetch()
  }

  updateEmpresa(empresa: IEmpresa) {
    const existing = this.empresas.find(x => x.id === empresa.id)
    if (existing) {
      Object.assign(existing, empresa)
    } else {
      this.empresas.unshift(empresa)
    }
    this.empresasMap.set(empresa.id, empresa)
  }

  removeEmpresa(id: number) {
    this.empresas = this.empresas.filter(x => x.id !== id)
    this.empresasMap.delete(id)
  }
}

export const postEmpresa = (data: IEmpresa) => {
  return POST({
    data,
    route: "empresas",
    refreshRoutes: ["empresas"]
  })
}

