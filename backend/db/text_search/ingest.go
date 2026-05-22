package text_search

import (
	"context"
	"fmt"
	"strconv"
	"strings"
)

// UpsertBatch upserts every record into {table}/p{partition}_s{statusGroup}.
//
// The forked Sonic supports BULKPUSH / BULKFLUSHO and PUSH now replaces
// (rather than appends) per-object tokens, so one batch collapses to at
// most two commands:
//
//  1. BULKFLUSHO on the opposite-status bucket for every record ID, so a
//     status flip never leaves a stale row in the previous bucket — Sonic
//     can't detect a flip on its own.
//  2. BULKPUSH on the current-status bucket for every record with non-empty
//     SearchText. Records with empty text fall back to a BULKFLUSHO on the
//     current bucket (Sonic rejects empty PUSH payloads).
//
// Each command line is chunked to stay below the buffer size advertised
// by Sonic in STARTED.
func UpsertBatch(ctx context.Context, table string, partition int32, statusGroup int8, records []Record) error {
	if len(records) == 0 {
		return nil
	}
	collection, currentBucket := CollectionAndBucket(table, partition, statusGroup)
	_, otherBucket := CollectionAndBucket(table, partition, 1-statusGroup)
	if err := validateIdentifier(collection); err != nil {
		return err
	}
	if err := validateIdentifier(currentBucket); err != nil {
		return err
	}
	if err := validateIdentifier(otherBucket); err != nil {
		return err
	}

	allIDs := make([]int64, 0, len(records))
	emptyIDs := make([]int64, 0)
	pushable := make([]Record, 0, len(records))
	for _, r := range records {
		allIDs = append(allIDs, r.ID)
		if r.SearchText == "" {
			emptyIDs = append(emptyIDs, r.ID)
		} else {
			pushable = append(pushable, r)
		}
	}

	initPools()
	conn, err := ingestMgr.acquire(ctx)
	if err != nil {
		return err
	}
	defer ingestMgr.release(conn)

	if err := bulkFlushO(ctx, conn, collection, otherBucket, allIDs); err != nil {
		conn.broken = true
		return fmt.Errorf("text_search: bulkFlushO other %s/%s: %w", collection, otherBucket, err)
	}
	if len(emptyIDs) > 0 {
		if err := bulkFlushO(ctx, conn, collection, currentBucket, emptyIDs); err != nil {
			conn.broken = true
			return fmt.Errorf("text_search: bulkFlushO current %s/%s: %w", collection, currentBucket, err)
		}
	}
	if len(pushable) > 0 {
		if err := bulkPush(ctx, conn, collection, currentBucket, pushable); err != nil {
			conn.broken = true
			return fmt.Errorf("text_search: bulkPush %s/%s: %w", collection, currentBucket, err)
		}
	}
	return nil
}

// UpsertRecord is the single-record convenience wrapper.
func UpsertRecord(ctx context.Context, table string, partition int32, statusGroup int8, recordID int64, searchText string) error {
	return UpsertBatch(ctx, table, partition, statusGroup, []Record{{ID: recordID, SearchText: searchText}})
}

// DeleteRecord removes one document from both s0 and s1 buckets.
func DeleteRecord(ctx context.Context, table string, partition int32, _ int8, recordID int64) error {
	return DeleteBatch(ctx, table, partition, []int64{recordID})
}

// DeleteBatch removes every recordID from both s0 and s1 buckets via
// BULKFLUSHO. The statusGroup the records previously lived in is not
// required — we flush both buckets so the caller doesn't need to know.
func DeleteBatch(ctx context.Context, table string, partition int32, ids []int64) error {
	if len(ids) == 0 {
		return nil
	}
	collection, bucketS0 := CollectionAndBucket(table, partition, 0)
	_, bucketS1 := CollectionAndBucket(table, partition, 1)
	if err := validateIdentifier(collection); err != nil {
		return err
	}
	if err := validateIdentifier(bucketS0); err != nil {
		return err
	}
	if err := validateIdentifier(bucketS1); err != nil {
		return err
	}

	initPools()
	conn, err := ingestMgr.acquire(ctx)
	if err != nil {
		return err
	}
	defer ingestMgr.release(conn)

	for _, bucket := range [2]string{bucketS0, bucketS1} {
		if err := bulkFlushO(ctx, conn, collection, bucket, ids); err != nil {
			conn.broken = true
			return fmt.Errorf("text_search: bulkFlushO %s/%s: %w", collection, bucket, err)
		}
	}
	return nil
}

