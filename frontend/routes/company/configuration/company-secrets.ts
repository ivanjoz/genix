// Mirrors CompanySecretsView in backend/invoicing/company_secrets_api.go. There is no
// field for the certificate or either password: no endpoint returns key material, so
// the panel only ever knows whose certificate it is and until when it works.
export interface ICompanySecrets {
  ID: number
  Name: string
  SolUser: string
  Environment: number
  CertSubject: string
  CertIssuer: string
  CertRUC: string
  CertValidFrom: number
  CertValidTo: number
  HasCert: boolean
  ss: number
  upd: number
}

// Environment values of types.SunatEnv* in the backend.
export const SUNAT_ENV_BETA = 1
export const SUNAT_ENV_PRODUCTION = 2

// What the form sends. The certificate travels base64 and only when a new file was
// picked; the passwords only when they were typed. Anything empty means "keep what is
// stored", which is why nothing here is ever prefilled from the server.
export interface ICompanySecretsForm {
  ID: number
  SolUser: string
  SolPassword: string
  Certificate: string
  CertPassword: string
  Environment: number
}

// The row emission will use: LoadActiveSecrets picks the active one of the highest id,
// and the panel has to show the same one, or it would describe a certificate that is
// not the one signing.
export const activeSecrets = (all: ICompanySecrets[]): ICompanySecrets | undefined =>
  all.filter(secrets => secrets.ss === 1).sort((a, b) => b.ID - a.ID)[0]

// Retired by a rotation. Kept visible because the certificate that signed last year's
// documents is what an audit asks about.
export const retiredSecrets = (all: ICompanySecrets[]): ICompanySecrets[] =>
  all.filter(secrets => secrets.ss !== 1).sort((a, b) => b.ID - a.ID)

// SUnixTime (int32, (unix - 1e9) / 2) back to unix seconds.
const sunixToUnix = (sunixTime: number) => 1e9 + sunixTime * 2

// Days left on the certificate, negative once it is expired. SUNAT rejects every
// document signed with an expired certificate, so this is a countdown to an outage,
// not a detail: the panel warns while there is still time to buy the next one.
export const certificateDaysLeft = (secrets: ICompanySecrets, nowUnixSeconds: number): number =>
  Math.floor((sunixToUnix(secrets.CertValidTo) - nowUnixSeconds) / 86400)

export const EXPIRY_WARNING_DAYS = 30

// Refuses what the endpoint would refuse anyway, plus the two states it accepts and
// emission does not: credentials with no SOL password and credentials with no
// certificate. Both save a row that can never sign.
export function validateSecretsForm(
  form: ICompanySecretsForm, current?: ICompanySecrets,
): string | undefined {
  if (!form.SolUser.trim()) {
    return "Enter the SOL user.|Indique el usuario SOL."
  }
  if (!current && !form.SolPassword) {
    return "Enter the SOL password.|Indique la clave SOL."
  }
  if (!current?.HasCert && !form.Certificate) {
    return "Upload the digital certificate.|Cargue el certificado digital."
  }
  return undefined
}
