package text_search

import (
	"context"
	"fmt"
	"strconv"
	"strings"
)

// Match is one result from a QUERY: a record key plus the relevance score
// GenixSearch assigned it. Weight is 0 when the daemon omits the "|<score>"
// half of a token.
type Match struct {
	ID     int32
	Weight float32
}

// SearchResult is the decoded payload of an EVENT QUERY frame (IDs only).
type SearchResult struct {
	IDs []int32
}

// Search runs a QUERY against {table}/p{partition}_s{statusGroup} and
// returns only the matching record IDs. Thin wrapper over SearchWeighted
// for callers that don't need the scores.
func Search(ctx context.Context, table string, partition int32, statusGroup int8, query string, limit, offset int) (*SearchResult, error) {
	matches, err := SearchWeighted(ctx, table, partition, statusGroup, query, limit, offset)
	if err != nil {
		return nil, err
	}
	ids := make([]int32, len(matches))
	for i, m := range matches {
		ids[i] = m.ID
	}
	return &SearchResult{IDs: ids}, nil
}

// SearchWeighted runs a QUERY against {table}/p{partition}_s{statusGroup}
// and returns matches with their GenixSearch scores. It checks out a
// search-mode connection, issues the QUERY as a PENDING->EVENT exchange via
// execPending, and decodes the EVENT payload. A broken connection is
// discarded by release; healthy ones return to the pool.
func SearchWeighted(ctx context.Context, table string, partition int32, statusGroup int8, query string, limit, offset int) ([]Match, error) {
	// An empty query would emit an invalid QUERY line — treat as no matches.
	if strings.TrimSpace(query) == "" {
		return nil, nil
	}
	initPools()
	conn, err := searchMgr.acquire(ctx)
	if err != nil {
		return nil, err
	}
	line := buildQueryLine(table, partition, statusGroup, query, limit, offset)
	res, err := conn.execPending(ctx, line)
	searchMgr.release(conn) // release discards the conn when execPending marked it broken
	if err != nil {
		return nil, fmt.Errorf("text_search: query %s: %w", table, err)
	}
	return decodeMatches(res.payload)
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

// decodeMatches parses payload tokens from an EVENT QUERY frame into Match
// values. GenixSearch emits each token as "<key>|<score>"; a token without
// a '|' separator is accepted as a bare key with zero weight for
// forward-compat.
func decodeMatches(payload []string) ([]Match, error) {
	matches := make([]Match, 0, len(payload))
	for _, tok := range payload {
		key := tok
		var weight float32
		if idx := strings.IndexByte(tok, '|'); idx >= 0 {
			key = tok[:idx]
			if scoreStr := tok[idx+1:]; scoreStr != "" {
				score, err := strconv.ParseFloat(scoreStr, 32)
				if err != nil {
					return nil, fmt.Errorf("text_search: bad score %q in EVENT QUERY payload: %w", tok, err)
				}
				weight = float32(score)
			}
		}
		n, err := strconv.ParseInt(key, 10, 32)
		if err != nil {
			return nil, fmt.Errorf("text_search: bad key %q in EVENT QUERY payload: %w", tok, err)
		}
		matches = append(matches, Match{ID: int32(n), Weight: weight})
	}
	return matches, nil
}

// decodeIDs is the IDs-only decoder retained for the unit tests and the
// Search wrapper's historical contract.
func decodeIDs(payload []string) ([]int32, error) {
	matches, err := decodeMatches(payload)
	if err != nil {
		return nil, err
	}
	ids := make([]int32, len(matches))
	for i, m := range matches {
		ids[i] = m.ID
	}
	return ids, nil
}
