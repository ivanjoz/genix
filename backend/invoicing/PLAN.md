# Plan — `facturago` (standalone library) + the `invoicing` integration in Genix

SUNAT electronic invoicing for Peru, split across two repositories:

- **`git@github.com:ivanjoz/facturago.git`** — a standalone, zero-dependency Go library that owns
  every piece of SUNAT knowledge: catalogs, totals, UBL 2.1 XML, XMLDSig, PKCS#12, the SOAP client,
  CDR parsing and error classification. It knows nothing about Genix, ScyllaDB or S3. Wired into
  this repo exactly like `genix-orm`: git submodule at `backend/facturago` + a `replace` directive.
- **`backend/invoicing/`** — the thin ERP side: tables, correlativo allocation, artifact storage,
  async emission and retries, HTTP handlers, and the mapping from a `SaleOrder` to a facturago
  document.

Documents in scope: **Factura (01)**, **Boleta (03)**, **Credit Note (07)**, **Debit Note (08)**,
**Daily Summary (RC)** and **Voiding Communication (RA)**.

**Decisions already made:**

| Topic | Decision |
| --- | --- |
| Distribution | Standalone repo, submodule + `replace`, same method as `genix-orm`. |
| Boundary | **facturago is stateless.** Pure functions plus one `Client` for the network. Genix owns all persistence. |
| Ambition | **Peru / SUNAT only.** Flat package, SUNAT vocabulary in the API. |
| Secrets | Genix decrypts and passes raw `.pfx` bytes + passwords per call. facturago never touches storage or encryption. |
| Dependencies | **Zero.** Standard library only, in both the library and the integration. |
| Sending | Asynchronous: the POST reserves the correlativo and returns; an async invoke emits; a cron-action retries. |
| Data source | A `SaleOrder` in Genix. |

The remote `git@github.com:ivanjoz/facturago.git` exists and is **empty** — this is a from-scratch
bootstrap (phase 0). Everything is written in English per `AGENTS.md`.

---

## 1. Ownership: who does what

| Concern | facturago | Genix `invoicing` |
| --- | --- | --- |
| SUNAT catalogs (01, 03, 06, 07, tributos) | ✅ | |
| Amount completion and coherence validation | ✅ | |
| Amount in words, rounding, Peru date handling | ✅ | |
| UBL 2.1 XML generation (canonical) | ✅ | |
| XMLDSig signature, PKCS#12 reading | ✅ | |
| Zip, file naming, QR string | ✅ | |
| SOAP transport, CDR parsing, error codes, retryability | ✅ | |
| Certificate and SOL credential **storage** | | ✅ `CompanySecrets` |
| Series and correlativo **allocation** | | ✅ ORM sequence |
| Issued-document persistence and state | | ✅ `InvoiceDocument` |
| XML / CDR artifacts in S3 | | ✅ |
| Async execution, retry scheduling, cron | | ✅ |
| HTTP handlers, permissions, tenancy | | ✅ |
| `SaleOrder` → document mapping | | ✅ |
| Frontend and route documentation | | ✅ |

The rule that keeps the split honest: **facturago never learns a Genix type, and Genix never
reimplements a SUNAT rule.** If a change needs both, it belongs on the wrong side of the line.

---

## 2. Reference audit

`/run/media/ivanjoz/projects/sunat-facturacion/` holds two MIT-licensed projects. Everything ported
lands in **facturago**; nothing from them lands in Genix.

- **`sunat-service/`** (Node 22 + TypeScript, ~10k lines) is the real source. It was validated end
  to end against SUNAT BETA — every case returned code `0` — and its `CLAUDE.md` records rules that
  each cost a rejection.
- **`greenter/`** (PHP) is the de-facto Peruvian standard, used as a structural reference plus one
  concrete data asset.

### 2.1 Ported from `sunat-service` → facturago

