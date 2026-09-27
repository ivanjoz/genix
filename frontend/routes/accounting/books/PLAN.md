# PLAN — Accounting Books (`/accounting/books`)

Status: **approved, ready to build.** Three books under one page: Ventas, Compras, Diario
Simplificado. Decisions in §6, phases in §5.

**The target is SIRE.** Ventas and Compras are built against the RVIE/RCE structures; PLE 14.x
and 8.x stay in this document as reference only and are not a build target. The one exception is
the **Diario Simplificado, which has no SIRE equivalent and can only go through PLE 5.2 + 5.4** —
that is not a choice, it is where SUNAT put it.

Sources, in order of authority: **RS 112-2021/SUNAT and its annexes** (provided by the user,
parsed in full — this is the SIRE/RVIE rulebook), the official PLE annexes
(`anexo1-361-2015.pdf` for 14.1/14.2, `anexo2-361-2015.pdf` for 8.1/8.2/8.3), the two docs in
`../docs/`, and the code in this repo. Every claim about the repo below was read, not assumed;
file:line references are given so each can be checked.

---

## 1. What SUNAT actually asks for

### 1.1 Two different worlds, and the book codes are not the same

This is the single most important thing to get right, and it is where most ERP implementations
go wrong:

| | PLE (reference only) | **SIRE / RVIE — the target** |
| --- | --- | --- |
| Ventas | 14.1 (34 campos) / 14.2 simplificado (25) — código `140100` | **Anexo 3 de RS 112-2021 — 33 campos informados — código `140400`** |
| Compras | 8.1 (41) / 8.3 simplificado (31) — código `080100`/`080200` | RCE (misma familia, resoluciones posteriores) |
| Periodo | `AAAAMM00` (8) | **`YYYYMM` (6)** |
| CUO / correlativo A-M-C | obligatorios | **no existen** — SUNAT asigna el CAR |
| Diario Simplificado 5.2 | **PLE. No está en SIRE.** | — |

**The RCE annex is in hand.** RS 040-2022 (Anexo 11, the file that replaces the RCE proposal:
`080400`, fields 1-37) and RS 138-2023 were provided and are summarized field by field in
`../docs/SUNAT-RCE-Diario-annexes-summary.md`. The Compras exporter targets Anexo 11. The PLE 8.3
list in §1.9 is reference only.

The user's intuition that the sales book "can be simplified" is right twice over: PLE has an
actual **format 14.2 "simplificado"** (drops exportación, descuentos, exonerado/inafecto, ISC,
IVAP, contrato), **and** the SIRE replacement file drops the CUO and correlativo machinery
entirely — SUNAT builds the proposal and assigns the key itself.

### 1.2 SIRE — Anexo 3: the flat file that replaces the RVIE proposal

Fields **1–33 are informed by the taxpayer**. Fields **34–40 SUNAT completes automatically from
the proposal** (nota 4). Fields **41–57** are free use. Separator `|`.

| # | Nemotécnico | Field | Len | Notes |
| --- | --- | --- | --- | --- |
| 1 | RUC | RUC del generador | 11 | obligatorio |
| 2 | ID | Razón social del generador | ≤1500 | debe pertenecer al RUC |
| 3 | Periodo | **`YYYYMM`** | 6 | 01≤MM≤12 |
| 4 | CAR SUNAT | Código de Anotación de Registro | 29 | **consignar vacío** — SUNAT lo completa |
| 5 | Fecha de emisión | `DD/MM/AAAA` | 10 | ≤ periodo del campo 3 |
| 6 | Fecha Vcto/Pago | vencimiento o pago | 10 | obligatorio sólo si campo 7 = `14` |
| 7 | Tipo CP/Doc. | tipo de comprobante | 2 | tabla de comprobantes |
| 8 | Serie del CDP | serie | ≤20 | 4 para 01/03/07/08 |
| 9 | Nro CP / Nro Inicial | número (o inicial del rango) | ≤20 | |
| 10 | Nro Final | final del rango | ≤20 | **sólo tipos `00`,`03`,`12`,`13`,`18`,`87`,`88`** |
| 11 | Tipo Doc Identidad | del cliente | 1 | condicional |
| 12 | Nro Doc Identidad | del cliente | ≤15 | condicional |
| 13 | Apellidos/Razón Social | del cliente | ≤1500 | condicional |
| 14 | Valor Facturado Exportación | | 12,2 | |
| 15 | BI Gravada | base imponible | 12,2 | sin ISC |
| 16 | Dscto BI | descuento de base | 12,2 | negativo, sólo `07`/`87` |
| 17 | IGV / IPM | | 12,2 | |
| 18 | Dscto IGV / IPM | | 12,2 | negativo, sólo `07`/`87` |
| 19 | Mto Exonerado | | 12,2 | |
| 20 | Mto Inafecto | | 12,2 | |
| 21 | ISC | | 12,2 | |
| 22 | BI Grav IVAP | | 12,2 | obligatorio si campo 7 = `49` |
| 23 | IVAP | | 12,2 | idem |
| 24 | **ICBPER** | impuesto a las bolsas de plástico | 12,2 | **obligatorio cuando campo 7 ∈ {01,03,07,08,12}; `0.00` si no hay** |
| 25 | Otros Tributos | cargos fuera de la base | 12,2 | |
| 26 | Total CP | importe total | 12,2 | |
| 27 | Moneda | ISO 4217 Alpha | 3 | |
| 28 | Tipo Cambio | | 1,3 | **obligatorio sólo si campo 27 ≠ `PEN`** |
| 29 | Fecha Emisión Doc Modificado | | 10 | sólo `07`/`08`/`87`/`88` |
| 30 | Tipo CP Modificado | | 2 | idem |
| 31 | Serie CP Modificado | | ≤20 | idem |
| 32 | Nro CP Modificado | | ≤20 | idem |
| 33 | ID Proyecto Operadores | consorcios sin contabilidad independiente | ≤50 | |
| 41–57 | CLU | campos de libre utilización | ≤200 | omitir palotes si no se usan |

