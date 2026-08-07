# Plan: derive the cloud mirror schema from `db.TableSchema` and drop the `col:` tag

> **Status: executed.** Five deviations from the plan as written, all forced by what the
> schema compiler actually accepts — see "Deviations during execution" at the end.

## Goal

1. Remove the `col:"..."` struct tag from record structs. Column names, partition/sort keys and
   secondary indexes for the cloud mirror (DynamoDB / Cloudflare D1) come from the table struct's
   `GetSchema()` — the same declaration the Scylla driver already uses.
2. Fix the live self-hosted bug: `UserTable` has no `PasswordHash` column, so `db.Insert` never
   persists it and `PostLogin` (`backend/security/login.go:122`) can never match a password when
   `BACKEND_PROVIDER=none`.
3. Add real Scylla indexes on `User` / `Email` so the self-hosted login stops doing an
   `AllowFilter()` partition scan.

## Why this is not a tag deletion

`backend/cloud/orm-meta.go:87` parses `col:` into `ColumnMeta{ColumnName, IsPK, IsSK, IsIndex,
DynamoIndex}`. That metadata is what builds the DynamoDB item keys (`orm-dynamodb.go:84-107`), the
D1 `CREATE TABLE` / `PRIMARY KEY` / `CREATE INDEX` statements (`orm-sqlite.go:229-269`) and the GSI
resolution in `Exec()` (`orm-dynamodb.go:263`). Deleting the tags with no replacement would leave
the mirror with no keys at all, rename three columns that call sites query by literal string, and
start mirroring the plaintext `Password` field (today excluded by `col:"-"`).

## Current state

Structs carrying `col:` tags:

| Struct | Embeds `db.TableStruct` | Used by the cloud ORM | Index slots in use |
| --- | --- | --- | --- |
| `coretypes.User` | yes | yes | ix1 `user`, ix2 `email`, ix3 `company_usuario`, ix4 `company_status_updated` |
| `securityTypes.Profile` | yes | yes | ix1 `company_updated`, ix2 `company_status_updated` |
| `configTypes.Company` | yes | yes | ix1 `ruc`, ix2 `email`, ix3 `updated` |
| `agentTypes.AgentMessage` | yes | **no** — nothing in `backend/agent` imports `app/cloud` | — |
| `exec.TestUsuario` | **no** | yes (`exec/test_cloud_orm.go`) | ix1 `email` |

`cloud/template.yml:350-390` declares exactly four shared string GSIs (`ix1`–`ix4`) plus one numeric
(`in5`) on the single Dynamo table. Slots are assigned per struct in tag declaration order, so the
derived order must stay ≤ 4 per struct.

The three `*Index` fields (`CompanyUserIndex`, `CompanyStatusIndex`, `CompanyUpdatedIndex`) are not
real data — they are zero-padded composite strings hand-built by `PrepareCloudSync()` so that a
Dynamo GSI (string-valued) supports a range scan. They are exactly what a `db.Index` of
`{Type: TypeView, Keys: Cols(Status.DecimalSize(1), Updated.DecimalSize(10))}` expresses, computed
by hand.

## Design

### 1. Thread the table type into the cloud ORM

`cloud.Insert[T any]` cannot reach `T`'s schema: `GetSchema()` on the record's embedded
`TableStruct` returns an empty `TableSchema` (`genix-orm/db/tablestruct.go:108`); only the *table*
struct overrides it, and the compiled column names only exist after `InitStructTable`.

Use the inference trick `db.Query` already uses (`backend/db/db.go:73`) — constrain the record type
so Go infers the table type from it:

```go
func Insert[RecordT db.Record[TableT, RecordT, D], TableT db.Schema[TableT], D db.Executor[TableT, RecordT]](
    records []RecordT) error
```

Call sites keep their current shape (`cloud.Insert([]coretypes.User{body})` still infers). Inside,
`db.MakeSchema[RecordT, TableT]()` returns a fully compiled `TableSchema` with resolved column
names. Same treatment for `Init`, `GetByID`, `Select`, and the `ORM[T]` / `QueryBuilder[T]`
interfaces plus `DynamoORM` / `SqliteORM`.

### 2. Replace `parseColumns` with `buildColumnsFromSchema`

`orm-meta.go` keeps `ColumnMeta` but fills it from the schema instead of the tag:

- **Column set** = the fields of the *table* struct, not the record struct. This is the key
  semantic change: the mirror stores exactly the ORM's columns. `Password` is excluded because it
  is not in `UserTable`; `PasswordHash` is included once added (step 4).
- **Column names** = `ColumnInfo.Name` (already honours `db:"..."` overrides and snake_case).
- **`IsPK`** = `schema.Partition`.
- **`IsSK`** = `schema.Keys` (all current mirrored tables have exactly one key column, so the
  existing "only one sk" rule holds; assert it rather than silently taking the first).
