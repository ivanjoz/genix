# Cache-group S3 snapshots (`db.TypeCacheGroupS3`)

Status: implemented, unit-tested; the Scylla integration test (§8) is not written yet. Prerequisite
done: index-group marks are written **before** the rows (`markIndexGroupsBeforeWrite`,
`genix-orm/scylla/index_groups.go`). The colbin check runs at the end of the table compile
(`requireCacheGroupSnapshotCodec`), with the record type from `BaseStructType()`.

## 1. Goal

A cache group (`db.TypeCacheGroup`) already lets a browser keep a group of records and skip it while
the group's `last_write` is unchanged. The first read of a group — a new browser, another user, a
cleared IndexedDB — still costs a Scylla read through a secondary index (index lookup plus base-table
reads), and every client pays it once per group.

`db.TypeCacheGroupS3` keeps the same per-group cache **on the server**, in the S3 Express bucket the
ORM already uses (`genix-orm--use1-az4--x-s3`, root `orm/`). The first client to read a settled group
pays the DB read and writes the snapshot; every later miss of that group, from any client, is served
from S3 with no DB read until the group is written again.

Opt-in per index. Nothing changes for the client: the `cc-gh` / `cc-lw` protocol, `RecordGroup`
and the frontend group cache stay as they are.

## 2. Declaration

```go
Indexes: []db.Index{
	{Type: db.TypeCacheGroupS3, Keys: db.Cols(e.Date)},
	{Type: db.TypeCacheGroupS3, Keys: db.Cols(e.Date, e.Status)},
}
```

- `TypeCacheGroupS3` creates exactly the indexes `TypeCacheGroup` creates (single key: a local index
  on the column; scalar hash: a local index on `zz_ig_`; set hash: a global index on `zz_igs_`) and
  writes the same `__index_updated` rows.
- There is no S3 variant of `TypeCacheGroupFromKey`: its fetch is a primary-key range read, already
  cheap, so a snapshot saves little there.
- The table must round-trip through colbin: every stored column with a struct field must be in the
  colbin schema (no `cb:"-"` on a stored column). Otherwise **the compile panics**. The managed
  columns without a field (`created`, `zz_ig_*`) are skipped, as for delta snapshots. Reason: a
  dropped column would come back zero from S3 and filled from the DB, so a result would depend on
  whether the cache was hit.

## 3. Read path (`queryIndexGroups`, `scylla/select_grouped.go`)

The freshness probe and `splitIndexGroupFetches` are unchanged. Groups the client holds with the
current label still go back as metadata only. For each group to fetch, on an S3 index:

| Group | Action |
|---|---|
| Unsettled (marked < `indexGroupSettleTicks`, 16 s, ago; label 0) | DB read, post-filtered, **never stored** |
| Settled, snapshot label == probe `last_write`, fingerprint matches, younger than 2 h | decode, post-filter in memory, return. **No DB read** |
| Settled, no file / different label / other fingerprint / decode error / older than 2 h | DB read of the whole group, store it, post-filter in memory |
| Settled, the DB read is empty | not stored |

- **Whole group, filtered in memory.** Today the post-filter runs inside `scanSelectQueryRows`. For an
  S3 group the scan runs without it and `recordMatchesPostFilter` is applied after, to the DB rows and
  to the decoded snapshot alike, so one snapshot serves every query shape that lands on the group.
- **GETs** run in parallel, capped at 16 at a time per request. A 30-day `BETWEEN` on `Date` is up to
  30 GETs.
- **S3 errors are misses**, logged: a failed GET reads the DB, a failed PUT is dropped. The request
  never fails because of the store.
- Grouped reads already reject `Select()`, `Limit`, `OrderDesc`, `GroupBy`: there is no projected or
  cut snapshot case.

## 4. Write rules

1. **Only a full DB read of a settled group writes a snapshot.** A snapshot is never written from
   another snapshot, nor from an unsettled read.
2. **The label is the probe's `last_write`** for that group, the same value the client gets as `lw`.
3. **`PutFileIfMatch` with the ETag the GET returned** (empty when the file did not exist). Two
   readers racing on a miss: one write wins, the other gets `ErrFilePreconditionFailed` and drops it.
   A slow reader holding an older label cannot overwrite a newer file; even if it could, the label no
   longer matches the probe, so the file is only a miss.
