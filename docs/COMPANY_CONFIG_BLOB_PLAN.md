# Company config blob — plan

A company's full operating configuration, compacted into one encrypted `.cb` file in object
storage, so any API can load everything an emission needs without touching the database.

Status: **plan, not implemented**. Awaiting approval.

## 1. What forced the layout

Two constraints found while reading the code, both of which move things from where they'd
naturally go:

**`config/types` cannot write to S3.** `docs/MODULE_BOUNDARIES.md` L2: a `<module>/types`
package may import "L0, L1, other `<module>/types` — and nothing else, ever". `cloud` is L3.
So the store/load half cannot live in `config/types`, even though that is where the data
lives. It goes in `cloud/`, which is L3 and may import both `config/types` and `s3.go`.
Both callers (`config` and `invoicing`) are L4 and already import `cloud`.

**The signing material does not need the `.pfx`.** `facturago.Sign` takes a
`Credential{PrivateKey *rsa.PrivateKey, Certificate *x509.Certificate}` (`facturago/pkcs12.go:25`).
The PKCS#12 container, its MAC, its bag attributes, the CA chain and the certificate password
are all transport wrapping that `ParsePKCS12` throws away. So the blob can carry the private
key as PKCS#8 DER and the leaf certificate as DER — roughly 2.5 KB instead of 4–6 KB, and one
fewer secret to carry (`CertPasswordEnc` never enters the blob).

## 2. Files

| File | Layer | Holds |
| --- | --- | --- |
| `config/types/company_config.go` | L2 | `CompanyConfig` struct + `BuildCompanyConfig(companyID)` — the DB reads and assembly. Pure of storage and crypto. |
| `cloud/company_config_blob.go` | L3 | `CompactAndStoreCompanyConfig(companyID)`, `LoadCompanyConfig(companyID)`, the object key, colbin, AES-128-GCM. |
| `cloud/company_config_blob_test.go` | L3 | Round trip, key derivation vector, tamper rejection. |

`BuildCompanyConfig` is split out so the assembly is unit-testable without S3 or a cipher, and
so the layer rule is satisfied by construction rather than by discipline.

## 3. The record

Every blob-local type carries explicit numeric `cb` ids. Not for determinism — colbin's hashed
ids are already deterministic — but so a field **rename** cannot move an id, and so every type
stays inside colbin's 4-bit id window (all ids ≤ 14, `colbin/README.md`), which is why the
record is four small structs rather than one flat 22-field one.

```go
// config/types/company_config.go
type CompanyConfig struct {
    Version       int8                     `cb:"1"` // blob format; an unknown value is refused
    CompanyID     int32                    `cb:"2"`
    SourceUpdated int32                    `cb:"3"` // max Updated of the rows it was built from
    GeneratedAt   int32                    `cb:"4"` // core.SUnixTime()
    Company       CompanyConfigCompany     `cb:"5"`
    Sunat         CompanyConfigSunat       `cb:"6"`
    Culqi         CompanyConfigCulqi       `cb:"7"`
    Parameters    []CompanyConfigParameter `cb:"8"`
}

type CompanyConfigCompany struct {
    RUC               string `cb:"1"`
    LegalName         string `cb:"2"`
    TradeName         string `cb:"3"`
    Address           string `cb:"4"`
    City              string `cb:"5"`
    Phone             string `cb:"6"`
    Email             string `cb:"7"`
    NotificationEmail string `cb:"8"`
}

type CompanyConfigSunat struct {
    SolUser        string                    `cb:"1"`
    SolPassword    string                    `cb:"2"` // decrypted; the blob itself is encrypted
    PrivateKeyDER  []byte                    `cb:"3"` // PKCS#8 DER from the active .pfx
    CertificateDER []byte                    `cb:"4"` // leaf X509 DER
    CertRUC        string                    `cb:"5"`
    CertValidTo    int32                     `cb:"6"`
    Environment    int8                      `cb:"7"`
    Series         []invoicing.InvoiceSeries `cb:"8"`
}

// Backend-side keys only. PubKeyLive/PubKeyDev are the browser's half and are absent.
type CompanyConfigCulqi struct {
    RsaKey   string `cb:"1"`
    RsaKeyID string `cb:"2"`
    KeyLive  string `cb:"3"`
    KeyDev   string `cb:"4"`
}

// A parameters row, minus what the blob already knows (CompanyID) and what no
// reader needs (Status, Updated, UpdatedBy).
type CompanyConfigParameter struct {
    Group    int32   `cb:"1"`
    Key      string  `cb:"2"`
    Value    string  `cb:"3"`
    ValueInt int32   `cb:"4"`
    Values   []int32 `cb:"5"`
}
```

