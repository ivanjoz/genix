import { GetHandler, POST } from '$ecommerce/node_modules/@sveltejs/kit/src/utils/http';

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

export class EmpresaParametrosService extends GetHandler {
    route = "empresa-parametros"
    useCache = { min: 10, ver: 1 }
    
    empresa = $state({
        SmtpConfig: {},
        CulquiConfig: {}
    } as IEmpresa)

    handler(response: any) {
      const record = (response[0] || {}) as IEmpresa
      console.log("empresa response::", record)

      record.SmtpConfig = record.SmtpConfig || {} as IEmpresaSmtp
      record.CulquiConfig = record.CulquiConfig || {} as IEmpresaCulqui
      this.empresa = record
    }

    constructor() {
        super()
        this.fetch()
    }
}

export const postEmpresaParametros = (data: IEmpresa) => {
  return POST({
    data,
    route: "empresa-parametros",
    refreshRoutes: ["empresa-parametros"]
  })
}
