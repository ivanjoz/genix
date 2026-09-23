# RATIONALE — crm

Design decisions for clients and providers, newest first.

## The short form of a document type lives in the Go catalog

**Context** — The till's picker sits *inside* the document-number input, so the two names that do
not fit — carné de extranjería and pasaporte — had to shorten. The obvious place to write the
abbreviation was the frontend copy of the list, where the picker is.

**Decision** — `IdentityDocOption` carries a `Label` field, bilingual like `Name` and set only on
the entries that need one (`4` → `F.ID|C.E.`, `7` → `PSP`). `sync_struct_interfaces` copies it
across with the rest of the row; nothing else reads it.

**Rationale** — The frontend array is generated: an abbreviation hand-written there is erased by
the next sync, silently. That makes "put it in the catalog" the only durable option, and the
catalog is already the single place the ids, codes and names are written down. The cost is a
purely presentational field in a backend type — it never reaches SUNAT, a book or the DB. The
remaining long names (sin RUC, cédula diplomática, doc. país de residencia) were left unlabelled:
they only appear in the CRM maintainer's wider field, and inventing Spanish abbreviations for them
is a copy decision, not a layout one.

## `IdentityDocType` id `9` carries SUNAT's code `"0"`

**Context** — The column went from SUNAT's Anexo 1 character to an `int8` so the UI could key on
it like every other option list. SUNAT's catalog includes a real code `"0"` (doc. tributario no
domiciliado sin RUC), but Go's `0` is also the zero value that `json:",omitempty"` drops and that
the derive-from-shape fallback keys on. One of the two meanings had to move.

**Decision** — `0` means *not declared*. SUNAT `"0"` became `IdentityDocNone = 9`. Every other id
is the SUNAT digit itself (1, 4, 6, 7) continuing through hex for `A`–`F` (10–15), so a raw DB
dump still reads as the code it prints for all but that one entry.

**Rationale** — The alternative was `IdentityDocNone = 0` with `-1` for undeclared, which keeps
the codes tidy and makes every other line worse: `-1` survives `omitempty` and travels on the
wire, and "was anything declared?" stops being the natural zero-value test the save path already
uses. Moving the rarest entry costs one irregular id, guarded by
`TestSunatCodeZeroIsIdentityDocNone`; moving the sentinel would cost a check at every call site.

## The catalog is generated into the frontend, not served from it

**Context** — `IdentityDocOptions` has to exist in Go (the save path validates against it, and
`SunatIdentityDocCode` converts through it) *and* in the UI, which shows it as a picker. Serving
it would have made the list a fetch; mirroring it by hand would have made it two lists.

**Decision** — Go owns it, and `scripts/generators/sync_catalogs.go` copies it into
`frontend/services/crm/client-provider.svelte.ts` at build time through a `//CATALOG:` marker.

**Rationale** — Serving does not actually buy a single source of truth, because the Go copy has to
exist for validation either way; it only moves the duplication to runtime and makes a till's
dropdown wait on a request. Generating gets the single source at zero runtime cost. What it costs
is a build step that must be re-run — a stale frontend list is now possible in a way a fetch would
have prevented, which is why the marker sits directly above the list with the command in the
comment beside it.

## One `ClientProvider` table, one owning module, two menus

**Context** — clients and providers are the same row discriminated by `Type` (1 = client,
2 = provider). They were in `business`, surfaced as two sibling pages. The roadmap adds a client
file with history, a call/visit log, an opportunity funnel, receivables, credit lines, campaigns
and loyalty — all client-side, none provider-side.

**Decision** — a new L4 module `app/crm` owns the table (`crm/types/client_provider.go`) and its
handlers (`crm/client_provider.go`). The **Clientes** page moved to `/crm/customers`; the
**Proveedores** page moved to `/logistics/suppliers`, next to purchase orders.

**Rationale** — splitting the table by `Type` would mean two tables, two delta syncs and a
migration, to serve a UI grouping. Keeping one table and letting the route live where the user
looks for it costs nothing: `logistics` imports `crm/types`, which is an ordinary L4 → L2 edge.
The frontend does the same — both pages render one `domain-components/ClientProviderMaintainer`
parameterized by type.

## `SaveClientProviders` stays in `types`

**Context** — `sales` creates or resolves the buyer while recording a sale, so it needs the
upsert. A module body may not import another module body.

**Decision** — `crm/types/client_provider_save.go`, unchanged but for its new path.

**Rationale** — same reasoning as before the move (see `backend/docs/MODULE_BOUNDARIES.md`); only
the owning module's name changed.
