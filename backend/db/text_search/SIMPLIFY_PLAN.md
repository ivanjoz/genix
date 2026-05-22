# text_search simplification — Sonic fork (BULKPUSH / BULKFLUSHO / replace-PUSH)

## What changes in the fork (relevant bits)

From `sonic/PROTOCOL.md` and `sonic/CHANGELOG.md`:

- `PUSH` is now **replace**, not append. Re-pushing the same `(collection, bucket, object)` overwrites tokens. There is also an internal FNV content-hash so identical repeated pushes are cheap.
- `BULKPUSH <collection> <bucket> <object> "<text>" [<object> "<text>"]...` — multi-object push in one frame, same replace semantics per object. Returns `RESULT <n>`.
- `BULKFLUSHO <collection> <bucket> <object> [<object>]...` — multi-object FLUSHO in one frame. Returns `RESULT <n>` (summed flushed records).
- Channel buffer / max line size: `BUFFER_SIZE = 20000`, `MAX_LINE_SIZE = 20002` (`sonic/src/channel/handle.rs:43-45`). The server advertises 20000 in `STARTED ... buffer(20000)`.

## What we drop

Current `ingest.go` does, per record:

```
FLUSHO  <col> <other-bucket>  <id>
FLUSHO  <col> <current-bucket> <id>     # <- no longer needed, PUSH replaces
PUSH    <col> <current-bucket> <id> "<text>"
```

That's 3 round trips per record. We can drop the second FLUSHO outright (replace-PUSH) and batch the other two with `BULKPUSH` / `BULKFLUSHO`.

The opposite-bucket FLUSHO still has to stay — Sonic can't detect a status flip (`s0 ↔ s1`) by itself; the record sits in the bucket where it was last pushed until something explicitly removes it.

## New ingest shape (one batch -> at most 3 commands, chunked by buffer)

`UpsertBatch(ctx, table, partition, statusGroup, records)`:

1. `BULKFLUSHO <col> <otherBucket>  id1 id2 …`   (every record's ID — protects against status flips)
2. `BULKFLUSHO <col> <currentBucket> id_with_empty_text …`   (only records whose normalized text is `""` — replaces the per-record FLUSHO)
3. `BULKPUSH   <col> <currentBucket> id "text" id "text" …`   (only records with non-empty text)

Each command is split into multiple frames when the line would exceed `conn.bufferSize`. For BULKPUSH a single oversized record's text is truncated (same rule we already have in `truncateForBuffer`, just adjusted for the BULKPUSH header).

`DeleteRecord` (and a new `DeleteBatch` for the bulk case) collapses to:

```
BULKFLUSHO <col> <bucketS0> id …
BULKFLUSHO <col> <bucketS1> id …
```

## Chunking rule (one place)

Helper `chunkLines(base, items, budget, append) -> []string` builds the command lines incrementally:

- start a line with `base` (`BULKFLUSHO <col> <bucket>` or `BULKPUSH <col> <bucket>`)
- for each item, compute its appended size; if it overflows the budget, flush the current line and start a new one
- per-record text in BULKPUSH is truncated on a word boundary when the item alone won't fit a fresh line (matches existing `truncateForBuffer` behavior)

Budget = `conn.bufferSize - 2` (CRLF). We don't need the +16 headroom we used before because no LANG() option is set.

## Files touched

- `backend/db/text_search/ingest.go` — rewrite `UpsertBatch`, `UpsertRecord`, `DeleteRecord`; add `DeleteBatch`; add internal `bulkFlushO` / `bulkPush` helpers.
- `backend/db/text_search/text.go` — `truncateForBuffer` becomes generic (`maxTextLen(bufferSize, headerLen)`) so both PUSH-style and BULKPUSH-style headers can use it. (Optional — can also be deleted if BULKPUSH chunker does its own arithmetic.)
- `backend/db/text_search/driver.go` — drop the stale "Upsert handling accommodates two Sonic quirks" doc block; it no longer applies.
- `backend/db/text_search/text_search_test.go` — update the ingest tests to expect the new wire frames (`BULKFLUSHO`, `BULKPUSH`). Add a chunking test where buffer size forces a split.

No call-site changes: `backend/db/text_search_index.go` keeps calling `UpsertBatch` with the same signature.

## Edge cases the new code preserves

- Empty `records` slice → no-op (no commands sent).
- All records have empty `SearchText` → only the two `BULKFLUSHO` lines are sent, no `BULKPUSH`.
- Single huge record whose text alone exceeds the buffer → word-boundary trim inside `BULKPUSH`. If even one token overflows, hard-trim. If the budget after the header is `<= 0`, skip the record (same as today).
- Connection error mid-batch → `ingestMgr.discard(conn)` (unchanged).
- `ErrProtocol` vs `*SonicError` distinction kept; only framing errors mark the conn broken.

## Test plan

1. `TestUpsertBatchSendsBulkCommands`: 2 records, both with text — expect `PING`, one `BULKFLUSHO ... otherBucket 1 2`, one `BULKPUSH ... currentBucket 1 "a" 2 "b"`.
2. `TestUpsertBatchEmptyTextOnlyFlushesBothBuckets`: 1 record with `""` — expect `BULKFLUSHO otherBucket 9`, `BULKFLUSHO currentBucket 9`, no `BULKPUSH`.
3. `TestUpsertBatchMixedSplit`: 3 records, two with text, one empty — both BULKFLUSHOs hit the right ID sets, BULKPUSH contains only the two non-empty.
4. `TestDeleteRecordUsesBulkFlushO`: single record — both buckets flushed via `BULKFLUSHO`.
5. `TestDeleteBatchSplitsByBuffer`: enough IDs that one BULKFLUSHO line would exceed the buffer; expect two BULKFLUSHO frames per bucket, all IDs covered.
6. `TestBulkPushTruncatesOversizeText`: one record whose text > budget, expect a BULKPUSH whose quoted text is a word-boundary-trimmed prefix.

Existing pure-helper tests (`CollectionAndBucket`, `NormalizeSearchText`, `quote`, `parseResult`, `decodeIDs`, `buildQueryLine`, `validateIdentifier`) stay unchanged.

## Out of scope for this PR

- Search/Suggest path is still `ErrNotImplemented`; no change.
- Pool sizing / liveness PING strategy — unchanged.
- The `BULKPUSH RESULT <n>` count is currently ignored (matches today's PUSH-returns-OK behavior). We can surface it later if a caller needs it.
