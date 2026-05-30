package db

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"app/db/text_search"

	"github.com/viant/xunsafe"
)

// IDWeight pairs a record key with the GenixSearch relevance score for a
// text-search hit (higher weight = better match). Returned by SearchTextIDs
// and SearchText so callers can rank results without reading ScyllaDB.
type IDWeight struct {
	ID     int32   `json:"id"`
	Weight float32 `json:"w"`
}

// resolveTextSearchTable compiles table E and asserts it declares a
// TextSearchColumn, returning the compiled ScyllaTable. Shared guard for
// SearchTextIDs / SearchText so a misconfigured table fails with a clear
// error instead of querying a non-existent index.
func resolveTextSearchTable[T TableBaseInterface[E, T], E TableSchemaInterface[E]]() (ScyllaTable, error) {
	scyllaTable := MakeScyllaTable[T, E]()
	if scyllaTable.textSearchIndex == nil {
		return scyllaTable, fmt.Errorf(`Table "%v": SearchText requires a TextSearchColumn in GetSchema`, scyllaTable.name)
	}
	return scyllaTable, nil
}

// SearchTextIDs queries table E's GenixSearch index for `query` within one
// partition + status group and returns matching record IDs with their
// relevance weights — no ScyllaDB read. statusGroup is 0 (status==0) or 1
// (active). limit<=0 lets the daemon apply its default page size.
func SearchTextIDs[T TableBaseInterface[E, T], E TableSchemaInterface[E]](partition int32, query string, statusGroup int8, limit int) ([]IDWeight, error) {
	scyllaTable, err := resolveTextSearchTable[T, E]()
	if err != nil {
		return nil, err
	}
	// Normalize the query the same way the write path normalized the indexed
	// text, so the daemon tokenizes both sides identically.
	normalized := text_search.NormalizeSearchText(query)
	if normalized == "" {
		return nil, nil
	}
	matches, err := text_search.SearchWeighted(context.Background(), scyllaTable.name, partition, statusGroup, normalized, limit, 0)
	if err != nil {
		return nil, err
	}
	weights := make([]IDWeight, len(matches))
	for i, m := range matches {
		weights[i] = IDWeight{ID: m.ID, Weight: m.Weight}
	}
	return weights, nil
}

// SearchText runs SearchTextIDs, then hydrates the full records from
// ScyllaDB into refSlice ordered by descending weight (best match first).
// It returns the id/weight ranking so callers keep the scores. Record IDs
// the index returns but the table no longer holds are simply absent.
func SearchText[T TableBaseInterface[E, T], E TableSchemaInterface[E]](refSlice *[]T, partition int32, query string, statusGroup int8, limit int) ([]IDWeight, error) {
	weights, err := SearchTextIDs[T, E](partition, query, statusGroup, limit)
	if err != nil || len(weights) == 0 {
		return weights, err
	}
	scyllaTable, err := resolveTextSearchTable[T, E]()
	if err != nil {
		return weights, err
	}

	ids := make([]int64, len(weights))
	for i, w := range weights {
		ids[i] = int64(w.ID)
	}
	fetched := []T{}
	if err := selectRecordsByPartitionIDs(scyllaTable, partition, ids, &fetched); err != nil {
		return weights, err
	}

	// Reorder fetched rows to match the weight ranking (the IN-clause select
	// returns rows in clustering-key order, not relevance order).
	rankByID := make(map[int32]int, len(weights))
	for i, w := range weights {
		rankByID[w.ID] = i
	}
	sort.SliceStable(fetched, func(a, b int) bool {
		return rankByID[recordKeyID(scyllaTable, &fetched[a])] < rankByID[recordKeyID(scyllaTable, &fetched[b])]
	})
	*refSlice = append(*refSlice, fetched...)
	return weights, nil
}

// recordKeyID reads a record's single primary-key value as int32, the same
// way the text-search write path reads it (the table's key column over an
// unsafe pointer to the record).
func recordKeyID[T any](scyllaTable ScyllaTable, record *T) int32 {
	return convertToInt32(scyllaTable.keys[0].GetRawValue(xunsafe.AsPointer(record)))
}

// selectRecordsByPartitionIDs loads records by primary key within one
// partition into refSlice, reusing the batched IN-clause select that
// QueryCachedIDs uses. No cache-version filtering — every requested ID is
// fetched.
func selectRecordsByPartitionIDs[T any](scyllaTable ScyllaTable, partition int32, ids []int64, refSlice *[]T) error {
	if len(ids) == 0 {
		return nil
	}
	if len(scyllaTable.keyspace) == 0 {
		scyllaTable.keyspace = connParams.Keyspace
	}
	partitionColumn := scyllaTable.partKey
	keyColumn := scyllaTable.keys[0]

	columnNames := make([]string, 0, len(scyllaTable.columns))
	for _, column := range scyllaTable.columns {
		if column.GetInfo().IsVirtual {
			continue
		}
		columnNames = append(columnNames, column.GetName())
	}

	for _, batch := range splitIDsIntoBatches(ids, queryCachedIDsMaxBatchSize) {
		queryValues := make([]any, 0, len(batch)+1)
		placeholders := make([]string, 0, len(batch))
		queryValues = append(queryValues, partition)
		for _, id := range batch {
			queryValues = append(queryValues, id)
			placeholders = append(placeholders, "?")
		}
		queryString := fmt.Sprintf(
			"SELECT %v FROM %v.%v WHERE %v = ? AND %v IN (%v)",
			strings.Join(columnNames, ", "),
			scyllaTable.keyspace, scyllaTable.name,
			partitionColumn.GetName(), keyColumn.GetName(),
			strings.Join(placeholders, ", "),
		)
		if err := scanSelectQueryRows(
			queryString, queryValues, buildDefaultScanColumns(columnNames),
			scyllaTable, refSlice, nil, nil, time.Now(),
		); err != nil {
			return err
		}
	}
	return nil
}
