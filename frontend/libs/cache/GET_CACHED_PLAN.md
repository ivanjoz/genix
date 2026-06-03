# Plan: `GETCached(route, updated)` — route-keyed cache invalidated by parent watermark

## Goal

Add a small, self-contained cache for "child list" endpoints whose freshness is
governed by a parent record's `updated` value. Today
`ExpensesRegister.svelte:75-82` fetches an expense's settling payments with no
cache:

```ts
payments = await getCashBankMovementByID({ documentID: expenseID })
```

We want a `GETCached(route, updated)` helper that:

- Returns the cached records when the cached `updated` still matches the caller's
  `updated`.
- Re-fetches from the server when `updated` changes (parent record was modified),
  then persists the new `{ route, updated, records }`.
- Caches in two layers: an in-memory static map (fast path) **and** IndexedDB
  (survives reloads), mirroring the existing cache modules' memory→IDB→server
  layering.

This is intentionally simpler than `cache-by-ids.svelte.ts` (no per-record `ccv`
protocol, no delta sync, no buffering) and than `delta-cache` (no watermark
status maps). One route → one stored blob → one `updated` stamp.

## Storage shape

IndexedDB row (exactly as the user specified):

```ts
interface ICacheQueryByIdRow<T = any> {
  route: string      // primary key — the full request route (incl. query string)
  updated: number    // parent watermark the cached records were valid for
  records: T[]        // the full server response list
  _fch?: number      // fetched-at (seconds) — for debugging / future TTL, optional
}
```

In-memory mirror: `Map<string /* route */, ICacheQueryByIdRow>`.

## Files

### 1. NEW — `frontend/libs/cache/cache-query-by-id.idb.ts`

Dexie wrapper, modeled on `cache-by-ids.idb.ts`:

- A `CacheQueryByIdDatabase extends Dexie` with one table
  `cacheQueryById: 'route'` (primary key = `route`, no compound key needed).
- Reuse the **same database** as `cache-by-ids` to avoid spawning another IDB
  database per company/env. Concretely: add a second store to
  `makeCacheByIDsDatabaseName(...)`'s database by bumping
  `CACHE_BY_IDS_DB_VERSION` 1 → 2 and adding `cacheQueryById: 'route'` to the
  `.stores({...})` block in `cache-by-ids.idb.ts`. *(Decision point — see
  "Open question" below; default is to reuse that DB.)*
- Exported helpers (scoped by `getCurrentDatabaseName()` exactly like the
  existing file):
  - `readQueryCacheRow(route): Promise<ICacheQueryByIdRow | undefined>`
  - `upsertQueryCacheRow(row: ICacheQueryByIdRow): Promise<void>`
  - `deleteQueryCacheRow(route): Promise<void>` (for explicit invalidation)
- Wrap every Dexie call in try/catch returning a safe fallback (`undefined` /
  no-op), matching the defensive style of `cache-by-ids.idb.ts`.

### 2. NEW — `frontend/libs/cache/cache-query-by-id.ts`

The public module (the currently-empty file the user pointed at). Exports:

```ts
export const GETCached = async <T = any>(
  route: string,
  updated: number,
): Promise<T[]> => { ... }
```

Logic:

1. **Memory hit** — look up `memoryCache.get(route)`. If found and
   `row.updated === updated`, return `row.records` immediately.
2. **In-flight dedup** — keep `Map<string, Promise<T[]>>` so two concurrent
   calls for the same `route` share one fetch (the existing modules all do this;
   the payments effect can fire more than once).
3. **IndexedDB hit** — `readQueryCacheRow(route)`. If found and
   `row.updated === updated`, promote into the memory map and return its
   `records`.
4. **Server fetch** — `await GET({ route })` (plain, `useCache:false` path —
   this helper *is* the cache, so it must not double-cache through the SW delta
   cache). Normalize the response the same way `cache-by-ids` does: accept a raw
   array, else `payload.records`, else `[]` with a warning.
