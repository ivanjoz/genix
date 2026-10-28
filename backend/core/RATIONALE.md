# RATIONALE — core

Design decisions, newest first.

## sonic runs from a fork that makes every field `omitempty` except pointers to numbers
**Context** — Every json tag in the project carried `omitempty`. Neither sonic nor `encoding/json/v2` has a global switch for it, and sonic 1.15.0 silently fell back to `encoding/json` on Go 1.27 (1.15.5 fixes that).
**Decision** — `go.mod` replaces `bytedance/sonic` with `github.com/ivanjoz/sonic` (branch `omitempty-default`, based on v1.15.5), which adds `sonic.SetOmitEmptyDefault`. `core/json_defaults.go` turns it on in `init()`. Every field is omitted when empty except `*int..`, `*uint..` and `*float..`: nil is `null`, a pointer to 0 is `0`. `*string`, `*bool`, `*slice`, `*map` and `*struct` are omitted when nil (`*bool` was not named in the request; it follows the "only *number is exempt" rule). A non-nil `*string` pointing to `""` is still sent. There is no opt-out tag. All `omitempty` tags were removed from the repo. The table generator and `check_tables` no longer expect `,omitempty` on `upd`/`ver`/`mdt`.
**Rationale** — The flag has to be set before the first encode because sonic caches compiled encoders per type, and `core` is the package everyone imports. It only reaches sonic: code that calls `encoding/json` directly now emits zero values and empty slices. Costs: a fork to rebase on each sonic release, and DynamoDB/SQLite JSON columns no longer store zero values (they read back as zero, so nothing changes on load).

## Every struct that carries `json` tags got `cb:"<json name>,<n>"` on every field
**Context** — colbin reads only `cb` tags and names a schema field by its Go name, but the
frontend reads the `json` names (`upd`, `ss`, …). Untagged, the self-describing message would
carry Go names and hash-derived ids that nobody controls.
**Decision** — A one-off AST pass added `cb:"<json name>,<n>"` to all fields of 191 structs that had
`json` tags and no `cb` yet (n counts from 1 in declaration order; `json:"-"` became `cb:"-"`;
embedded fields untouched). New response structs must carry the tag too.
**Rationale** — The names make the decoded objects match the JSON ones and the numbered ids keep
keys narrow. Cost: ids are positional, so reordering or inserting a field changes the wire, and
any colbin blob already stored (ORM blob columns, `ExportRecordsColbin` backups) of a re-tagged
type no longer decodes: reprocess or wipe it, pre-alpha. `DemoStruct4/5`, `embeddingResponse` and
`d1Result` have fields colbin cannot carry (`[]float32`, `[]map`, only-skipped) — none is a route
response.

## Responses are plain JSON unless `Accept` announces `application/vnd.colbin`
**Context** — minijson was unconditional (`?raw=1` was the only way out) and the frontend sent no
signal. Smartberry negotiates through `Accept`.
**Decision** — `MakeResponse` answers `colbin.MarshalSelfDescribing` with `IsBinary` and
`Content-Type: application/vnd.colbin` when `Accept` contains the media type, else sonic JSON.
`?raw=1` and `WantsRawJSON` are gone. A nil slice is colbin `null`; the frontend maps it to `[]`.
**Rationale** — `Accept` is CORS-safelisted (no preflight changes) and a wildcard never matches, so
curl, webhooks and tests keep plain JSON. minijson stays only for the published webpage snapshot.

## The DataFrame lock is a queuing fareward lock through a new AcquireLockWithOptions
**Context** — A read that finds a DataFrame partition due must wait up to 40 s for the compaction
under way, but `fareward.AcquireLock` fixes a 5 s wait and a 15 s lease.
**Decision** — fareward's Go client exports `AcquireLockWithOptions(ctx, action, identifier,
LockOptions)`. `acquireDataFrameLock` calls it with `ActionCompactDataFrame` (renamed from
`ActionMaterializeDataFrame`, same number 6), keyed on fnv64 of the frame partition's folder, the
wait and lease the ORM asks for (40 s each; the daemon clamps leases to 60 s) and at most 16
waiters per key. Past 16 a read is refused the lock and answers from the log, which stays correct.
**Rationale** — The daemon's own queue does the waiting: no polling of a lock timestamp. Costs one
exported function in the fareward repo, which must be pushed there.
