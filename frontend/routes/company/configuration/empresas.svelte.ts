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
  // The district of the fiscal address, which is the INEI ubigeo itself. It
  // replaced a free-text city: SUNAT validates this code on every document the
  // company issues, and a typed city name is not one.
  CityID: number
  CulqiConfig: ICompanyCulqi
  // The SUNAT series travel inline on the company: there are at most 99 and every
  // emission needs one, so they are not worth a table of their own.
  InvoiceSeries: IInvoiceSeries[]
  // The checked flag ids from backend/company_flags.toml, and only those. Saved as int16
  // on the backend, which the Flags tab never has to know.
  Flags: number[]
  ss: number
  upd: number
}

export class EmpresaParametrosService extends GetHandler {
    route = "company-parametros"
    // ver 2: the record gained InvoiceSeries. A copy cached before that field existed
    // would show a company with no series while the server has six, which reads as
    // configuration having been lost.
    // ver 3: the free-text City became CityID, the district's ubigeo. A cached copy
    // would keep feeding the old string into a selector that expects a number.
    // ver 4: the record gained Flags. A copy cached before it would show every flag
    // unchecked, and saving from that tab would then clear the ones that are on.
    useCache = { min: 10, ver: 4 }

    empresa = $state({
        CulqiConfig: {},
        InvoiceSeries: [],
        Flags: []
    } as unknown as ICompany)

    handler(response: any) {
      const record = (response[0] || {}) as ICompany
      // The backend omits CulqiConfig when empty, but the Ecommerce inputs bind into it directly.
      record.CulqiConfig = record.CulqiConfig || {} as ICompanyCulqi
      // Omitted when the company has none, and the series table binds into it.
      record.InvoiceSeries = record.InvoiceSeries || []
      // Omitted when no flag is checked, which is every company until somebody checks one.
      record.Flags = record.Flags || []
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