| Source | Target | Kind of port |
| --- | --- | --- |
| `domain/catalogs.ts` | `catalogs.go` | **Literal.** Tributos (IGV 1000, ISC 2000, ICBPER 7152, EXP 9995, GRA 9996, EXO 9997, INA 9998, OTROS 9999), catalog-07 → affectation map, catalogs 01 and 06. |
| `domain/totals.ts` | `totals.go` | **Literal.** Including the golden rule — *what the caller sent is never overwritten*, only absent values are filled — and the tolerance `ValidateDocument` grants to global discounts, charges and prepayments. |
| `util/money.ts` | `money.go` | **Literal.** **Reworked to integers.** Half-away-from-zero rounding survives, on scaled integers instead of float64. |
| `util/numero-letras.ts` | `number_to_words.go` | **Literal.** Legend 1000, currency table included. |
| `util/dates.ts` | `dates.go` | **Literal.** Peru is UTC-5, no DST. The library takes a `time.Time` from the caller; Genix supplies it from `core.Now()` (effective clock). |
| `ubl/namespaces.ts`, `common.ts`, `sale.ts` | `xml_writer.go`, `xml_common.go`, `xml_sale.go` | **Structure copied node by node, mechanics rewritten.** `xmlbuilder2` → the canonical writer (§4.3). |
| `ubl/summary.ts` | `xml_summary.go` | **Structure copied literally.** RC and RA. |
| `ubl/sign.ts` | `sign.go` | **Reimplementation.** Go has no `xml-crypto` equivalent (§4.4). |
| `sunat/endpoints.ts` | `endpoints.go` | **Literal.** Beta and production URLs for `cpe`, `otros`, `consulta`. |
| `sunat/errors.ts` | `errors.go` | **Literal.** `0` accepted · `0100–1999` retryable · `2000–3999` rejected, never retried · `4000+` accepted with observations. |
| `sunat/soap.ts` | `soap.go`, `client.go` | **Literal.** `wsse:UsernameToken` envelope, and the detail that an HTTP 500 can carry a legitimate `Fault` in its body. |
| `sunat/cdr.ts` | `cdr.go` | **Literal**, on `encoding/xml` in read mode — safe here, only `ResponseCode`, `Description` and `Note` are read. |
| `sunat/zip.ts` | `zip.go` | **Trivial**, on `archive/zip`. |
| `security/certificate.ts` | `pkcs12.go` | **Reimplementation.** `node-forge` → in-repo reader + `crypto/x509` (§4.5). |
| `services/qr.ts` | `qr.go` | **String only.** The PNG is the consumer's problem. |
| `services/naming.ts` | `naming.go` | **Literal.** `RUC-TYPE-SERIES-NUMBER`, `RUC-RC-YYYYMMDD-###`, `RUC-RA-YYYYMMDD-###`. |
| `services/emitter.ts` | `emit.go` | **Reduced.** Only the pure part survives: build → sign → zip. Persistence, queue and storage were the Node service's job and are Genix's here. |
| `security/secrets.ts`, `repositories/*`, `http/*`, `pdf/*`, `storage/*`, `queue/*` | — | **Not ported.** Out of the library's scope by design. |

### 2.2 Taken from `greenter`

| Source | Use |
| --- | --- |
| `packages/xcodes/src/data/CodeErrors.xml` (1712 codes) | **Ported as data** into `error_codes.txt` + `//go:embed`. This is what turns rejection `2335` into an actionable message. |
| `packages/xml/src/Xml/Templates/*.twig` | Element-order reference while writing each generator. |
| `packages/validator/src/Validator/Loader/*` | Checklist of per-entity validations. |

### 2.3 SUNAT rules the port must preserve

Each cost a real rejection against BETA. Each goes in as a comment **and** a test:

1. **Element order.** The XSDs enforce it: in factura/boleta `cac:Signature` comes **before**
   `cac:AccountingSupplierParty`. One element out of place is a rejection.
2. **Daily summary:** the `RC-YYYYMMDD-###` id is built from the **summary date**, not the
   generation date — otherwise rejection `2346`.
3. **Perception:** `percentage` is a factor (`0.02` = 2 %) written verbatim into
   `MultiplierFactorNumeric`; dividing by 100 is rejection `2798`.
4. **Credit Note (07):** its monetary totals declare only `PayableAmount`.
5. **CDATA** in `RegistrationName`, item `Description` and `Note`.
6. When a reference template and the web service disagree, **SUNAT wins**.

### 2.4 What gets deleted from this repo

`backend/invoicing/invoice.go` and `invoice_types.go` (inherited from `backend/billing/`) model the
UBL with `encoding/xml` struct tags. **Removed**, for two unfixable reasons: `encoding/xml` controls
neither attribute order, nor prefix declarations, nor self-closing tags, so its output is not
canonical and could never be signed; and UBL element order is conditional, which struct tags cannot
express. Their replacement lives in facturago.

---

## 3. Integration idiom — mirroring `genix-orm`

### 3.1 Wiring

```
.gitmodules
  [submodule "backend/facturago"]
      path = backend/facturago
      url = git@github.com:ivanjoz/facturago.git
      branch = main

backend/go.mod
  require github.com/ivanjoz/facturago v0.0.0
  replace github.com/ivanjoz/facturago => ./facturago
```

Single Go module — unlike `genix-orm`, which is split into root + `db` + `dynamo` because it has
swappable drivers. facturago has one implementation of one thing, so a second module would be
indirection with no payoff.

The submodule tracks `main`, so the parent needs no pointer bump; edits inside `backend/facturago`
must be **committed and pushed inside the submodule** (`AGENTS.md` §7). Builds read the checked-out
files through `replace`, so what is on disk is what compiles — the same loop as the ORM.

`AGENTS.md` §7 (repo map and the submodules note) and the `backend/` line get updated to list
facturago as the third submodule.

### 3.2 Configuration handoff

Genix "sets the configuration and facturago does the rest", in the same shape as
`scylla.SetScyllaConnection`: one call at boot, in `backend/main.go`, next to the DB connection.

```go
// backend/main.go — process-wide defaults, set once.
facturago.Configure(facturago.Config{
    Timeout: 45 * time.Second,      // per SUNAT call
    Logf:    core.Logf,             // nil = silent
})
```

Everything tenant-specific is passed **per call**, never configured globally: a Lambda serves many
companies and a cached issuer would leak across tenants. That is the one deliberate difference from
the ORM, whose connection genuinely is process-wide.

---

## 4. facturago — the library

### 4.1 Repository layout