- **Table name** = `schema.Name`, replacing `toSnakeCase(structName)`. Note this renames the D1
  tables: `user` → `users`, `profile` → `profiles`, `company` → `companies` (see Migration).
- **Hash prefix** = keep `getStructHashPrefix(structName)` as-is so existing Dynamo `pk` values
  stay valid.

### 3. Derive the GSIs from `schema.Indexes`

Each entry in `schema.Indexes` becomes one `ixN` slot, in declaration order. The mirror computes
the attribute value the same way `PrepareCloudSync` does today:

```
value = <partition value> "_" <key1> "_" <key2> ...
```

with each key rendered as `%0*d` when the column declares `.DecimalSize(n)`, raw otherwise. The
partition prefix is included unless the index overrides `Index.Partition`. This reproduces
byte-for-byte:

- `fmt.Sprintf("%d_%s", CompanyID, User)` → index `{Keys: Cols(User)}`
- `fmt.Sprintf("%d_%d_%020d", CompanyID, Status, Updated)` → index
  `{Keys: Cols(Status.DecimalSize(1), Updated.DecimalSize(10))}`

`PrepareCloudSync()` and the three synthetic `*Index` record fields are then deleted, along with
their `PrepareCloudSync()` call sites.

### 4. Query resolution in `Exec()`

Call sites stop naming synthetic columns and name real ones instead. `Exec()` matches the set of
conditions to an index whose keys are pinned in order — all `Equals` except optionally a trailing
range operator, which becomes a composite `BETWEEN` over the padded prefix.

| Call site | Today | After |
| --- | --- | --- |
| `login.go:97` | `Where("empresa_id").Equals(id).Where("company_usuario").Equals(concat)` | `Where("company_id").Equals(id).Where("user").Equals(body.User)` |
| `usuarios.go:112` | `Where("empresa_id")…Where("company_status_updated").GreaterEqual(concat)` | `Where("company_id").Equals(id).Where("status").Equals(1).Where("updated").GreaterEqual(updated)` |
| `perfiles.go:30` | `Where("company_updated").GreaterEqual(…)` | `Where("empresa_id").Equals(id).Where("updated").GreaterEqual(…)` |
| `perfiles.go:33` | `Where("company_status_updated").GreaterEqual(…)` | `Where("empresa_id").Equals(id).Where("status").Equals(1).Where("updated").GreaterEqual(…)` |
| `empresas.go:46` | `Where("updated").GreaterEqual(…)` | unchanged (now padded — see Migration) |

`splitQueryConditions` / `findLogicalPartitionColumn` in `query_conditions.go` keep their role;
`findLogicalPartitionColumn` reads `IsPK` which is now schema-derived.

### 5. Schema declarations

`coretypes.User` — drop every `col:` tag; drop `CompanyUserIndex`, `CompanyStatusIndex` and
`PrepareCloudSync`. `UserTable` gains `PasswordHash db.Col[UserTable, string]`. `GetSchema()`
gains:

```go
Indexes: db.Cols… // in this order, so ix1=user, ix2=email, ix3=status+updated
{Type: db.TypeLocalIndex, Keys: db.Cols(e.User)},
{Type: db.TypeLocalIndex, Keys: db.Cols(e.Email)},
{Type: db.TypeView, Keys: db.Cols(e.Status.DecimalSize(1), e.Updated.DecimalSize(10))},
```

`Password` stays on the record with `json:"-"`-style handling as today (it is write-only input,
zeroed at `usuarios.go:221`) and is simply absent from `UserTable`.

`securityTypes.Profile` — drop `col:` tags, the two `*Index` fields and `PrepareCloudSync`; add the
two matching indexes to `ProfileTable.GetSchema()` (which currently declares none).

`configTypes.Company` — drop `col:` tags; add indexes for `RUC`, `Email`, `Updated`.

`agentTypes.AgentMessage` — drop the `col:` tags outright; nothing mirrors this table.

`exec.TestUsuario` — does not embed `db.TableStruct`, so it cannot satisfy the new constraint.
Convert it into a real paired `TestUsuario`/`TestUsuarioTable` declaration (the smoke test in
`exec/test_cloud_orm.go` otherwise stops compiling).

### 6. Self-hosted login

With `TypeLocalIndex` on `User`, drop `AllowFilter()` from the Scylla branch at `login.go:100-102`.

## Migration impact (mirror deployments only)

Self-hosted (`BACKEND_PROVIDER=none`, the current `credentials.json`) is unaffected apart from the
`PasswordHash` column addition and the new indexes.

For `aws` / `cloudflare` deployments these are **data-shape changes and need a mirror rebuild**:

