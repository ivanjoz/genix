# Cache Version Feature Plan

## Goal
Implement automatic cache-version tracking for tables with `SaveCacheVersion: true`, using `cache_version` records keyed by `(partition, table_id)` and assigning per-record cache version (`ccv`) on selects.

## Scope
1. Add runtime support in ORM write flow (`Insert`, `Update`, `UpdateExclude`) to increment cache-version groups.
2. Add runtime support in ORM read flow (`Select/Exec`) to assign `ccv` from `cache_version`.
3. Enforce strict validations and panic on invalid schema/model configuration.
4. Ensure `cache_version` table model is valid and included in deployment controllers.

## Implemented Steps
1. Added table runtime flag:
`ScyllaTable.saveCacheVersion` now mirrors `TableSchema.SaveCacheVersion`.

2. Added dedicated feature module:
Created `db/cache_version.go` with:
- model/schema validation for cache-version feature usage.
- `[]byte <-> map[uint8]uint8` conversion helpers.
- DB access helpers for loading/saving cache versions.
- write hook (`updateCacheVersionsAfterWrite`) to increment versions by `uint8(id)` group.
- read hook (`assignCacheVersionsAfterSelect`) to assign `ccv` to loaded records.

3. Integrated write hooks:
After successful DB write:
- `Insert` updates cache version groups.
- `Update` updates cache version groups.
- `UpdateExclude` updates cache version groups.

4. Integrated read hooks:
- `selectExec` validates cache-version model requirements early.
- `selectExec` ensures partition and key columns are always selected internally when cache-version is enabled.
- `selectExec` assigns `ccv` to all fetched records before returning.

5. Fixed cache-version table model declaration:
- `CacheVersionTable` column generics now use `CacheVersionTable`.
- partition column changed to `int64` to support `int32|int64` source partitions.
- fixed recursive schema bug (`Partition: e.Partition`).

6. Added deploy registration:
- `exec/migrate.go` now includes `makeDBController[db.CacheVersion]()`.

## Validation Rules Enforced
When `SaveCacheVersion` is enabled:
1. Table must have exactly one key.
2. Key type must be `int16`, `int32`, or `int64`.
3. Partition must exist.
4. Partition type must be `int32` or `int64`.
5. Model must include a `uint8` field named `CacheVersion` or with JSON tag `ccv`.
6. Missing/invalid configuration panics immediately.

## Behavior Summary
1. Table ID is computed as `BasicHashInt(table_name)`.
2. Cache groups are computed as `uint8(record_id)`.
3. Increment wraps naturally (`255 -> 0`) using `uint8` overflow.
4. Encoded byte layout is `[group, version, group, version, ...]`.
5. Serialization order is stable (sorted by group ID) to avoid non-deterministic byte output.

## Open Questions
1. `table_id` hashing:
Should `table_id` hash use only `table_name` (current implementation) or `keyspace.table_name`?

2. Update coverage:
Should `InsertOrUpdate` increment cache versions once per successful operation branch (current behavior, inherited from `Insert` and `UpdateExclude`)?

3. Select with explicit field projection:
Current implementation auto-includes partition and key fields internally to compute `ccv`. Is this acceptable for your API semantics?

4. Legacy existing `cache_version` table schema:
If your DB already has `partition int` (`int32`) instead of `bigint` (`int64`), do you want a migration step now or keep backward compatibility logic in code?
