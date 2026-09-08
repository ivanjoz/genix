import { GetHandler, POST } from '$libs/ui-runtime.svelte';
import { Notify } from '$libs/helpers';
import { tr } from '$core/store.svelte';
import type { IInvoiceSeries } from './invoice-series';
import pkg from 'notiflix'
const { Loading } = pkg;

// Access id from backend/access.toml. It gates the "My Company" tab of this route, whose
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
  // The SUNAT series travel inline on the company: there are at most 99 and every
  // emission needs one, so they are not worth a table of their own.
  InvoiceSeries: IInvoiceSeries[]
  ss: number
  upd: number
}

export class EmpresaParametrosService extends GetHandler {
    route = "company-parametros"
    // ver 2: the record gained InvoiceSeries. A copy cached before that field existed
    // would show a company with no series while the server has six, which reads as
    // configuration having been lost.
    useCache = { min: 10, ver: 2 }

    empresa = $state({
        CulqiConfig: {},
        InvoiceSeries: []
    } as unknown as ICompany)

    handler(response: any) {
      const record = (response[0] || {}) as ICompany
      // The backend omits CulqiConfig when empty, but the Ecommerce inputs bind into it directly.
      record.CulqiConfig = record.CulqiConfig || {} as ICompanyCulqi
      // Omitted when the company has none, and the series table binds into it.
      record.InvoiceSeries = record.InvoiceSeries || []
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

// Email is deliberately absent: seeded and imported companies carry none, and the global email
// index on the companies table tolerates a blank value. PostEmpresaParametros enforces the same
// three. Labels are repeated here so a refusal can name the field the user has to go fix.
const requiredFields: [keyof ICompany, string][] = [
  ["Name", "Name|Nombre"],
  ["RUC", "RUC"],
  ["LegalName", "Legal Name|Razón Social"],
]

// Shared by the My Company and Store tabs: both edit one company record and one endpoint writes it
// whole, so saving from the Store tab must still carry (and therefore validate) the legal fields.
export async function saveCompanyParameters(company: ICompany) {
  const missing = requiredFields
    .filter(([field]) => ((company[field] as string) || "").trim().length === 0)
    .map(([, label]) => tr(label))

  if (missing.length > 0) {
    Notify.failure(`${tr("Missing required data:|Faltan datos a guardar:")} ${missing.join(", ")}`)
    return
  }

  Loading.standard(tr("Saving...|Guardando..."))
  try {
    await postEmpresaParametros(company)
    Notify.success(tr("Data saved successfully|Datos guardados correctamente"))
  } catch (error) {
    // Error handled by POST
  }
  Loading.remove()
}
