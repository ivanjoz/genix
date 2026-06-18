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
  Name: string
  LegalName: string
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

export class EmpresaParametrosService extends GetHandler {
    route = "company-parametros"
    useCache = { min: 10, ver: 1 }

    empresa = $state({
        SmtpConfig: {},
        CulquiConfig: {}
    } as ICompany)

    handler(response: any) {
      const record = (response[0] || {}) as ICompany
      console.log("empresa response::", record)

      record.SmtpConfig = record.SmtpConfig || {} as ICompanySmtp
      record.CulquiConfig = record.CulquiConfig || {} as ICompanyCulqui
      this.empresa = record
    }

    constructor() {
        super()
        this.fetch()
    }
}

export const postEmpresaParametros = (data: ICompany) => {
  return POST({
    data,
    route: "company-parametros",
    refreshRoutes: ["company-parametros"]
  })
}
