# Plan — numeric `IdentityDocType` + `leftOptions` selector on `Input`

**Goal.** At `sales/sale_order_create`, the walk-in client form must let the cashier pick the
identity document type next to the *Documento / RUC* box. The picker is a new `leftOptions`
prop on `packages/genix-ui/form/Input.svelte`. Per your decision, `ClientProvider.IdentityDocType`
becomes an **`int8`** with a numeric catalog, instead of the SUNAT character string it is today.

**Scope note.** `IdentityDocType` and `ClientProviderSnapshot` are entirely uncommitted
(`git diff` on `client_provider.go`, and `client_provider_snapshot.go` is untracked). Nothing
has shipped, so this is a change to in-flight code, not a migration of released data.

**Decisions taken** (all yours, recorded here so the plan reads standalone):

1. `IdentityDocType` becomes `int8` with a numeric catalog.
2. `0` means *undeclared*; SUNAT's own code `"0"` gets id **9**.
3. The frontend list is **generated from the Go list**, not hand-mirrored.
4. The dev DB is truncated rather than given a read-time fallback.
5. The till offers DNI / RUC / CE / Pasaporte; the CRM maintainer offers all 11.
6. The type auto-derives from the number typed until the cashier picks one.

---

## 1. The catalog

Today the column stores SUNAT's Anexo 1 character verbatim — `"0" "1" "4" "6" "7" "A"…"F"` —
precisely so it travels CRM → facturago XML → sales book with no conversion. Going numeric adds
one conversion at each of those two boundaries. That is the cost; the benefit is a normal
`int8` id the UI can key on like every other option list in the app.

### The ids

New file `backend/crm/types/identity_doc_type.go`:

```go
const (
    IdentityDocDNI        int8 = 1  // SUNAT "1"
    IdentityDocForeign    int8 = 4  // SUNAT "4" — carné de extranjería
    IdentityDocRUC        int8 = 6  // SUNAT "6"
    IdentityDocPassport   int8 = 7  // SUNAT "7"
    IdentityDocNone       int8 = 9  // SUNAT "0" — doc. tributario no domiciliado sin RUC
    IdentityDocDiplomatic int8 = 10 // SUNAT "A"
    IdentityDocResidence  int8 = 11 // SUNAT "B"
    IdentityDocTIN        int8 = 12 // SUNAT "C"
    IdentityDocIN         int8 = 13 // SUNAT "D"
    IdentityDocTAM        int8 = 14 // SUNAT "E"
    IdentityDocPTP        int8 = 15 // SUNAT "F"
)
```

Two rules behind those numbers:

- **The id equals the SUNAT digit wherever SUNAT uses a digit** (1, 4, 6, 7), and continues
  through the hex values for `A`–`F` (10–15). For the four types that cover essentially every
  real row, the stored int *reads* as the code it prints.
- **`0` is reserved for "not declared"**, so `json:",omitempty"` and the derive-from-shape
  fallback keep working. SUNAT's own code `"0"` therefore cannot be id 0 — it gets **9**, the
  one free slot below the hex range. This is the single id that does not read as its own code,
  and the generator's round-trip test is what keeps it honest.

### Conversion at the two boundaries

```go
func SunatIdentityDocCode(docType int8) string  // 1→"1", 6→"6", 9→"0", 10→"A", 0→""
func IdentityDocTypeFromSunatCode(code string) int8
```

`IdentityDocTypeFromSunatCode` has no runtime caller — it exists for the facturago round-trip
test. If that reads as dead weight I will drop it and assert against a literal table instead.

### The catalog slice

The one source of truth, in the same file. `Name` carries the app's `"EN|ES"` bilingual form,
the same convention `personTypeOptions` already uses:

```go
type IdentityDocOption struct {
    ID   int8   `json:",omitempty"`
    Code string `json:",omitempty"` // SUNAT Anexo 1
    Name string `json:",omitempty"`
}

var IdentityDocOptions = []IdentityDocOption{
    {ID: IdentityDocDNI, Code: "1", Name: "DNI|DNI"},
    {ID: IdentityDocRUC, Code: "6", Name: "RUC|RUC"},
    {ID: IdentityDocForeign, Code: "4", Name: "Foreign ID|Carné de extranjería"},
    {ID: IdentityDocPassport, Code: "7", Name: "Passport|Pasaporte"},
    // … the remaining seven, in the order the pickers should show them
}
```