4. **Max age 2 h.** The snapshot keeps the time its DB read began (`ReadTime`). Older than
   `RecordSnapshotMaxAgeTicks` (3600 ticks, 2 h) it is a miss even with a matching label: the group is
   read from the DB and the file rewritten. A requested group pays at most one DB read per 2 h; a group
   nobody requests pays nothing.

## 5. File

Key, under the store root:

```
cache-group/<table>/<partition>/<indexID>/<group hash>.cb
```

Layout: the shared snapshot codec (§6), with `[schema fingerprint u64 LE][label uvarint][read time
uvarint][colbin of []Record]`; for a group the label is `last_write`.

## 6. Shared codec

Delta snapshots and group snapshots use one codec in `db/cache`: `DeltaSnapshot` /
`EncodeDeltaSnapshot` / `DecodeDeltaSnapshot` become `RecordSnapshot` / `EncodeRecordSnapshot` /
`DecodeRecordSnapshot` with the fields `SchemaFingerprint`, `Cursor` (delta: the C; group:
`last_write`), `ReadTime`, `Records`. `DeltaSnapshotMaxAgeTicks` / `IsDeltaSnapshotExpired` become
`RecordSnapshotMaxAgeTicks` / `IsRecordSnapshotExpired`, one 2 h limit for both. The eligibility and fingerprint helper (`deltaSnapshotCodecFor`)
becomes `recordSnapshotCodecFor` and serves both. Keys stay feature-specific (`DeltaSnapshotKey`,
`CacheGroupSnapshotKey`).

## 7. Files

- `genix-orm/db/schema.go`: `TypeCacheGroupS3` (next free value), doc, `IndexTypeNames`.
- `genix-orm/scylla/aliases.go`, `backend/db/db.go`: alias.
- `genix-orm/scylla/reflect.go`, `index_groups.go`: route to `registerIndexGroup`, set
  `indexGroupInfo.snapshotInS3`, the colbin eligibility panic.
- `genix-orm/db/cache/delta_snapshot.go` → shared codec (§6); `cache_group_snapshot.go` for the key.
- `genix-orm/scylla/cache_group_snapshot.go` (new): read-or-serve-or-store for one group, called from
  `fetchIndexGroupRecordGroups`.
- `genix-orm/scylla/delta_snapshot.go`: move to the shared codec.
- `genix-orm/scylla/ORM_INTERNALS.md`: the S3 variant in the group freshness section.
- `backend/sales/types/sales.go`: opt in (§10).

## 8. Tests

- Codec round trip; a truncated header is an error.
- Eligibility: a stored column tagged `cb:"-"` panics at compile.
- Unit, with `utils.MemoryFileStore` and a fixed clock:
  - a settled miss reads the DB and stores the file with the probe's label;
  - the next read with the same label serves the file, no DB read;
  - a different label is a miss and rewrites the file;
  - a matching label older than 2 h is a miss and rewrites the file;
  - an unsettled group is read from the DB and never stored;
  - the post-filter applies to a served snapshot;
  - an empty group is not stored;
  - a stale ETag drops the PUT without failing the request.
- Integration (`GENIX_ORM_SCYLLA_TEST`), next to `TestDeltaSnapshotServesTheNextReplace`: write,
  settle, read (stores), read again from a "new client" (served), write, settle, read (new label).

## 9. Known gaps (shared with the browser cache, wider with S3)

A stale group can be read by a settled fetch in two cases:

- A data phase that lands more than 16 s after its mark.
- `LocalOne` writes and reads: a lost mutation (a replica that never got the write) is not bounded by
  time. Reading at `LocalQuorum` alone does not close it while writes are `LocalOne`.

In S3 the max age (§4.4) bounds both: a stale snapshot is served to every first read of the group for
at most 2 h, then the DB read replaces it. Browsers are not covered: one holding the group under the
same label never refetches, and keeps it stale until the group's next write. Accepted.

## 10. Rollout

- Opt in: three cache groups of `SaleOrder` — `Date`, `Date+Status`, `Date+ClientID`. The product
  set group stays `TypeCacheGroup`.
- No schema change in Scylla; no keyspace reseed needed for this feature.
- No S3 lifecycle rule for now (noted): snapshots of old groups stay until removed by hand.

## 11. Before implementing

- Done: delta snapshots already run on the new window code (`serveDeltaSnapshot` plans with
  `cache.PlanDeltaSync` from the file's Cursor and ReadTime); the codec rename (§6) only moved them
  onto the shared helpers (`scylla/record_snapshot.go`).
