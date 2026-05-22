package text_search

import (
	"context"
	"fmt"
	"strconv"
)

// SearchResult is the decoded payload of an EVENT QUERY frame.
type SearchResult struct {
	IDs []int64
}

// Search runs a Sonic QUERY against {table}/p{partition}_s{statusGroup}.
// Wired but disabled for this PR — returns ErrNotImplemented from the
// exported surface so callers know the read path is deferred. The
// internal helpers (buildQueryLine / decodeIDs) are unit-tested so the
// follow-up PR can flip the body to call execPending directly.
func Search(ctx context.Context, table string, partition int32, statusGroup int8, query string, limit, offset int) (*SearchResult, error) {
	_ = buildQueryLine(table, partition, statusGroup, query, limit, offset)
	return nil, ErrNotImplemented
}

// Suggest runs a Sonic SUGGEST and returns the candidate words. Same
// staging as Search: deferred to the read PR.
func Suggest(ctx context.Context, table string, partition int32, statusGroup int8, word string) ([]string, error) {
	_ = buildSuggestLine(table, partition, statusGroup, word)
	return nil, ErrNotImplemented
}

// buildQueryLine composes the wire command for a QUERY. Pulled out so
// tests can pin the framing without dialing a live Sonic.
func buildQueryLine(table string, partition int32, statusGroup int8, query string, limit, offset int) string {
	collection, bucket := CollectionAndBucket(table, partition, statusGroup)
	line := fmt.Sprintf("QUERY %s %s %s", collection, bucket, quote(query))
	if limit > 0 {
		line += fmt.Sprintf(" LIMIT(%d)", limit)
	}
	if offset > 0 {
		line += fmt.Sprintf(" OFFSET(%d)", offset)
	}
	return line
}

// buildSuggestLine composes the wire command for SUGGEST.
func buildSuggestLine(table string, partition int32, statusGroup int8, word string) string {
	collection, bucket := CollectionAndBucket(table, partition, statusGroup)
	return fmt.Sprintf("SUGGEST %s %s %s", collection, bucket, quote(word))
}

// decodeIDs parses the payload tokens from an EVENT QUERY frame into
// int64 record IDs. Sonic returns them as decimal strings.
func decodeIDs(payload []string) ([]int64, error) {
	ids := make([]int64, 0, len(payload))
	for _, tok := range payload {
		n, err := strconv.ParseInt(tok, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("text_search: bad object_id %q in EVENT QUERY payload: %w", tok, err)
		}
		ids = append(ids, n)
	}
	return ids, nil
}
