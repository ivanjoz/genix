# PLAN — Extract `backend/db` into the `genix-orm` repo as a git submodule

**Status:** awaiting approval — no code changed yet.

## Goal

Move `backend/db` (~17,900 LOC, 42 files + `text_search/`) out of the genix repo into
`github.com/ivanjoz/genix-orm` as the `scylla` package, then consume it back from genix as a
git submodule at `backend/genix-orm/`.

## Approved decisions

| Decision | Choice |
|---|---|
| Package clause | `package scylla`, in directory `genix-orm/scylla/` (the empty `cassandra/` dir gets renamed) |
| Call sites | `db.X` → `scylla.X` (1447 occurrences, 84 files) — **not** an import alias |
| colbin | Whole backend migrates to `github.com/ivanjoz/colbin`; `backend/libs/colbin` is deleted |
| Submodule location | `backend/genix-orm/` + `replace github.com/ivanjoz/genix-orm => ./genix-orm` |

## Verified facts (measured, not assumed)

- `backend/db` imports **only two** internal packages: `app/db/text_search` (moves with it) and
  `app/libs/colbin` (`converter.go:16`, `reflect_accessors.go:11`).
- colbin is used only as `colbin.Marshal` / `colbin.Unmarshal` on `[]byte` — **no colbin types
  appear in any `db` signature**, so the import swap is a 2-line change.
- `backend/libs/colbin` vs `github.com/ivanjoz/colbin`: identical except doc comments, an added
  `doc.go`, and test/README drift. The standalone version is a **superset** (adds a single-value
  layout beside the columnar records layout). Wire-compatible.
- Config is injected via `db.SetScyllaConnection(ConnParams{...})` (`connection.go:29`). No app
  globals, no `go:embed`, no relative paths. Sole env read: `os.Getenv("MAX_CLUSTERING_KEY")`
  (`select_helpers.go:221`).
- External deps: `gocql`, `viant/xunsafe`, `fatih/color`, `kr/pretty`, `golang.org/x/sync/errgroup`.
- 84 files outside `db` import `app/db`; 57 distinct `db.X` symbols; 1447 call sites.
- No identifier named `db` or `scylla` exists outside `db/` → `\bdb\.` rewrite has no shadowing
  hazard. `dynamodb.` does **not** match `\bdb\.` (no word boundary). One string-literal hit to
  fix by hand: `exec/test_selects.go:97`.
- `cloud/`, `db-backup/`, `p2p/` (separate modules) do **not** import `app/db` — unaffected.
- `genix-orm/go.sum` has **no gocql entry** — must be added.

## Open questions to settle at review time

1. **Git history** — plain copy loses `backend/db`'s history. Preserving it needs
   `git filter-repo`/`git subtree split` into a branch that is then merged into genix-orm.
   *Proposal: plain copy.* The history stays reachable in genix. Say so if you want it preserved.
2. **Stale plan docs in `db/`** — `MIGRATE_TO_PREPARED_WRITES_PLAN.md`,
   `SELECT_STATEMENT_PRECOMPUTE_PLAN.md`, `SONIC_INDEX_PLAN.md` are completed-work plans.
   *Proposal: move `ORM_INTERNALS.md` to `genix-orm/scylla/`, delete the three plan docs*
   (AGENTS.md: pre-alpha, remove deprecated stuff).

---

## Phase 1 — Build the `scylla` package in genix-orm (standalone, no genix changes)

Work happens in `/run/media/ivanjoz/projects/genix-orm`.

1. `git mv cassandra scylla` (dir is currently empty and may be untracked — create `scylla/`).
2. Copy `backend/db/*.go` → `genix-orm/scylla/`, and `backend/db/text_search/` →
   `genix-orm/scylla/text_search/`. Copy `ORM_INTERNALS.md` too.
3. Rewrite package clause `package db` → `package scylla` in all `scylla/*.go`.
4. Rewrite internal imports:
   - `"app/db/text_search"` → `"github.com/ivanjoz/genix-orm/scylla/text_search"` (3 files:
     `deploy.go`, `text_search_index.go`, `text_search_query.go`)
   - `"app/libs/colbin"` → `"github.com/ivanjoz/colbin"` (2 files)
5. `go.mod`: add `github.com/gocql/gocql v1.6.0`, `github.com/fatih/color`, `github.com/kr/pretty`,
   `golang.org/x/sync`, and
   `replace github.com/gocql/gocql v1.6.0 => github.com/scylladb/gocql v1.13.0`
   (a dependency's `replace` is ignored by the consuming main module, so genix keeps its own —
   this one exists purely so genix-orm builds and tests on its own).
6. `go mod tidy && go build ./... && go vet ./...`
7. `go test ./scylla/...` — the unit tests (`group_by_test.go`, `memory_usage_test.go`,
   `select_helpers_test.go`, `update_counter_test.go`, `cache_version_generic_test.go`, …) have no
   app-level dependencies, so they should pass unmodified. **Tests that need a live ScyllaDB are the
   expected exception** — record which ones skip/fail for the same reason they do today.
