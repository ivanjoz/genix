# RATIONALE — crm

Design decisions for clients and providers, newest first.

## One `ClientProvider` table, one owning module, two menus

**Context** — clients and providers are the same row discriminated by `Type` (1 = client,
2 = provider). They were in `business`, surfaced as two sibling pages. The roadmap adds a client
file with history, a call/visit log, an opportunity funnel, receivables, credit lines, campaigns
and loyalty — all client-side, none provider-side.

**Decision** — a new L4 module `app/crm` owns the table (`crm/types/client_provider.go`) and its
handlers (`crm/client_provider.go`). The **Clientes** page moved to `/crm/customers`; the
**Proveedores** page moved to `/logistics/suppliers`, next to purchase orders.

**Rationale** — splitting the table by `Type` would mean two tables, two delta syncs and a
migration, to serve a UI grouping. Keeping one table and letting the route live where the user
looks for it costs nothing: `logistics` imports `crm/types`, which is an ordinary L4 → L2 edge.
The frontend does the same — both pages render one `domain-components/ClientProviderMaintainer`
parameterized by type.

## `SaveClientProviders` stays in `types`

**Context** — `sales` creates or resolves the buyer while recording a sale, so it needs the
upsert. A module body may not import another module body.

**Decision** — `crm/types/client_provider_save.go`, unchanged but for its new path.

**Rationale** — same reasoning as before the move (see `backend/docs/MODULE_BOUNDARIES.md`); only
the owning module's name changed.
