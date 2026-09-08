import { describe, expect, test } from 'bun:test';
import {
  activeSecrets, certificateDaysLeft, retiredSecrets, SUNAT_ENV_BETA, validateSecretsForm,
  type ICompanySecrets, type ICompanySecretsForm,
} from './company-secrets';

const makeSecrets = (fields: Partial<ICompanySecrets>): ICompanySecrets => ({
  ID: 1, Name: "", SolUser: "MODDATOS", Environment: SUNAT_ENV_BETA,
  CertSubject: "", CertIssuer: "", CertRUC: "20100000001",
  CertValidFrom: 0, CertValidTo: 0, HasCert: true, ss: 1, upd: 0,
  ...fields,
})

const makeForm = (fields: Partial<ICompanySecretsForm>): ICompanySecretsForm => ({
  ID: 0, SolUser: "MODDATOS", SolPassword: "clave", Certificate: "base64",
  CertPassword: "clave", Environment: SUNAT_ENV_BETA,
  ...fields,
})

describe('activeSecrets', () => {
  test('takes the active row of the highest id, the one emission signs with', () => {
    const all = [
      makeSecrets({ ID: 1, ss: 0 }),
      makeSecrets({ ID: 2, ss: 1 }),
      makeSecrets({ ID: 3, ss: 1 }),
    ]
    expect(activeSecrets(all)?.ID).toBe(3)
  })

  test('is undefined when every row was retired', () => {
    expect(activeSecrets([makeSecrets({ ID: 1, ss: 0 })])).toBeUndefined()
  })
})

test('retiredSecrets lists what a rotation left behind, newest first', () => {
  const all = [makeSecrets({ ID: 1, ss: 0 }), makeSecrets({ ID: 2, ss: 0 }), makeSecrets({ ID: 3, ss: 1 })]
  expect(retiredSecrets(all).map(secrets => secrets.ID)).toEqual([2, 1])
})

describe('certificateDaysLeft', () => {
  // SUnixTime is (unix - 1e9) / 2, which is what the column holds.
  const asSunix = (unixSeconds: number) => (unixSeconds - 1e9) / 2
  const now = 1_800_000_000

  test('counts the days that remain', () => {
    const secrets = makeSecrets({ CertValidTo: asSunix(now + 10 * 86400) })
    expect(certificateDaysLeft(secrets, now)).toBe(10)
  })

  test('goes negative once the certificate expired', () => {
    const secrets = makeSecrets({ CertValidTo: asSunix(now - 3 * 86400) })
    expect(certificateDaysLeft(secrets, now)).toBe(-3)
  })
})

describe('validateSecretsForm', () => {
  test('accepts a complete first upload', () => {
    expect(validateSecretsForm(makeForm({}), undefined)).toBeUndefined()
  })

  test('refuses credentials with no SOL user', () => {
    expect(validateSecretsForm(makeForm({ SolUser: "  " }), undefined)).toContain("SOL")
  })

  test('refuses a first record with no SOL password, which could never authenticate', () => {
    expect(validateSecretsForm(makeForm({ SolPassword: "" }), undefined)).toBeDefined()
  })

  test('refuses a first record with no certificate, which could never sign', () => {
    expect(validateSecretsForm(makeForm({ Certificate: "" }), undefined)).toBeDefined()
  })

  test('leaves both passwords and the certificate optional once a record exists', () => {
    const stored = makeSecrets({ HasCert: true })
    const form = makeForm({ ID: stored.ID, SolPassword: "", CertPassword: "", Certificate: "" })
    expect(validateSecretsForm(form, stored)).toBeUndefined()
  })

  test('still demands a certificate when the stored record has none', () => {
    const stored = makeSecrets({ HasCert: false })
    const form = makeForm({ ID: stored.ID, SolPassword: "", Certificate: "" })
    expect(validateSecretsForm(form, stored)).toBeDefined()
  })
})