It is **not** served over HTTP. The frontend copy is generated at build time (§3).

---

## 2. Backend changes

| File | Change |
|---|---|
| `crm/types/identity_doc_type.go` | **new** — constants, `IdentityDocOptions`, both converters, `DeriveIdentityDocType` moved here (it is catalog logic, not snapshot logic) |
| `crm/types/client_provider.go` | `IdentityDocType string` → `int8`; same for `ClientProviderTable` col |
| `crm/types/client_provider_snapshot.go` | same two changes; drop the old `IdentityDoc*` string consts and `IdentityDocTypes`; `SelfParse` hashes the int8 |
| `crm/types/client_provider_save.go` | validate `== 0 → derive`, else must be in the catalog; error text lists names, not raw codes |
| `invoicing/types/sale_order_to_cpe.go` | `DocType: crm.SunatIdentityDocCode(...)` at both call sites; `AnonymousDocType` stays the **string** `"0"` — it feeds `model.Party`, which is facturago's public API |
| `accounting/types/sales_book.go` | `row.BuyerDocType = crm.SunatIdentityDocCode(identity.IdentityDocType)`. `SalesBookRow.BuyerDocType` stays a string: it is the printed book column |
| tests in `crm/types`, `accounting/types`, `invoicing/types` | updated to the new constants; `TestIdentityDocTypeMatchesFacturagoCatalog` becomes a check that every `IdentityDocOptions[i].Code` round-trips through both converters and matches facturago's `model.IDDoc*` |

`facturago` is untouched — it is a public repo whose `model.Party.DocType` is SUNAT's own
string format, and it names no consumer. The conversion belongs on our side of that line.

**`AnonymousDocType` stops being an alias.** Today it is `= crm.IdentityDocNone`, which after
this change would be the int `9` — but it feeds `model.Party.DocType` and must stay the string
`"0"`. A Go `const` cannot call `SunatIdentityDocCode`, so it becomes the literal `"0"` with a
test asserting `AnonymousDocType == crm.SunatIdentityDocCode(crm.IdentityDocNone)`. That keeps
it a const and makes the two drifting apart a build failure rather than a wrong filed book.

**Hash change — checked, and no migration was needed.** `ClientProviderSnapshot.Hash` is computed
over `IdentityDocType`, so changing the column type shifts every existing hash, and stale rows
would silently stop deduplicating. The dev DB turned out to hold none: `client_provider_snapshot`
counts **0 rows**, and no `client_provider` row carries a doc type (both checked with `fn-db`).
Nothing was truncated and nothing was cleared.

Worth knowing for the next type change: the ORM **does not migrate a column's type**. It compares
the live type against the struct and only logs (`genix-orm/scylla/deploy.go:945`), so a column that
had already been deployed as `text` would have stayed `text` and quietly mismatched. This one was
safe because it had never been written.

To populate the columns, run the existing backfill rather than anything by hand:

```bash
cd backend && go run . fn-backfill-identity-snapshots          # reports, writes nothing
cd backend && go run . fn-backfill-identity-snapshots apply    # writes
```

---

## 3. Go → TS catalog generator

The repo already has a Go→TS generator: `scripts/generators/sync_struct_interfaces.go` rewrites
frontend interfaces from backend structs, driven by `//STRUCT:<module>.<Type>` markers (four in
use, e.g. `sale_order.svelte.ts:44`). I extend it rather than add a script, so:

- **no dispatcher change** — `go run ./scripts sync_struct_interfaces` keeps working as-is;
- the marker set stays in one place, documented in that file's header comment, which is how
  `route_ids_generator.go` documents itself too.

New marker, parallel in spelling to the existing one:

```ts
//CATALOG:crm.IdentityDocOptions
export const IDENTITY_DOC_OPTIONS: { id: number; code: string; name: string }[] = [
  { id: 1, code: '1', name: 'DNI|DNI' },
  { id: 6, code: '6', name: 'RUC|RUC' },
  // … generated, do not edit
]
```

The generator resolves `crm.IdentityDocOptions` to the `var` declaration in a `*/types` folder,
reads the composite literal through `go/ast`, and replaces the TS declaration that follows the
marker — the same locate-and-replace strategy the struct path already uses. Const identifiers in
the Go literal (`IdentityDocDNI`) resolve to their values, which the existing named-basic-type
pass already does for field types.

