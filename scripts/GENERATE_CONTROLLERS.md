# Generate Controllers Script

Regenerates `backend/exec/controllers.generated.go` by scanning the backend for every base struct that embeds `db.TableStruct[XTable, X]`.

## Why

`MakeScyllaControllers()` is the single source of truth that `db.DeployScylla(...)` (invoked by `fn-homologate`) uses to create/migrate every ScyllaDB table. Hand-maintaining the list drifts as tables are added or removed — this script rebuilds it deterministically.

## Usage

```bash
./app.sh generate_controllers
```

The script runs automatically as part of `deploy.sh` option **[5] Recrear Tablas**, immediately before `fn-homologate`. You do not normally need to invoke it directly.

## What It Does

1. Walks `backend/` recursively, parsing every `.go` file with `go/parser`.
2. For each struct declaration, checks whether its first field is an embedded `db.TableStruct[XTable, X]` **and** the second type argument matches the struct's own name. That match selects only the "base" half of each `X` / `XTable` pair.
3. Emits one `makeDBController[<Type>]()` entry per detected base struct, sorted alphabetically by the qualified reference (`alias.TypeName`).
4. Builds a standardized import block:
   - Packages ending in `/types` get a `<parentDir>Types` alias (e.g. `app/configuracion/types` → `configuracionTypes`).
   - `app/types` gets `appTypes`.
   - All others use the bare package name (e.g. `app/core`, `app/db`).
   - Types defined in `app/exec` (the output package itself) are referenced without a qualifier.
5. Formats the output with `go/format` and writes it to `backend/exec/controllers.generated.go`.

## Skipped Paths

- `backend/db/` — ORM-internal tables (`Increment`, `CacheVersion`) are bootstrapped by `db.Init()` and must NOT be registered again here.
- `vendor/`, `node_modules/`, `.git/`, `*_test.go`, and the output file itself.

## After Adding or Removing a Table

No manual step is required: the next `./app.sh generate_controllers` run (or any execution of `deploy.sh` option [5]) will pick the change up.
