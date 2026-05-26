package text_search

import (
	"context"
	"fmt"
	"strconv"
	"strings"
)

// SearchResult is the decoded payload of an EVENT QUERY frame.
type SearchResult struct {
	IDs []int32
}

// Search runs a QUERY against {table}/p{partition}_s{statusGroup}.
// Wired but disabled until the read path lands — returns
// ErrNotImplemented so callers know it is deferred. buildQueryLine and
// decodeIDs are unit-tested so the follow-up PR can flip the body to
// call execPending directly.
func Search(ctx context.Context, table string, partition int32, statusGroup int8, query string, limit, offset int) (*SearchResult, error) {
	_ = buildQueryLine(table, partition, statusGroup, query, limit, offset)
	return nil, ErrNotImplemented
}

// buildQueryLine composes the wire command for a QUERY. Pulled out so
// tests can pin the framing without dialing a live daemon.
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

// decodeIDs parses payload tokens from an EVENT QUERY frame into
// int32 record IDs. GenixSearch emits each token as "<key>|<score>";
// only the key half is consumed here. Tokens without a '|' separator
// are accepted as a bare key for forward-compat.
func decodeIDs(payload []string) ([]int32, error) {
	ids := make([]int32, 0, len(payload))
	for _, tok := range payload {
		key := tok
		if idx := strings.IndexByte(tok, '|'); idx >= 0 {
			key = tok[:idx]
		}
		n, err := strconv.ParseInt(key, 10, 32)
		if err != nil {
			return nil, fmt.Errorf("text_search: bad key %q in EVENT QUERY payload: %w", tok, err)
		}
		ids = append(ids, int32(n))
	}
	return ids, nil
}