Four rules that change the design:

- **Nota 3 — un comprobante anulado o rechazado se anota con importe `0.00`, no se omite.**
  Electronic CPs whose CDR is not "aceptado", CPs with *baja comunicada*, and annulled physical
  CPs go into the book as rows with fields 14–26 at zero. This maps one-for-one onto
  `InvoiceDocument.State` (§2.1).
- **Nota múltiple:** fields 29–32 accept several modified documents, comma-separated in
  correlative order; the length cap rises to 1500 for those.
- **Consolidación diaria de boletas electrónicas** lives in field 10 (`Nro Final`), and only for
  the listed types. Same rule as PLE, different field number.
- **Tipo de cambio is only required when the currency is not PEN** — unlike PLE 14.x, where it
  was required whenever the currency field carried data.

### 1.3 SIRE — file naming (Tabla 6)

```
LE RRRRRRRRRRR AAAA MM 00 140400 CC O I M G [N] .TXT
   └ RUC (11)                    │  │ │ │ │  └ correlativo de ajustes posteriores
                                 │  │ │ │ └ G = '2' fijo (generado por MIGE IGV)
                                 │  │ │ └ moneda: 1 soles · 2 dólares
                                 │  │ └ contenido: 1 con información · 0 sin
                                 │  └ operaciones: 0 baja RUC · 1 operativa · 2 cierre del libro
                                 └ oportunidad: 01 acepta · 02 **reemplaza** · 03 ajustes ·
                                   04/05 ajustes de periodos anteriores al nuevo sistema
```

Mask for the replacement file: `LERRRRRRRRRRRAAAAMM0014040002OIM2.TXT`. Note positions 20-21
(`DD`) are `00` for the RVIE, and the book identifier is **`140400`**, not `140100`.

### 1.4 The CAR — and why it unblocks the Diario Simplificado

**Tabla 7 / Anexo C.** The CAR is 27 characters, built purely from the comprobante:

```
RUC del emisor (11) + tipo CP (2) + serie (4) + número (10)
```

Left-pad with zeros when shorter; take the rightmost digits when longer; an empty serie becomes
zeros. SUNAT assigns it automatically from the proposal — **but it is fully deterministic, so
the ERP can compute the CAR of its own documents without any SIRE round-trip.** For a Genix
document that is `company.RUC` + `01|03|07|08` + `SeriesCode` + `Correlativo` zero-padded to 10.

That matters because of **Anexo B**, which redefines field 20 of the Libro Diario (5.1) and the
Libro Diario de Formato Simplificado (5.2):

| Situation | Field 20 content |
| --- | --- |
| Taxpayer **not** on the RVIE module | `códigoLibro & campo1 & campo2 & campo3` — `140100` for ventas, `080100`/`080200` for compras |
| Taxpayer **on** the RVIE module (SIRE) | **the CAR (27 chars)** |
| Entry is **consolidated** | empty, in both cases |

So the Diario's link to the sales book is computable from data the ERP already holds. This was
the open question in my first pass; it is now closed.

