# Frontend Services & Backend Integration Guide

In this project, **Services** act as the glue between the Svelte frontend and the Go backend. They encapsulate HTTP communication, state management (using Svelte 5 runes), and caching logic.

## 1. Core Concepts

### Service vs. Connector
While they are technically "connectors," the project convention is to name them `*Service` (e.g., `ProductosService`, `CajasService`). They are typically co-located with their respective routes or placed in `pkg-services/` if shared across multiple modules.

### HTTP Utilities (`$core/http.svelte`)
- **`GetHandler`**: A base class for services that need automated caching and synchronization (Delta Cache). Best for master data.
- **`GET`**: A functional wrapper for fetching data. It can be used for one-off requests or reports. If `useCache` is provided, it will utilize the Service Worker cache.
- **`POST`**: Used for creating or updating records. It includes a `refreshRoutes` feature to invalidate caches.

---

## 2. Cached Services (Master Data)

Cached services are used for "Master Data" (Productos, Sedes, Almacenes) that change infrequently and should be available offline or load instantly.

### Implementation Pattern
These services extend `GetHandler` and implement the `handler` method.

```typescript
import { GetHandler } from '$core/http.svelte';

export class MyService extends GetHandler {
  route = "my-entity"
  useCache = { min: 5, ver: 1 } // min: TTL in minutes, ver: Cache version

  // Reactive state using Svelte 5 runes
  records: IEntity[] = $state([])
  recordsMap: Map<number, IEntity> = $state(new Map())

  // This is called automatically when data is received (from cache or network)
  handler(result: IEntity[]): void {
    // Note: Backend uses 'ss' for status. 
    // 1 = Active, 0 = Deleted (only returned during sync)
    this.records = result.filter(x => x.ss > 0) 
    this.recordsMap = new Map(result.map(x => [x.ID, x]))
  }

  constructor() {
    super()
    this.fetch() // Triggers the offline -> refresh cycle
  }
}
```

### How Delta Cache Works
1. **Offline Fetch**: `GetHandler.fetch()` first retrieves data from the local cache (IndexedDB via Service Worker). The `handler` is called immediately with this stale data.
2. **Refresh Fetch**: Simultaneously, it sends a request to the backend with an `upd` (updated) query parameter.
3. **Backend Logic**:
   - If `upd == 0`: Backend returns all active records (`ss > 0`).
   - If `upd > 0`: Backend returns ONLY records changed since that timestamp, including deleted ones (`ss == 0`).
4. **Merge**: The Service Worker merges the delta into the local cache and the `handler` is called again with the updated dataset.

---

## 3. Report & On-demand Services

Report services are used for complex queries, historical data (e.g., sales reports, movements), or datasets too large to cache entirely.

### Option A: Standard GET (No Cache)
Used for data that must always be fresh or uses dynamic filters.

```typescript
import { GET } from '$core/http.svelte';

export const getMyReport = async (filters: LogFilters): Promise<ILog[]> => {
  let route = `my-report?start=${filters.start}&end=${filters.end}`
  
  try {
    const result = await GET({ route })
    return result.records
  } catch (error) {
    console.error("Report Error:", error)
    throw error
  }
}
```

### Option B: Functional GET with Cache
Useful for data that doesn't need a full class/state management but benefits from caching.

```typescript
import { GET } from '$core/http.svelte';

export const getStaticConfig = () => {
  return GET({ 
    route: "config", 
    useCache: { min: 60, ver: 1 } 
  })
}
```

---

## 4. Modifying Data (`POST`)

When updating data, it is critical to keep the cached services in sync.

```typescript
import { POST } from '$core/http.svelte';

export const saveEntity = (data: IEntity) => {
  return POST({
    data,
    route: "my-entity",
    // Invalidate these routes in the Service Worker cache
    refreshRoutes: ["my-entity", "related-summary"]
  })
}
```

- **`refreshRoutes`**: List of routes that should be re-fetched or marked as stale. When a `GetHandler` service for that route is active, it will receive the updated data automatically on its next cycle.

---

## 5. Summary Checklist for New Services

- [ ] **Choose the pattern**: `GetHandler` for master data/sync, `GET` for reports/logs.
- [ ] **Define Interfaces**: Create TypeScript interfaces for the backend response (matching the Go structs).
- [ ] **State Management**: Use `$state` runes for reactivity.
- [ ] **Status Filtering**: Remember that in `GetHandler`, the backend might return records with `ss: 0` (deleted) during sync; filter them out in your `handler`.
- [ ] **Route Definition**: Ensure the `route` matches the key in the backend's `ModuleHandlers`.
- [ ] **Cache Versioning**: Increment `useCache.ver` if the data structure changes to force a full re-sync.
