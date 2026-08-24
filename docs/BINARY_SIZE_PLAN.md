# Binary Size — Plan

Status: **Phases 0, 1 and 2 are implemented and verified. Phase 3 is partially implemented (write
path and row scan done, outer read path outstanding). Phase 4 was built, measured negative and
reverted. Phase 5 was rejected on inspection — it breaks two live API routes.**

Implemented: **44,236,960 → 29,819,040, −14,417,920 (−32.6%)**. What landed: `backend/thirdparty/{protobuf,gocql}` (see its `README.md` and
`RATIONALE.md`), the two `replace` directives in `backend/go.mod`, and the shipping build tags in
`cloud/main.go` and `scripts/deploy_vps.go`.

One change from the plan as written: open decision #2 resolved *better* than either option offered.
Chain 2 is cut by routing `internal/impl`'s proto1 probes through `reflect.Value.MethodByName`
instead of `reflect.Type.MethodByName` — a concrete struct method registers no interface methodsig,
which is the entire mechanism. Same method, same zero receiver, **no proto1 loss and no
type-assertion rewrite**. The caveat in Phase 1 below no longer applies.

Phases 3–5 remain projections from symbol inventories and are labelled as such.

Baseline: **44,236,960**, the tree at `c132e325`, which already includes both phases of
`backend/genix-orm/COL_INSTANTIATION_PLAN.md`. Build used everywhere below:

```sh
cd backend && GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -ldflags '-s -w' -o /tmp/main .
```

This document supersedes `backend/genix-orm/BINARY_SIZE_FINDINGS.md` §3 and §7, which name the wrong
cause for the largest item. The corrections are in §7 below; the rest of that document (the cost
model, the killed hypotheses H1/H2, the `Col[*T, E]` measurements) still stands.

---

## 1. Where the bytes are

`.text` is under half the binary. Any lever that duplicates *functions* is paid for roughly twice,
because `.gopclntab` carries a name and pc-value tables for every one of them.

| Section | Baseline | After Phases 0–2 | Delta |
| --- | --- | --- | --- |
| `.text` | 19,671,412 | 14,275,300 | −5,396,112 |
| `.gopclntab` | 17,408,302 | 11,735,859 | −5,672,443 |
| `.go.type` | 4,786,360 | 4,017,296 | −769,064 |
| `.rodata` | 1,469,593 | 1,384,729 | −84,864 |

genix-orm is 18,408 of 52,591 symbols and **3,387,390 of 5,588,581 bytes of symbol names (61%)**.
Counting `.text` alone, as the earlier analysis did, understates every lever here by about 2×.

---

## 2. Root cause: two independent chains disable dead-method elimination

The linker keeps a single program-global `reflectSeen` flag
(`cmd/link/internal/ld/deadcode.go:462`). While it is set, **every exported method of every reachable
type is retained**, so the ORM's 258 `Col` instantiations and 54 `TableStruct` instantiations carry
their full method sets whether or not anything calls them. It is boolean: every source must go or
nothing changes.

Both live sources run through protobuf, which enters the binary through exactly one importer —
`github.com/qdrant/go-client/qdrant` is the **sole** importer of both grpc and protobuf in the whole
dependency graph.

**Chain 1 — `descfmt` plus a Go 1.27 addition to `reflect`.**
Go 1.27 added the iterators `reflect.Value.Methods()` and `reflect.(*rtype).Methods()`. They call
`v.Method(i)` with a non-constant index and are absent from the exclusion list in
`cmd/compile/internal/walk.usemethod` (`expr.go:1026`), so they carry `AttrReflectMethod`. Then
`protobuf/internal/descfmt/stringer.go:265` does `rv.MethodByName("Methods")`. A *constant* name
emits `R_USENAMEDMETHOD`, and the linker's `genericIfaceMethod` map is keyed **by name only** — so it
retains every exported method named `Methods` on every reachable type, `reflect.Value.Methods`
included. Reaching that symbol sets `reflectSeen`.

