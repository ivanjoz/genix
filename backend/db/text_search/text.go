package text_search

import (
	"fmt"
	"regexp"
	"strings"
)

// Record is one row to push into a GenixSearch index. ID is the
// signed 32-bit key the server stores directly (no string interning).
// SearchText should already be normalized via NormalizeSearchText.
type Record struct {
	ID         int32
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

// CollectionAndBucket maps an ORM (table, partition, statusGroup)
// triple to the GenixSearch collection/bucket pair. Format:
//
//	collection = {tableName}
//	bucket     = p{partitionID}_s{statusGroup}
//
// The "p" prefix keeps the bucket valid when partition_id is 0
// (identifiers must start with a non-digit). Negative partition
// values are not expected; abs() defensively avoids producing a
// leading "-".
func CollectionAndBucket(tableName string, partitionID int32, statusGroup int8) (string, string) {
	id := partitionID
	if id < 0 {
		id = -id
	}
	return tableName, fmt.Sprintf("p%d_s%d", id, statusGroup)
}

// validIdentifier matches the subset of bytes the daemon accepts for
// collection / bucket names. Pre-flight check so a malformed name
// can't slip into a PUSHI/POPI command line.
var validIdentifier = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

func validateIdentifier(name string) error {
	if !validIdentifier.MatchString(name) {
		return fmt.Errorf("text_search: invalid identifier %q", name)
	}
	return nil
}

// NormalizeSearchText lowercases ASCII, folds Spanish accents to their
// base letter (á→a, ñ→n, ...) and keeps [a-z0-9]. Whitespace is the only
// word separator: runs collapse to a single space. Any other character
// (punctuation, symbols like '&', '-', '.') is DROPPED, not turned into a
// space, so "Marca Za&Co" → "marca zaco" rather than "marca za co".
//
// This must fold accents the same way the daemon's normalizer does
// (genix-search src/pair/mod.rs normalize_char). Folding client-side keeps
// PUSHI payloads free of bytes that would need protocol escaping; the key
// point is that the accent becomes its base letter — never a separator —
// so "Jamón" and "Jamon" both tokenize to "jamon" on write and on query.
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
		case isWhitespace(r):
			if !prevSpace {
				b.WriteByte(' ')
				prevSpace = true
			}
		default:
			if folded := foldAccent(r); folded != 0 {
				b.WriteByte(folded)
				prevSpace = false
			}
			// Any other rune (punctuation, symbols) is dropped entirely so
			// it merges the surrounding letters instead of splitting them.
		}
	}
	out := b.String()
	if len(out) > 0 && out[len(out)-1] == ' ' {
		out = out[:len(out)-1]
	}
	return out
}

// isWhitespace reports whether r should act as a word separator. Limited to
// the common ASCII/Unicode spacing runes; everything else is handled by the
// accent-fold / drop path in NormalizeSearchText.
func isWhitespace(r rune) bool {
	switch r {
	case ' ', '\t', '\n', '\r', '\f', '\v', ' ':
		return true
	}
	return false
}

// foldAccent maps a Spanish accented letter to its base ASCII letter,
// mirroring the daemon's normalize_char so both sides tokenize identically.
// Returns 0 for runes that are not accented Latin letters.
func foldAccent(r rune) byte {
	switch r {
	case 'á', 'à', 'ä', 'â', 'Á', 'À', 'Ä', 'Â':
		return 'a'
	case 'é', 'è', 'ë', 'ê', 'É', 'È', 'Ë', 'Ê':
		return 'e'
	case 'í', 'ì', 'ï', 'î', 'Í', 'Ì', 'Ï', 'Î':
		return 'i'
	case 'ó', 'ò', 'ö', 'ô', 'Ó', 'Ò', 'Ö', 'Ô':
		return 'o'
	case 'ú', 'ù', 'ü', 'û', 'Ú', 'Ù', 'Ü', 'Û':
		return 'u'
	case 'ñ', 'Ñ':
		return 'n'
	}
	return 0
}