---

## 4. `Input.svelte` — the `leftOptions` prop

Genix-UI is generic, so **no Peru business logic goes in it**. New props:

```ts
/** Options for the compact selector in the field's left slot. */
leftOptions?: { id: number; code?: string; name: string }[];
/** Where the picked option's `id` is written, on the same `saveOn` object as `save`. */
saveLeft?: keyof T;
/** Auto-picks an option from the current value while the user has not chosen one.
 *  The rule lives with the caller — this component only calls it. */
deriveLeftOption?: (value: string | number) => number | undefined;
```

Behaviour:

- Renders through `FieldShell`'s existing `prefix` snippet: a compact button showing the
  selected option's `name`, plus a caret, opening a small option list positioned via the
  shell's `overlay` snippet (the shell root is already the positioned ancestor).
- `deriveLeftOption` runs on every keystroke **until the user opens the selector and picks**;
  after that their choice is sticky for the life of the field. This is what makes the shown
  type equal the saved one instead of letting the backend fallback disagree with the UI.
- Registers with the agent registry as its own entry so `agent-browser` can drive it.

Two supporting changes in `form/field-shell.module.css`, both additive:

- `.prefix` gains `pointer-events: auto` for interactive prefixes (today it is `none`, correct
  for decorative icons; I will scope the override to a `has-interactive-prefix` class so plain
  icon prefixes still cannot eat clicks meant for the value).
- `.has-prefix .inp { padding-left: 30px }` becomes `padding-left: var(--input-prefix-width, 30px)`,
  since "DNI ▾" is wider than one glyph. Input sets the token from the measured prefix width.

**`genix-ui` is a submodule** — I will commit and push inside it, not just at the root.

---

## 5. Frontend consumers

- `services/crm/client-provider.svelte.ts`
  - the `IdentityDocType` object of string codes is replaced by the **generated**
    `IDENTITY_DOC_OPTIONS` (§3).
  - `IClientProvider.IdentityDocType` and `IClientProviderSnapshot.IdentityDocType`: `string` → `number`.
- `routes/sales/sale_order_create/+page.svelte` — the *Documento / RUC* `Input` gets
  `leftOptions` (DNI, RUC, CE, Pasaporte — filtered from the generated list),
  `saveLeft="IdentityDocType"` and a `deriveLeftOption` implementing 8 digits → DNI, 11 → RUC.
- `routes/sales/sale_order_create/sale_order.svelte.ts` — `ClientInfo` seeds
  `IdentityDocType: 0` so the derive path owns it until the cashier picks.
- `domain-components/ClientProviderMaintainer.svelte` — `IdentityDocType: ''` → `0`, and the
  form gets the **full** generated list so the types the till omits stay reachable.
- `routes/accounting/books/` — no change. It reads `BuyerDocType` off the backend row, which
  is still the printed SUNAT string.

---

## 6. Verification — what was actually run

- `go build ./...` clean; `go test ./crm/... ./invoicing/... ./accounting/...` all pass.
- `go run ./scripts sync_struct_interfaces` writes the catalog, and a second run reports
  "no changes needed" — idempotent.
- `svelte-check`: 10 pre-existing errors, none in any file this work touched.
- `bun test routes/sales/sale_order_create routes/accounting/books` — 38 pass, 0 fail.
- `agent-browser` on `/sales/sale_order_create`, with the selector registered as its own handle:
  - it offers exactly DNI / RUC / Carné de extranjería / Pasaporte with ids 1, 6, 4, 7;
  - it opens on DNI;
  - typing `20123456789` moves it to RUC on its own;
  - picking *Pasaporte* first and then typing the same number leaves it on Pasaporte.
- Screenshotted open, to confirm the dropdown anchors under the selector and does not clip.

- `go run ./scripts check_tables` — 53 table struct pairs, no findings.

## 7. RATIONALE.md entries I will write

- `backend/crm/RATIONALE.md` — why the ids are what they are, why `0` is undeclared and SUNAT
  `"0"` is 9, and what the two boundary conversions cost.
- `scripts/RATIONALE.md` — why the catalog is generated rather than served or mirrored.
- `frontend/packages/genix-ui/form/RATIONALE.md` — why the derive rule is injected rather than
  built in, and the two CSS tokens.