// FlushBucket clears an entire (table, partition, statusGroup) index.
// Exposed for the deferred backfill / cleanup tooling; not used on the
// hot write path.
func FlushBucket(ctx context.Context, table string, partition int32, statusGroup int8) error {
	collection, bucket := CollectionAndBucket(table, partition, statusGroup)
	if err := validateIdentifier(collection); err != nil {
		return err
	}
	if err := validateIdentifier(bucket); err != nil {
		return err
	}
	initPools()
	conn, err := ingestMgr.acquire(ctx)
	if err != nil {
		return err
	}
	defer ingestMgr.release(conn)
	line := fmt.Sprintf("FLUSHB %s %s", collection, bucket)
	if _, err := conn.exec(ctx, line); err != nil {
		conn.broken = true
		return fmt.Errorf("text_search: flushB %s/%s: %w", collection, bucket, err)
	}
	return nil
}

// commandBudget returns the number of bytes a command line may use,
// leaving room for the trailing CRLF Sonic writes around frames.
func commandBudget(c *sonicConn) int {
	bs := c.bufferSize
	if bs <= 0 {
		bs = 20000
	}
	return bs - 2
}

// bulkFlushO sends BULKFLUSHO <col> <bucket> id1 id2 ... splitting into
// as many command frames as needed to stay under the conn's buffer.
func bulkFlushO(ctx context.Context, c *sonicConn, collection, bucket string, ids []int64) error {
	if len(ids) == 0 {
		return nil
	}
	base := "BULKFLUSHO " + collection + " " + bucket
	budget := commandBudget(c)

	var sb strings.Builder
	sb.Grow(budget)
	sb.WriteString(base)
	for _, id := range ids {
		idStr := strconv.FormatInt(id, 10)
		need := 1 + len(idStr)
		if sb.Len()+need > budget && sb.Len() > len(base) {
			if _, err := c.exec(ctx, sb.String()); err != nil {
				return err
			}
			sb.Reset()
			sb.WriteString(base)
		}
		sb.WriteByte(' ')
		sb.WriteString(idStr)
	}
	if sb.Len() > len(base) {
		if _, err := c.exec(ctx, sb.String()); err != nil {
			return err
		}
	}
	return nil
}

// bulkPush sends BULKPUSH <col> <bucket> <id> "<text>" ... batched to
// fit conn.bufferSize. NormalizeSearchText guarantees the payload
// contains no characters that quote() escapes, so the quoted form is
// exactly len(text)+2 bytes — we can size each entry without rendering it.
//
// If a single record's text would exceed the budget on a fresh line, the
// text is trimmed at the last space below the limit (or hard-cut for a
// pathological single huge token).
func bulkPush(ctx context.Context, c *sonicConn, collection, bucket string, records []Record) error {
	if len(records) == 0 {
		return nil
	}
	base := "BULKPUSH " + collection + " " + bucket
	budget := commandBudget(c)

	var sb strings.Builder
	sb.Grow(budget)
	sb.WriteString(base)
	for _, r := range records {
		idStr := strconv.FormatInt(r.ID, 10)
		text := r.SearchText
		entry := 1 + len(idStr) + 1 + len(text) + 2 // " " id " " "text"
		if sb.Len()+entry > budget && sb.Len() > len(base) {
			if _, err := c.exec(ctx, sb.String()); err != nil {
				return err
			}
			sb.Reset()
			sb.WriteString(base)
		}
		// Even on a fresh line the record may not fit — trim text.
		maxText := budget - len(base) - 1 - len(idStr) - 1 - 2
		if maxText <= 0 {
			continue
		}
		if len(text) > maxText {
			if cut := strings.LastIndexByte(text[:maxText], ' '); cut > 0 {
				text = text[:cut]
			} else {
				text = text[:maxText]
			}
			if text == "" {
				continue
			}
		}
		sb.WriteByte(' ')
		sb.WriteString(idStr)
		sb.WriteByte(' ')
		sb.WriteString(quote(text))
	}
	if sb.Len() > len(base) {
		if _, err := c.exec(ctx, sb.String()); err != nil {
			return err
		}
	}
	return nil
}
