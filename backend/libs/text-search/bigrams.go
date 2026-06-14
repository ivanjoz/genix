// Package textsearch ports the genix-search Spanish bigram word encoder
// (genix-search src/pair/{mod,spanish_bigrams}.rs) into Go and packs a whole
// phrase into a single compact []uint8 buffer.
//
// Per-word encoding mirrors genix-search exactly so the bytes match what the
// daemon stores: text is normalized (lowercase, Spanish accents folded to their
// base letter, repeated adjacent letters collapsed, runs split on any
// non-alphanumeric rune), then each word becomes a sequence of "units":
//
//   - a letter pair "ab" -> a single byte via the LETTER_PAIR_ROWS table
//     (letterPairCode); an odd trailing letter pairs with the end marker '$'.
//   - a digit run -> two bytes: numFlag (255) followed by a quantized bucket.
//
// EncodeTextBigrams keeps at most maxUnitsPerWord (4) units per word — extra
// bigrams of an over-long word are dropped — and lays the phrase out as:
//
//	[ wordCount ][ packed 2-bit lengths ][ word units... ]
//	     1 byte    ceil(wordCount/4) bytes   sum(len) bytes
//
// wordCount is the boundary marker: it both states how many 2-bit length fields
// follow and lets a reader compute where the unit bytes begin —
// BigramUnitsOffset(wordCount). Each 2-bit field holds (unitLen-1), i.e. a value
// 0..3 standing for a word length of 1..4 units. Fields are packed big-endian
// within each byte, four words per byte (word 0 in the high bits).
package textsearch

// numFlag marks the start of a two-byte numeric unit (255 + quantized bucket),
// matching genix-search NUM_FLAG.
const numFlag uint8 = 255

// maxUnitsPerWord caps how many bigram units we keep per word. genix-search
// allows up to 8 (its "long" signature); here a word is clamped to 4 units so
// every word's length fits in the 2-bit header field. Extra units are dropped.
const maxUnitsPerWord = 4

// maxWords caps the phrase at 255 words so wordCount fits in the single leading
// byte. Words past the cap are ignored.
const maxWords = 255

// letterPairRow is one row of the genix-search letter-pair table: a first
// letter and the ordered groups of valid second letters (including the '$' end
// marker). The group index within the flattened table is the emitted code.
type letterPairRow struct {
	first  byte
	groups []string
}

