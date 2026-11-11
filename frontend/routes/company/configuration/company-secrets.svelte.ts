import { GetHandler, POST } from '#libs/ui-runtime.svelte.ts';
import type { ICompanySecrets, ICompanySecretsForm } from './company-secrets';

// Every row, active and retired, synced by delta. min: 0 syncs on every load after painting the
// cached rows: a stale certificate state is the one thing this panel must never show, since it
// decides whether the company can invoice.
export class CompanySecretsService extends GetHandler<ICompanySecrets> {
  route = "company-secrets"
  useCache = { min: 0, ver: 1 }

  all: ICompanySecrets[] = $state([])

  handler(response: ICompanySecrets[]) {
    this.all = response || []
  }

  constructor() {
    super()
    this.fetch()
  }

  // The reply is the saved row as the panel may see it — never the certificate or the passwords.
  // A rotation also retires the previous row, so the list is brought up to date from the delta
  // that comes back with the write rather than patched with the reply.
  save(form: ICompanySecretsForm): Promise<ICompanySecrets> {
    return this.writeAndSync<ICompanySecrets>("company-secrets", form)
  }
}

// Asks SUNAT for the CDR of a document that does not exist: an answer of "no record"
// means the credentials were accepted, which is the whole point. Nothing is emitted.
export async function testCompanySecrets(): Promise<{ Ok: boolean, Code: string, Message: string }> {
  return await POST({
    data: {},
    route: "company-secrets-test",
  })
}