```
facturago/
  go.mod                  module github.com/ivanjoz/facturago — no require block at all
  README.md  LICENSE      public-facing; README derived from this section
  doc.go                  package doc: the pipeline in one paragraph

  types.go                Issuer, Document, Line, Party, Address, Totals, DocType, Environment
  catalogs.go             tributos, catalog 07 → affectation, catalogs 01/06
  money.go                round / format2 / formatUpTo
  dates.go                Peru date and time formatting
  number_to_words.go      legend 1000
  totals.go               CompleteTotals + ValidateDocument

  xml_writer.go           canonical writer (C14N 1.0 by construction)
  xml_common.go           shared blocks: signature, supplier, customer, TaxSubtotal
  xml_sale.go             Invoice / CreditNote / DebitNote
  xml_summary.go          SummaryDocuments (RC) and VoidedDocuments (RA)

  pkcs12.go               .pfx/.p12 → private key + certificate
  sign.go                 XMLDSig: digest, SignedInfo, RSA-SHA256, ds:Signature
  emit.go                 Emit(): build → sign → zip
  naming.go  qr.go        file names and the normative QR string

  client.go               Client: SendBill · SendSummary · GetStatus · GetStatusCDR
  soap.go  cdr.go  zip.go  endpoints.go
  errors.go               Severity, Classify, IsRetryable, SunatError
  error_codes.go          //go:embed error_codes.txt
  error_codes.txt         1712 codes, "code<TAB>description"

  testdata/               golden XML per document type, two .pfx fixtures, sample CDRs
  *_test.go               unit + golden tests
  beta_test.go            live BETA test, behind an env guard
  example_test.go         godoc examples — these are the documentation
```

Flat, grouped by domain, greppable file names. No `internal/`, no interfaces, no constructors that
only wrap a struct literal: `AGENTS.md` §4 applies to this repo too.

### 4.2 Public API

Five types and a handful of standalone functions. The whole library is usable without reading its
source:

```go
// Configure sets process-wide defaults. Optional: the zero config is usable.
func Configure(cfg Config)

// Issuer is who signs and sends. Built per emission by the consumer; never cached by facturago.
type Issuer struct {
    RUC, LegalName, TradeName string
    Address                   Address
    SolUser, SolPassword      string
    PKCS12                    []byte // raw .pfx as uploaded
    PKCS12Password            string
    Environment               Environment // Beta | Production
}

// Document is one CPE to issue. Correlativo is allocated by the consumer.
type Document struct {
    Type        DocType // Factura | Boleta | CreditNote | DebitNote
    Series      string  // "F001"
    Correlativo int64
    IssuedAt    time.Time
    Currency    string
    Customer    Party
    Lines       []Line
    AffectedDoc *AffectedDoc // notes only
    Totals      Totals       // optional: CompleteTotals fills what is absent
    Legends     []Legend
    Detraction  *Detraction
    Perception  *Perception
    Prepayments []Prepayment
    Charges, Discounts []Charge
}

// ── pure: no network, no disk ────────────────────────────────────────────────
func CompleteTotals(doc *Document) error
func ValidateDocument(doc *Document) []error          // pre-flight, before spending a SUNAT call
func ParsePKCS12(pfx []byte, password string) (*Credential, error)
func BuildXML(iss Issuer, doc Document) ([]byte, error)
func Sign(xml []byte, cred *Credential) (*Signed, error)
func Emit(iss Issuer, doc Document) (*Emission, error) // CompleteTotals → validate → build → sign → zip
func FileName(ruc string, t DocType, series string, correlativo int64) string
func QRText(iss Issuer, doc Document, digest string) string

// Emission is everything the consumer must persist.
type Emission struct {
    FileName string // 20123456789-01-F001-123
    XML      []byte // signed, canonical
    Digest   string // DigestValue — the CPE hash, also the QR's last field
    Zip      []byte
}

// ── network ──────────────────────────────────────────────────────────────────
type Client struct{ HTTP *http.Client } // zero value works
func (c *Client) SendBill(iss Issuer, e *Emission) (*CDR, error)
func (c *Client) SendSummary(iss Issuer, e *Emission) (ticket string, err error)
func (c *Client) GetStatus(iss Issuer, ticket string) (*Status, error)
func (c *Client) GetStatusCDR(iss Issuer, t DocType, series string, n int64) (*Status, error)

// ── classification ───────────────────────────────────────────────────────────
func Classify(code string) Severity        // Accepted | Observed | Rejected | Failed
func IsRetryable(err error) bool           // a rejection is never retryable: the XML is invalid
func ErrorDescription(code string) string  // the embedded 1712-code catalog
```

`Emit` is the whole pure pipeline in one call; the individual steps stay exported because Genix
needs `BuildXML` alone for a preview, and `ParsePKCS12` alone to validate a certificate at upload.

### 4.3 Canonical XML by construction

The piece everything else depends on. XMLDSig does not sign bytes, it signs the **canonical form**
(C14N 1.0, `REC-xml-c14n-20010315`). Rather than implement a canonicalizer, the writer **emits XML
that is already canonical**, so the verifier's C14N is the identity transform and the digest closes
by construction. `xml_writer.go` is ~150 lines with these invariants:

1. UTF-8, `\n` endings, no XML declaration inside the signed body (it is prepended when the file is
   stored; C14N excludes it).
2. No comments, no processing instructions, no whitespace between elements.
3. **Never** self-closing tags: `<cbc:Note></cbc:Note>`, never `<cbc:Note/>`.
4. Attributes sorted: namespace declarations first by prefix, then the rest by (namespace URI, local
   name).
