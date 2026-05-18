import { GetHandler, POST } from '$libs/http.svelte';

export interface ICompanySmtp {
  Host: string
  Port: string
  User: string
  Password: string
  Email: string
}

export interface ICompanyCulqui {
  RsaKey: string
  RsaKeyID: string
  LlaveLive: string
  LlaveDev: string
  LlavePubLive: string
  LlavePubDev: string
}

export interface ICompany {
  id: number
  Email: string
  Nombre: string
  RazonSocial: string
  RUC: string
  Telefono: string
  Representante: string
  Direccion: string
  Ciudad: string
  SmtpConfig: ICompanySmtp
  CulquiConfig: ICompanyCulqui
  ss: number
  upd: number
}

export class EmpresasService extends GetHandler {
  route = "empresas"
  useCache = { min: 5, ver: 1 }

  empresas: ICompany[] = $state([])
  empresasMap: Map<number, ICompany> = $state(new Map())

  handler(response: any) {
    // Following the original structure: response.Records contains the array
    const empresas = (response?.Records || response || []) as ICompany[]
    
    this.empresas = empresas.map(empresa => {
      empresa.SmtpConfig = empresa.SmtpConfig || {} as ICompanySmtp
      empresa.CulquiConfig = empresa.CulquiConfig || {} as ICompanyCulqui
      return empresa
    })
    
    this.empresasMap = new Map(this.empresas.map(x => [x.id, x]))
  }

  constructor() {
    super()
    this.fetch()
  }

  updateEmpresa(empresa: ICompany) {
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

export const postEmpresa = (data: ICompany) => {
  return POST({
    data,
    route: "empresas",
    refreshRoutes: ["empresas"]
  })
}