**`invoicing.InvoiceSeries` deliberately gets no `cb` tags, and neither does `CulqiConfig`.**
`genix-orm/scylla/converter.go:784` colbin-marshals any non-`[]byte` struct column, so both of
those are *already on disk* encoded with hashed ids. Adding explicit ids would reassign them and
make every stored company row undecodable. `InvoiceSeries` is embedded in the blob as-is: its
ids hash from its field names, so a rename would shift them — but a rename already breaks the
database column today, so the blob inherits that fragility rather than adding a new one.
`CulqiConfig` is not embedded at all; `CompanyConfigCulqi` above copies the four fields the
backend actually uses.

Sources: `companies` (identity, `InvoiceSeries`, `CulqiConfig`), `company_secrets` (the **active**
row only — `LoadActiveSecrets`' selection), `parameters` (all rows for the company).

The three encrypted columns are decrypted with `core.Decrypt` during the build. The blob is a
single sealed unit; nested encryption would mean two keys to rotate for one artifact.

## 4. Object key

`c{companyID}_cf_{tag}.cb`, e.g. `c42_cf_1kQ9x3FfA2w.cb`.

`tag` is the fareward MAC strategy (`fareward/go/siphash`, already a dependency):

```go
hasher := siphash.New(siphash.DeriveKey([]byte(core.Env.SECRET_PHRASE)))
hasher.WriteString("company-config-v1")           // domain separation
binary.Write(hasher, binary.BigEndian, companyID)
tag := base64.RawURLEncoding.EncodeToString(binary.BigEndian.AppendUint64(nil, hasher.Sum64()))
```

64-bit SipHash-2-4 output, big-endian, base64url without padding — 11 characters. Deterministic,
so a reader computes the name and does one GET; unguessable without the key, so listing or id
enumeration yields nothing.

Keyed on `SECRET_PHRASE`, same as the AES key below. **Consequence to accept:** rotating
`SECRET_PHRASE` orphans every blob (both the name and the key). Readers fall back to a rebuild
on miss, so the system self-heals on first read — it does not break, it just pays one rebuild
per company.

Path: `company-config/` prefix in the configured bucket, `ContentType: application/octet-stream`,
`CacheControl: no-store`.

## 5. Encryption — AES-128-GCM

A new helper beside the blob code, not `core.Encrypt` (which is AES-256-GCM):

- Key: `sha256(SECRET_PHRASE + "company-config-v1")[:16]` — 16 bytes, AES-128.
- Nonce: 12 random bytes, prepended to the ciphertext, same layout as `core.Encrypt`.
- AAD: the object key string. This binds a blob to its filename, so a blob copied over another
  company's object fails to open instead of decrypting into the wrong company's config.

Wire format: `[nonce:12][ciphertext+tag]`. The colbin payload is the plaintext.

## 6. Write-through

`CompactAndStoreCompanyConfig(companyID)` is called after a successful commit in:

| Handler | File |
| --- | --- |
| `PostEmpresa` | `config/empresas.go:13` |
| `PostEmpresaParametros` | `config/empresas.go:141` |
| `PostParametros` | `config/parametros.go:27` |
| `PostInvoiceSeries` | `invoicing/invoice_series_api.go:19` |
| `PostCompanySecrets` | `invoicing/company_secrets_api.go:80` |

A failed upload **logs and does not fail the request**. The user's save already committed;
refusing it afterwards would be a lie about what happened, and the next save — or the first
reader, below — repairs the blob.

## 7. Read

`LoadCompanyConfig(companyID) (*config.CompanyConfig, error)`:

1. GET the object; decrypt; colbin-unmarshal; check `Version`.
2. On miss, on a decrypt failure, or on an unknown `Version`: rebuild via
   `CompactAndStoreCompanyConfig` and return the fresh record.

That fallback is what makes the cache safe to be wrong: a missed write-through trigger, a
rotated key or a format bump costs one rebuild, never a failed emission. No TTL — the blob is
authoritative until a mutation replaces it.

## 8. facturago change (submodule)

`model.Issuer` gains `PrivateKeyDER []byte` and `CertificateDER []byte`; `signAndPack`
(`facturago/emit.go:66`) builds the `Credential` from those when `PKCS12` is empty, and keeps
the existing `.pfx` path otherwise. Generic and consumer-agnostic, which is what that repo
requires. Must be committed **and pushed inside the submodule**.

`invoicing/issuer.go` then builds its `model.Issuer` from `LoadCompanyConfig` instead of three
`core.Decrypt` calls — one GET replacing a company read, a secrets read and three decrypts.

## 9. Not in scope

- No `Content-MD5`/ETag concurrency control: two saves racing write the same derived content.
- No cross-region replication or lifecycle rules on the prefix.
- Backend validation of `IssueSeriesID` on sale creation stays open (separate TODO).

## 10. Settled

- §8 (facturago + `invoicing/issuer.go`) is **in scope** for this change, so the blob is
  exercised by the live emission path rather than shipping unread.
- Numeric `cb` ids on the blob-local types; nothing tagged that the ORM already persists.
- Keyed on `SECRET_PHRASE`.
