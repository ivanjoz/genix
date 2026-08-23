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
- Once the approach is settled, carry out the mechanical steps without asking permission for each one. Think as long as the problem needs, run independent searches and tool calls in parallel. Report what you found and what you changed.
- **Record decisions in `RATIONALE.md`.** Every design decision or assumption goes into a `RATIONALE.md` in the module or feature folder it affects, next to the code it explains. Append newest first; create the file if it doesn't exist. Three headings, concise — no essays:

  ```markdown
  ## <short title of the decision>
  **Context** — the problem, and what constraint made it a decision at all.
  **Decision** — what was chosen and implemented.
  **Rationale** — why this over the alternative, and what it costs.
  ```

  **This is the review surface.** The human reads `RATIONALE.md` before committing anything, so a decision missing from it is a decision that ships unreviewed. Read the existing file before changing an area — it explains why the code looks the way it does. This is for developers; `DOCUMENTATION.md` is a separate, support-facing artifact and is not a substitute.

- Report honestly. If a build fails or you skipped part of the scope, say so plainly with the output.

- **Language: always English.** The user is bilingual and may write in Spanish — reply in English regardless. Write everything in English: code, identifiers, comments, commit messages and `.md` docs.

## 2. RESEARCH & DEBUGGING

- **One-shot research:** gather enough in one pass to form a hypothesis. Don't re-search the same thing without writing code in between.
- **Debug with logs, deliberately:** when stuck, the fastest path is to instrument heavily — add debug logs across the suspect path, run it, read the output, and let the trace tell you where the assumption broke. Prefer this over guessing at fixes.
- **Verify, don't assume.** See section 4.

## 3. CODE RULES

- **ALWAYS CHECK SKILLS FIRST:** review the available skills list at the start of every task.
- Read the relevant `.md` doc (section 6) before working in an area.
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

## 6. VERIFY YOUR WORK

Independence only works with a closed feedback loop. Run the check that covers what you touched:

| Changed | Run |
| --- | --- |
| Backend Go | `cd backend && go build ./...` then `go vet ./...` |
| Backend tests | `cd backend && go test ./<pkg>/...` |
| DB table structs | `cd scripts && go run . check_tables` (skill: `static-project-validation`) |
| Frontend | `cd frontend && bun run check` (svelte-kit sync + svelte-check) |
| Frontend build | `cd frontend && bun run build` |
| server_utils (Rust) | `cd server_utils && cargo build` / `cargo test` |
| Real app behaviour | skill: `agent-browser` — drives the running app, reads the page, screenshots what renders |

Operational scripts run through the dispatcher: `cd scripts && go run . <script_name>`. Deploys go through `./deploy.sh` (TUI). `app.sh` is deprecated.

## 7. REPOSITORY MAP

```
backend/          Go API. Domain packages: sales, logistics, finance, invoicing,
                  business, security, agent, webpage. core/ = shared helpers,
                  db/ = the ONLY ORM entry point, exec/ = entrypoints, tests/, docs/
backend/genix-orm/     git submodule (github.com/ivanjoz/genix-orm) — separate repo
backend/facturago/     git submodule (github.com/ivanjoz/facturago) — separate repo,
                       public: SUNAT electronic invoicing, names no consumer
frontend/         SvelteKit app. routes/ core/ services/ domain-components/ libs/ styles/
frontend/packages/genix-ui/  git submodule (github.com/ivanjoz/genix-ui) — separate repo
frontend/webpage/ Independent public storefront app (own build)
server_utils/     Rust daemon: credit limiter, lock service, request log, SSE bridge
scripts/          Go script dispatcher + deployer TUI + configure.py
cloud/  db-backup/  webpage-renderer/   standalone Go/JS services
docs/             Cross-cutting design docs (SECURITY_PLAN, EXTRA_CREDITS_PLAN, ...)
config.toml       Local config (not committed). config.example.toml is the template.
```

**Submodules:** `genix-orm`, `genix-ui` and `facturago` are separate git repos, all tracking `main` (see `.gitmodules`). Edits there must be committed **and pushed inside the submodule** — a root-level commit does not publish them. No pointer bump is needed in the parent repo: it follows the submodule's `main`. Builds read the checked-out files directly (`go mod replace` for the ORM and facturago, a vite alias for the UI), so what is on disk is what compiles.

`facturago` is **public**. It implements SUNAT and names no consumer: no ERP identifiers, no tenancy, no business rules from this repo. A leak there is a leak in public.