// letterPairRows is adapted from LETTER_PAIR_ROWS in genix-search
// src/pair/spanish_bigrams.rs. Order is significant: the byte code for a pair is
// its 1-based position in the flattened (row, group) sequence. Letters sharing a
// group collapse to the same code, so they match fuzzily as the second letter of
// a bigram.
//
// On top of the upstream layout we apply three rules:
//
//   - Five phonetic confusion classes always share a single group within every
//     row, so similar-sounding letters never split across codes: {i,y}, {m,n},
//     {c,k,q}, {s,z,x}, and {b,v}. (q sounds like the hard c/k in Spanish — always
//     "qu"; b and v are pronounced identically.)
//   - No group ever combines two vowels (a,e,i,o,u). Since vowels are the highest
//     -probability letters, this keeps the most common second letters on distinct
//     codes. ({i,y} is allowed — only one of its members is a vowel.)
//   - The table is expanded to the full 254-code budget (codes 0 and 255 are
//     reserved as padding and numFlag). Splitting every two-vowel group overshoots
//     the ceiling, so the lowest-frequency consonant pairs (per Spanish letter
//     frequency) were merged back to land exactly on 254 — the minimum-loss way
//     to fit the ceiling.
//
// The upstream 'p' row also had a typo — duplicate 'y', missing 'f' — fixed here
// so each row is a clean permutation of the 25 other letters plus the '$' end
// marker.
var letterPairRows = []letterPairRow{
	{'a', []string{"r", "mn", "sxz", "l", "d", "ckq", "g", "tf", "bv", "p", "j", "iy", "u", "o", "e", "hw$"}},
	{'b', []string{"o", "a", "iy", "e", "lr", "usxz", "dwmn", "gtckq", "hfj", "pv$"}},
	{'c', []string{"o", "a", "he", "iykq", "rtu", "lsxz", "nmbv", "gdf", "jpw$"}},
	{'d', []string{"o", "a", "iy", "e", "ur", "hsxz", "wtl", "vbckq", "fjpmn", "g$"}},
	{'e', []string{"sxz", "mn", "r", "lckq", "tf", "a", "giy", "dp", "jbvo", "uwh$"}},
	{'f', []string{"o", "r", "e", "iy", "a", "u", "ltp", "hsxz", "vdbmn", "gjckq", "w$"}},
	{'g', []string{"a", "e", "riy", "u", "o", "lhmn", "dsxz", "pbv", "tf", "kcjqw$"}},
	{'h', []string{"a", "o", "e", "iy", "ut", "wmn", "lr", "fckqsxzbv", "dgj", "p$"}},
	{'i', []string{"mn", "a", "ckq", "dl", "to", "e", "psxzbv", "g", "rf", "ujwhy$"}},
	{'j', []string{"a", "o", "hiy", "umn", "dckqbv", "efg", "lp", "rtsxz", "w$"}},
	{'k', []string{"iy", "e", "asxz", "o", "rut", "gfmn", "lwh", "dpjcq", "bv$"}},
	{'l', []string{"a", "o", "e", "iy", "usxz", "tmnckq", "dgbv", "p", "fhw", "rj$"}},
	{'m', []string{"a", "e", "op", "iybv", "usxz", "ftckq", "gr", "dnhl", "jw$"}},
	{'n', []string{"a", "t", "o", "e", "diy", "gckqsxz", "ubv", "jf", "rl", "mhpw$"}},
	{'o', []string{"mn", "r", "lsxz", "ut", "gckq", "jde", "pbv", "afh", "iyw$"}},
	{'p', []string{"a", "iy", "e", "o", "r", "uhsxz", "ltf", "ckqbv", "djmn", "wg$"}},
	{'q', []string{"u", "sxz", "ack", "dbv", "efg", "hjiy", "lmn", "oprt", "w$"}},
	{'r', []string{"a", "e", "iy", "o", "tu", "dgmn", "vb", "fpckqsxz", "l", "hjw$"}},
	{'s', []string{"a", "iy", "ot", "eckq", "pu", "hnm", "dl", "fwgbv", "rxjz$"}},
	{'t', []string{"e", "a", "iy", "ru", "lhsxz", "ockq", "wpmn", "df", "gbjv$"}},
	{'u', []string{"r", "e", "lmn", "tsxz", "a", "iyckq", "dbv", "pgj", "of", "hw$"}},
	{'v', []string{"e", "a", "o", "iy", "uckq", "ltmn", "dr", "bfsxz", "ghj", "pw$"}},
	{'w', []string{"iy", "oh", "e", "a", "umnsxzbv", "rdtlckq", "fgj", "p$"}},
	{'x', []string{"t", "iy", "p", "e", "a", "ockq", "fu", "lhszbv", "rdmn", "gjw$"}},
	{'y', []string{"o", "a", "e", "sxzbv", "dltumn", "prckq", "gif", "whj$"}},
	{'z', []string{"a", "ou", "e", "iyckq", "pmnsx", "thl", "dfgbv", "jrw$"}},
}

// spanishStopwords are common Spanish connectors/articles that carry no search
// signal and are dropped before encoding. Direct port of genix-search
// is_spanish_stopword (src/pair/mod.rs).
var spanishStopwords = map[string]struct{}{
	"a": {}, "al": {}, "con": {}, "de": {}, "del": {}, "desde": {}, "el": {},
	"en": {}, "es": {}, "esa": {}, "ese": {}, "eso": {}, "esos": {}, "esta": {},
	"esto": {}, "fue": {}, "ha": {}, "la": {}, "las": {}, "le": {}, "les": {},
	"lo": {}, "los": {}, "me": {}, "mi": {}, "mis": {}, "ni": {}, "no": {},
	"nos": {}, "o": {}, "para": {}, "por": {}, "que": {}, "se": {}, "ser": {},
	"si": {}, "sin": {}, "son": {}, "soy": {}, "su": {}, "sus": {}, "tu": {},
	"y": {},
}

// isSpanishStopword reports whether an already-normalized token should be
// dropped as a common connector word.
func isSpanishStopword(token string) bool {
	_, ok := spanishStopwords[token]
	return ok
}