Anexos A and B also add a **"Código de Anotación de Registro"** column to the *physical*
formats 5.1 and 5.2 — so the on-screen Diario should carry that column too, not just the TXT.

### 1.5 SIRE — reference tables (Anexo 1)

**Tipo de documento de identidad:** `0` doc.trib.no.dom.sin.ruc · `1` DNI · `4` carné de
extranjería · `6` RUC · `7` pasaporte · `A` céd. diplomática · `B` doc. país de residencia ·
`C` TIN · `D` IN · `E` TAM · `F` PTP.

**Estado de validez del comprobante (Tabla 9):** `Activo` (CDR aceptado) · `Baja` (CDR aceptado
con baja comunicada) · `Rechazado` (el XML no pasó validaciones, el número se inutiliza) ·
`Autorizado` (físico con autorización de imprenta).

**Reglas generales (Tabla 5):** separador `|`; negativos con formato `- #.##`; texto libre entre
palotes que **no puede contener `|`, `/`, `\`**; "alfanumérico" = A–Z más `( ) , .`; longitudes
y módulo 11 por tipo de documento de identidad.

**Series, de la tabla de comprobantes:** factura electrónica → serie `E001` o `FXXX`; boleta
electrónica → `EB01` o `BXXX`; número hasta 8 dígitos, numérico mayor a cero. See §2.1 for a
divergence in our validator.

**Un aviso sobre las referencias cruzadas:** the published resolution numbers its own tables
inconsistently — field 27 says "Tabla N.º 3 Tipo de moneda" while the index lists Tabla 2 as
moneda, and field 11 cites "tabla 2 y 6" for tipo de documento de identidad while the index
lists Tabla 1. **Go by the table title, never by the number cited in a field rule.**

### 1.6 SIRE — the API (Anexo 6)

REST over TLS, **OAuth 2.0** with JWT tokens derived from the SOL credentials.

- `POST` a ZIP containing a single flat file plus the file's **SHA-256** hash → returns a
  **ticket**.
- `GET` with the ticket → JSON with the process state and, when finished, the *Reporte de
  Información Recibida* plus a file listing the rejected records.

This is a realistic phase-5 target and matches what
`../docs/SUNAT-Libro-Ventas-RVIE-SIRE-ERP.md` §5.5 describes.

### 1.7 PLE — what still applies

For a taxpayer **not** yet on SIRE, Ventas and Compras are still PLE 14.1/14.2 and 8.1/8.3,
with `AAAAMM00`, CUO, and the `A`/`M`/`C` correlativo. Those field lists were parsed from the
official annexes and are reproduced in §1.8/§1.9 below in condensed form.

The **Diario Simplificado 5.2 is PLE in both worlds** — it has no SIRE equivalent. Its
companion 5.4 (plan contable) is obligatorio in January and the first time the book is
generated. One TXT row per account line, Debe and Haber mutually exclusive per row, Σ Debe =
Σ Haber per CUO, correlativo prefixed `A` (apertura) / `M` (movimiento) / `C` (cierre).

### 1.8 PLE 14.2 (Registro de Ventas simplificado) — condensed

1 Periodo `AAAAMM00` · 2 CUO (≤40, llave) · 3 Nº correlativo `A/M/C` (2–10, llave) · 4 Fecha de
emisión · 5 Fecha de vencimiento (sólo tipo `14`) · 6 Tipo CP · 7 Serie · 8 Número · 9 Número
final (consolidación diaria de boletas) · 10-12 tipo/nro/nombre del cliente · 13 BI gravada ·
14 IGV/IPM · 15 otros conceptos · 16 total · 17 moneda · 18 tipo de cambio · 19-22 documento
modificado · 23 error tipo 1 · 24 medio de pago · 25 estado `0/1/2/8/9` · 26-50 libre uso.

Fields 10/11/12 are **optional when campo 16 < 700.00 y campo 6 ∈ {03,12}** — the same S/ 700
threshold already coded as `BoletaIdentifiedFrom = 70_000` cents
(`invoicing/types/customer_identity.go:19`). The two must never drift.

14.1 adds exportación, descuentos de BI e IGV, exonerado, inafecto, ISC, IVAP y contrato.

### 1.9 PLE 8.3 (Registro de Compras simplificado) — condensed

1-3 as above · 4 fecha de emisión **del comprobante del proveedor** (obligatorio) · 5 fecha de
vencimiento · 6 tipo CP · 7 serie · 8 número · 9 número final · 10-12 tipo/RUC/razón social del
proveedor · 13 BI con derecho a crédito fiscal · 14 IGV · 15 otros conceptos · **16 total, y
debe ser exactamente 13+14+15** · 17 moneda · 18 tipo de cambio · 19-22 documento modificado ·
23-24 constancia de detracción · 25 marca de retención · 26 clasificación tabla 30 (>1500 UIT) ·
27-29 marcas de error · 30 medio de pago · 31 estado `0/1/6/7/9`.

---

## 2. What the repo can feed these books today

### 2.1 Ventas — mostly there

Mapping against **Anexo 3** (the SIRE target); the PLE 14.x mapping differs only in field
numbering and in needing a CUO.

| Anexo 3 field | Source | Status |
| --- | --- | --- |
| 1-2 RUC y razón social del generador | `config.Company.RUC` (`invoicing/issuer.go:117`) | ✅ |
| 3 Periodo `YYYYMM` | the page's month selector | ✅ |
| 4 CAR | vacío on the replacement file; computable for the Diario (§1.4) | ✅ |
| 5 Fecha de emisión | `InvoiceDocument.IssueDate` (UnixDay) | ✅ |
| 7 Tipo CP | `InvoiceSeries.DocType` → `01/03/07/08` | ✅ |
| 8 Serie | `InvoiceSeries.SeriesCode` | ⚠️ **not on the document** — the row carries only the series id in the last two digits of its own id (`invoice_document.go:155`); the code lives inline on the company record |
| 9 Número | `Correlativo` | ✅ |
| 10 Nro final | consolidation of sub-700 boletas | 🆕 to build |
| 11 Tipo doc identidad | `ClientProviderSnapshot.IdentityDocType` | ✅ **stored as SUNAT's own character — no conversion** |
| 12-13 Nro doc y razón social | `InvoiceDocument.ClientSnapshotID` → `ClientProviderSnapshot` | ✅ **frozen at issuance** |
| 15 BI gravada | `TaxableAmount` | ✅ |
| 17 IGV | `TaxAmount` | ✅ |
| 19 Exonerado / 20 Inafecto | `ExemptAmount` / `UnaffectedAmount` | ✅ |
| 24 ICBPER | — | ⚠️ no concept in the ERP → emit `0.00` (legal, but wrong the day they sell bolsas) |
| 26 Total | `TotalAmount` | ✅ |
| 27 Moneda | `Currency` (1=PEN, 2=USD) → `PEN`/`USD` | ✅ |
| 28 Tipo de cambio | — | ❌ **not stored** (only needed when Currency ≠ PEN) |
| 29-32 Doc modificado | `AffectedDocID` → that row + its series | ✅ resolvable |
| Nota 3 — importe `0.00` | `State`: `InvoiceRejected`→Rechazado, `InvoiceVoided`→Baja, `InvoiceAccepted/Observed`→Activo (`invoice_document.go:15-31`) | ✅ maps cleanly |

**The identity problem is solved** (`crm/types/client_provider_snapshot.go`). `ClientProvider`
now carries `IdentityDocType` — SUNAT's catalog character verbatim, so the same value travels
from CRM to the XML to the book with no conversion — plus a `SnapshotID` pointing at an
immutable, content-addressed `ClientProviderSnapshot` row. `InvoiceDocument.ClientSnapshotID`
pins it at issuance (`document_reserve.go:105`), and a resend reads the pin instead of
re-resolving the customer (`sale_order_to_cpe.go`, `readBuyerIdentity`). A client renamed in CRM
can no longer rewrite a filed book.

Three smaller things remain:

1. **The anonymous boleta pins nothing** — `ClientSnapshotID` stays `0` and the document
   declares `CLIENTES VARIOS` / `00000000` / doc type `0`. That is correct (Anexo 3 makes
   fields 11-13 optional below S/ 700 for tipo `03`), but the book mapper has to reproduce it,
   and `anonymousDocNumber` / `anonymousCustomerName` are unexported in `invoicing/types`
   (`sale_order_to_cpe.go:177-180`). They need exporting, or a small `DeclaredBuyer()` helper.
2. **Existing rows carry none of this.** `IdentityDocType` and `SnapshotID` are filled on save
   (`client_provider_save.go`), so a client nobody edits keeps them empty, and every
   `InvoiceDocument` written before the column has `ClientSnapshotID = 0`. Per CLAUDE.md §3 this
   wants a **backfill script** — derive the doc type, resolve the snapshots, stamp the existing
   documents — not a read-time fallback in the book mapper. No such script exists yet.
3. **`ValidateSeries` is stricter than SUNAT.** It requires exactly 4 characters starting with
   `F` or `B` (`invoicing/types/invoice_series.go:158-169`). SUNAT also allows `E001` for
   facturas and `EB01` for boletas. Not a bug today — the seeded series are F/B — but a company
   migrating in with an `E001` series cannot be configured.

**Sales without a comprobante** are reachable: a sale whose id ends in `00` was never registered
for electronic issuance (`SaleOrder.SeriesID()`, `sales/types/sale_order_id.go:86`). Those are
*not* book rows — the Libro de Ventas is a book of comprobantes — but the user explicitly wants
them visible, so they belong on the page as a separate, clearly-labelled control block.

### 2.2 Compras — ✅ built (phase 6)

**The supplier's identity is solved**, the same way as the customer's:
`PurchaseOrder.ProviderSnapshotID` is frozen when the order is created
(`logistics/purchase-order-management.go:253`) and never moves on edit. That covers
**fields 10/11/12** — tipo de documento, RUC y razón social del proveedor.

What the order still cannot answer is *which comprobante* the purchase was. `PurchaseOrder`
holds `Date` (the order's date, not the supplier document's issue date), `TotalAmount`,
`TaxAmount`, `DebtAmount` and one free-text `InvoiceNumber`. `Expense`
(`finance/types/expenses.go:86-114`) holds no document fields at all.

Missing against 8.3, none of it derivable:

- field 4 — issue date of the *supplier's* comprobante
- field 6 — tipo de comprobante (`01`, `03`, `07`, `08`, `14` recibo de servicios públicos…)
- fields 7/8 — serie and número, currently one unparsed string
- fields 13/14 — BI con derecho a crédito fiscal e IGV. `PurchaseOrder.TaxAmount` is a declared
  column that the save path never writes (`logistics/purchase-order-management.go:334` lists the
  updated columns; `TaxAmount` is not among them), and `Expense` has no tax field at all
- field 16 — must equal 13+14+15 exactly, so it cannot be a rounded total
- fields 17/18 — moneda y tipo de cambio (`Expense.CurrencyType` exists; `PurchaseOrder` has none)
- fields 23/24 — constancia de detracción
- field 31 — estado de la anotación

**Decided: the columns go on `PurchaseOrder`, `Expense` and `Asset`** (confirmed 2026-09-24). A
recibo por honorarios, an electricity bill and a fixed-asset factura are all RCE rows, so without
them the book would miss those purchases.

The column set, mapped to **RCE Anexo 11** (revised from the PLE 8.3 draft against the annex):

| Column | Anexo 11 field | Notes |
| --- | --- | --- |
| `DocType int8` | 7 | SUNAT catalog — reuse `invoicing.DocType*` values (`01`,`03`,`07`,`08`…). `0` = the purchase has no comprobante |
| `DocSeries string` | 8 | separated from the number at last |
| `DocNumber int64` | 10 | numeric on every type the ERP books; `int64` because supermarket tickets run past `int32`. Only the series is alphanumeric |
| `DocIssueDate int16` | 5 | UnixDay — the **supplier's** issue date, not the order's. **Also picks the RCE period** |
| `TaxableAmount int32` | 15 | base DG. A company that only makes taxed sales never uses DGNG/DNG (17-20) |
| `TaxAmount int32` | 16 | IGV — already a column, currently never written |
| `UntaxedAmount int32` | 21 | 🆕 exonerado + inafecto together |
| `OtherAmount int32` | 24 | charges outside the base |
| `CurrencyType int8` | 26 | existing column on `Expense`/`Asset`, new on `PurchaseOrder` — same name on all three |
| `ExchangeRate int32` | 27 | ×1000; required when currency ≠ PEN. SBS weighted-average selling rate of `DocIssueDate` |

Dropped from the draft: **`DetractionCode`/`DetractionDate`**. Anexo 11 has no detracción field
(field 38 is a SUNAT-filled mark, not sent).

Plus `ProviderSnapshotID` on `Expense` and `Asset` (fields 12-14; `PurchaseOrder` already had
it). Field 6 reads the existing `DueDate` (`PaymentDate` on orders), so no new column. The rules
are one function, `finance.NormalizePurchaseDocument` (`backend/finance/types/purchase_document.go`);
the form block is `$domain/PurchaseDocumentFields.svelte`; the book is `books.purchases.ts` +
`books.purchases.txt.ts`, read from `GET.purchases-book`. Existing rows are normalized by
`fn-backfill-purchase-documents`.

Field 25 (Total CP) is computed as 15+16+21+23+24, never stored. ICBPER (23) is written as
`0.00`. `InvoiceNumber` (free text) is replaced by `DocSeries`/`DocNumber` and deleted, since
pre-alpha keeps no compatibility.

**Rows the exporter omits** (Anexo 11 note 2): canceled orders, and documents with baja, reverted
or annulled by a type-02 credit note. **This is the opposite of RVIE nota 3.** Supplier credit
notes go in as negative amounts in 15-25.

**Purchases without a comprobante** (`DocType = 0`) are not RCE rows. The page hides them behind a
checkbox with a warning icon. They are excluded from totals and the TXT, and they are not an
export blocker.

None of this is optional: a Libro de Compras assembled from what exists today would have its
mandatory columns blank, which is worse than no book.

### 2.3 Diario Simplificado — nothing exists

No chart of accounts, no journal, no entries. `backend/accounting/` is assets + depreciation
only. Field 20, previously the hardest part, is now solved (§1.4) — the CAR is computable from
the document itself.

Per §6 this book is **derived at read time**: the PCGE account mapping is a parameterizable table
in code, and the sources are the ones that already exist — invoice documents (Dr Caja/Cliente,
Cr Ventas, Cr IGV), purchases (Dr Compra/Gasto, Dr IGV crédito, Cr Proveedor), expenses,
depreciation rows (`Expense.Type = 4`), and cash-bank movements. The CUO is the source record's
id, so it is stable and unique without a counter. Debe = Haber is a unit-tested invariant of the
mapping, not a database constraint.

---

## 3. The page

```
frontend/routes/accounting/books/
  +page.svelte          Selector row, book switching, tables. No business logic
  BookToolbar.svelte    The three selectors + export buttons
  books.svelte.ts       $state + the service calls that feed it
  books.ts              Pure: period math, row building, SUNAT field mapping, consolidation
  books.utils.ts        DD/MM/AAAA, importes, CAR padding, pipe escaping. No business logic
  books.excel.ts        ExcelTableColumn sets, one per book
  books.test.ts         Vitest over books.ts
  RATIONALE.md
  DOCUMENTATION.md