8. Commit and push to `git@github.com:ivanjoz/genix-orm.git` (needed before the submodule can pin a
   commit).

**Gate:** Phase 1 must build and test green in isolation before genix is touched at all.

## Phase 2 — Wire the submodule into genix

1. `git submodule add git@github.com:ivanjoz/genix-orm.git backend/genix-orm`
2. `backend/go.mod`: add
   `require github.com/ivanjoz/genix-orm v0.0.0` +
   `replace github.com/ivanjoz/genix-orm => ./genix-orm`
   Keep the existing scylladb/gocql replace.
3. **Snapshot the module graph before** — `go list -m all > /tmp/mods-before.txt` — so the bump in
   step 7 is measurable rather than guessed.
4. Rewrite the 84 consumer files:
   - import line `"app/db"` → `"github.com/ivanjoz/genix-orm/scylla"`
   - import line `"app/db/text_search"` → `".../scylla/text_search"` (`main.go`, `exec/init.go`)
   - call sites: `\bdb\.` → `scylla.` (1447), then hand-fix the string literal at
     `exec/test_selects.go:97`
5. `git rm -r backend/db`
6. colbin migration: `config/hello.go`, `security/login.go`, `exec/demo.go`,
   `core/usuario-accesos.go` → `github.com/ivanjoz/colbin`; then `git rm -r backend/libs/colbin`.
7. `go mod tidy`, then **diff the module graph** against the snapshot. Two expected bumps to confirm
   are harmless:
   - `viant/xunsafe` v0.10.3 → v0.11.0 (genix-orm requires it)
   - the `dynamo` package's AWS SDK (`aws-sdk-go-v2` v1.43.0, `dynamodb` v1.61.0 vs genix's v1.25.2 /
     v1.30.1) **should be pruned** because genix imports only `scylla/`. If pruning does not kick in,
     the fallback is to give `dynamo` its own `go.mod` inside genix-orm.
8. `go build ./... && go vet ./...` in `backend/`.

## Phase 3 — Tooling, CI, docs

1. `scripts/` (separate module, loads backend source via `go/packages` — the hardcoded path strings
   must change):
   - `validation/check_tables.go:65` — type check `app/db.TableStruct` →
     `github.com/ivanjoz/genix-orm/scylla.TableStruct`
   - `table/create_edit_table.go:128` — emitted `import "app/db"` → new path
   - `controllers/controllers_generator.go:367` (+ the skip-list comment at `:98`, alias logic at
     `:385`/`:389`) — `requiredPackages` key `"app/db"` → new path
   These back the `create-database-tables` and `static-project-validation` skills, so **the table
   workflow breaks if this step is skipped**.
2. `.github/workflows/ci-deploy.yml:23` — add `with: submodules: recursive` to `actions/checkout@v4`.
3. Docs: `AGENTS.md` (points at `backend/db/ORM_INTERNALS.md`), `README.md`,
   `scripts/GENERATE_CONTROLLERS.md`, `scripts/CREATE_EDIT_TABLE.md`, `docs/CACHE_VERSIONED_BY_IDS.md`,
   and `backend/docs/ORM_*.md`. Historical `PLAN_*.md` files at the root are left as-is (they
   describe past states).
4. Check `app.sh` / `deploy.sh` / `start.js` — grep found **no** `backend/db` references, so
   probably nothing to do; confirm the Lambda build still packages correctly.

## Phase 4 — Verification

- `cd backend && go build ./... && go vet ./... && go test ./...`
- `cd genix-orm && go test ./scylla/...`
- Run the `static-project-validation` skill (`scripts/validation/check_tables.go`) — must still
  discover every `db.TableStruct` embedder.
- Run the controllers generator and confirm the emitted import block uses the new path.
- Dry-run a table creation via the `create-database-tables` skill.
- Fresh-clone smoke test: `git clone --recurse-submodules` into a temp dir and build, to prove the
  submodule + replace wiring works for a new checkout (and for CI).
- Confirm the deploy build (`deploy.sh`) produces a working artifact.

## Risks

| Risk | Mitigation |
|---|---|
| AWS SDK bump leaks in from `genix-orm/dynamo` | Module-graph diff in Phase 2.7; fallback = separate `go.mod` for `dynamo` |
| `xunsafe` v0.10.3 → v0.11.0 breaks reflection code | `db` uses xunsafe heavily; Phase 1 tests run against v0.11.0 *before* genix is touched |
| colbin format drift between repos | Verified byte-format identical today; risk is *future* drift — genix-orm and genix must pin the same colbin version |
| 1447-site rewrite introduces a subtle miss | `\bdb\.` is unambiguous here (no shadowing identifiers, `dynamodb.` excluded); compiler catches the rest |
| Contributors forget `--recurse-submodules` | CI flag + a README note |
| ORM iteration now needs two commits | The `replace` keeps edits local; only publishing needs the second commit |

## Rollback

Nothing is destructive until Phase 2.5 (`git rm -r backend/db`). Up to that point genix is
untouched. After that, `git revert` of the genix commit restores `backend/db` in full; the genix-orm
repo can be left in place.