// EncodeTextBigrams normalizes a Spanish phrase and packs every word's bigram
// units into one buffer. The layout is:
//
//	out[0]                      = wordCount (the boundary marker)
//	out[1 : 1+ceil(N/4)]        = 2-bit length per word (value = unitLen-1)
//	out[BigramUnitsOffset(N):]  = concatenated word units, in word order
//
// Each word contributes between 1 and maxUnitsPerWord (4) units; a word longer
// than that is truncated. Returns nil when the phrase has no encodable words.
func EncodeTextBigrams(phrase string) []uint8 {
	tokens := normalizeTokens(phrase)
	if len(tokens) == 0 {
		return nil
	}
	if len(tokens) > maxWords {
		tokens = tokens[:maxWords]
	}

	// Encode each word first so we know its unit length for the header.
	wordUnits := make([][]uint8, 0, len(tokens))
	totalUnits := 0
	for _, token := range tokens {
		if isSpanishStopword(token) {
			continue
		}
		units := encodeWordUnits(token)
		if len(units) == 0 {
			continue
		}
		wordUnits = append(wordUnits, units)
		totalUnits += len(units)
	}
	if len(wordUnits) == 0 {
		return nil
	}

	wordCount := len(wordUnits)
	headerBytes := lengthHeaderBytes(wordCount)
	out := make([]uint8, 0, 1+headerBytes+totalUnits)

	// Boundary marker: word count. A reader derives the unit-region start from
	// it via BigramUnitsOffset.
	out = append(out, uint8(wordCount))

	// Packed 2-bit lengths: four words per byte, word 0 in the high bits.
	header := make([]uint8, headerBytes)
	for i, units := range wordUnits {
		field := uint8(len(units)-1) & 0b11 // unitLen 1..4 -> 0..3
		shift := 6 - uint(2*(i%4))
		header[i/4] |= field << shift
	}
	out = append(out, header...)

	// Word unit bytes, in order.
	for _, units := range wordUnits {
		out = append(out, units...)
	}

	return out
}

// lengthHeaderBytes returns the number of bytes needed to hold wordCount 2-bit
// length fields (four per byte).
func lengthHeaderBytes(wordCount int) int {
	return (wordCount + 3) / 4
}

// BigramUnitsOffset returns the index into an EncodeTextBigrams buffer where the
// concatenated word-unit bytes begin, given the wordCount stored in out[0].
func BigramUnitsOffset(wordCount int) int {
	return 1 + lengthHeaderBytes(wordCount)
}

// WordUnitLength reads back the unit length (1..4) of word i from a header byte
// slice (the bytes between the wordCount marker and the unit region).
func WordUnitLength(header []uint8, i int) int {
	shift := 6 - uint(2*(i%4))
	return int((header[i/4]>>shift)&0b11) + 1
}

// encodeWordUnits turns one already-normalized token into at most
// maxUnitsPerWord bigram/numeric units, mirroring genix-search
// encode_normalized_token (clamped to 4 units instead of 8).
func encodeWordUnits(token string) []uint8 {
	if token == "" {
		return nil
	}
	bytes := []byte(token)
	units := make([]uint8, 0, maxUnitsPerWord)
	i := 0
	for i < len(bytes) && len(units) < maxUnitsPerWord {
		c := bytes[i]
		switch {
		case isDigit(c):
			// A numeric run needs two units (flag + bucket); stop if they
			// won't both fit, matching genix-search's `units.len()+2 > N` guard.
			if len(units)+2 > maxUnitsPerWord {
				return units
			}
			start := i
			for i < len(bytes) && isDigit(bytes[i]) {
				i++
			}
			units = append(units, numFlag, numericBucket(parseUint(bytes[start:i])))
		case isLower(c):
			if i+1 < len(bytes) && isLower(bytes[i+1]) {
				units = append(units, letterPairCode(c, bytes[i+1]))
				i += 2
			} else {
				units = append(units, letterPairCode(c, '$'))
				i++
			}
		default:
			i++
		}
	}
	return units
}

// letterPairCode returns the 1-based byte code for the pair (first, second)
// using letterPairRows. Code 0 is reserved as padding/no-unit, so codes start
// at 1. second is either an ASCII lowercase letter or the '$' end marker.
func letterPairCode(first, second byte) uint8 {
	code := uint8(1)
	for _, row := range letterPairRows {
		if row.first == first {
			for _, group := range row.groups {
				if containsByte(group, second) {
					return code
				}
				code++
			}
		} else {
			code += uint8(len(row.groups))
		}
	}
	// The table covers every ASCII lowercase first letter; reaching here means
	// `first` was not lowercase, which the caller must prevent.
	return 0
}

