## The company config blob is split across `config/types` and `cloud`

**Context** — `CompactAndStoreCompanyConfig` has to read three tables and write to object storage,
and both `config` and `invoicing` call it. A module body may not import another module body, so it
cannot live in either. The obvious home — `config/types`, next to the record — is illegal too: L2
may import "L0, L1, other `<module>/types` — and nothing else, ever", and `cloud` is L3.

**Decision** — The record and `AssembleCompanyConfig` (pure, takes rows already read) live in
`config/types/company_config.go`. The reads, colbin, AES and S3 live in `cloud/company_config_blob.go`.
`cloud` is the first non-test file in that package to import a module's types.

**Rationale** — It is the only split the layer table permits, and it happens to be the one that
makes the assembly unit-testable without a database or a bucket. The cost is that the two halves of
one feature are in different packages; `fd company_config` finds both, which is why the filenames
match.

## Nothing the ORM already persists gets a numeric `cb` id

**Context** — The blob's own types carry explicit `cb:"1"`-style ids so a field rename cannot move
an id and every type stays inside colbin's 4-bit id window. The obvious next step is to tag
`InvoiceSeries` and `CulqiConfig` too, since the blob embeds them.

**Decision** — Neither is tagged. `CompanyConfigCulqi` copies the four Culqi fields the backend
uses instead of embedding `CulqiConfig`; `InvoiceSeries` is embedded as it is, with hashed ids.

**Rationale** — `genix-orm/scylla/converter.go:784` colbin-marshals any non-`[]byte` struct column,
so both of those are already on disk encoded under hashed ids. Giving them explicit ones would
reassign every field and make every stored company row undecodable — a silent data loss with no
error to catch it. The cost is that `InvoiceSeries` keeps hashed ids inside the blob, so renaming
one of its fields shifts them; that rename already breaks the database column today, so the blob
adds no new fragility.

## The blob is sealed with the object name as AAD, and a failed upload never fails the save

**Context** — Two questions the format left open: what binds a blob to the company it belongs to,
and what happens when the upload fails after the user's save already committed.

**Decision** — The GCM additional authenticated data is the object key string, so a blob moved onto
another company's key fails to open rather than decrypting. `StoreCompanyConfigAsync` logs and
swallows every error; no call site can fail on it.

**Rationale** — Without the AAD binding, an attacker able to copy objects within the bucket could
give a company somebody else's SOL credentials, and nothing in the payload would object. On the
write path, the alternative — failing the request — would report a save that actually happened as
having failed; the blob self-heals on the next save or the first read, so the error is recoverable
and the lie would not be.

## The per-company SMTP panel is removed; outgoing mail stays platform-wide

**Context** — The Parameters page showed an "SMTP parameters for notifications" card writing
`Company.SmtpConfig` (Host / Port / User / Password / Email). Nothing ever read it: every caller of
`core.SendEmail` — sign-up verification, the contact form — takes its credentials from
`core.Env.SMTP_*`, loaded from the `[smtp]` section of `config.toml`. The panel had also never
worked end to end: the Go struct field was named `Post` while the form bound `save="Port"`, so the
port never even deserialized. Worse, the record is served by the standard `company-parametros`
delta-cache endpoint, so a saved SMTP password shipped in plaintext to any client allowed to read
company parameters.

**Decision** — Deleted the panel, the `ICompanySmtp` interface in both `configuration/parameters`
and `system/companies`, and the `SmtpConfig` field and type from `config/types/empresas.go`. The
two-column page grid collapses to the single form grid, since the right column existed only to hold
that card.

**Rationale** — A control that stores a secret it never uses is worse than no control: it invites an
operator to enter real mail credentials, then leaks them over an endpoint whose read permission is
much broader than "may configure mail". Wiring it up for real is the alternative, but per-tenant
SMTP needs a write-only password field, a fallback chain to the global config and a send-test path —
work that only pays off once tenants actually want mail from their own domain. Cost: the Scylla
column `smtp_config` is now orphaned. Pre-alpha, so it is left in place rather than migrated; the
ORM only reads declared columns and `check_tables` passes.
