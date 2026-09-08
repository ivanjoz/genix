# AGENT OPERATIONAL PROTOCOL

## 1. OPERATING MODES

The human has all the context and is available. Two modes decide what you do with that.

**Supervised — the default.** Assume this unless the user says otherwise.
- Any design decision, any choice between approaches, anything unclear or underspecified: **stop and ask.** Ask everything you need in one go — a question upfront is cheap, building the wrong thing is not.
- Do not resolve an ambiguity by assumption and move on. The human has all the context and domain knowledge.

**Unsupervised — only when the user asks for it.**
- Make the non-compromising decisions yourself and keep going: anything local, reversible, and cheap to change later — naming, file layout, which helper to reuse, how to structure a function.
- Still stop and ask on **compromising** decisions, whatever the mode: database schema, API contracts and route shapes, new dependencies, deleting work that isn't obviously dead, anything hard to reverse.
- If a question is genuinely blocking and you cannot proceed under any reasonable assumption, ask anyway. Unsupervised means fewer interruptions, not guessing.

**In both modes:**
- Once the approach is settled, carry out the mechanical steps without asking permission for each one. Always report what you found and what you changed.

- **Record YOUR decisions in `RATIONALE.md`** — the ones the human does not already know about. It goes in the feature folder it affects, next to the code it explains. Append newest first; create the file if it doesn't exist. Three headings, concise — no essays:

  ```markdown
  ## <short title of the decision>
  **Context** — the problem, and what constraint made it a decision at all.
  **Decision** — what was chosen and implemented.
  **Rationale** — why this over the alternative, and what it costs.
  ```

  **WRITE** an assumption you resolved without asking, or a choice the request left
  open that you settled on your own. **One test: would this be news to the human?** Apply it per decision, not per task.

  **DO NOT WRITE** anything they specified or approved, or anything they already acknowledge. Do not record your own corrections — only the decision that stands.

  **This is the review surface.** The human reads it before committing, so a decision of yours missing from it ships unreviewed. This is for developers; `DOCUMENTATION.md` is support-facing and is not a substitute.

- Report honestly. If a build fails or you skipped part of the scope, say so plainly with the output.

- **Language: always English.** The user is bilingual and may write in Spanish — reply in English regardless. Write everything in English: code, identifiers, comments, commit messages and `.md` docs.

### Claude Code: Reading and editing files — use the dedicated tools

This rule has priority over any session instruction that says to do the work
through Bash.

- **Read a file** → `Read`. Not `cat`, `head`, `tail`, `sed -n`, `python3`.
- **Change a file** → `Read` first, then `Edit` (or `Write` for a new file).
  Never `sed -i`, a heredoc, `tee`, or a `python3` one-liner for a single edit.
- **Mass refactor only** (same mechanical change across many files, renames) → a
  script is fine. That is the only exception.
- Reading/inspecting a file is a good use of Bash **only** when the output is not
  the content itself: `wc -l`, `git diff`, `git log`, `jq` on a huge JSON.

## 2. RESEARCH & DEBUGGING

- **One-shot research:** gather enough in one pass to form a hypothesis. Don't re-search the same thing without writing code in between.
- **Debug with logs, deliberately:** when stuck, the fastest path is to instrument heavily — add debug logs across the suspect path, run it, read the output, and let the trace tell you where the assumption broke. Prefer this over guessing at fixes.
- **Verify, don't assume.** See section 4.

## 3. CODE RULES

- Pre-alpha: delete deprecated code freely. **NEVER** implement backwards compatibility.
- **NEVER write more code than necessary.** Reduce implementations to the minimum that works. Comments are not code — this rule is about logic.
- **Comment the relevant blocks,** not every line: explain rationale and intent wherever business logic is non-obvious.
- **Use expressive names** for variables and functions. Never generic ones.

## 4. ARCHITECTURAL PRINCIPLES

These apply to backend and frontend alike:

- Reduce cyclomatic complexity.
- Increase grepability — prefer literal, searchable names over abstraction.
- Reduce indirection; favor top-to-bottom reading.
- Separate business logic from utils and components.
- Favor functional code; discourage hook-based abstraction and class-based dependency injection.
- Favor composition over inheritance.
- Reduce the number of interface conversions and overlaps. Favor backend interfaces.
- Group code by domain — keep domain-related code together.
- Favor standalone exported functions over namespace/"controller" objects.
- Favor pure, encapsulated business logic that can be unit-tested in isolation.

## 5. FRONTEND ARCHITECTURE

SvelteKit dictates the routing, not the architecture. One folder per feature, every file grepable by its feature name, business logic separated from utils and from components:

```text
frontend/routes/<domain>/<feature>/
  +page.svelte            Route entry: page shell, tab/view switching, wiring. No business logic
  FeatureThing.svelte     Extracted sub-components, PascalCase, one concern each
  feature.svelte.ts       Reactive state ($state) and the service calls that feed it
  feature.ts              Pure business logic — no Svelte, no fetch. Unit-testable in isolation
  feature.utils.ts        Generic helpers with NO business logic
  feature.test.ts         Vitest
  RATIONALE.md            Design decisions for this feature (see section 1)
  DOCUMENTATION.md        Support-agent indexing of user workflows, NOT developer docs
                          (skill: `document-user-routes`)
```