// numericBucket quantizes a number to the nearest bucket boundary, returning the
// bucket code (1..254). Port of genix-search numeric_bucket.
func numericBucket(number uint64) uint8 {
	bestCode := uint8(1)
	bestDistance := ^uint64(0)
	for code := uint8(1); code <= 254; code++ {
		boundary := numericBoundary(code)
		distance := absDiff(number, boundary)
		if distance < bestDistance {
			bestCode = code
			bestDistance = distance
		}
	}
	return bestCode
}

// numericBoundary maps a bucket code to its representative value. Codes 0 and
// 255 are reserved (padding and numFlag). Port of genix-search numeric_boundary.
func numericBoundary(code uint8) uint64 {
	switch {
	case code >= 1 && code <= 81:
		return uint64(code - 1)
	case code >= 82 && code <= 121:
		return 80 + uint64(code-81)*5
	case code >= 122 && code <= 181:
		return 280 + uint64(code-121)*15
	case code >= 182 && code <= 221:
		return 1180 + uint64(code-181)*25
	case code >= 222 && code <= 241:
		return 3180 + uint64(code-221)*50
	default: // 242..=254
		return 8180 + uint64(code-241)*100
	}
}

// normalizeTokens splits a phrase into normalized words, mirroring
// genix-search `normalize`: fold accents and case, collapse repeated adjacent
// letters, and break a token on any rune that is neither a letter nor a digit.
// Digits do not collapse and reset the repeat tracker.
func normalizeTokens(input string) []string {
	var tokens []string
	var current []byte
	lastLetter := byte(0)

	flush := func() {
		if len(current) > 0 {
			tokens = append(tokens, string(current))
			current = current[:0]
		}
		lastLetter = 0
	}

	for _, r := range input {
		nc, ok := normalizeChar(r)
		switch {
		case ok && isLower(nc):
			if lastLetter != nc {
				current = append(current, nc)
			}
			lastLetter = nc
		case ok && isDigit(nc):
			current = append(current, nc)
			lastLetter = 0
		default:
			flush()
		}
	}
	flush()
	return tokens
}

// normalizeChar lowercases ASCII letters, folds Spanish accented vowels and ñ to
// their base ASCII letter, and passes ASCII digits through. Returns ok=false for
// anything else (acts as a word separator). Port of genix-search normalize_char.
func normalizeChar(r rune) (byte, bool) {
	switch {
	case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
		return byte(r), true
	case r >= 'A' && r <= 'Z':
		return byte(r) + ('a' - 'A'), true
	}
	switch r {
	case 'á', 'à', 'ä', 'â', 'Á', 'À', 'Ä', 'Â':
		return 'a', true
	case 'é', 'è', 'ë', 'ê', 'É', 'È', 'Ë', 'Ê':
		return 'e', true
	case 'í', 'ì', 'ï', 'î', 'Í', 'Ì', 'Ï', 'Î':
		return 'i', true
	case 'ó', 'ò', 'ö', 'ô', 'Ó', 'Ò', 'Ö', 'Ô':
		return 'o', true
	case 'ú', 'ù', 'ü', 'û', 'Ú', 'Ù', 'Ü', 'Û':
		return 'u', true
	case 'ñ', 'Ñ':
		return 'n', true
	}
	return 0, false
}

func isLower(c byte) bool { return c >= 'a' && c <= 'z' }
func isDigit(c byte) bool { return c >= '0' && c <= '9' }

func containsByte(s string, c byte) bool {
	for i := 0; i < len(s); i++ {
		if s[i] == c {
			return true
		}
	}
	return false
}

func absDiff(a, b uint64) uint64 {
	if a > b {
		return a - b
	}
	return b - a
}

// parseUint reads an all-digit byte slice into a uint64, saturating to the max
// value on overflow (matching genix-search, which maps unparseable runs to
// u64::MAX before bucketing).
func parseUint(digits []byte) uint64 {
	const max = ^uint64(0)
	var n uint64
	for _, d := range digits {
		dv := uint64(d - '0')
		if n > (max-dv)/10 {
			return max
		}
		n = n*10 + dv
	}
	return n
}