5. **Persist** — build `row = { route, updated, records, _fch: nowSeconds() }`,
   write to memory map and `upsertQueryCacheRow(row)`, then return `records`.

Invalidation is implicit: any `updated` mismatch in steps 1/3 falls through to
the server fetch, which overwrites the stored row. Optionally also export
`invalidateQueryCache(route)` that clears both layers, for callers that mutate
the child list directly.

Add a `clearQueryByIdCache()` that clears the memory map (and is called from the
existing `clearCacheByIDs()` in `cache-by-ids.svelte.ts` so a global cache reset
also drops this cache).

### 3. EDIT — `frontend/routes/finance/cajas/cajas.svelte.ts`

`getCashBankMovementByID` (line 122) currently always calls `GET`. Add an
optional `updated` arg and route through `GETCached` when present:

```ts
export const getCashBankMovementByID = async (
  args: { documentID?: number, referenceID?: number, updated?: number },
): Promise<ICashBankMovement[]> => {
  let route = "cash-bank-movement-by-id?"
  if (args.documentID) route += `document-id=${args.documentID}`
  else if (args.referenceID) route += `reference-id=${args.referenceID}`
  else throw ("Debe enviar un documentID o un referenceID.")

  if (typeof args.updated === 'number') {
    return GETCached<ICashBankMovement>(route, args.updated)
  }
  // ...existing uncached path unchanged...
}
```

Note: the current uncached path returns `result.movimientos` (a wrapped key),
not a raw array. So `GETCached`'s normalization must also try
`payload.movimientos` — OR (cleaner) keep the unwrap in the caller by having
`GETCached` return the raw response and letting the caller pick the array.
**Decision:** make `GETCached` generic over the list only and have the *caller*
pass the already-unwrapped list is not possible (caching happens before unwrap).
So `GETCached` will accept an optional `pick: (payload) => T[]` extractor,
defaulting to the array/`.records` heuristic. The cajas caller passes
`p => p.movimientos || []`.

### 4. EDIT — `frontend/routes/finance/expenses/ExpensesRegister.svelte`

`loadPayments` (line 75) passes the parent expense's watermark. The expense
record is `form`; use its update field as `updated`. Verify the field name on
`IExpense` (likely `upd`/`updated`/`Updated`) before wiring — pass that:

```ts
const loadPayments = async (expenseID: number) => {
  paymentsLoading = true
  try {
    payments = await getCashBankMovementByID({
      documentID: expenseID,
      updated: form.Updated,   // <-- confirm exact field on IExpense
    })
  } finally {
    paymentsLoading = false
  }
}
```

When a payment is registered (`postExpensePayment`), the expense's `updated`
advances, so the next `loadPayments` automatically misses the cache and
re-fetches. No manual invalidation needed in the happy path.

## Open questions to confirm before coding

1. **Which IDB database?** Reuse the `cache-by-ids` database (bump its version,
   add a store) vs. a brand-new dedicated database. Default in this plan: reuse,
   to keep one DB per company/env. Confirm acceptable.
2. **`updated` comparison:** strict `===` (any change invalidates) vs `>=`
   (cached is valid if at least as new). Default: `===`, matching "invalidate
   when the updated changes". Confirm.
3. **Exact `updated` field name on `IExpense`** (and that it actually advances
   on payment). Need to read `expenses.svelte.ts` to confirm.
4. **Response unwrap:** confirm the `pick`/extractor approach for the
   `{ movimientos: [...] }` shape vs. standardizing the endpoint to return a raw
   array.

## Out of scope

- No TTL/staleness timer (freshness is driven solely by `updated`).
- No per-record versioning, delta sync, or service-worker integration.
- No changes to the backend `cash-bank-movement-by-id` handler.

## Test plan

- Open a paid expense → payments load (server hit, row persisted in IDB).
- Re-open same expense → memory hit, no network (check console / Network tab).
- Reload page, re-open → IDB hit, no network.
- Register a new payment (expense `updated` advances) → cache miss → re-fetch.