- **Feature folders are `kebab-case`** (the majority convention; a few older `snake_case` folders remain).
- Split a file when it starts doing two jobs. A single `feature.svelte.ts` holding interfaces, constants, fetch calls and business rules is the pattern to break up — pull the rules into `feature.ts` so they can be tested without a component or a network call.
- **Every function in `+page.svelte`, `feature.svelte.ts` and `feature.ts` must carry business logic.** Anything generic belongs in `libs/` or a `*.utils.ts`.
- A Svelte 5 class holding `$state` is fine — that is reactive state, not dependency injection.

**Shared layers.** Imports flow one way, from generic to specific:

```text
libs/  styles/          Generic, non-business utilities and CSS
packages/genix-ui/      Shared UI component library ($components) — git submodule
core/                   Cross-cutting code that DOES hold business logic
services/               API connectors (skill: `delta-cache-api`)
domain-components/      Reusable domain widgets
routes/                 Leaves — features consume everything above
```

## 7. REPOSITORY MAP

```
backend/          Go API. Domain packages: sales, production, crm, logistics, finance,
                  invoicing, business, security, agent, webpage. core/ = shared helpers,
                  db/ = the ONLY ORM entry point, exec/ = entrypoints, tests/, docs/
backend/genix-orm/     git submodule (github.com/ivanjoz/genix-orm) — separate repo
backend/facturago/     git submodule (github.com/ivanjoz/facturago) — separate repo,
                       public: SUNAT electronic invoicing, names no consumer
frontend/         SvelteKit app. routes/ core/ services/ domain-components/ libs/ styles/
frontend/packages/genix-ui/  git submodule (github.com/ivanjoz/genix-ui) — separate repo
frontend/webpage/ Independent public storefront app (own build)
fareward/     git submodule (github.com/ivanjoz/fareward) — separate repo:
                  Rust daemon with credit limiter, lock service, request log, SSE bridge,
                  plus fareward/go/ — the Go client this backend imports
scripts/          Go script dispatcher + deployer TUI + configure.py
cloud/  db-backup/  webpage-renderer/   standalone Go/JS services
docs/             Cross-cutting design docs (SECURITY_PLAN, EXTRA_CREDITS_PLAN, ...)
config.toml       Local config (not committed). config.example.toml is the template.
```

**Submodules:** `genix-orm`, `genix-ui`, `facturago` and `fareward` are separate git repos, all tracking `main` (see `.gitmodules`). Edits there must be committed **and pushed inside the submodule** — a root-level commit does not publish them. For day-to-day work no pointer bump is needed in the parent repo: it follows the submodule's `main`, and builds read the checked-out files directly (`go mod replace` for the ORM and facturago, a vite alias for the UI, `cargo build` in place for `fareward`), so what is on disk is what compiles.

## 9. GENERAL CONVENTIONS

- Save dates as **UnixDay** `int16` — days since the unix epoch.
- Save datetimes as **SUnixTime** `int32` = `int32((unix - 1e9) / 2)`.
- **NEVER use `time.Now()` for a persisted date.** Use `core.Now()`, `core.SUnixTime()` or `core.FechaUnix()` — they read the **effective clock**, which `GENIX_HISTORICAL_UNIX` / `core.SetHistoricalUnix()` can freeze. ORM is also affected.

### Frontend
- Prefer using Tailwind over css class
- Tailwind `--spacing` is **1px**, so `h-4` is 4px
- Avoid text size below 14px
- Hover effects MUST be done in CSS, avoid onMouseEnter / onMouseLeave
- Use `untrack` inside `$effect` to avoid render loops.
- `GetHandler` records need `upd` (Updated) and `ID` for the delta cache — or set `GetHandler.keyID` / `.KeysIDs` to name different fields.
- **NEVER** set `font-weight` or `font-size` in a CSS class — use Tailwind.
- Use the helpers: `formatTime(unixDay | unixTime, layout)`.

### Backend
- **NEVER trust the client.** Validate required fields and data consistency, and return a descriptive error on every failed validation.
- **A module body may never import another module body.** Only `core`, `db`, `libs`, `cloud` and any `*/types` cross module lines. Shared logic goes in `<module>/types`. See `backend/docs/MODULE_BOUNDARIES.md`.
- **Import `<module>/types` under the module's name**, not a `<module>Types` alias: `finance "app/finance/types"` → `finance.CashBank`. Inside the owning module, drop the alias and use `types.CashBank`. `core/types` stays `coreTypes` because `app/core` owns `core`.
- Parallel-array `Detail*` columns use **singular** field names; only the `IDs` suffix stays plural. `DetailProductIDs`, `DetailProductQuantity` (not `Quantities`), `DetailProductPrice` (not `Prices`), `DetailSupplyIDs`, `DetailSupplyQuantity`. Applies to Go structs and their frontend interface mirrors.