- D1 table names change (`user` → `users`, `profile` → `profiles`, `company` → `companies`).
- D1 column `empresa_id` → `company_id` for `User` only. Keeping `empresa_id` is not an option: the
  name would have to come from a `db:"empresa_id"` tag, which would also rename the *Scylla*
  column, which today is `company_id`. `Profile` already carries `db:"empresa_id"` and is unchanged.
- The synthetic `company_usuario` / `company_status_updated` / `company_updated` columns disappear;
  their values move into the `ixN` attributes, which is where Dynamo already stored them.
- `Company.updated` GSI values become zero-padded. Today `empresas.go:46` compares unpadded integer
  strings lexicographically, which is wrong for mixed-width values; padding fixes it but invalidates
  previously written index values.
- Dynamo `pk`/`sk` values are unchanged (same hash prefix, same key columns), so the base items
  survive; only the `ixN` attributes need rewriting, which a re-`Insert` of every row does.

`cloud/template.yml` needs no change — slot count stays within ix1–ix4 for every struct.

## Files touched

- `backend/cloud/orm-meta.go` — `parseColumns` → `buildColumnsFromSchema`; composite index-value builder
- `backend/cloud/orm-core.go` — generic signatures on `Init` / `Insert` / `GetByID` / `Select`, `ORM[T]`, `QueryBuilder[T]`
- `backend/cloud/orm-dynamodb.go` — schema-derived keys, index matching in `Exec`
- `backend/cloud/orm-sqlite.go` — schema-derived DDL and upsert
- `backend/cloud/query_conditions.go` — index matching helper
- `backend/core/types/users.go` — tags, `PasswordHash` column, indexes, remove `PrepareCloudSync`
- `backend/security/types/perfiles.go` — same
- `backend/config/types/empresas.go` — same
- `backend/agent/types/agent_messages.go` — remove dead `col:` tags
- `backend/security/login.go`, `backend/security/usuarios.go`, `backend/security/perfiles.go`,
  `backend/config/empresas.go`, `backend/exec/init.go`, `backend/exec/test_cloud_orm.go` — call sites

## Verification

1. `go build ./...` and `go vet ./...` in `backend/`.
2. `cd scripts && go run . check_tables` (the `static-project-validation` skill).
3. Self-hosted round trip: create a user via `PostUsuarios`, then `PostLogin` — this is the bug in
   goal 2 and currently fails.
4. `exec/test_cloud_orm.go` smoke test against a configured provider, if one is reachable.

## Deviations during execution

1. **D1 never builds a composite string.** The composite is a DynamoDB-only workaround for the
   single-table design's four shared string GSIs. D1 indexes real columns, so an index declaration
   becomes `CREATE INDEX ... ON table(partition, key1, key2)` and a query is a plain WHERE over the
   real columns. This left `sqliteQueryBuilder.Exec` essentially unchanged.
2. **Index-key widths are inferred when the schema cannot declare them.** `compileSchemaView`
   rejects `.DecimalSize()` on a view's *leading* column — it derives that width from the columns
   after it. The mirror still needs an explicit width per component to pad with, so
   `inferKeyDigits` falls back to the widest decimal form of the Go type (`int8`→3, `int32`→10,
   `int64`→19). Consequence: `status` pads to 3 digits, so a User index value reads
   `7_001_0000001234` rather than the old `7_1_0000000000000001234`. Both sides of the mirror build
   it through the same function, so the format is self-consistent.
3. **Company's RUC and Email indexes were dropped, not ported.** `TypeLocalIndex` dereferences the
   table's partition column, and Company has none (it is global), so a local index panics at
   compile time. A global index would have worked, but nothing in the codebase ever queries a
   company by RUC or email — those two GSI slots were declared and never used. Company now has one
   index, the `updated` delta view.
4. **`exec/test_cloud_orm.go` was deleted rather than converted.** `RunCloudORMTest` was never
   called from anywhere — not registered as an `fn-` command, not referenced in any test. It was a
   scratch smoke test for the `col:` tag mechanism this change removes.
5. **Added `cloud/orm-meta_test.go`.** The composite index value is now a storage contract, so its
   format, the ix-slot assignment and the range bounds are pinned by tests. It also compiles all
   three mirrored schemas, which is what caught deviations 2 and 3.

## Remaining operational step

`password_hash` is a new column. Run `./deploy.sh 6` (or `5` then `fn-init`) so `DeployScylla`
issues the `ALTER TABLE ... ADD` and `fn-init` re-seeds the admin/system users with a hash that now
persists. Until that runs, self-hosted login stays broken exactly as it is today.

For an `aws`/`cloudflare` deployment the mirror also needs a rebuild: the D1 table names, the User
partition column name and every `ixN` value changed. Since the mirror is a mirror — the primary
database is the source of truth — the simplest path is to drop the D1 tables, re-run `cloud.Init`,
and re-`Insert` users, profiles and companies.
