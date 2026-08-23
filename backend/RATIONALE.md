## `system/` was never a module — it is now `libs/servermetrics/`

**Context** — `backend/system/` sat at module level next to `sales`, `finance` and the rest, but
it declared no `ModuleHandlers`, had **zero** `app/` imports, and contained only OS metric
collection (`ServerMetricsCollector`, `CollectGoHeapPackageReport`, snapshot structs). Its only
consumer, `config`, imported it as `servermetrics "app/system"` — an alias whose sole job was to
correct the folder name. Under the module-boundary rule (`MODULE_BOUNDARIES_PLAN.md`) a module
body may not import another module body, so `config -> system` read as a violation of a rule it
was never really breaking.

**Decision** — moved to `libs/servermetrics/`, package renamed `system` -> `servermetrics`. The
two importers in `config` now use a bare `"app/libs/servermetrics"` with no alias. Stale path
references updated in `server_utils/PLAN_SERVER_METRICS.md`,
`server_utils/src/sysmetrics/collector.rs`, the repo map in `AGENTS.md`, and — the one that
would have broken a test — the evidence `path:` entries in
`frontend/routes/system/server-panel/DOCUMENTATION.md`, which the ragdocs parser validates by
path and content hash.

**Rationale** — `libs/` over `core/` because this reads the OS, not the domain, and `libs/` is
where non-business generics live; `core/` holds cross-cutting code that does carry business
logic. Classifying it as a foundation package rather than a module is what makes the boundary
rule honest: the import was never the problem, the folder's position was. Cost is a moved
folder and the four external references above, none of which are load-bearing at build time
except the DOCUMENTATION.md evidence paths.

