# RATIONALE — crm (frontend)

Design decisions for the Clientes (CRM) module, newest first.

## The client/provider maintainer became a domain component

**Context** — `CustomersView.svelte` is a fully parameterized maintainer for the `ClientProvider`
table: it takes `clientProviderType`, a page title and a layer title, and renders the list, the
filter and the side-layer form. `routes/business/customers/+page.svelte` and
`routes/business/suppliers/+page.svelte` were siblings and the second imported the first with
`../customers/CustomersView.svelte`.

**Decision** — `domain-components/ClientProviderMaintainer.svelte`. Both pages —
`routes/crm/customers/` and `routes/logistics/suppliers/` — now import
`$domain/ClientProviderMaintainer.svelte`.

**Rationale** — after the split the two pages are in different modules, so the relative import
would have become a route importing another module's route internals. The component was already a
reusable domain widget in everything but location; `domain-components/` is where those live. The
alternative — duplicating the view per module — would double the maintenance of a form that has
one backend contract.

## `ClientProviderService` and `CountryCitiesService` moved to `services/`

**Context** — `ClientProviderService` was imported by 10 files across `sales`, `logistics` and
`accounting`. `CountryCitiesService` lived in `routes/business/branches-warehouses/` and is used by
four routes; the maintainer needs it, and a `domain-components/` file cannot import from `routes/`
without inverting the layering.

**Decision** — `services/crm/client-provider.svelte.ts` and
`services/business/country-cities.svelte.ts`. `WarehousesService`, `postSite` and `postWarehouse`
stayed in the branches-warehouses route folder.

**Rationale** — both are API connectors with no page state, so `services/` is their home. The
country-cities move was forced by the maintainer's relocation rather than chosen, which is why the
rest of `branches-warehouses.svelte.ts` was left alone: splitting it further is a separate call.
