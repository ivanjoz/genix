package textsearch

import (
	"reflect"
	"strconv"
	"strings"
	"testing"
)

// lp is a test shorthand for the reference letter-pair code.
func lp(a, b byte) uint8 { return letterPairCode(a, b) }

func TestLetterPairCodesAreStable(t *testing.T) {
	// Anchored to row 'a' having 15 groups (after the phonetic-class merge, the
	// vowel split, and the frequency-based merges that fit the 254 budget): 'r'
	// is the first group, '$' rides the last group ("w$"), and row 'b' starts
	// immediately after.
	cases := []struct {
		a, b byte
		want uint8
	}{
		{'a', 'r', 1},
		{'a', '$', 15},
		{'b', 'o', 16},
	}
	for _, c := range cases {
		if got := lp(c.a, c.b); got != c.want {
			t.Errorf("letterPairCode(%q,%q) = %d, want %d", c.a, c.b, got, c.want)
		}
	}
}

// TestLetterPairCodeCapacity enforces the hard invariant: byte 0 is reserved as
// padding and byte 255 as numFlag, so every letter-pair code must land in
// 1..254. If this fails the table has too many groups — merge one back.
func TestLetterPairCodeCapacity(t *testing.T) {
	total := 0
	for _, row := range letterPairRows {
		total += len(row.groups)
	}
	if total > 254 {
		t.Errorf("letter-pair table has %d codes; must be <= 254 (255 is numFlag, 0 is padding)", total)
	}
	if c := lp('z', '$'); c == numFlag {
		t.Errorf("last code letterPairCode('z','$') = %d collides with numFlag (255)", c)
	}
}

func TestNormalizeTokens(t *testing.T) {
	cases := []struct {
		in   string
		want []string
	}{
		{"CAÑA 100LTS", []string{"cana", "100lts"}},
		{"Arroz--Extra", []string{"aroz", "extra"}},
		{"COOOLL", []string{"col"}},
		{"camisa de manga larga", []string{"camisa", "de", "manga", "larga"}},
	}
	for _, c := range cases {
		if got := normalizeTokens(c.in); !reflect.DeepEqual(got, c.want) {
			t.Errorf("normalizeTokens(%q) = %v, want %v", c.in, got, c.want)
		}
	}
}

func TestEncodeWordUnits(t *testing.T) {
	// "sol" -> [so, l$]
	if got := encodeWordUnits("sol"); !reflect.DeepEqual(got, []uint8{lp('s', 'o'), lp('l', '$')}) {
		t.Errorf("sol units = %v", got)
	}
	// "z" -> [z$]
	if got := encodeWordUnits("z"); !reflect.DeepEqual(got, []uint8{lp('z', '$')}) {
		t.Errorf("z units = %v", got)
	}
	// "100lts" -> [numFlag, bucket(100), lt, s$]
	if got := encodeWordUnits("100lts"); !reflect.DeepEqual(got, []uint8{numFlag, numericBucket(100), lp('l', 't'), lp('s', '$')}) {
		t.Errorf("100lts units = %v", got)
	}
	// Over-long word truncates to 4 units: "camisa" -> ca, mi, sa + (truncated)
	if got := encodeWordUnits("camisa"); len(got) != 3 {
		t.Errorf("camisa expected 3 units, got %v", got)
	}
	if got := encodeWordUnits("extraordinario"); len(got) != maxUnitsPerWord {
		t.Errorf("long word should clamp to %d units, got %d", maxUnitsPerWord, len(got))
	}
}

func TestNumericBucketBoundaries(t *testing.T) {
	cases := []struct {
		n    uint64
		want uint8
	}{
		{0, 1}, {80, 81}, {83, 82}, {85, 82}, {9580, 254}, {20000, 254},
	}
	for _, c := range cases {
		if got := numericBucket(c.n); got != c.want {
			t.Errorf("numericBucket(%d) = %d, want %d", c.n, got, c.want)
		}
	}
}

