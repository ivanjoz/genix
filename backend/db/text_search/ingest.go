package text_search

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// UpsertBatch upserts every record into {table}/p{partition}_s{statusGroup}.
//
// GenixSearch has no bulk POPI, so per batch we do:
//
//  1. A pipelined POPI sweep on the opposite-status bucket for every
//     record ID — guards against status flips (the server can't detect
//     a flip on its own).
//  2. A pipelined POPI sweep on the current-status bucket for records
//     whose SearchText is empty (PUSHI rejects empty payloads).
//  3. One or more PUSHI lines on the current-status bucket batching
//     the non-empty records, chunked to fit the channel buffer.
//
// Pipelining keeps each sweep to a single network round trip.
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

	allIDs := make([]int32, 0, len(records))
	emptyIDs := make([]int32, 0)
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

	if err := popIPipeline(ctx, conn, collection, otherBucket, allIDs); err != nil {
		return fmt.Errorf("text_search: popI other %s/%s: %w", collection, otherBucket, err)
	}
	if len(emptyIDs) > 0 {
		if err := popIPipeline(ctx, conn, collection, currentBucket, emptyIDs); err != nil {
			return fmt.Errorf("text_search: popI current %s/%s: %w", collection, currentBucket, err)
		}
	}
	if len(pushable) > 0 {
		if err := pushI(ctx, conn, collection, currentBucket, pushable); err != nil {
			return fmt.Errorf("text_search: pushI %s/%s: %w", collection, currentBucket, err)
		}
	}
	return nil
}

// UpsertRecord is the single-record convenience wrapper.
func UpsertRecord(ctx context.Context, table string, partition int32, statusGroup int8, recordID int32, searchText string) error {
	return UpsertBatch(ctx, table, partition, statusGroup, []Record{{ID: recordID, SearchText: searchText}})
}

// DeleteRecord removes one document from both s0 and s1 buckets.
func DeleteRecord(ctx context.Context, table string, partition int32, _ int8, recordID int32) error {
	return DeleteBatch(ctx, table, partition, []int32{recordID})
}

// DeleteBatch removes every recordID from both s0 and s1 buckets via
// pipelined POPI. The statusGroup the records previously lived in is
// not required — we sweep both buckets so the caller doesn't need to
// know.
func DeleteBatch(ctx context.Context, table string, partition int32, ids []int32) error {
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
		if err := popIPipeline(ctx, conn, collection, bucket, ids); err != nil {
			return fmt.Errorf("text_search: popI %s/%s: %w", collection, bucket, err)
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
		return fmt.Errorf("text_search: flushB %s/%s: %w", collection, bucket, err)
	}
	return nil
}

// commandBudget returns the number of bytes a command line may use,
// leaving room for the trailing LF the server reads as the line
// separator.
func commandBudget(c *searchConn) int {
	bs := c.bufferSize
	if bs <= 0 {
		bs = 20000
	}
	return bs - 1
}

// popIPipeline writes one POPI command per id on the same connection,
// flushes once, then reads N replies in order. This collapses N
// single-key removes into a single network round trip.
//
// Each reply must be RESULT 1 (existed) or RESULT 0 (missing); both
// are valid. A ProtocolError reply from the server is non-fatal for
// the connection — we surface it but keep the pool entry usable.
// Any framing failure marks the conn broken.
func popIPipeline(ctx context.Context, c *searchConn, collection, bucket string, ids []int32) error {
	if len(ids) == 0 {
		return nil
	}
	if c.broken {
		return fmt.Errorf("%w: connection already broken", ErrProtocol)
	}
	if deadline, ok := ctx.Deadline(); ok {
		_ = c.netConn.SetDeadline(deadline)
	} else {
		_ = c.netConn.SetDeadline(time.Now().Add(commandTimeout))
	}
	for _, id := range ids {
		line := "POPI " + collection + " " + bucket + " " + strconv.FormatInt(int64(id), 10)
		if _, err := c.writer.WriteString(line); err != nil {
			c.broken = true
			return err
		}
		if err := c.writer.WriteByte('\n'); err != nil {
			c.broken = true
			return err
		}
	}
	if err := c.writer.Flush(); err != nil {
		c.broken = true
		return err
	}
	var firstProtoErr error
	for range ids {
		line, err := readLine(c.reader)
		if err != nil {
			c.broken = true
			return err
		}
		res, perr := parseResult(line)
		if perr != nil {
			if _, ok := perr.(*ProtocolError); ok {
				if firstProtoErr == nil {
					firstProtoErr = perr
				}
				continue
			}
			c.broken = true
			return perr
		}
		if res.kind != kindResult {
			c.broken = true
			return fmt.Errorf("%w: expected RESULT for POPI, got %v", ErrProtocol, res.kind)
		}
	}
	return firstProtoErr
}

// pushI sends one or more PUSHI <col> <bucket> <id> "<text>" ... lines
// batched to fit conn.bufferSize. PUSHI replaces the entry keyed by
// (collection, bucket, id), so we don't need a prior POPI on the same
// bucket.
//
// NormalizeSearchText guarantees the payload contains no characters
// that quote() escapes, so the quoted form is exactly len(text)+2
// bytes — we can size each entry without rendering it.
//
// If a single record's text would exceed the budget on a fresh line,
// the text is trimmed at the last space below the limit (or hard-cut
// for a pathological single huge token).
func pushI(ctx context.Context, c *searchConn, collection, bucket string, records []Record) error {
	if len(records) == 0 {
		return nil
	}
	base := "PUSHI " + collection + " " + bucket
	budget := commandBudget(c)

	var sb strings.Builder
	sb.Grow(budget)
	sb.WriteString(base)
	for _, r := range records {
		idStr := strconv.FormatInt(int64(r.ID), 10)
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