```

### 3.1 Selector row, exactly as specified

```
[ SearchSelect: Libro ▾ ]  [ DateInput: Fecha ]  [ SearchSelect: Mes ▾ ]   … [Excel] [TXT]
```

- **Libro** — `Ventas` · `Compras` · `Diario Simplificado`.
- **Fecha** — `DateInput`, single day, the same component the invoicing page uses
  (`../invoicing/+page.svelte:115`).
- **Mes** — `Día específico` plus the last 24 months, newest first, labelled `SET 2026`.
  Picking a month **disables the DateInput** and loads the whole period; picking `Día
  específico` re-enables it and loads one day. The SUNAT period (`202609` for SIRE, `20260900`
  for PLE) is shown next to the title, so what is on screen and what would be exported are
  visibly the same thing.

A single day is **not** a filable book. The day view is a working/checking view and the page
says so; only a full month produces an export. That keeps the period field honest.

### 3.2 Tables

One `VTable` per book, columns in SUNAT field order with the field number in the header tooltip,
so an accountant can tick the book against the annex column by column. Totals row at the bottom
(BI gravada, IGV, total) — that is the figure that must tie to the Formulario 621.

Rows for documents in state *Rechazado* or *Baja* are shown with amounts at `0.00` and a state
chip, because that is literally what goes in the file (nota 3).

The Diario carries a **CAR** column, matching the physical format in Anexo A/B.

Below the Ventas book, a second read-only block: **Ventas sin comprobante**, with its own total,
labelled as *not part of the book*.

---

## 4. Backend

One new handler in `backend/accounting/`, which may query `invoicing/types`, `sales/types`,
`crm/types` and `config/types` tables directly — those are leaf packages, so
`docs/MODULE_BOUNDARIES.md` is satisfied without touching a module body.

```
GET.sales-book?period=YYYYMM   (whole month)
GET.sales-book?date=<unixDay>  (single day)
```

Returning the period's comprobantes as **raw numbers**, not mapped rows: the amounts, the dates,
the `ClientSnapshotID`, and the series id the document already carries in the tail of its own id.
The fat columns of an `InvoiceDocument` — CDR notes, ticket, digest, transport error — are not
part of a book and are left behind.

**Both joins happen in the browser**, which is the house rule:

1. **The series** — and with it the document type and the series code — comes off the company
   record the app already holds. `/accounting/invoicing` resolves it the same way
   (`invoicing/+page.svelte`), so the handler never re-reads the company.
2. **The buyer's identity** goes through the by-ids cache
   (`packages/genix-ui/cache/CACHE_BY_IDS.md`) against `GET.client-provider-snapshot-ids`. A
   snapshot is immutable, so a browser that has one keeps it forever and the same few hundred
   identities are never downloaded twice.

The date range still has to be a server read: `GetInvoices` is a whole-company delta sync
(`invoicing/invoice_api.go:20-31`) and a book is a period slice. It goes through the `IssueDate`
index group, because Scylla refuses a range on a plain secondary index.

The SUNAT mapping therefore lives in `books.ts` — pure, no Svelte and no fetch, so the rules
(S/700, contiguous-run consolidation, nota-3 zeroing) are testable with no component and no
network.

**The TXT writer follows it into the browser** (`books.txt.ts`). The consolidated rows only exist
here, so writing the file here is what keeps the preview and the filed file the same bytes and
leaves one implementation of the consolidation rules. The backend gains no route in phase 4; in
phase 5 it receives the file text, zips it, hashes it and uploads it to SIRE.

---

## 5. Phases

| # | Scope | Blocked by |
| --- | --- | --- |
| — | ~~Customer/supplier snapshot + `IdentityDocType`~~ | ✅ **done** — `client_provider_snapshot.go` |
| 0 | Export the declared-buyer constants; fix `ValidateSeries` for `E001`/`EB01` (§2.1) | — |
| 1 | Page shell, three selectors, month/day logic, menu entry, `access.toml` entry | — |
| 2 | **Libro de Ventas** read-only + Excel: backend `GET.accounting-book&book=ventas`, **Anexo 3** mapping, contiguous-run consolidation, nota-3 zeroing, "ventas sin comprobante" block | — |
| 3 | Backfill script for pre-snapshot rows (§2.1, point 2) | — |
| 4 | ~~**TXT export** — Anexo 3 replacement file with the Tabla 6 filename~~ | ✅ **done** — `books.txt.ts` |
| 5 | SIRE API upload (Anexo 6: OAuth2 + ZIP + SHA-256 + ticket polling) | — |
| 6 | ~~**Libro de Compras**: the §2.2 column set on `PurchaseOrder`, `Expense`, `Asset`, form capture, Anexo 11 mapping + TXT, "sin comprobante" checkbox~~ | ✅ **done** — supplier credit notes (07/08) still open, see §6 |
| 7 | **Diario Simplificado**: PCGE chart (codes + names, feeds 5.4) and mapping, read-time derivation, 5.2 + 5.4 TXT, field 20 via CAR | open decisions: chart of accounts fixed in code or per company, periodic vs permanent inventory, counter-accounts for the unspecified/withdrawal/loss/count cash movements, CUO prefixing, a PEN rate for USD documents |

The full 5.2 (21 fields) and 5.4 (8 fields) structures are in `Estructura del PLE.xls`,
summarized in `../docs/SUNAT-RCE-Diario-annexes-summary.md` §3.3. Every entry line carries a
document type and number (`00` plus an internal number when there is no comprobante), and that
number is part of the row's unique key.

**Two things phase 4 could not close**, both visible on the page as export blockers rather than
guessed at:

- **Field 28, tipo de cambio.** `InvoiceDocument` stores no exchange rate, and the field is
  mandatory the moment a row is not in PEN. A period holding a non-PEN comprobante refuses to
  export. Closing it means a column and a source for the SUNAT rate of the issue date.
- **The sign of a nota de crédito.** The writer negates fields 15/17/19/20/26 for tipo `07`.
  Fields 16 and 18 ("Dscto BI"/"Dscto IGV", negative, only `07`/`87`) may be where the annex
  actually wants the reduction. `salesBookRowSign` is the single place to change it.

---

## 6. Decisions

### Settled

- **The RCE file replaces the proposal (Anexo 11)**, the same path as Ventas (Anexo 3).
- **Target: SIRE.** Ventas against Anexo 3 of RS 112-2021, Compras against the RCE annex. PLE
  14.x/8.x are reference only. The Diario Simplificado goes through PLE 5.2/5.4 because SUNAT
  offers no alternative.
- **Compras must be SUNAT-compliant** — a filable Anexo 11 file, not a preview.
- **Purchase document columns on `PurchaseOrder`, `Expense` and `Asset`.** Column set in §2.2.
  `Expense` also needs the due date (RCE field 6, mandatory for type `14`).
- **The PCGE chart of accounts is fixed in code**, not configurable per company.
- **Periodic inventory.** A purchase posts Dr 60 / Dr 40 / Cr 42, and its warehouse entry posts
  Dr 20 / Cr 61. A sale posts only the income. Cost of sales (Dr 69 / Cr 20 = opening + purchases −
  closing) is posted at period end, valuing the closing stock at stock × last purchase price. No
  per-sale costing.
- **Cash-movement counter-accounts.** The cash side is 101 (till) or 1041 (bank account). A
  transfer is 101 ↔ 1041. It is stored as two rows (source outflow + destination inflow,
  `BuildCashTransferMovements`), and **the Diario posts it once, from the negative row**.
  Withdrawals, losses and physical counts are classified by the user when
  the movement is created. The account is stored in `CashBankMovement.AccountCode int16`, and the
  options are listed in `finance/types/cash_movement_account.go` (done).
- **The RCE period is the month of `DocIssueDate`.** No separate book-period column.
- **Purchases without a comprobante**: shown behind a checkbox with a warning icon, never part of
  the book.

### Open

- **Supplier credit notes (`07`).** A credit note needs fields 28-32 (the modified document).
  The user placed it on the purchase-order table; still to settle is whether it is its own
  `purchase_order` row pointing at the original order, or extra columns on the original order.
  Until then the purchase doc types are 01/02/03/12/14 and the book has no negative rows.
- The three §2.1 follow-ups are approved: export the declared-buyer constants, write the
  backfill, relax `ValidateSeries` to accept `E001`/`EB01`.

- **Diario Simplificado: derived at read time.** No `account_move` tables. Entries are built on
  the fly from invoice documents, purchases, expenses and cash movements through a
  parameterizable PCGE mapping. The consequence, accepted knowingly: it **shows** the book but
  cannot **file** it — a filed 5.2 needs manual adjustments, apertura/cierre and estados `8`/`9`,
  and none of those have anywhere to live. The real ledger is a later, separate decision.
- **Boletas: always consolidated.** One row per day per series for sub-S/700 boletas, using
  Anexo 3 field 9 (`Nro Inicial`) and field 10 (`Nro Final`). No toggle.

### What "always consolidated" actually means

A consolidated row declares a **contiguous range of correlativos**, so the grouping is not simply
"all sub-700 boletas of the day". Within each `(fecha, serie)`, order by correlativo and emit
**maximal contiguous runs** of boletas that are both under S/ 700 **and** in state Activo.
Anything else breaks the run and becomes its own detailed row:

- a boleta **≥ S/ 700** — SUNAT requires it detailed with the buyer's identity;
- a boleta that is **Rechazado** or with **baja comunicada** — nota 3 puts it in the book at
  `0.00`, which it cannot be while hidden inside somebody else's range;
- a gap in the correlativo sequence.

So a day is not one row but *n* rows, and a run of length 1 is written as a plain detailed row
rather than a range. Fields 11-13 (identity) stay empty on a consolidated row. This is the single
densest rule in the mapper and is where the unit tests should concentrate.

---

## 7. Sources

- **RS 112-2021/SUNAT y anexos** (provided by the user: `RS 112-2021.pdf`,
  `anexo-112-2021.pdf`) — Anexo 1 tablas, Anexo 2 propuesta/complemento, **Anexo 3 reemplazo del
  RVIE**, Anexo 4-5 ajustes posteriores, Anexo 6 API, Anexo 7 obligados, Anexos A/B/C formatos
  5.1/5.2 y CAR. [Texto en El Peruano](https://elperuano.pe/NormasElperuano/2021/07/31/1977652-1/1977652-1.htm)
- SUNAT, [Anexo N.° 1 RS 361-2015 — Registro de Ventas e Ingresos Electrónico (14.1 / 14.2)](https://www.sunat.gob.pe/legislacion/superin/2015/anexo1-361-2015.pdf)
- SUNAT, [Anexo N.° 2 RS 361-2015 — Registro de Compras Electrónico (8.1 / 8.2 / 8.3)](https://www.sunat.gob.pe/legislacion/superin/2015/anexo2-361-2015.pdf)
- SUNAT, [Formato 5.2 — Libro Diario de Formato Simplificado](http://contenido.app.sunat.gob.pe/insc/Libros+y+Registros/Informacion+m%C3%ADnimo+formatos/FORMATO_5_2.pdf)
- SUNAT, [RVIE — Registro de Ventas e Ingresos Electrónicos](https://cpe.sunat.gob.pe/node/159)
- `../docs/SUNAT-Libro-Ventas-RVIE-SIRE-ERP.md`, `../docs/SUNAT-Libro-Diario-Formato-Simplificado.md`
