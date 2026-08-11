# Genix

**An open-source, self-hostable ERP + E-commerce platform for small businesses**

Genix is a multi-tenant SaaS ERP that can also run as a single local binary for one company. It covers point of sale, inventory, cash management, product & client management, projected cash flows, and an AI-assisted e-commerce website builder. Every tenant can export a full backup of their own data at any time.

> Status: **pre-alpha**, under active development. Interfaces and schemas may change without backwards-compatibility guarantees.

---

## Philosophy

- **The user owns the data.** The tenant is the actual business/user. Data isolation, privacy, and portability are design constraints: every tenant can export a complete backup of their data at any time (see [Data portability](#data-portability--backups)).
- **Performance and efficiency by construction.** We prefer strictness over convenience. Go (compiled, statically typed) over flexible interpreted languages; ScyllaDB with explicit partition keys over a database where every access pattern is implicitly possible. Access patterns are declared up front and served by a partition, index, or view — `ALLOW FILTERING` is treated as a defect. The strict data model, enforced by struct rules and a compile-time query API, is a deliberate trade of flexibility for predictable, low-latency access at scale.
- **Flexible deployment.** The same codebase runs as a multi-tenant SaaS (AWS Lambda + ScyllaDB) or as a single local binary for one company on a laptop / on-prem server. See [Hybrid architecture](#hybrid-architecture-lambda-or-single-binary).
- **Open source.** Licensed under **GPL v3**.
- **Minimal by design.** Favor the smallest amount of code that solves the problem; no backwards-compatibility cruft while in pre-alpha.

---

## Tech stack

| Layer | Technology |
|---|---|
| Backend | **Go 1.26**, single `app` binary |
| Primary database | **ScyllaDB / Apache Cassandra** (via a custom ORM) |
| Cloud data mirror | *optional* — **DynamoDB** (`aws`) or **Cloudflare D1 / SQLite** (`cloudflare`); off when ScyllaDB-only |
| Object storage | pluggable — **Cloudflare R2** or **S3** today; self-hosted / local planned |
| Frontend | **SvelteKit 2 / Svelte 5**, TailwindCSS v4, Bun |
| Storefront delivery | Static SSR/prerender → **Cloudflare** (custom domains) |
| AI | **OpenRouter** (OpenAI-compatible), curated model registry |
| Text search | **GenixSearch** daemon (Spanish bigram encoder) |

The frontend is a monorepo with a strict dependency hierarchy (`libs` → `ui-components` → `core` → `services` → `domain-components`), and an independent, lightweight **storefront** app that never imports admin/business code.

### Configurable providers

`config.toml` separates backend data from file delivery:

- **`providers.backend`** — `aws` uses DynamoDB; `cloudflare` uses D1 / SQLite; `none` disables the optional mirror and reads auth/tenant data directly from ScyllaDB on a self-hosted backend.
- **`providers.cdn`** — `aws` uses S3 + CloudFront; `cloudflare` uses R2 as the public asset origin.

For example, `providers.backend=aws` and `providers.cdn=cloudflare` runs the backend on AWS Lambda + DynamoDB while files are stored and served by Cloudflare R2.

The data mirror only holds auth & tenant master tables (users, companies, profiles) so they are reachable outside Scylla; all business data lives in ScyllaDB regardless of provider.

---

## Technologies built for this project

Components developed in-house for Genix. Each is implemented (not a stub) unless noted.

### Custom ScyllaDB ORM
A reflection-optimized ORM for ScyllaDB/Cassandra, developed in its own repo
([genix-orm](https://github.com/ivanjoz/genix-orm)) and consumed here as a git submodule at
`backend/genix-orm/`. Clone with `--recurse-submodules` (or run `git submodule update --init`
afterwards), otherwise the backend will not build — `backend/go.mod` resolves the ORM through a
`replace` pointing at that directory, so local ORM edits apply without a publish cycle.

The `scylla` package (`backend/genix-orm/scylla/`) provides:
- **Two-struct `TableStruct` pattern** — each entity is a `XRecord` (persisted fields) + `XRecordTable` (typed fluent query columns), giving a **compile-time-safe query API** (`Equals`, `In`, `Between`, `Contains`, `GroupBy`, `Select`, `ExecScan`, …). Schema is declared in Go via `GetSchema()`.
- **Reflection elimination in hot paths** — per-column getter/setter function pointers compiled once (via `viant/xunsafe` + pointer arithmetic) instead of `reflect` in loops; cached per type.
- **Capability-based query routing** — normalized predicate signatures are matched at compile time to the best base key / secondary index / materialized view, avoiding accidental `ALLOW FILTERING`.
- **Advanced key/index strategies** — integer key-packing with autoincrement slots, Base62 concatenated keys, local/global indexes, hash & radix-range **views**, and composite-bucket indexes for bounded-range queries.
- **Delta cache versioning** — per-group cache-version counters power incremental client sync (`QueryCachedIDs`): the frontend only fetches records changed since its last sync.
- **Schema homologation & deploy** — `DeployScylla` diffs declared vs. live schema and applies missing columns/indexes/views automatically.

### minijson — compact JSON responses

API responses use [`minijson`](https://github.com/ivanjoz/minijson), a standalone Go module
that writes struct field metadata once and encodes records as compact positional JSON arrays.
The matching `@ivanjoz/minijson` TypeScript decoder is consumed by `genix-ui`, so backend and
frontend share one versioned wire-format implementation.

### Three-layer frontend cache
The frontend (`frontend/libs/cache/`) minimizes network traffic and moves relational "joins" to the client via three complementary caches, each memory → IndexedDB → server:

1. **Delta cache** (`delta-cache.*`) — for maintainer / master-data list pages. Each response key is stored as independent IndexedDB rows; on refetch the client sends its last watermark and the server returns only rows whose `upd` changed, plus `*_IDsToRemove` tombstone flags. Backed by the ORM's cache-version machinery.
2. **Query-block cache** (`cache-query-by-id.ts`, `GETCached(route, updated)`) — caches a whole response block keyed by a preset combination of query fields (the request route + query string), invalidated by a parent record's `updated` watermark. When the parent changes, the block is refetched and overwritten; no per-record protocol.
3. **Cache-by-IDs** (`cache-by-ids.*`) — resolves individual records by `ID` with backend delta validation via the `ccv` cache-version. Requests are batched into one call per route. This is how joins happen on the client: a page fetches its primary data, then resolves referenced dictionaries/lookups by ID from cache — fetching only the IDs that are missing or stale.

### colbin — columnar delta binary serializer
A serializer (`backend/libs/colbin/`) that replaced CBOR project-wide. Transposes slices-of-structs from row layout to columnar (SoA) layout and applies frame-of-reference delta encoding plus bit-level bit-packing per numeric column. Benchmarks at ~3× smaller and ~2.8× faster to decode than CBOR on ERP row batches. Used by the ORM to persist complex blob columns. Struct tag: `cb:"name"` (`cb:"-"` to skip).

### Agentic capabilities (backend-driven)
The agent loop runs on the backend. A user sends a request from an in-app chat widget; the backend runs an LLM tool-calling loop (OpenRouter) that either drives the live ERP page or authors website HTML. Two loops share the infrastructure:
- **ERP chat/navigation loop** (`backend/agent/chat_loop.go`) — tools `get_page`, `get_menu`, `navigate`, `invoke_batch`, and a `finish` terminator; bounded iterations with tool-result truncation and rolling context windows. Transport is SSE + POST per browser tab; history persisted in ScyllaDB with per-user token accounting.
- **Website-building loop** (`backend/agent/webpage/`) — an intent classifier with a relevance gate, a deterministic content-preservation gate (diffs old vs. new HTML as ASTs, enforcing a keep/add/modify/replace policy per text/image/icon), an aesthetic critic, plus SVG-generation and image-selection subagents.
- **External HTTP API** (`POST /agent`) lets external agents (e.g. Claude Code) drive the ERP directly.

### Agent-navigable UI components
A browser-side registry (`frontend/ui-components/agent/registry.ts`) exposes the ERP UI to the agent. Each interactive component registers itself (`window.__agent`) with a stable method vocabulary (`search`, `select`, `setValue`, `click`, `open`/`close`) and mirrors its state into DOM attributes (`data-id`, `data-value`, `aria-label`, `data-menu-root`). The agent reads a sanitized HTML snapshot and calls methods only for actions it cannot perform by reading the DOM, so one loop operates any page without per-page integration code.

### AI-assisted WYSIWYG e-commerce builder
A section-based website/storefront builder (`frontend/routes/webpage-builder/`) with an HTML↔AST editing engine, CSS auto-scoping, palette color absorption, 20+ section templates, mobile preview, and custom components (`ProductGrid`, `ImageEffect`, `Slider`, `Icon`). AI "Build page" and "Edit section" modes plug into the shared agent chat and merge model output into the live canvas. Pages deploy to Cloudflare with custom-domain provisioning.

### Hybrid architecture (Lambda or single binary)
See [below](#hybrid-architecture-lambda-or-single-binary).

### Text search
A Spanish bigram word encoder ported to Go (`backend/libs/text-search/`) that produces byte-identical index keys to the external GenixSearch daemon, plus a TCP driver and ORM integration (`backend/genix-orm/scylla/text_search/`): a table with a `TextSearchColumn` gets relevance-ranked search that returns IDs without a Scylla read.

---

## Hybrid architecture (Lambda or single binary)

The **same Go binary** runs three ways, selected at runtime — no separate builds:

1. **AWS Lambda** — detected via `AWS_LAMBDA_FUNCTION_NAME` / `AWS_EXECUTION_ENV`; served through API Gateway v2. Config from a compressed `CONFIG` env var. Data on ScyllaDB (VPS/EC2) + DynamoDB + S3.
2. **Standalone binary / self-host** — an `http.Server` on `:3589` (CORS, SSE, slowloris timeouts). Config from `config.toml`. Deployed as a `systemd` service behind Nginx (HTTP/3, TLS 1.3 0-RTT). See [`DEPLOYMENT.md`](DEPLOYMENT.md).
3. **Exec / function mode** — a `{"fn_exec": …}` body or `fn…` CLI arg runs a registered function (incl. a cron scheduler) instead of HTTP routing; cross-invocation calls transparently become `lambda.Invoke` in the cloud or a local HTTP POST when self-hosted.

Both HTTP transports converge on a single `core.HandlerArgs` → `mainHandler` path, with module handler maps registered per domain (`sales`, `finance`, `logistics`, `business`, `security`, `webpage`, `agent`, …), token auth, and YAML-driven access control.

---

## Multi-tenancy

- A **tenant = a Company** (`empresa_id` / `CompanyID`). The tenant is the actual business/user.
- **Isolation by partition key** — nearly every ScyllaDB table is physically partitioned by `empresa_id`, so a tenant's rows live in one partition and every query is scoped to it. The cloud ORM enforces a mandatory `empresa_id` filter to prevent cross-company reads.
- The **auth token carries the tenant** (`CompanyID` packed + hash-validated); access levels are loaded per (company, user).
- Company/User/Profile master records live in the configured data mirror (DynamoDB or Cloudflare D1) so they are reachable outside Scylla, and are also mirrored into ScyllaDB. With a vendor-free provider they live in ScyllaDB alone.

---

## Data portability & backups

Each tenant can export, download, and restore all of their data.

- **Create** (`POST backup-create`) — iterates every registered table, exports each tenant's rows as pipe-delimited base64 CSV (typed headers), zstd-compresses each table, and bundles them into a single `.tar` in S3 under `backups/{companyID}/…`.
- **List / Download** — the Backups page lists the tenant's archives and downloads the raw `.tar` directly from the CDN.
- **Restore** (`POST backup-restore`) — downloads the tar, decompresses, maps entries to tables by name, and re-inserts the records.
- UI: `frontend/routes/configuration/backups/`.

---

## Roadmap & current status

Legend: ✅ done · 🟡 partial / in progress · ⬜ not started

### Foundations & platform
- ✅ Custom ScyllaDB ORM (fluent queries, views, delta cache, schema deploy)
- ✅ Three-layer frontend cache (delta cache, query-block cache, cache-by-IDs) with client-side joins
- ✅ colbin columnar serializer (replaced CBOR project-wide)
- ✅ Hybrid runtime: AWS Lambda / single binary / exec+cron modes
- ✅ Multi-tenancy by `empresa_id` partitioning + tenant-scoped auth
- ✅ Independent providers via `providers.backend` (DynamoDB / D1) and `providers.cdn` (S3 / R2)
- 🟡 Vendor-free provider (`local` / `none`): ScyllaDB-only, no data mirror, self-hosted object storage
- ✅ Text search (GenixSearch bigram encoder + ORM integration)
- ✅ Full per-tenant data backup, download & restore
- ✅ Security: users, access profiles, YAML access-control
- ✅ Bilingual UI (EN | ES) i18n across production routes

### Agentic platform
- ✅ Agent-navigable UI component registry (DOM contract + method vocabulary)
- ✅ In-app AI chat widget (SSE + POST, ScyllaDB history)
- ✅ ERP navigation/tool-calling agent loop
- ✅ External HTTP agent API (drive the ERP from outside tools)
- ✅ Per-user token accounting (collected)
- ⬜ Streaming LLM responses (currently blocking)
- ⬜ Token-budget **enforcement** / quotas (data collected, no policy yet)

### Product & client management
- ✅ Products: catalog, categories, brands, attributes, images, sub-units, Excel bulk import/export
- ✅ Customers (person type, docs, geo address, search)
- ✅ Suppliers
- ✅ Branches & warehouses (+ visual warehouse layout editor)

### Inventory / Logistics
- ✅ Stock movements per warehouse (manual in/out, lot & serial)
- ✅ Purchase orders (create, report, goods receipt)
- ✅ Supply / replenishment planning
- ✅ Supplies & materials
- ✅ Warehouse movements

### Point of Sale & Sales
- ✅ POS / sale order creation (search, cart, register + warehouse, customer assign/create)
- ✅ Sales order management (pending payment / pending delivery / finished; register payment & delivery)
- ✅ Sales report + charts (by product, daily summary)
- 🟡 Sales charts — weekly summary view pending
- ✅ Sales planning (per-product weekly base quantities + 52-week seasonality curves)

### Cash management & finance
- ✅ Cash & bank registers (income/expense, reconciliation / cuadre)
- ✅ Cash-bank movements report
- ✅ Expenses (one-time + recurring scheduled, partial payments)
- 🟡 **Projected cash flow** — sales-projection groundwork exists (Sales Planning); the cash-flow projection report itself is **not yet built** (report tab is a placeholder). *This is a headline product goal still to be delivered.*

### E-commerce
- ✅ WYSIWYG website/storefront builder (sections, templates, AST engine, mobile preview)
- ✅ AI page-building & section-editing (classifier + content-preservation gate)
- ✅ Custom-domain deploy to Cloudflare
- ✅ Storefront rendering (products, search, categories, sliders)
- ✅ Client-side cart + Culqi payment UI
- ✅ Shipping-cost configuration per department / province / district (admin side)
- 🟡 Delivery price-per-city **applied at storefront checkout** (config exists; cart integration partial)
- ⬜ **E-commerce order persistence** (checkout shows a thank-you screen but does not yet save the order to the backend)
- ⬜ **Customer accounts** (storefront account creation / login / order history — currently skeleton)

### System & configuration
- ✅ Companies management
- ✅ Cron actions
- ✅ Server panel
- ✅ Parameters & configuration
- ✅ Backups & restore

---

## Repository layout

```
backend/          Go backend (single `app` binary)
  db/             Custom ScyllaDB ORM (+ text_search driver)
  agent/          Agentic loops (chat + webpage), LLM/OpenRouter client
  webpage/        Website/storefront content, config, Cloudflare deploy
  sales/ finance/ logistics/ business/ security/ system/ billing/
  cloud/          DynamoDB / SQLite mirror ORM, S3, Lambda invoke
  exec/           Function/cron mode, backup & restore
  libs/colbin/    Columnar delta binary serializer
  libs/text-search/  Spanish bigram encoder
frontend/         SvelteKit 2 / Svelte 5 monorepo
  routes/         Admin backoffice pages
  webpage/        Independent customer-facing storefront app
  ui-components/  UI atoms + agent registry
  core/ services/ domain-components/ libs/
scripts/          Table-creation, validation & build utilities
```

---

## Documentation

- [`AGENTS.md`](AGENTS.md) — operational protocol & project rules
- [`DEPLOYMENT.md`](DEPLOYMENT.md) — Lambda & self-host (systemd + Nginx) deployment
- [`backend/genix-orm/scylla/ORM_INTERNALS.md`](backend/genix-orm/scylla/ORM_INTERNALS.md) · [`backend/docs/ORM_DATABASE_QUERY.md`](backend/docs/ORM_DATABASE_QUERY.md) — ORM deep dive
- [`backend/docs/CREATE_API_HANDLERS.md`](backend/docs/CREATE_API_HANDLERS.md) — API handler guide
- [`backend/libs/colbin/README.md`](backend/libs/colbin/README.md) — colbin wire format
- [`frontend/FRONTEND.md`](frontend/FRONTEND.md) — frontend architecture
- [`frontend/docs/UI_COMPONENTS.md`](frontend/docs/UI_COMPONENTS.md) · [`frontend/ui-components/agent/AGENTIC_COMPONENTS.md`](frontend/ui-components/agent/AGENTIC_COMPONENTS.md) — UI & agent registry

---

## License

[GNU GPL v3](LICENSE).
