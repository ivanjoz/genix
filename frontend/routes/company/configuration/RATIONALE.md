## My Company split into two cards: Company Parameters and Culqi Configuration

**Context** — `CompanyTab.svelte` was one flat 24-column grid: eight company fields in 14/10 pairs,
then an `Ecommerce` sub-heading and six Culqi fields, all on the same background. Two unrelated
concerns read as one long form, and the Culqi block looked like a continuation of the legal data
rather than payment-gateway credentials.

**Decision** — Two `<section>` cards side by side (`lg:col-span-12`, stacking below `lg`), each
`rounded-[12px] border border-slate-200 bg-white p-16 shadow-sm` — the card style already used by
`system/server-panel`, `system/observability` and the sales charts. The left card keeps the
`Company Parameters|Parámetros de la Empresa` heading and its eight fields; the right card is
titled `Culqi Configuration|Configuración Culqi`, replacing the `Ecommerce` sub-heading. The page
header row now holds only the Save button, right-aligned, because the title moved into the card.

**Rationale** — Cards over a single grid with sub-headings because the two groups have different
audiences: one is filled once by whoever registers the company, the other only by a tenant that
actually sells online. Inside each half-width card the fields go full width (RUC and Phone are the
one 12/12 pair), since a 14/10 split inside half a viewport left labels clipping. Both cards share
one Save button and one request — splitting them would need two endpoints, and the backend writes
the company record whole. The cost is that a tenant with no online store still sees six empty
Culqi fields taking half the page.

## ICompany realigned to types.Company, and Email dropped from the required set

**Context** — `/company/configuration` refused every save with "Faltan datos a guardar." on a
seeded company, and ten of its fields were dead. `ICompany` still carried the Spanish field names
(`Telefono`, `Direccion`, `Ciudad`, `Representante`, `CulquiConfig`, `Llave*`) from before
`types.Company` was renamed to English, and the POST body is unmarshalled straight into that
struct — so those ten values were silently discarded on save and always reloaded blank, with each
save clearing whatever was stored. The refusal itself was a separate problem: `Email` was in the
required set, company 1 has none, and the message named no field, so the form showed a red toast
with no invalid input to point at (`Input` only paints its red state after a blur).

**Decision** — `ICompany`/`ICompanyCulqi` now mirror `config/types/empresas.go` exactly (`Phone`,
`Address`, `City`, `Representative`, `CulqiConfig`, `PubKeyDev`/`KeyDev`/`PubKeyLive`/`KeyLive`,
and `ID` rather than `id`), with `CompanyTab.svelte`'s `save=` props updated to match. `Email` is
no longer required, in the form or in `PostEmpresaParametros`. The remaining three required fields
live in a `requiredFields` list paired with their labels, so a refusal names the empty ones:
"Faltan datos a guardar: RUC, Razón Social".

**Rationale** — Aligning the interface to the Go struct rather than translating in `handler()`
keeps one spelling of each field across the wire and makes the drift greppable next time the
struct changes; the cost is that the interface is now coupled to Go's JSON names, which is what the
POST body already required. Email is optional because seeded and imported companies have none and
the global email index on `companies` tolerates a blank value — public sign-up sets it on its own
path, so requiring it here only blocked admins from editing anything else. Naming the missing
fields was chosen over forcing `Input`'s red state, which would need a new prop in the `genix-ui`
submodule for a message the toast can carry on its own.

## Parameters and Backups merged into a two-tab route at /company/configuration

**Context** — `/configuration/parameters` (My Company) and `/configuration/backups` were two menu
options under the **Mi Empresa** menu group, each one page with a single job. Both are
company-level settings a tenant admin visits rarely and together, and `configuration/` existed as
a route domain holding nothing else.

**Decision** — One route, `frontend/routes/company/configuration`, replacing the `configuration/`
domain folder entirely. `+page.svelte` is the shell: an `OptionsStrip` switches between
`CompanyTab.svelte` (the parameters form) and `BackupsTab.svelte` (the backup list, download and
restore). The `Backups` menu option is gone; the surviving `Configuration|Configuración` option
points at the new route and its description mentions backups. `empresas.svelte.ts` and
`backups.svelte.ts` moved unchanged except for the two access-id constants the shell reads.

**Rationale** — The route path was chosen over the alternatives (`configuration/company`, keeping
`configuration/parameters`) because the feature is the company's configuration, and `company/` is
now the domain that owns it. `OptionsStrip` is used directly in the page body instead of `Page`'s
`options` strip: this page has exactly two views and no need for the header strip's cross-session
persistence, and the request was for the in-page component. The cost is that the selected tab is
not remembered between visits, unlike routes that use `Page options`.

## Both access ids kept; the tab, not the route, is what gets gated

**Context** — `access.toml` id 1 (`Configuración`, the company-parameters APIs) and id 4
(`Backups`, `POST.backup-create` / `POST.backup-restore`) were separate accesses on separate
routes. `canAccessRoute` grants a route if **any** access mapped to it passes, so pointing both ids
at one route would let a user holding only `Backups` open the company-parameters form.

**Decision** — Both ids now list `frontend_routes: "company/configuration"` and both are kept. The
shell filters its own strip with `security.checkAcceso(CONFIGURATION_ACCESS_ID, 1)` and
`security.checkAcceso(BACKUPS_ACCESS_ID, 1)`, so a user is only offered the tabs they can read and
lands on the first one that survives the filter. `backend_apis` stay split, so the server-side
check is unchanged.

**Rationale** — Same reasoning as `security/users-profiles`: merging the two ids would silently
promote every profile holding only one of them, since accesses are persisted in `profiles.accesos`
as `id*10 + level`. Per-tab gating keeps stored permissions meaning exactly what they meant before.
The cost is that route-level access is coarser than what the user sees — the tab strip is the real
boundary in the UI, while the backend still enforces each API separately.

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
