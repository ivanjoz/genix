package text_search

import (
	"fmt"
	"regexp"
	"strings"
)

// Record is one row to push into a Sonic index. ID becomes the Sonic
// object_id (formatted as decimal). SearchText should already be
// normalized via NormalizeSearchText.
type Record struct {
	ID         int64
	SearchText string
}

// PickStatusGroup centralizes the s0/s1 rule: status == 0 -> group 0,
// everything else -> group 1.
func PickStatusGroup(status int8) int8 {
	if status == 0 {
		return 0
	}
	return 1
}

// CollectionAndBucket maps an ORM (table, partition, statusGroup) triple
// to the Sonic collection/bucket pair. Format:
//
//	collection = {tableName}
//	bucket     = p{partitionID}_s{statusGroup}
//
// The "p" prefix keeps the bucket valid when partition_id is 0 (Sonic
// identifiers must start with a non-digit). Negative partition values
// are not expected; abs() defensively avoids producing a leading "-".
func CollectionAndBucket(tableName string, partitionID int32, statusGroup int8) (string, string) {
	id := partitionID
	if id < 0 {
		id = -id
	}
	return tableName, fmt.Sprintf("p%d_s%d", id, statusGroup)
}

// validIdentifier matches the subset of bytes Sonic accepts for
// collection / bucket names. Pre-flight check so a malformed name
// can't slip into a PUSH/FLUSHO command line.
var validIdentifier = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

func validateIdentifier(name string) error {
	if !validIdentifier.MatchString(name) {
		return fmt.Errorf("text_search: invalid identifier %q", name)
	}
	return nil
}

// NormalizeSearchText lowercases and strips everything outside [a-z0-9 ],
// then collapses runs of whitespace to a single space. This matches the
// pre-tokenization the FTS5 backend used to do and keeps PUSH payloads
// free of bytes that would need protocol escaping.
func NormalizeSearchText(s string) string {
	if s == "" {
		return ""
	}
	var b strings.Builder
	b.Grow(len(s))
	prevSpace := true
	for _, r := range s {
		switch {
		case r >= 'A' && r <= 'Z':
			b.WriteByte(byte(r) + ('a' - 'A'))
			prevSpace = false
		case (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9'):
			b.WriteRune(r)
			prevSpace = false
		default:
			if !prevSpace {
				b.WriteByte(' ')
				prevSpace = true
			}
		}
	}
	out := b.String()
	if len(out) > 0 && out[len(out)-1] == ' ' {
		out = out[:len(out)-1]
	}
	return out
}
