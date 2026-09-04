# recompute_user_accesos

Rebuilds `users.accesos_computed` and `users.accesos_sub_computed` for every user of every active
company, from the profiles and direct access grants they hold.

```bash
cd scripts && go run . recompute_user_accesos
```

Also available in the `./deploy.sh` TUI under **Base de Datos → Recomputar Accesos de Usuarios**.

## Why it exists

The two columns are the authorization blobs `fareward` caches to gate every route. Their grant word
(`[14 bits accesoID][2 bits nivel-1]`) is unchanged, but the release that added sub-accesses changed
the column's **container and byte order**: little-endian `[]uint16` became big-endian `[]byte`, and
`accesos_sub_computed` appeared beside it.

The word survived, its byte order did not. Read big-endian, a little-endian blob still decodes — into
accesses nobody granted — so without this run every user is denied everything. See
`docs/SUB_ACCESSES_PLAN.md` for the format and `backend/core/accesos-blob.go` for the only encoder.

## What it does

For each active company:

1. Reads every user in the partition, **including inactive ones** — their blobs are what a
   reactivation would authorize with, and nothing else recomputes them.
2. Merges each user's profiles through `buildAccesosComputedFromPerfiles` (highest nivel per access,
   union of sub-access masks), then folds in their direct `AccessLevelIDs` exactly as
   `PostUsuarios` does. Leaving that second source out would strip every access granted to a user
   directly rather than through a profile.
3. Encodes both blobs with `core.EncodeAccesosGrants` and writes only the users whose bytes changed,
   naming only those two columns in the `UPDATE`.
4. Sends one `InvalidateUserAccess(companyID, InvalidateAllCompanyUsers)` per company that had
   writes, so the daemon re-reads instead of serving its cache until the TTL expires.

User 1 is expected to come out unchanged with empty blobs: its grants are synthesized at login by
`buildBootstrapAdminAccesos` and never stored, which is why `resolveRouteAccess` lets it bypass the
daemon.

## Re-running it

Safe and idempotent. It recomputes from the profiles, which are the source of truth, so a second run
writes the same bytes and reports `0 reescritos`. That also makes it the repair tool for any user
whose blobs drifted — not only a one-off migration.

## Where it belongs in a deploy

1. `fn-homologate` (deploy.sh → **Recrear Tablas**) to add the new columns.
2. Backend **and** the `fareward` daemon together — the `fareward:v9` domain makes a mixed pair fail
   at the first frame.
3. **This script.**
4. Frontend.

Between steps 2 and 3 every authenticated request is denied, so keep the window short.