func TestEncodeTextBigramsLayout(t *testing.T) {
	out := EncodeTextBigrams("camisa de manga larga")
	if out == nil {
		t.Fatal("expected non-nil encoding")
	}

	wordCount := int(out[0])
	if wordCount != 3 { // "de" is a dropped stopword
		t.Fatalf("wordCount = %d, want 3", wordCount)
	}

	unitsOffset := BigramUnitsOffset(wordCount)
	header := out[1:unitsOffset]
	if len(header) != 1 { // 3 words -> 1 header byte
		t.Fatalf("header len = %d, want 1", len(header))
	}

	// Rebuild expected per-word units and verify both the 2-bit lengths and the
	// concatenated unit bytes.
	wantWords := [][]uint8{
		encodeWordUnits("camisa"),
		encodeWordUnits("manga"),
		encodeWordUnits("larga"),
	}

	pos := unitsOffset
	for i, want := range wantWords {
		gotLen := WordUnitLength(header, i)
		if gotLen != len(want) {
			t.Errorf("word %d length field = %d, want %d", i, gotLen, len(want))
		}
		got := out[pos : pos+gotLen]
		if !reflect.DeepEqual([]uint8(got), want) {
			t.Errorf("word %d units = %v, want %v", i, got, want)
		}
		pos += gotLen
	}
	if pos != len(out) {
		t.Errorf("trailing bytes: consumed %d of %d", pos, len(out))
	}
}

func TestEncodeTextBigramsEmpty(t *testing.T) {
	if out := EncodeTextBigrams("   --  !! "); out != nil {
		t.Errorf("expected nil for non-encodable phrase, got %v", out)
	}
}

func TestHeaderSizingEightWords(t *testing.T) {
	// 8 words -> 16 length bits -> 2 header bytes (the spec's example).
	out := EncodeTextBigrams("uno dos tres cuatro cinco seis siete ocho")
	if int(out[0]) != 8 {
		t.Fatalf("wordCount = %d, want 8", out[0])
	}
	if got := lengthHeaderBytes(8); got != 2 {
		t.Fatalf("lengthHeaderBytes(8) = %d, want 2", got)
	}
	if BigramUnitsOffset(8) != 3 {
		t.Fatalf("BigramUnitsOffset(8) = %d, want 3", BigramUnitsOffset(8))
	}
}

// --- Lossy decoder (test-only) ----------------------------------------------
//
// The encoding is lossy: a unit byte identifies a first letter plus a *group*
// of possible second letters, not the exact second letter. To decode we look
// the code back up and pick a representative second letter (the first non-'$'
// char of its group). Numeric units decode to their bucket's boundary value.

// decodePairCode reverses letterPairCode: code -> (firstLetter, secondGroup).
func decodePairCode(code uint8) (byte, string) {
	c := uint8(1)
	for _, row := range letterPairRows {
		for _, group := range row.groups {
			if c == code {
				return row.first, group
			}
			c++
		}
	}
	return '?', ""
}

func decodeWord(units []uint8) string {
	var b strings.Builder
	for j := 0; j < len(units); j++ {
		u := units[j]
		if u == numFlag {
			if j+1 < len(units) {
				b.WriteString(strconv.FormatUint(numericBoundary(units[j+1]), 10))
				j++
			}
			continue
		}
		first, group := decodePairCode(u)
		b.WriteByte(first)
		for k := 0; k < len(group); k++ {
			if group[k] != '$' { // '$' is the end marker, not a real letter
				b.WriteByte(group[k])
				break
			}
		}
	}
	return b.String()
}

func decodeTextBigrams(out []uint8) []string {
	if len(out) == 0 {
		return nil
	}
	wordCount := int(out[0])
	offset := BigramUnitsOffset(wordCount)
	header := out[1:offset]
	pos := offset
	words := make([]string, 0, wordCount)
	for i := 0; i < wordCount; i++ {
		n := WordUnitLength(header, i)
		units := out[pos : pos+n]
		pos += n
		words = append(words, decodeWord(units))
	}
	return words
}

func TestEncodeDecodeSpanishPhrases(t *testing.T) {
	phrases := []string{
		"Jamón del país de mayor calidad",
		"El pato más rico del Perú en provincia",
		"La cocina de don Pepe es la mejor",
	}
	for _, phrase := range phrases {
		encoded := EncodeTextBigrams(phrase)
		decoded := decodeTextBigrams(encoded)
		t.Logf("\n  original: %q\n  encoded : %v (%d bytes)\n  decoded : %q",
			phrase, encoded, len(encoded), strings.Join(decoded, " "))
	}
}