5. Exact C14N escaping — text: `&`, `<`, `>`, `\r`; attributes: `&`, `<`, `"`, `\t`, `\n`, `\r`.
   **`>` is not escaped in attributes**, `'` is never escaped.
6. Every prefix (`cac`, `cbc`, `ds`, `ext`, `sac`) is declared at the root and every one is used, so
   C14N has no superfluous declaration to prune.
7. CDATA is **expanded** by C14N, so text nodes are written already-escaped instead of wrapped in
   `<![CDATA[…]]>`, which would yield a different digest than the verifier computes.

One test per invariant, one golden file per document type.

### 4.4 XMLDSig (`sign.go`)

Enveloped RSA-SHA256 signature, SHA-256 digest, inserted into
`ext:UBLExtensions/ext:UBLExtension/ext:ExtensionContent`.

```
1. Build the document with an empty <ext:ExtensionContent></ext:ExtensionContent>.
2. DigestValue = base64(SHA256(document))   ← enveloped transform: no ds:Signature present yet.
3. Build ds:SignedInfo with that reference (URI="").
4. SignatureValue = base64(RSA-PKCS1v15(SHA256(c14n(SignedInfo)))).
5. Insert ds:Signature into ExtensionContent. No other byte moves, so step 2 stays valid.
```

**The trap:** inclusive C14N propagates **every in-scope namespace declaration** onto the apex node
of the signed subtree. Canonicalizing `ds:SignedInfo` for signing therefore requires adding the
declarations inherited from the root (the document's default namespace plus `cac`, `cbc`, `ds`,
`ext`), sorted by prefix — even though they do **not** appear on `SignedInfo` in the stored
document. This is the number-one cause of "invalid signature"; SUNAT's verifier adds them back.

`ds:KeyInfo` carries `X509SubjectName` and `X509Certificate` (base64 DER, no PEM headers). All of it
resolves with `crypto/rsa`, `crypto/sha256` and `crypto/x509`.

### 4.5 PKCS#12 (`pkcs12.go`)

`golang.org/x/crypto/pkcs12` is frozen and only understands PBE with RC2/3DES, so it fails on modern
`.pfx` files that use PBES2 + AES. The reader lives in the repo, on `encoding/asn1`:

- `ContentInfo` → `AuthenticatedSafe` → `keyBag` / `pkcs8ShroudedKeyBag` / `certBag`.
- **PBKDF1-SHA1** (the PKCS#12 legacy KDF) for 3DES/RC2, **PBKDF2-HMAC-SHA1/SHA256** for PBES2 with
  AES-128/192/256-CBC. PBKDF2 is ~20 lines over `crypto/hmac`, so `x/crypto/pbkdf2` is not needed.
- MAC verification against the password, so a wrong password says "incorrect password" instead of
  surfacing as corrupt DER.
- The leaf is the certificate whose public key matches the private key (chained `.pfx` files also
  carry intermediates).
- `Credential` exposes `RUC` (subject OID `2.5.4.5`), `Subject`, `Issuer`, `Serial`, `NotBefore`,
  `NotAfter` — everything a consumer needs to validate an upload and warn about expiry.

`GenerateSelfSigned(ruc, name, password)` produces a BETA-only test certificate, using the same
PKCS#12 code in write mode.

### 4.6 Tests

- **Unit**: totals (the 14 cases already validated in BETA — taxed, exempt, free, global and
  per-line discount, prepayments, detraction, export, ICBPER, ISC, IVAP, contingency, and an invoice
  with 5 affectation types), amount in words, rounding, code classification.
- **Golden files**: the XML of each type, byte for byte. A writer change that moves one byte breaks
  the test — exactly right, since it would also break the signature.
- **Canonicality**: property tests over the writer.
- **Signature**: verified with `crypto/rsa.VerifyPKCS1v15` over the produced document, plus a
  comparison against an XML signed by `xml-crypto` with the same test key.
- **PKCS#12**: one 3DES fixture and one AES/PBES2 fixture.
- **BETA** (`beta_test.go`, guarded by `FACTURAGO_BETA=1`, skipped by default): issues against the
  SUNAT test environment with the public credentials `20000000001` / `MODDATOS`. It is the only test
  that counts: nothing is done until it returns `code 0`.

Known BETA quirks that are **not** bugs: `getStatus` answers `Failed to establish a backside
connection`, and valid submissions intermittently get `401`. Both classify as retryable transport.

---

## 5. Genix side — `backend/invoicing/`

```
backend/invoicing/
  PLAN.md
  main.go                  ModuleHandlers + init() (cron-action and exec registration)
  types/
    company_secrets.go     CompanySecrets  / CompanySecretsTable   (table 26)
    invoice_document.go    InvoiceDocument / InvoiceDocumentTable  (table 42)
    invoice_series.go      InvoiceSeries   / InvoiceSeriesTable    (table 53)
    invoice_summary.go     InvoiceSummary  / InvoiceSummaryTable   (table 54)
  issuer.go                CompanySecrets → facturago.Issuer (decrypt, per call)
  sale_order_to_cpe.go     SaleOrder + ClientProvider + Product + Site → facturago.Document
  emitter.go               allocate → Emit → SendBill → persist artifacts and state
  emit_worker.go           async exec handler + cron-action retry handler
  invoice_api.go  invoice_series_api.go  company_secrets_api.go
```

Eleven files instead of the thirty the single-repo plan needed: catalogs, money, dates, words,
totals, the four XML files, sign, pkcs12, soap, cdr, zip, qr, naming and the error catalog all moved
to facturago.

### 5.1 Tables

Schema IDs **26, 42, 53, 54**, confirmed by `check_tables` rather than by reading — two earlier
candidates collided with tables a grep had missed.
Paired-struct convention from the `create-database-tables` skill; validated with
`cd scripts && go run . check_tables`.

**`CompanySecrets` — table 26, `company_secrets`.** SOL credentials and the **PKCS#12 in a column**,
encrypted at rest. One row per credential set; the autoincrement `ID` allows certificate rotation,
leaving the old one as history (`Status = 0`).

```go
type CompanySecrets struct {
    db.TableStruct[CompanySecretsTable, CompanySecrets]
    CompanyID int32  `json:",omitempty"`
    ID        int32  `json:",omitempty"`
    // 1 = SUNAT CPE credentials. Leaves room for other per-company secrets.
    Type      int8   `json:",omitempty"`
    Name      string `json:",omitempty"`

    // Secondary SOL user: in clear, because on its own it authenticates nothing.
    SolUser string `json:",omitempty"`
    // Encrypted with core.Encrypt (AES-256-GCM). These never leave the backend.
    SolPasswordEnc  []byte `json:"-"`
    CertificateEnc  []byte `json:"-"` // the .pfx/.p12 exactly as uploaded
    CertPasswordEnc []byte `json:"-"`

    // Filled from facturago.ParsePKCS12 at upload time: lets the UI warn about expiry
    // without decrypting the key material again.
    CertSubject, CertIssuer, CertSerial, CertRUC string
    CertValidFrom, CertValidTo                   int32 // SUnixTime

    // 1 = beta, 2 = production. Chooses the endpoint and blocks production while the
    // certificate is self-signed.
    Environment int8 `json:",omitempty"`

    Status         int8  `json:"ss,omitempty"`
    Updated        int32 `json:"upd,omitempty"`
    UpdatedVersion int32 `json:"upv,omitempty"`
    UpdatedBy      int32 `json:",omitempty"`
    Created        int32 `json:",omitempty"`
    CreatedBy      int32 `json:",omitempty"`
}

func (e CompanySecretsTable) GetSchema() db.TableSchema {
    return db.TableSchema{
        ID:                 26,
        Name:               "company_secrets",
        Partition:          e.CompanyID,
        Keys:               db.Cols(e.ID.Autoincrement(0)),
        SaveUpdatedVersion: true,
        FixedValues:        []db.FixedValues{{Col: e.Status, Values: []int64{0, 1}}},
        Indexes: []db.Index{
            // Resolves "the active CPE credentials of this company" in one read.
            {Type: db.TypeLocalIndex, Keys: db.Cols(e.Type)},
        },
    }
}
```

A typical `.pfx` is 3–6 KB, so the row fits comfortably in a Scylla cell and saves an S3 round trip
per emission.

**`InvoiceDocument` — table 42, `invoice_document`.** **The correlativo is allocated by the ORM**:
the series is packed into the key and Scylla's `counter`-backed sequence hands out numbers per
series, atomically — no distributed lock.

```go
Keys:              db.Cols(e.ID),
KeyIntPacking:     db.Cols(e.DocTypeSeries.DecimalSize(6), e.Autoincrement(8)),
AutoincrementPart: e.DocTypeSeries,
```

`DocTypeSeries = DocType*1000 + SeriesID` (§5.2), `Correlativo = ID % 1e8`. The SUNAT identity of the
document **is** its primary key, so a double submission cannot duplicate numbering.

| Column | Type | Note |
| --- | --- | --- |
| `CompanyID`, `ID` | `int32`, `int64` | partition and packed key |
| `DocTypeSeries`, `DocType`, `SeriesID`, `SeriesCode`, `Correlativo` | | `DocType` = 1, 3, 7, 8 |
| `IssueDate`, `IssueTime` | `int16` UnixDay, `int32` SUnixTime | from `core.FechaUnix()` |
| `SaleOrderID` | `int64` | the originating sale |
| `ClientID`, `ClientDocType`, `ClientDocNumber`, `ClientName` | | copied at emission: the CPE is immutable even if the client record changes |
| `Currency` | `int8` | 1 = PEN, 2 = USD |
| `TotalAmount`, `TaxAmount`, `TaxableAmount`, `ExemptAmount`, `UnaffectedAmount`, `FreeAmount` | `int64` | **cents**, like the rest of the ERP |
| `DetailProductIDs`, `DetailQuantity`, `DetailUnitValue`, `DetailValue`, `DetailIgvAmount`, `DetailIgvType`, `DetailUnitCode`, `DetailDescription` | parallel slices | singular naming per `AGENTS.md` §9 |
| `AffectedDocID`, `NoteReasonCode`, `NoteReason` | | 07/08 only |
| `State` | `int8` | 0 voided · 1 pending · 2 queued · 3 accepted · 4 accepted with observations · 5 rejected · 6 exception |
| `SunatCode`, `SunatDescription`, `SunatNotes` | | from the CDR |
| `DigestValue` | `string` | `Emission.Digest`; feeds the QR |
| `XmlPath`, `CdrPath` | `string` | S3 keys |
| `RetryCount`, `LastError` | | |
| audit | | `Status`, `Updated`, `UpdatedVersion`, `UpdatedBy`, `Created`, `CreatedBy` |

No serialized-payload column: a retry re-sends the signed XML already in S3, which is the exact
artifact SUNAT must receive — rebuilding it could yield a different digest. Only a document that
never reached the signing step is rebuilt from its row.

```go
FixedValues: []db.FixedValues{{Col: e.State, Min: 0, Max: 6}},
Indexes: []db.Index{
    // Free: range scan over the packed-key prefix. Lists a whole series.
    {Type: db.TypeInheritFromKey, Keys: db.Cols(e.DocTypeSeries), UseIndexGroup: true},
    {Type: db.TypeLocalIndex, Keys: db.Cols(e.IssueDate)},
    {Type: db.TypeLocalIndex, Keys: db.Cols(e.SaleOrderID)}, // "was this sale invoiced?"
    {Type: db.TypeDelta, Keys: db.Cols(e.State)},            // frontend incremental sync
},
```

**`InvoiceSeries` — table 53, `invoice_series`.** The series a company has enabled: which are valid,
which site they belong to, and the real alphanumeric code. `ID = DocType*1000 + SeriesID`, with
`SeriesID` (1..999) assigned locally — the SUNAT code (`F001`, `B001`, `FC01`) is alphanumeric and
cannot serve as a numeric key. Columns: `CompanyID`, `ID`, `DocType`, `SeriesID`, `SeriesCode`,
`SiteID`, `WarehouseID`, `IsDefault`, `Status`, `Updated`, `UpdatedVersion`, `UpdatedBy`.

**`InvoiceSummary` — table 54, `invoice_summary`.** RC and RA, which SUNAT makes asynchronous by
design: `sendSummary` returns a ticket, polled with `getStatus`. Packed `ID`:
`ReferenceDate.DecimalSize(5)` + `Autoincrement(3)` with `AutoincrementPart: e.ReferenceDate` — the
`###` of `RC-YYYYMMDD-###`. RC and RA share the day counter (SUNAT only requires the identifier not
to repeat; sharing leaves harmless gaps and saves a key column). Columns: `CompanyID`, `ID`, `Type`
(1 = RC, 2 = RA), `ReferenceDate`, `IssueDate`, `XmlName`, `Ticket`, `State`, `SunatCode`,
`SunatDescription`, `DocumentIDs []int64`, `XmlPath`, `CdrPath`, `RetryCount`, `LastError`, audit.

### 5.2 New columns on existing tables

Phase 1 runs on company-level defaults; these land with the UI that fills them (phase 6). None is
blocking:

| Table | Column | Purpose |
| --- | --- | --- |
| `Site` | `SunatAnnexCode string` | `codLocal` of the fiscal address (`0000` = main). |
| `ClientProvider` | `DocumentType int8` | Catalog 06. Until then **derived**: 11 digits → RUC (`6`), 8 → DNI (`1`), else `0`. |
| `Product` | `SunatUnitCode string` | Catalog 03 (`NIU`, `KGM`, `ZZ`…). Default `NIU`. |
| `Product` | `SunatIgvType int8` | Catalog 07. Default `10`. This is what enables selling exempt or unaffected. |

The **ubigeo already exists**: `Site.CityID` is the 6-digit INEI code, and the codebase already
treats it as such (`CityID/100` = province, `CityID/10000` = department) — exactly what
`cbc:ID` of `RegistrationAddress` wants.

### 5.3 Emission pipeline

```
POST.invoice  (sale-order-id, doc-type, series-id)
   │
   ├─ validate access, company, sale not yet invoiced, credentials valid
   ├─ SaleOrderToDocument() → facturago.Document
   ├─ facturago.CompleteTotals + ValidateDocument   ← fails BEFORE spending a SUNAT call
   ├─ db.Insert(InvoiceDocument)                    ← the ORM assigns the correlativo (atomic)
   ├─ cloud.ExecLambda{FuncToExec: "fn-emit-cpe", InvokeAsEvent: true}
   └─ returns the document in State = 1 (pending)

fn-emit-cpe   (async invoke; also the body of the cron-action retry)
   ├─ load InvoiceDocument + CompanySecrets
   ├─ IssuerFromSecrets(): core.Decrypt → facturago.Issuer, in memory, never cached
   ├─ facturago.Emit(issuer, doc)                   → Emission{FileName, XML, Digest, Zip}
   ├─ S3: cpe/{companyID}/{yyyymm}/{fileName}.xml
   ├─ facturago.Client.SendBill(issuer, emission)   → CDR
   ├─ S3: cpe/{companyID}/{yyyymm}/R-{fileName}.zip
   └─ map facturago.Classify(cdr.Code) → State, persist code, description, digest, paths
        ├─ facturago.IsRetryable(err) → core.ScheduleCronAction(action 5, docID) at 5 / 15 / 30 min
        └─ rejection (2000–3999) → State = 5, never retried: that XML will always be invalid
```

**Why an async invoke and not only the cron-action:** cron frames are aligned to 5 minutes, and a
point of sale cannot wait that long for its CDR. `cloud.ExecLambda` with `InvokeAsEvent` starts the
emission immediately (locally it POSTs back to the backend, so both runtimes behave alike); the
cron-action is the **retry net**, where it earns its keep — it survives a panic, caps at 10 attempts
and leaves an auditable row.

**A retry keeps its correlativo.** The row is persisted before signing, so a transport failure never
burns a number.

### 5.4 API and access

| Method.route | Behavior |
| --- | --- |
| `GET.company-secrets` | Lists secrets **without key material**: type, SOL user, environment, certificate metadata. |
| `POST.company-secrets` | Create/update. Opens the `.pfx` with `facturago.ParsePKCS12`, checks validity and that its RUC matches the company, encrypts, stores. |
| `POST.company-secrets-test` | Smoke test: `GetStatusCDR` for a non-existent document. Confirms SOL credentials without issuing anything. |
| `GET.invoice-series` / `POST.invoice-series` | Series per document type and site. |
| `GET.invoices` | Listing with delta (`upv`) and filters by date, series, state. |
| `GET.invoice-by-ids` | Detail by IDs (`fetch-record-by-id-api` pattern). |
| `GET.invoice-xml` | Downloads the signed XML or the CDR from S3. |
| `POST.invoice` | Issues from a `SaleOrder`. Returns the document in pending state. |
| `POST.invoice-retry` | Re-queues a document in exception state, same correlativo. |
| `POST.invoice-void` | Voids: RA for facturas, RC with state `3` for boletas. |
| `POST.invoice-note` | Credit/debit note over an accepted document. |
| `GET.invoice-summary-status` | Polls an RC/RA ticket. |

Access entries in `backend/access_list.yml` (group 7 "Contabilidad"; IDs 1–34 are taken):

```yaml
  - id: 35
    name: "Facturación Electrónica"
    group: 7
    levels: 14
    frontend_routes: "accounting/invoicing,accounting/invoice-series"
    backend_apis: "POST.invoice,POST.invoice-retry,POST.invoice-void,POST.invoice-note,POST.invoice-series,GET.invoices,GET.invoice-xml"
  - id: 36
    name: "Credenciales SUNAT"
    group: 1
    levels: 14
    frontend_routes: "configuration/company-secrets"
    backend_apis: "GET.company-secrets,POST.company-secrets,POST.company-secrets-test"
```

`GET.company-secrets` and `GET.invoice-xml` are mapped deliberately: an unmapped GET stays open to
any session, and neither of these can be.

**IDs to reserve:** cron-action `5` (1–4 are in use); exec handler `fn-emit-cpe`.

### 5.5 Security

1. **The certificate never leaves the backend.** No endpoint returns it, not even encrypted; the
   columns carry `json:"-"`.
2. **`core.Decrypt` has a bug** (`core/helpers.go:1477`): given an explicit key it ignores it and
   uses `MakeCipherKey()`. Nothing passes an explicit key today, so nothing is broken, but this
   module must not lean on that parameter. Fixed along the way.
3. The master key is `SECRET_PHRASE`. Rotating it invalidates every stored secret — documented next
   to the deployment notes.
4. Logs redact `SolPassword`, `CertPassword` and the certificate. facturago's `Logf` hook never
   receives the SOAP envelope whole: it carries the SOL password in the `wsse` header.
5. Every `InvoiceDocument` read filters by the token's `CompanyID`; the issuer RUC is compared with
   the authenticated company before signing.
6. `POST.invoice` takes the **lock service** (`core.AcquireLock`) keyed by `(action, saleOrderID)`:
   two clicks cannot produce two documents for one sale.

---

## 6. Frontend and route documentation

Three feature folders following `AGENTS.md` §5 — kebab-case, business rules split out of reactive
state so Vitest can reach them with no component and no network:

```
frontend/routes/accounting/invoicing/
  +page.svelte              Page shell, tabs, wiring. No business logic
  InvoiceDetailLayer.svelte Side panel: CDR, SUNAT notes, XML/CDR download
  InvoiceEmitModal.svelte   Issue from a sale: series, document type, client check
  invoicing.svelte.ts       $state + service calls
  invoicing.ts              Pure rules: actions allowed per state, note eligibility,
                            rejection code → user message, displayed totals
  invoicing.utils.ts        Generic formatting, no business logic
  invoicing.test.ts         Vitest over invoicing.ts
  DOCUMENTATION.md          page_id: accounting.invoicing

frontend/routes/accounting/invoice-series/      … page_id: accounting.invoice-series
frontend/routes/configuration/company-secrets/  … page_id: configuration.company-secrets
```

Connector in `frontend/services/` per `SERVICES_GUIDE.md`: `invoices` is a cached (delta) service
watermarked on `upv`; `invoice-xml` is a plain report call. Menu entries go in
`frontend/core/modules.ts` with the bilingual `English|Español` naming already used there.

**Route documentation** follows the `document-user-routes` skill — read
`backend/agent/PLAN_ROUTE_DOCUMENTATION.md` first, start from the skill's
`DOCUMENTATION.template.md`, and mirror the granularity of
`frontend/routes/finance/cash-banks/DOCUMENTATION.md`. Specific to this module:

- Stable `page_id`s as above; one `DOC-ID` per capability (`capability.emit-invoice`,
  `capability.void-document`, `capability.upload-certificate`…), kept stable across edits.
- Written **from verified evidence**, tracing frontend → service → handler → business function, and
  only after phase 5 works against BETA. A plan is not evidence.
- Domain vocabulary the mixed prose needs: `comprobante`, `boleta`, `factura`, `nota de crédito`,
  `serie y correlativo`, `CDR`, `certificado digital`, `clave SOL`, `resumen diario`,
  `comunicación de baja`, `anulación`, `rechazado`, `observado`, `ambiente beta`.
- Page-specific rules belong there — a rejection cannot be retried, a correlativo is never reused, a
  voided boleta needs a daily summary. Generic tenancy, permissions and cent storage stay out.
- Validate without indexing:
  `./deploy.sh index_documentation -mode validate -document frontend/routes/accounting/invoicing/DOCUMENTATION.md`.
  Qdrant ingestion only on explicit request.

---

## 7. Phases

| # | Where | Deliverable | Verification |
| --- | --- | --- | --- |
| 0 | both | **Bootstrap.** `facturago` skeleton (go.mod, README, LICENSE, doc.go) → first commit → push to the empty remote. Submodule added at `backend/facturago`, `replace` in `backend/go.mod`, `.gitmodules` and `AGENTS.md` §7 updated. **Pushing is outward-facing: needs your go-ahead.** | `cd backend && go build ./...` |
| 1 | facturago | Domain: `types.go`, `catalogs.go`, `money.go`, `dates.go`, `number_to_words.go`, `totals.go` + tests. | `go test ./...` inside the submodule |
| 2 | facturago | Crypto: `xml_writer.go` (+ canonicality tests), `pkcs12.go` (+ 3DES and AES/PBES2 fixtures), `sign.go` verified against a reference signature. | `go test ./...` |
| 3 | facturago | XML and transport: `xml_sale.go`, `xml_summary.go`, `emit.go`, `client.go`, `soap.go`, `cdr.go`, `zip.go`, `errors.go`, `error_codes.go`, `qr.go`, `naming.go`. Golden files per type. **Live BETA test green.** | `go test ./...`, then `FACTURAGO_BETA=1 go test -run TestBeta` |
| 4 | Genix | ✅ The 4 tables + deletion of `invoice.go` / `invoice_types.go`. | `go build ./...`, `go vet ./...`, `cd scripts && go run . check_tables` |
| 5 | Genix | ✅ `issuer.go`, `sale_order_to_cpe.go`, `emitter.go`, `emit_worker.go`, the three API files, access entries, `sunat.Configure`. **Verified end to end:** boletas B001-2 and B001-3 issued from real sales and accepted by SUNAT beta with code 0. | `go build ./...` + a real emission against BETA from the ERP |
| 6 | Genix | Notes, RC and RA: `POST.invoice-note`, `POST.invoice-void`, the ticket cycle on cron-action. | `go test ./invoicing/...` + BETA |
| 7 | Genix | Frontend: the three feature folders, the connector, menu entries, and the SUNAT columns on `Product`, `ClientProvider`, `Site`. | `cd frontend && bun run check`, `bun run build`, skill `agent-browser` |
| 8 | Genix | One `DOCUMENTATION.md` per route, from shipped behavior. | `./deploy.sh index_documentation -mode validate` |

Phases 1–3 are self-contained inside facturago and can be finished and pushed before Genix touches
the library at all — which is the point of splitting the repo: the hard, testable part has no ERP
dependencies. Every phase is a commit that builds and passes its checks; submodule phases are
committed **and pushed inside `backend/facturago`**.

---

## 8. Risks and open points

1. **Boletas: individual submission or daily summary?** The web service accepts both and
   `sunat-service` validated both in BETA. The regulation talks about the daily summary; most
   providers now send each boleta individually for an immediate CDR. The plan implements individual
   submission and keeps RC for voiding — **worth confirming with your accountant before phase 5**.
2. **Amounts crossing the boundary.** Both sides are integer cents, so nothing is converted: the
   ERP's `int32` cents are what `facturago.Cents` holds. Quantities and rates are scaled integers
   too (`Units(3)`, `Percentage(18)`). Where a line does not divide into whole cents — wholesale,
   ten thousand items at S/ 0.0847 — the mapper passes the line total and the library derives the
   unit value SUNAT reads by exact long division.
3. **Certificate expiry.** An expired `.pfx` rejects every emission. Phase 5 includes a daily cron
   warning at 30 and 7 days, using `CertValidTo`.
4. **SUNAT latency inside the async invoke.** The worker Lambda needs a timeout ≥ 60 s (SUNAT can
   take 40 s) — to be checked in the deployment configuration.
5. **The `.pfx` living in the row.** 3–6 KB per company is negligible, but if long-chain
   certificates appear the blob moves to S3 and the row keeps only the path. Nothing in facturago
   changes: it receives bytes either way.
6. **facturago is public.** The repo will carry SUNAT knowledge and nothing of Genix — no ERP
   identifiers, no tenancy, no business rules from this repo. Worth keeping in mind on every commit
   to the submodule; a leak there is a leak in public.
7. **Language of user-facing errors.** `AGENTS.md` mandates English for code, comments and docs;
   `CREATE_API_HANDLERS.md` says handler error messages are Spanish for consistent user feedback,
   and every existing handler does that. This plan keeps handler messages in Spanish to match the
   product surface, while facturago — a public library — returns English errors. Say the word if you
   want the handler layer switched or routed through `tr()`.
