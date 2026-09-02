import { GetHandler, POST } from '$libs/ui-runtime.svelte';

// Access id from backend/access_list.yml. It gates the "My Company" tab of this route, whose
// other tab (Backups) carries its own id.
export const CONFIGURATION_ACCESS_ID = 1

// Field names mirror config/types/empresas.go exactly: the POST body is unmarshalled straight
// into types.Company, so any spelling drift here is silently dropped instead of erroring.
export interface ICompanyCulqi {
  RsaKey: string
  RsaKeyID: string
  KeyLive: string
  PubKeyLive: string
  KeyDev: string
  PubKeyDev: string
}

export interface ICompany {
  ID: number
  Email: string
  Name: string
  LegalName: string
  RUC: string
  Phone: string
  Representative: string
  Address: string
  City: string
  CulqiConfig: ICompanyCulqi
  ss: number
  upd: number
}

export class EmpresaParametrosService extends GetHandler {
    route = "company-parametros"
    useCache = { min: 10, ver: 1 }

    empresa = $state({
        CulqiConfig: {}
    } as ICompany)

    handler(response: any) {
      const record = (response[0] || {}) as ICompany
      // The backend omits CulqiConfig when empty, but the Ecommerce inputs bind into it directly.
      record.CulqiConfig = record.CulqiConfig || {} as ICompanyCulqi
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
