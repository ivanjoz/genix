# Text Search Index ORM Plan

## Goal

Enable `TableSchema.TextSearchColumn` so the ORM automatically maintains a derived search table for records whose configured text field changes.

Example:

```go
// Purpose: Tell the ORM which text column must feed the derived search index.
// Rationale: Product search should be maintained by the write path, not by caller code.
TextSearchColumn: e.Name,
```

For table `product` and column `name`, the ORM creates and maintains:

```text
product_name_search_idx
```

## Physical Table

The derived table should contain:

```sql
CREATE TABLE keyspace.product_name_search_idx (
    partition_id int,
    hash int,
    bigrams list<tinyint>,
    status tinyint,
    id [base_table_id_type],
    PRIMARY KEY ((partition_id), hash, id)
);
```

Notes:

- `partition_id` mirrors the base table partition column.
- `hash` is the `int32` hash of one generated word-combination.
- `bigrams` stores only the first compact bigram bucket from each normalized word that has no numbers.
- `status` mirrors the base record status when the table has a status column.
- `id` mirrors the base table single key column type and is used to locate stale index rows for the same base record.
- A local index is required on `id` for maintenance reads/deletes scoped by partition:

```sql
-- Purpose: Fetch current search-index rows for one base record before rewriting them.
-- Rationale: Updates need to delete hashes that disappeared from the new text.
CREATE INDEX product_name_search_idx__id_index
ON keyspace.product_name_search_idx ((partition_id), id);
```

## Metadata Changes

Add a dedicated descriptor to `ScyllaTable`:

```go
// Purpose: Store compiled metadata for the write-maintained text-search table.
// Rationale: The write path must avoid reflection/schema recomputation per record.
type textSearchIndexInfo struct {
    tableName string
    sourceColumn IColInfo
    partitionColumn IColInfo
    idColumn IColInfo
    statusColumn IColInfo
}
```

Add `textSearchIndex *textSearchIndexInfo` to `ScyllaTable`.

Compilation rules in `makeTable`:

- If `schema.TextSearchColumn == nil`, do nothing.
- Require a partition column.
- Require exactly one base key column.
- Require the key column to be numeric and reuse its CQL type in the index table.
- Require the text search column to be `string`.
- Resolve `status` by column name `status`; if absent, write `status = 0`.

## Deploy Changes

During `Deploy`:

1. Create the base table as today.
2. If `table.textSearchIndex != nil`, check `system_schema.columns` for `keyspace.[table]_[column]_search_idx`.
3. Create the search index table when missing.
4. Create the maintenance index on `(partition_id), id` when missing.
5. Do not alter old incompatible search-index tables automatically in the first version; log the mismatch and fail loudly if a required column has a wrong type.

## Text Parsing

Create `text_search_index_parser.go`.

Normalization:

- Lowercase.
- Convert accents to plain ASCII.
- Convert `ñ` to `n`.
- Keep pure-number tokens as-is for hash generation, for example `100`.
- Drop one-letter words.
- Drop common Spanish connectors from `CommonSpanishWords`.
- Collapse repeated adjacent letters before bigram extraction, for example `interregional` -> `interegional`.
- For mixed alphanumeric tokens, keep normalized letters and digits for hash generation.
- Keep only the first 12 normalized words before generating hashes and bigrams.
- Store at most one bigram per word: only the first letter-letter bigram found in each normalized word without numbers.
- Skip words containing any number when generating the `bigrams` list.

Implementation shape:

```go
// Purpose: Convert user-facing text into deterministic search tokens.
// Rationale: Search-index writes must produce the same hashes on every node.
func parseTextSearchWords(rawText string) []string
```

```go
// Purpose: Convert each normalized non-numeric word into its first compact bigram bucket id.
// Rationale: Words containing numbers participate in hashes but not in bigram filtering.
func makeTextSearchBigrams(words []string) []uint8
```

The `[]uint8` value is converted to `[]int8` only at DB bind time because CQL uses `tinyint`. With the 12-word cap, each row stores at most 12 bigram values, and fewer when words contain numbers.

## Hash Generation

For the first 12 normalized words, generate combinations of 1, 2, and 3 words across all words in the text:

- Single word: `["aceite"]`
- Pairs: `["aceite", "oliva"]`, `["aceite", "extra"]`, `["oliva", "extra"]`
- Triples: `["aceite", "oliva", "extra"]`

Rules:

- Preserve word order from the source text, but combine all available words within the first 12 normalized words, not only consecutive words.
- Deduplicate hashes per record.
- Hash the joined combination with a stable delimiter, for example `aceite\x1foliva`.
- Use the existing `BasicHashInt` if it is the project standard for `int32` hash groups; otherwise add a small stable `int32` hash helper beside the parser.

Implementation shape:

```go
// Purpose: Build all searchable word-combination hashes for one text value.
// Rationale: Search queries can match one, two, or three relevant product words.
func buildTextSearchRows(partitionID int64, baseID int64, status int8, rawText string) []textSearchIndexRow
```

## Write Path

Hook after the base insert/update batch succeeds, before cache-version update:

```text
base insert/update batch
sync index groups
sync table-backed views
sync text-search index
update cache versions
```

Reasoning:

- The base record must exist before derived search rows are visible.
- Search index failures should return an error so callers know the write was not fully indexed.
- Cache versions should be updated after all derived write maintenance succeeds.

Maintenance algorithm:

1. For each record, build new rows from `TextSearchColumn`.
2. Fetch existing rows with:

```sql
SELECT hash, id FROM product_name_search_idx
WHERE partition_id = ? AND id IN (...);
```

3. Compare existing `(partition_id, id, hash)` with new hashes.
4. Delete missing hashes.
5. Insert new or changed rows.
6. For matching hashes, update only if `bigrams` or `status` changed.

Batch behavior:

- Use `gocql.UnloggedBatch`, same as current derived writes.
- Group maintenance fetches by partition to keep queries bounded.
- Add `DebugFull` logs with table name, records count, inserted count, deleted count, updated count.

## Query Path Later

This plan only covers table creation and write maintenance.

A later query API can expose:

```go
// Purpose: Search records by normalized text using the derived search table.
// Rationale: Callers should not know the physical search-index table name.
func SearchText[T any](partition any, rawQuery string, limit int) ([]int64, error)
```

The read path should parse the query with the same parser and fetch candidate IDs by `hash`.

## Tests

Add focused tests in `backend/db`:

- Parser removes connectors, one-letter words, accents, and `ñ`.
- Parser preserves pure numbers.
- Bigram conversion uses `BigramMap`, skips words containing numbers, keeps only the first bigram per remaining word, and is deterministic.
- Combination generation creates 1, 2, and 3-word hashes without duplicates.
- Schema compilation rejects invalid `TextSearchColumn` configurations.
- Deploy script generation creates `product_name_search_idx` with the expected columns.
- Write sync inserts new search rows and deletes stale rows when text changes.

Run:

```bash
# Purpose: Compile the db package quickly after ORM changes.
# Rationale: This catches generic/reflection signature errors before running slower integration tests.
go test ./db -run '^$' -count=1
```

## Open Questions

1. Resolved: `status` always mirrors a column literally named `status`; if the table has no such column, write `0`.
2. Resolved: `id` in the derived table preserves the exact base key CQL type (`int`, `bigint`, etc.).
3. Resolved: word combinations use all ordered combinations from the normalized text, preserving source order.
4. Should a deleted/inactive base row remove search-index rows, or keep them with `status` changed?
5. Should pure-number words participate in hashes only, or also contribute any synthetic bigram bucket?
