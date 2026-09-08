## The secrets form object is replaced, not mutated, so `Input` re-reads it

**Context** — A stored SOL user came back blank in the panel: the row held `MODDATOS`, the GET
returned it, and the field still rendered empty. `Input` re-reads `saveOn` only when the object
*identity* changes — its effect returns early while `lastSaveOn === saveOn`
(`packages/genix-ui/form/Input.svelte:155`). The panel held one `form` object for the component's
whole life and assigned `form.SolUser` once the GET resolved, which is after the inputs mounted, so
they kept the empty value they started with.

**Decision** — The sync effect assigns a whole new `form` object instead of setting three fields on
the existing one. `Input` was left alone.

**Rationale** — The identity guard is not a bug to fix: without it the effect would re-run on every
keystroke, since typing writes back into `saveOn[save]`. Replacing the object is the mechanism that
guard was built to allow, and it fixes this at the call site instead of changing a component every
form in the app renders. The cost is that the effect now also clears `SolPassword` and
`CertPassword`; it only fires when the edited row's id changes — first load, or after a rotation —
and both are moments the passwords are meant to be empty anyway.

## The credentials panel refuses what the endpoint accepts, and reads uncached

**Context** — `POST company-secrets` only requires the SOL user. It will happily store credentials
with no password and no certificate, and the panel could have mirrored that. The page around it
reads through the delta cache, so the credentials would naturally have been read the same way.

**Decision** — `validateSecretsForm` refuses to create a record without a SOL password *and*
without a certificate, and `CompanySecretsService` reads `company-secrets` with a plain `GET`, no
`useCache`. Passwords are never prefilled: an empty one means "keep the stored one", which is what
the endpoint does with it.

**Rationale** — a credentials row missing either piece cannot sign or authenticate, so accepting it
only moves the failure to the first real sale, where it surfaces as a SUNAT rejection at the till
instead of a message on the form that created it. On existing records both stay optional, because
the stored ones are still there. The uncached read costs one request per visit to a page nobody
opens often, and buys the one thing this panel cannot get wrong: showing a certificate that was
rotated or expired in another tab as if it were still the one signing.

## The certificate panel lives in the invoicing column

**Context** — `CompanyTab` was a two-column grid, company form on the left and series on the right.
A third `col-span-12` section would have flowed into the left column, under the company form.

**Decision** — the right half is now a `flex` column holding the series and the certificate, and
`InvoiceSeriesTable` lost the `col-span` it used to carry (its parent places it now).

**Rationale** — the certificate signs the series, and the two are read together when somebody sets
up invoicing; the company's legal fields are a different job. The cost is that `InvoiceSeriesTable`
is no longer droppable straight into a 24-column grid — it needs a parent that positions it.

## The series maintainer is a form plus a list, because a cell cannot hold a select

**Context** — The request was a `TableGrid` maintainer for the invoicing series, inline on the
company record. Two of the five fields are choices — the document type and the branch — and the
obvious shape is a grid whose rows are edited in place.

**Decision** — `InvoiceSeriesTable.svelte` is an add form (type, code, branch) above a list. The grid
edits nothing: it shows the series, moves the default on a cell click, and retires a row. `required`
is deliberately **not** set on the three add fields.

**Rationale** — `CellInput` only renders `text` and `number`. `ITableColumn` declares `cellOptions`
and `onCellSelect`, but `TableGrid` never forwards them, so a select cell renders as a free-text box
that writes a site name where a `SiteID` belongs. Extending the shared component was out of scope for
a small maintainer, and typing a branch by hand is worse than picking it from a list. The cost is that
correcting a series means retiring it and adding it again — acceptable while a company has a handful,
and the alternative is editing the component library.

`required` came off after seeing it rendered: `SearchSelect` paints a red warning triangle the moment
a required field is empty, so a form nobody had touched yet showed two errors. Adding validates all
three anyway and names the one that is missing, which says more than a glyph does.

## Adding a series does not save it

**Context** — The series live on the company record, and the tab already has one Save button.

**Decision** — `Agregar` appends to `company.InvoiceSeries` in memory. Nothing reaches the backend
until the tab's existing Save is pressed. The panel says so in its subtitle.

**Rationale** — One record, one save. A separate write would mean two save paths for one row and a
half-saved tab whenever the second failed, and `POST company-parametros` writes the company whole —
so a series-only save would have to send every other field along regardless. The cost is that leaving
the tab loses unsaved series with no warning, which is the same behaviour every other field on it
already has.

## The branch list is fetched here rather than reused

**Context** — The site list already exists in `WarehousesService`, in the branches-warehouses route.

**Decision** — `invoice-series.svelte.ts` declares its own two-field `IInvoiceSeriesSite` and reads
`locations-warehouses` itself.

**Rationale** — Routes are leaves; importing one route's service into another is the dependency the
architecture rules out, and it would have pulled in warehouses and their layouts to render a
dropdown of names. The endpoint is cached, so the second reader costs nothing. The duplication is
two fields of a shape this panel already only partly uses.

## Culqi moved to a Store tab, gated on the same access, fed by one shared service

**Context** — The request was a third tab, `Store|Tienda`, holding the Culqi configuration split
into a Pruebas section and a Live section. Three things it left open: which access gates the new
tab, how a second tab editing the same record gets its data, and where the single RSA pair goes
when the other credentials are split by environment.

**Decision** — `StoreTab.svelte` is gated on `CONFIGURATION_ACCESS_ID`, the same id as My Company,
so no new entry in `backend/access.toml`. `EmpresaParametrosService` is now instantiated once in
`+page.svelte` and passed to both tabs as a prop; `saveEmpresa` moved out of `CompanyTab.svelte`
into `empresas.svelte.ts` as `saveCompanyParameters(company)`, and both tabs' Save buttons call it.
Inside the Culqi card, `Culqi Pruebas` (amber) and `Culqi Live` (emerald) are two boxes side by
side holding a public/private key each, with `RsaKeyID`/`RsaKey` in a third, neutral box below
labelled `Encriptación RSA` and marked optional. The RSA key ID sits on its own grid row rather
than beside the textarea, because a field bottom-aligns in its grid cell by design
(`field-shell`'s `margin-top: auto`) and would otherwise drop to the textarea's baseline.

**Rationale** — One access rather than a new id: the tab writes the same company record through
`POST company-parametros`, so a separate id would claim a boundary the backend does not enforce,
and every profile holding `Configuración` would silently lose the Culqi form until regranted. One
service instance rather than two because the endpoint writes the record **whole**: two instances
would each hold their own copy and whichever tab saved last would revert the other tab's edits. The
cost is that the Store tab's Save validates Name/RUC/Razón Social — fields that live on the other
tab — and refuses naming them; the alternative, a Culqi-only endpoint, is a backend change the
request did not ask for. The RSA pair is outside both environment boxes because the record stores
one pair for both, and duplicating it into each would suggest two.

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
