# Plan — Notifications & Process Layer

## Goal
A header button-layer (the red box top-right in the Productos screenshot, next to the
⚙ gear / ↻ reload buttons) that is **always present** and accumulates two kinds of items:

- **Notifications** — static messages (name + text) carrying a `type`:
  `1 = info` (default), `2 = warn`, `3 = error`.
- **Processes** — long-running tasks with a live `status`: `1 = in progress`, `2 = done`,
  `0 = canceled`. A process **cannot be created with `status = 0`**, and the user **cannot**
  cancel it from the UI — `status = 0` is only ever set programmatically by the producer.

The layer must survive reloads (persisted in IndexedDB via Dexie), and expose convenience
methods any component can call — first consumer is `ImageUploader.svelte` (image upload progress).

## Public API (synchronous IDs)
```ts
// type: 1 info (default) | 2 warn | 3 error
addNotification(name: string, text: string, type?: 1 | 2 | 3): number
// status: 1 in-progress | 2 done (0 invalid on create)
addProcess(name: string, text: string, status: 1 | 2): number
// "" for name/text keeps the previous value; status omitted keeps previous
updateProcess(id: number, name?: string, text?: string, status?: 0 | 1 | 2): void
```

### ID generation (must return synchronously)
- A module-level in-memory counter seeded **once** at load:
  `counter = (Math.floor(Date.now() / 1000) % 1_000_000) * 1000`
  (last 6 digits of unix-seconds × 1000).
- Each `addNotification` / `addProcess` does `return ++counter` — pure sync, no `await`.
- IndexedDB persistence happens **after** the ID is returned (fire-and-forget write).
  This keeps callers fully synchronous while still durable.

## Files

### 1. `frontend/core/notifications.idb.ts`  (new — mirrors `core/agent/chat_history.idb.ts`)
Dexie DB scoped per company+env: db name `${companyID}_notifications_${env}`.

```ts
export interface NotificationRow {
  id: number            // the sync counter id (NOT ++id autoincrement — we own it)
  kind: 1 | 2           // 1 = notification, 2 = process
  name: string
  text: string
  type: 1 | 2 | 3       // notification kind: 1 info | 2 warn | 3 error (unused for processes)
  status: 0 | 1 | 2     // process kind: 1 in-progress | 2 done | 0 canceled (unused for notifications)
  createdAt: number     // Date.now()
  updatedAt: number
}
```
Store: `notifications: 'id, createdAt'` (we supply `id`, so no `++`).
Helpers: `loadNotifications()`, `putNotification(row)`, `patchNotification(id, patch)`,
and `pruneNotifications()` — keep the newest **500** rows (delete the rest by `createdAt`).

### 2. `frontend/core/notifications.svelte.ts`  (new — reactive store + API)
- `export const notifications = $state(new SvelteMap<number, NotificationRow>())`
  (keyed by id; the UI derives sorted lists / active-process count from it).
- Seed the counter once at module load.
- `addNotification` / `addProcess` / `updateProcess` mutate the map (instant, reactive)
  and call the `.idb.ts` write without awaiting.
- `updateProcess`: empty-string `name`/`text` → keep previous; `undefined` status → keep
  previous; reject unknown ids (no-op + `console.warn`, per AGENTS.md logging rule).
- A one-time `hydrate()` on first import loads persisted rows into the map so the user sees
  prior items after reload.

### 3. `frontend/domain-components/NotificationsButton.svelte`  (new — UI)
- Built on the existing `ButtonLayer` (same as the ⚙ settings button).
- Trigger button: `icon-info` in a `w-40 h-40 rounded-full bg-white/10` circle (matches the
  gear/reload siblings); shows a small badge with the count of **in-progress** processes
  (status 1). **Loading effect = animated rotating border** around the circular button while
  any process is in progress (a conic-gradient/`::before` ring spinner on the button edge — NOT
  a separate spinner inside). The border is static when nothing is running.
- Layer content: list of items newest-first — processes first (spinner for status 1, ✓ for 2,
  ✕ for 0) then notifications (color-coded by type: info/warn/error); each shows name + text +
  relative time via `formatTime`.

### 4. `frontend/domain-components/Header.svelte`  (edit)
Insert `<NotificationsButton />` in the **Right Actions** block, between the `pm-loading`
indicator and the settings `ButtonLayer` (the red-box position).

### 5. `frontend/ui-components/files/ImageUploader.svelte`  (edit — first consumer)
Replace the standalone progress UX wiring with process tracking:
- On upload start: `const pid = addProcess(fileName, tr('Uploading...|Subiendo...'), 1)`.
- On `onUploadProgress`: `updateProcess(pid, '', tr('Uploading X%...'), 1)`.
- On success: `updateProcess(pid, '', tr('Uploaded|Subida'), 2)`.
- On error: `updateProcess(pid, '', tr('Upload failed|Error al subir'), 0)`.
(The in-card overlay/progress bar stays as-is for immediate local feedback; the header layer
is the global accumulator.)

## Resolved decisions
1. `addNotification` carries a **type** (`1` info default / `2` warn / `3` error), not a status.
   Only **processes** have a live `status` + `updateProcess`.
2. Persistence is **per company+env** (DB name `${companyID}_notifications_${env}`).
3. Retention: keep the newest **500** rows; prune the rest on hydrate.
4. Trigger icon: **`icon-info`**. Loading effect is an **animated rotating border** on the
   circular button while any process is in progress.
5. **No user cancel** — `status = 0` is set only by the producing code.