## 8. KEY DOCUMENTATION

**Project & deployment**
- `README.md` — ERP + Ecommerce platform for small businesses (Go backend, Svelte frontend; frontend is migrating off Solid.js)
- `DEPLOYMENT.md` — AWS Lambda + ScyllaDB on VPS/EC2, and self-host via systemd

**Backend** (Go + ScyllaDB/Cassandra)
- `backend/docs/CREATE_API_HANDLERS.md` — **MUST read before creating any API.** The `updated` delta param, query examples, conventions
- `backend/docs/ORM_DATABASE_QUERY.md` — model definitions, CRUD, query building
- `backend/db/` — the single ORM entry point. All app code imports `app/db`, never a driver. `db/driver.go` names the database; repointing it switches the whole project
- `backend/genix-orm/db/` — driver-agnostic layer: schema, columns, predicates, accessors, the `Executor` contract
- `backend/genix-orm/scylla/ORM_INTERNALS.md` — driver internals: memory model, reflection engine, query optimization

**server_utils (Rust)** — one daemon: credit limiter, lock service, request log, SSE bridge. Rarely changes; read only when working on it.
- `server_utils/README.md` — the entry point. The `*_WALKTHROUGH.md` files explain the limiter and lock service end-to-end. `docs/SECURITY_PLAN.md` and `docs/EXTRA_CREDITS_PLAN.md` cover authorization and the extra-credit pool; `scripts/configure/CONFIGURE_SERVER_UTILS.md` covers deployment
- Go client: `backend/core/server_utils/`, re-exported via `backend/core/server_utils_api.go`. Authorization policy stays in Go.

**Frontend**
- `frontend/FRONTEND.md` — monorepo architecture, directory structure, package system, dev workflow
- `frontend/docs/UI_COMPONENTS.md` — Page, OptionsStrip, Layer/Modal, form components, VTable, services
- `frontend/docs/SERVICES_GUIDE.md` — **read before creating a service/connector.** Cached (delta) vs. report services
- `frontend/webpage/ECOMMERCE.md` — storefront: thumbhash, routes, CSS hashing

**Scripts**
- `scripts/SCRIPTS.md` — the dispatcher convention; **read before adding a script**
- `scripts/CREATE_EDIT_TABLE.md` — creating tables / adding columns. **USE ALWAYS**
- `scripts/CHECK_TABLES_SCRIPT.md` — validates data-model conventions for the ORM
- `scripts/AGENT_BROWSER.md` — self-authenticating headless browser for driving the real app
- `scripts/GENERATE_ERP_HISTORY.md` — seeds past-dated ERP history; documents the global effective clock
- `scripts/DEPLOYER.md` — the `deploy.sh` TUI and its fixed execution order
- `scripts/configure/CONFIGURE_SERVER.md` — backend on a VPS: systemd, binary auto-reload, Nginx proxy
- `scripts/configure/CONFIGURE_DB.md` — data host: ScyllaDB, GenixSearch, Qdrant

## 9. GENERAL CONVENTIONS

- Save dates as **UnixDay** `int16` — days since the unix epoch.
- Save datetimes as **SUnixTime** `int32` = `int32((unix - 1e9) / 2)`.
- **NEVER use `time.Now()` for a persisted date.** Use `core.Now()`, `core.SUnixTime()` or `core.FechaUnix()` — they read the **effective clock**, which `GENIX_HISTORICAL_UNIX` / `core.SetHistoricalUnix()` can freeze so seed generators write past-dated records. The ORM's own `created`/`updated` columns follow the same clock.

### Backend
- **NEVER trust the client.** Validate required fields and data consistency, and return a descriptive error on every failed validation.
- Parallel-array `Detail*` columns use **singular** field names; only the `IDs` suffix stays plural. `DetailProductIDs`, `DetailProductQuantity` (not `Quantities`), `DetailProductPrice` (not `Prices`), `DetailSupplyIDs`, `DetailSupplyQuantity`. Applies to Go structs and their frontend interface mirrors.

### Frontend
- Use `untrack` inside `$effect` to avoid render loops.
- `GetHandler` records need `upd` (Updated) and `ID` for the delta cache — or set `GetHandler.keyID` / `.KeysIDs` to name different fields.
- Tailwind `--spacing` is **1px**, so `h-4` is 4px.
- **NEVER** set `font-weight` or `font-size` in a CSS class — use Tailwind.
- Use the helpers: `formatTime(unixDay | unixTime, layout)`.
