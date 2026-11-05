import { GetHandler, POST } from '#libs/ui-runtime.svelte.ts'

// Mirrors backend/hr/types/legal_parameters.go. Shared by every company of the country; only the
// SaaS company writes them (System → Parámetros Laborales).

export interface ILegalParameter {
  ID: number
  Key: number
  /** UnixDay the value takes effect. */
  ValidFrom: number
  /** Cents or basis points, per the key's unit (LEGAL_PARAMETER_OPTIONS). */
  Value: number
  ss: number
  upd: number
}

export interface IAfpRate {
  ID: number
  /** YYMM. */
  Period: number
  PensionSystem: number
  /** Basis points. */
  FundRate: number
  PremiumRate: number
  FlowCommission: number
  MixedCommission: number
  ss: number
  upd: number
}

interface ILegalParametersResponse {
  Parameters?: ILegalParameter[]
  AfpRates?: IAfpRate[]
}

/** Legal values and AFP rates in one response (`hr-legal-parameters`), one cursor per key. */
export class LegalParametersService extends GetHandler {
  route = 'hr-legal-parameters'
  useCache = { min: 5, ver: 1 }

  Parameters: ILegalParameter[] = $state([])
  AfpRates: IAfpRate[] = $state([])

  handler(result: ILegalParametersResponse): void {
    this.Parameters = (result.Parameters || [])
      .filter((parameter) => parameter.ss > 0)
      .sort((left, right) => left.Key - right.Key || right.ValidFrom - left.ValidFrom)
    this.AfpRates = (result.AfpRates || [])
      .filter((afpRate) => afpRate.ss > 0)
      .sort((left, right) => right.Period - left.Period || left.PensionSystem - right.PensionSystem)
  }

  constructor(init = false) {
    super()
    if (init) this.fetch()
  }
}

export const postLegalParameter = (parameter: ILegalParameter): Promise<ILegalParameter> =>
  POST({ data: parameter, route: 'hr-legal-parameters', refreshRoutes: ['hr-legal-parameters'] })

export const postAfpRates = (afpRates: IAfpRate[]): Promise<IAfpRate[]> =>
  POST({ data: afpRates, route: 'hr-afp-rates', refreshRoutes: ['hr-legal-parameters'] })
