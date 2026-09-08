import { GET, POST } from '$libs/ui-runtime.svelte';
import type { ICompanySecrets, ICompanySecretsForm } from './company-secrets';

// Read uncached, unlike the rest of this page. A stale certificate state is the one
// thing this panel must never show: it decides whether the company can invoice, and
// it changes exactly as often as somebody rotates a certificate.
export class CompanySecretsService {
  all: ICompanySecrets[] = $state([])
  isLoading = $state(true)

  constructor() {
    void this.load()
  }

  async load() {
    this.isLoading = true
    try {
      const response = await GET({
        route: "company-secrets",
        errorMessage: "Error al obtener las credenciales SUNAT.",
      })
      this.all = (response?.CompanySecrets || []) as ICompanySecrets[]
    } catch (error) {
      // Reported by GET.
    }
    this.isLoading = false
  }
}

// The reply is the saved row as the panel may see it — never the certificate or the
// passwords. A rotation answers with the new row, so the caller reloads instead of
// patching what it holds: the row this replaced is now retired and has to move too.
export async function postCompanySecrets(form: ICompanySecretsForm): Promise<ICompanySecrets> {
  const response = await POST({
    data: form,
    route: "company-secrets",
  })
  return response as ICompanySecrets
}

// Asks SUNAT for the CDR of a document that does not exist: an answer of "no record"
// means the credentials were accepted, which is the whole point. Nothing is emitted.
export async function testCompanySecrets(): Promise<{ Ok: boolean, Code: string, Message: string }> {
  return await POST({
    data: {},
    route: "company-secrets-test",
  })
}