**Chain 2 — `xunsafe` plus protobuf's `internal/impl`.**
`xunsafe.Field` (`field.go:20`) embeds the `reflect.Type` *interface*, so the compiler generates a
promoted `MethodByName(string)` wrapper whose argument is not constant → `AttrReflectMethod`. It
becomes reachable because `protobuf/internal/impl` calls `MethodByName` **through the `reflect.Type`
interface** when probing for proto1 legacy hooks, which registers that methodsig in `ifaceMethod`,
which matches the wrapper's signature.

Measured — neither chain alone changes anything:

| Build | Binary |
| --- | --- |
| Baseline | 44,236,960 |
| `-tags lambda.norpc,grpcnotrace` + gocql fork | 43,581,600 |
| + chain-1 patch only (unmodified xunsafe) | 43,581,600 — no change |
| + chain-2 patch only (unmodified descfmt) | 43,581,600 — no change |
| + **both** | **32,178,336** |

`text/template` (via gocql's `recreate.go`) is **not** a flipper: `ToCQL` is unreachable, so
`evalField` never links. It is only dead weight.

---

## 3. Phases

Ordered by dependency, not by size. Phases 0–2 are measured; 3–5 are projected and only become
worth their cost **after** Phase 1, which is what turns the ORM's concrete method sets back into
shape stencils.

| Phase | Change | Saving | Status |
| --- | --- | --- | --- |
| **0** | `-tags lambda.norpc,grpcnotrace` | −393,216 | **shipped** |
| **1** | Vendored protobuf fork — cuts both chains | **−11,403,264** | **shipped** |
| **2** | gocql fork without `recreate.go` | −262,144 | **shipped** |
| **3** | Type-erase the scylla engine below `Executor` | **−2,359,296 so far**; ~1.5 MB left | **partially shipped** |
| **4** | Split `TableStruct`; pointer-shape the query half | **+65,536 — measured negative** | **rejected** |
| **5** | `ScyllaController` / deploy surface behind a build tag | ~0.6 MB + pclntab share | **rejected — breaks live routes** |

Phases 0–2 together: **44,236,960 → 32,178,336, −12,058,624 (−27.3%)**, with **no application code
changed and genix-orm untouched.** Reproduced from a clean `backend/thirdparty/regenerate.sh` run.

Phase 3 so far: **32,178,336 → 29,819,040, −2,359,296.** `executeInsertUpdateBatch` fell from
766,236 bytes across 378 stencils to nothing measurable, and `scanSelectQueryRows` from 219,193
across 108 stencils to 3,633 bytes in 2 symbols. Design decisions are in
`backend/genix-orm/scylla/RATIONALE.md`.

**Still generic, and the remaining ~1.5 MB of Phase 3** — the outer read path, which allocates and
merges intermediate `[]T` slices and so needs the real element type: `executeBoundSelectQueries`
(196,951), `selectRecordsByPartitionIDs` (222,210), `execIndexGroupQuery` (218,700), `Merge`
(218,104), `preloadExistingRecordsBySingleKey` (179,561), `QueryCachedIDs` (165,913),
`db.InitStructTable` (168,152), `CsvToRecords`. Erasing these needs the sink to build new
destination slices at runtime (`reflect.MakeSlice`), which is a different and slower shape than
`recordSink` — worth doing, but not a continuation of the same edit.

Verified on the implemented tree: `go build ./...` and `go vet ./...` clean on `backend`, `cloud` and
`scripts`; `genix-orm` and `genix-orm/db` module tests pass; `accounting`, `libs`, `finance`, `exec`
and `core` tests pass; `check_module_imports` reports no violations across 44 packages;
`check_tables` reports **"Found 53 table struct pairs"**, identical to before.

---

### Phase 0 — build tags

Free, no code change, ship independently of everything else.

- `-tags lambda.norpc` — AWS's own flag. Drops the legacy local-RPC entry path, not the Lambda
  runtime. Removes `net/rpc`, an independent `reflectSeen` source.
- `-tags grpcnotrace` — grpc's own flag (`trace_notrace.go`). Removes `golang.org/x/net/trace` and
  with it `html/template`.

Add both to `deploy.sh` and to every build path that produces a shipping binary, or they silently
apply to some builds and not others.

`grpcnotrace` becomes moot if Track B (§5) is taken instead of Phase 1. `lambda.norpc` is needed
either way.

**Verify:** `GOOS=linux GOARCH=arm64 go list -tags 'lambda.norpc,grpcnotrace' -deps . | grep -xE 'net/rpc|html/template|golang.org/x/net/trace'` returns nothing.

---

### Phase 1 — vendored protobuf fork

The whole of §2, cut by one fork. `replace` is module-granular, so the module is forked even though
the diff is six lines.

**Placement:** `backend/thirdparty/protobuf/`, with `replace google.golang.org/protobuf => ./thirdparty/protobuf`
in `backend/go.mod`. A directory replace needs no `go.sum` entry.

`go build ./...` and the deploy build then produce the same binary. The `-modfile=go.prod.mod`
alternative keeps git cleaner but makes development builds differ from shipping builds, which
defeats the point of the verification table in `CLAUDE.md` §6 — rejected for that reason.

**Bulk.** Trimmed to the 33 packages the build actually reaches, with `_test.go` and `testdata`
stripped:

```
1.6 MB · 146 .go files · 39,385 lines     (upstream module: 15 MB, 491 files)
```

Do not forget `internal/editiondefaults/editions_defaults.binpb` — it is `go:embed`-ed and the
build fails without it.

**The diff.**

`internal/descfmt/stringer.go:265` — behaviour-identical, produces the same method value:

```go
// before
{rv.MethodByName("Methods"), "Methods"},
// after
{reflect.ValueOf(t.Methods), "Methods"},
```

`internal/impl/message.go` (2 sites) and `internal/impl/legacy_message.go` (3 sites) — the proto1
legacy probes for `XXX_OneofFuncs`, `XXX_OneofWrappers` and `ExtensionRangeArray` route to a stub in
a new `internal/impl/zz_legacy_lookup.go`:

```go
func legacyOneofLookupDisabled() (reflect.Method, bool) { return reflect.Method{}, false }
```

**Superseded caveat.** The first working version of this patch stubbed the three probes out, which
cut the chain but silently removed proto1 legacy oneof and extension-range discovery. As shipped it
does not: the probes go through `reflect.Value.MethodByName` on a zero value of the same type, which
is a concrete struct method and so registers no interface methodsig, while calling the same method on
the same receiver. Identical binary size, no semantic loss, nothing to decide.

Stubbing the whole `descfmt` file instead of the one line is worth a further −196,608 (the ~45 other
constant method names it registers, each retaining same-named methods program-wide). Not worth
losing descriptor formatting for.

**Verify.** A round-trip test on a real qdrant message exercises the patched `internal/impl` paths
and the `descfmt` path together — `proto.Marshal`/`Unmarshal`/`Equal`, `protojson.Marshal`, `%v` on
a descriptor and on a field list, and a oneof accessor. All pass on the forked tree. Keep that test
in the repo so a protobuf bump that drops the patch is caught.

**Upstream.** Both halves are reportable, and the first one matters far beyond this repo: as
written, *every Go binary that links protobuf loses dead-method elimination program-wide*. The Go
side — `reflect.Value.Methods` / `(*rtype).Methods` missing from `usemethod`'s exclusion list — looks
like a genuine regression from when those iterators were added. If either lands, drop the
corresponding half of the fork.

---

### Phase 2 — gocql fork without `recreate.go`

`recreate.go` holds package-level `template.Must(...)` vars and provides only
`KeyspaceMetadata.ToCQL()`, a scylla-manager helper that nothing in this repo references. Deleting
the file removes `text/template` entirely: **−262,144**.

Needs a fork of the existing `replace github.com/gocql/gocql v1.6.0 => github.com/scylladb/gocql v1.13.0`,
so it carries the same vendoring question as Phase 1 at 1/50th the payoff. Land it with Phase 1 or
not at all.

---

### Phase 3 — type-erase the scylla engine below `Executor`

*Projected.* Do not start before Phase 1: with `reflectSeen` set, concrete method sets dominate and
erasure buys a fraction of this.

After Phase 1, genix-orm still holds **5,125,520 bytes of text**, and the same code compiled exactly
once would be **469,230**. The gap — about **4.66 MB, plus its `.gopclntab` share** — is one engine
stencilled 54 times, once per `(table, record)` pair:

| Function | Bytes | Stencils | Per stencil |
| --- | --- | --- | --- |
| `scylla.executeInsertUpdateBatch` | 551,936 | 378 | 1,460 |
| `scylla.(*ScyllaController).*` | 513,040 | 636 | 806 |
| `scylla.appendUpdateQueriesToBatch` | 172,144 | 108 | 1,593 |
| `scylla.Merge` | 159,968 | 54 | 2,962 |
| `scylla.execIndexGroupQuery` | 158,112 | 54 | 2,928 |
| `scylla.selectRecordsByPartitionIDs` | 153,792 | 54 | 2,848 |
| `scylla.scanSelectQueryRows` | 153,120 | 108 | 1,417 |
| `scylla.handlePreInsert` | 141,120 | 54 | 2,613 |
| `db.InitStructTable` | 135,696 | 54 | 2,513 |
| `scylla.preloadExistingRecordsBySingleKey` | 134,416 | 54 | 2,489 |

**The type parameters are already vestigial.** `scanSelectQueryRows[E]` (`select.go:526`) uses `E`
for exactly two things — `new(E)` and the final `append` — because every column read and write
already goes through precompiled `func(unsafe.Pointer) any` accessors on the runtime descriptor. The
insert path is the same. Go stencils it 54 times only because `E` is a value-struct shape.

**Shape.** Keep the typed API exactly as it is and put one small generic veneer at the boundary:

```go
type recordSink struct {
	elemType reflect.Type
	elemSize uintptr
	base     func() unsafe.Pointer      // slice data pointer
	grow     func(n int) unsafe.Pointer // extend by n, return ptr to the first new element
	len      func() int
}

func makeRecordSink[E any](dst *[]E) recordSink { … }   // ~150 B × 54 ≈ 8 KB total
```

Everything below `Executor.Select` / `.Insert` takes `recordSink` + `ScyllaTable` and compiles once.

- **Zero query-API change.** `db.Query(&recs).X.Equals(1).Exec()` is untouched; the erasure happens
  below the `Executor` boundary.
- **Per-row cost** is one indirect call instead of an inlined `append`, against a CQL row decode.
  Measure it on the insert path before converting the read path.
- **Incremental.** Convert one function at a time and re-measure; there is no flag day.

---

### Phase 4 — split `TableStruct` — REJECTED, measured negative

Built and reverted. `db.(*TableStruct)` fell 287,601 → 190,182, but `db.(*queryBuilder)` arrived at
138,134 across **594 concrete symbols**, 11 per table. The shape collapse worked perfectly — **1
stencil, 3,242 bytes, for all 55 tables** — but promotion through embedding needs a concrete
forwarding wrapper per instantiation, so one symbol per method per table became two. Net **+44,320**
of ORM text, **+65,536** on the binary.

The lesson generalises: shape sharing pays only when the per-instantiation *body* is large. The same
move was worth 766,236 bytes in `scylla`, where bodies run 2,000–4,000 bytes, and is negative in
`db`, where they average 246. Full write-up in `backend/genix-orm/db/RATIONALE.md`, including the
`*T is pointer to type parameter` constraint rule that shapes any future attempt.

---

### Phase 5 — admin surface behind a build tag — REJECTED

The plan assumed `ScyllaController` was reachable only from CLI entrypoints. It is not.
`exec.ModuleHandlers` is wired into `main-handlers.go:29` and registers three live HTTP routes:

```go
"GET.backups":         GetBackups,
"POST.backup-restore": RestoreBackup,   // -> MakeScyllaControllers() at exec/restore.go:47
"POST.backup-create":  CreateBackup,    // -> SaveBackup -> MakeScyllaControllers() at exec/backup.go:24
```

Tagging the controllers out of the shipping binary would leave both write routes compiling,
deploying, and then finding zero tables to back up or restore. The ~480 KB is not worth losing
backup and restore over HTTP, so the controllers stay. Revisit only as part of the agent/admin split
in §5, where the endpoints would move rather than disappear.

---

## 4. Combined outlook

| Stage | Binary |
| --- | --- |
| Baseline | 44,236,960 |
| After Phases 0–2 (measured) | 32,178,336 |
| After Phase 3 write path + row scan (measured) | **29,819,040** |
| After the rest of Phase 3 (projected) | ~28 MB |

---

## 5. Track B — replace the qdrant Go client instead of forking protobuf

An alternative to Phase 1, not a complement. It is ~3 MB better and costs application code.

`qdrant/go-client` cannot work without protobuf — `qdrant.PointStruct`, `qdrant.Value`,
`qdrant.Filter` are `protoc-gen-go` output embedding `protoimpl.MessageState`, with no JSON
transport and no build tag. But the Qdrant *server* serves a feature-equivalent REST/JSON API on
6333, and since that client is the sole importer of grpc and protobuf, dropping it removes
**1,527,744 bytes of text** (qdrant 315,744 + grpc 524,288 + protobuf 687,712, ~3 MB with pclntab and
type shares) **on top of** the Phase 1 saving, and needs no vendored fork at all.

The scope is far smaller than the ~60 qdrant identifiers in the tree suggest, because the write path
already lives in separate binaries (`agent/cmd/documentation-index`, `agent/cmd/documentation-search`).
What the API binary reaches, via `agent/route_turn.go:57-67`, is three calls:

| Call | REST endpoint |
| --- | --- |
| `client.Query` (`agent/knowledge/search.go:168`) | `POST /collections/{c}/points/query` |
| `client.GetCollectionInfo` (`agent/knowledge/qdrant.go:145`) | `GET /collections/{c}` |
| `client.CollectionExists` (`agent/knowledge/qdrant.go:136`) | `GET /collections/{c}/exists` |

Everything else — `CreateCollection`, `CreateFieldIndex`, `Upsert`, `OverwritePayload`, `Delete`,
`ScrollAndOffset` — is `agent/knowledge/index.go` and `EnsureCollection`, i.e. the indexing path that
already runs out of process.

So the API binary needs one non-trivial REST call: the hybrid query — two prefetches (dense vector,
sparse `Document` with IDF) fused by RRF, a `must` filter of keyword and bool matches, and a
`with_payload` include list. Roughly 150–200 lines of structs plus `net/http`. `knowledge.Store` is
already the seam.

The open design question if this is taken: whether `Store` becomes two types (a read-only
`SearchStore` in the API, the full one in the indexer) or one interface with two implementations.

A third option, cheaper still in code and more expensive in operations, is to move the whole
agent/RAG path to its own deployable and have `route_turn.go` call it. No client to write; two
services to run.

---

## 6. Open decisions

These block landing and are not mine to make.

1. ~~Vendoring a third-party module into the tree at all~~ — **done.** `check_module_imports` needed
   no exclusion: it reports no violations across 44 packages, because the forks are nested modules
   and `./...` skips them.
2. ~~proto1 loss vs. type-assertion rewrite~~ — **resolved**, neither was needed. See the header.
3. **Phase 1 or Track B** — still open, but no longer blocking: Phase 1 shipped, and Track B would
   *replace* it (delete the protobuf fork) rather than add to it. Track B is worth ~3 MB more.
4. ~~Phase 5 and the agent split~~ — Phase 5 **rejected**: it breaks `POST.backup-restore` and
   `POST.backup-create`. The agent split (§5) is still open and is now the only route to that ~480 KB,
   because it moves the endpoints rather than deleting them.
5. **Phase 3/4 and the language version.** Generic methods exist in the go1.27 toolchain but need
   `go 1.27` as the *language* version; `backend/go.mod` declares `go 1.26` with
   `toolchain go1.27.0`. Neither phase needs them — the lever is pointer type arguments and
   type erasure, not generics — but the bump is a prerequisite if they are ever wanted.

Each phase lands with its own `RATIONALE.md` entry in the folder it touches, per `CLAUDE.md` §1.
Phases 1 and 2 additionally need a note in `backend/thirdparty/` recording what was patched and why,
because a protobuf or gocql bump that silently drops the patch costs 12 MB with no test failure —
which is exactly what the round-trip test in Phase 1 is for.

---

## 7. Corrections to `backend/genix-orm/BINARY_SIZE_FINDINGS.md`

Its §3 table names three `reflectSeen` sources. Two are wrong and two are missing:

| Claim in §3 | Reality |
| --- | --- |
| `aws-lambda-go` → `net/rpc`, free flag | Correct. |
| `gocql/recreate.go` → `text/template`, a flipper needing a fork | **Not a flipper.** `ToCQL` is unreachable so `evalField` never links. Still 262,144 of dead code. |
| `qdrant` → `grpc` → `x/net/trace` → `html/template`, "not a flag, architecture question" | **`-tags grpcnotrace` is grpc's own flag** and removes it for free. |
| — | **Missing: `protobuf/internal/descfmt`** (chain 1). |
| — | **Missing: `xunsafe.Field`'s promoted `reflect.Type` wrappers** (chain 2). |

Its §7 says the TODO is "~2.4 MB text + up to 1.6 MB names, measured on a 2-table program". On the
real tree it is **12,058,624 bytes**, and it is not an architecture question — it is a six-line
patch to one vendored dependency.

`COL_INSTANTIATION_PLAN.md`'s prediction that Phase A's value "multiplies by about 4× if the TODO is
ever resolved" is confirmed: `db.Col` falls from 1,672,704 to 313,152 (−81%) once pruning is on, and
that only happens because `Col[*T, E]` already collapsed the shapes.

---

## 8. Reproducing

```sh
# production target
cd backend && GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -ldflags '-s -w' -o /tmp/main .

# section sizes — .gopclntab is as large as .text; do not measure text alone
size -A /tmp/main

# symbol inventory (needs an unstripped build)
go build -o /tmp/uns . && go tool nm -size /tmp/uns > /tmp/syms.txt

# is dead-method elimination off? count concrete Col instantiations.
# ~8,256 means off, ~2,053 means on. Binary size alone is a poor signal: removing a
# template package shrinks the binary without flipping the flag.
rg 'db\.\(?\*?Col\[' /tmp/syms.txt | rg -v 'go\.shape' | wc -l

# who flips it: any non-constant reflect.{Type,Value}.Method / .MethodByName in a reachable
# package, plus any type embedding the reflect.Type interface
GOOS=linux GOARCH=arm64 go list -deps -f '{{.Dir}}' . | sort -u | while read -r d; do
  grep -HnE '\.MethodByName\(|\.Method\(' "$d"/*.go 2>/dev/null | grep -v _test.go
done

# linker edges into a retained method — shows the type descriptor that kept it alive
go build -ldflags '-dumpdep' -o /dev/null . 2>/tmp/dep.txt
rg 'UsedInIface.*-> .*db\.\(\*?Col\[' /tmp/dep.txt | head
```

---

## 9. Out of scope — do not retry

Carried forward from `BINARY_SIZE_FINDINGS.md`, all measured at zero or negative:

- **De-reflecting `InitStructTable`** (H1). Byte-identical result; would add `unsafe` pointer
  arithmetic to the schema binder for nothing.
- **Removing `Coln` boxing from schema declarations** (H2). Slightly negative. The linker counts the
  method set, not the boxing.
- **`Col[E]` with the table on the method.** Not expressible; `T` appears only in return position.
- **A public `ColRef` / `db.Mod()` declaration API.** Rejected: rewrites 54 declaration sites to
  recover ~765 KB that Phase 1 makes moot anyway.
- **Per-row hot paths.** Not where the bytes are. Phase 3 touches the row loop only through one
  indirect call and must be measured, not assumed.
- **Splitting `TableStruct` into a shape-collapsed `queryBuilder`.** Measured **+65,536**. The shape
  collapse works (1 stencil for 55 tables); the promotion wrappers cost more than the shared bodies
  save. See `backend/genix-orm/db/RATIONALE.md`.
- **Tagging `ScyllaController` out of the shipping binary.** Breaks `POST.backup-restore` and
  `POST.backup-create`, which are live routes via `exec.ModuleHandlers`.
- **`go build -overlay` for dependency patches.** Refuses any path under `GOMODCACHE`:
  `files beneath GOMODCACHE must not be replaced`. It works for GOROOT and for your own tree only.
