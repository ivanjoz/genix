package libs

import (
	"errors"
	"strings"
	"sync"
	"unicode"
)

var (
	reverseSilabas map[string]uint8
	once           sync.Once
)

type EncodedPhrase struct {
	Content       []uint8
	WordsSize     []uint8
	WordsSizeBits []uint8
	Is4BitSize    bool
}

// EnsureInitialized makes sure the syllable map is fully populated and the reverse index is built.
func EnsureInitialized() {
	once.Do(func() {
		if len(silabas) <= 30 { 
			makeSilabas()
		}
		
		reverseSilabas = make(map[string]uint8)
		for id, list := range silabas {
			for _, s := range list {
				reverseSilabas[s] = id
			}
		}
	})
}

func NormalizeText(input string) string {
	var sb strings.Builder
	for _, r := range input {
		// 1. Skip '0'
		if r == '0' {
			continue 
		}
		
		// 2. Keep Space
		if unicode.IsSpace(r) {
			sb.WriteRune(' ')
			continue
		}

		// 3. Keep Hyphen (mapped to ID 4)
		if r == '-' {
			sb.WriteRune('-')
			continue
		}
		
		// 4. Handle Accents & Case
		switch r {
		case 'á': r = 'a'
		case 'é': r = 'e'
		case 'í': r = 'i'
		case 'ó': r = 'o'
		case 'ú': r = 'u'
		case 'ü': r = 'u'
		case 'ñ': r = 'ñ'
		}
		
		if unicode.IsUpper(r) {
			r = unicode.ToLower(r)
		}

		// 5. Keep Alphanumeric, Remove everything else (., _, etc)
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			sb.WriteRune(r)
		}
	}
	return sb.String()
}

func TruncateNumberSequences(input string) string {
	var sb strings.Builder
	digitCount := 0
	for _, r := range input {
		if unicode.IsDigit(r) {
			if digitCount < 2 {
				sb.WriteRune(r)
				digitCount++
			}
		} else {
			digitCount = 0
			sb.WriteRune(r)
		}
	}
	return sb.String()
}

// EncodePhrase converts a text phrase into a structured EncodedPhrase.
func EncodePhrase(phrase string) (*EncodedPhrase, error) {
	EnsureInitialized()
	
	normalized := NormalizeText(phrase)
	truncated := TruncateNumberSequences(normalized)
	
	words := strings.Fields(truncated)
	
	var content []uint8
	var wordsSize []uint8
	maxWordSize := 0
	
	for _, word := range words {
		// Rule: If word length >= 6 and ends with 's', remove 's'
		if len(word) >= 6 && strings.HasSuffix(word, "s") {
			word = word[:len(word)-1]
		}

		ids := findBestSplit(word)
		if len(ids) == 0 {
			continue
		}
		
		// Truncate to max 16 syllables per word (limit of 4-bit encoding)
		if len(ids) > 16 {
			ids = ids[:16]
		}
		
		count := len(ids)
		content = append(content, ids...)
		wordsSize = append(wordsSize, uint8(count))
		
		if count > maxWordSize {
			maxWordSize = count
		}
	}
	
	is4Bit := maxWordSize > 4
	packed, err := packSizes(wordsSize, is4Bit)
	if err != nil {
		return nil, err
	}
	
	return &EncodedPhrase{
		Content:       content,
		WordsSize:     wordsSize,
		WordsSizeBits: packed,
		Is4BitSize:    is4Bit,
	}, nil
}

// packSizes packs the word lengths into bytes based on the mode.
// 2-bit mode: 4 sizes per byte. Encoding: size - 1. (Range 1-4 -> 0-3)
// 4-bit mode: 2 sizes per byte. Encoding: size - 1. (Range 1-16 -> 0-15)
func packSizes(sizes []uint8, is4Bit bool) ([]uint8, error) {
	var packed []uint8
	
	if is4Bit {
		// 4-bit mode: 2 items per byte
		// Byte = (Size1 << 4) | Size2
		for i := 0; i < len(sizes); i += 2 {
			s1 := sizes[i]
			s2 := uint8(0)
			if i+1 < len(sizes) {
				s2 = sizes[i+1]
			}
			
			if s1 < 1 || s1 > 16 { return nil, errors.New("word size out of range for 4-bit mode") }
			if s2 != 0 && (s2 < 1 || s2 > 16) { return nil, errors.New("word size out of range for 4-bit mode") }
			
			v1 := s1 - 1
			v2 := uint8(0)
			if s2 != 0 {
				v2 = s2 - 1
			}
			
			b := (v1 << 4) | v2
			packed = append(packed, b)
		}
	} else {
		// 2-bit mode: 4 items per byte. (Size 1-4 -> 0-3)
		// Byte = (s1<<6) | (s2<<4) | (s3<<2) | s4
		for i := 0; i < len(sizes); i += 4 {
			var chunk [4]uint8
			for j := 0; j < 4; j++ {
				if i+j < len(sizes) {
					s := sizes[i+j]
					if s < 1 || s > 4 { return nil, errors.New("word size out of range for 2-bit mode") }
					chunk[j] = s - 1
				} else {
					chunk[j] = 0 // padding
				}
			}
			b := (chunk[0] << 6) | (chunk[1] << 4) | (chunk[2] << 2) | chunk[3]
			packed = append(packed, b)
		}
	}
	return packed, nil
}

type splitResult struct {
	ids   []uint8
	score int 
}

var memoSplit map[string]*splitResult

func findBestSplit(word string) []uint8 {
	memoSplit = make(map[string]*splitResult)
	res := solveSplit(word)
	if res == nil {
		return []uint8{}
	}
	return res.ids
}

func solveSplit(s string) *splitResult {
	if s == "" {
		return &splitResult{ids: []uint8{}, score: 0}
	}
	if res, ok := memoSplit[s]; ok {
		return res
	}

	var best *splitResult

	for length := 3; length >= 1; length-- {
		if len(s) < length {
			continue
		}

		sub := s[:length]
		id, exists := reverseSilabas[sub]
		if !exists {
			continue
		}

		restResult := solveSplit(s[length:])
		if restResult == nil {
			continue
		}

		currentScore := restResult.score + 1
		if length == 1 {
			currentScore += 100
		}

		newIds := append([]uint8{id}, restResult.ids...)
		candidate := &splitResult{
			ids:   newIds,
			score: currentScore,
		}

		if best == nil || candidate.score < best.score {
			best = candidate
		}
	}

	memoSplit[s] = best
	return best
}

type ProductEncoded struct {
	Head uint8
	ProductID []uint8
	BrandID uint8
	CategoriesIDs []uint8
	WordCount []uint8
	TextEncoded []uint8
}

func (e ProductEncoded) Encode() []uint8 {
	// Reserve exact-ish capacity to avoid repeated reallocations while appending fields.
	totalSize := 1 + len(e.ProductID) + len(e.CategoriesIDs) + len(e.WordCount) + len(e.TextEncoded)
	if (e.Head & (1 << 2)) != 0 { // bit 3 in spec (1-based): has_brand
		totalSize++
	}

	encodedRecord := make([]uint8, 0, totalSize)

	// Header byte always leads the record.
	encodedRecord = append(encodedRecord, e.Head)
	encodedRecord = append(encodedRecord, e.ProductID...)

	// Brand byte is emitted only when the has_brand flag is set in the header.
	if (e.Head & (1 << 2)) != 0 {
		encodedRecord = append(encodedRecord, e.BrandID)
	}

	// Variable sections are appended in fixed order for deterministic decoding.
	encodedRecord = append(encodedRecord, e.CategoriesIDs...)
	encodedRecord = append(encodedRecord, e.WordCount...)
	encodedRecord = append(encodedRecord, e.TextEncoded...)

	return encodedRecord
}

func EncodeProduct(productID int32, brandID int16, categoriesIDs []int16, text string) ProductEncoded {
	// Normalize IDs with lossy conversion to fixed byte widths.
	normalizedProductID := uint16(uint32(productID))
	normalizedBrandID := uint8(uint16(brandID))

	// Encode the searchable text into phoneme IDs and packed per-word sizes.
	encodedPhrase, phraseError := EncodePhrase(text)
	if phraseError != nil {
		return ProductEncoded{
			ProductID: []uint8{uint8(normalizedProductID), uint8(normalizedProductID >> 8)},
		}
	}

	// Keep 1..4 categories to match the 2-bit category count in the header.
	normalizedCategoryIDs := make([]uint8, 0, 4)
	for _, currentCategoryID := range categoriesIDs {
		if len(normalizedCategoryIDs) == 4 {
			break
		}
		normalizedCategoryIDs = append(normalizedCategoryIDs, uint8(uint16(currentCategoryID)))
	}
	if len(normalizedCategoryIDs) == 0 {
		normalizedCategoryIDs = append(normalizedCategoryIDs, 0)
	}

	// Header allows 1..4 mask bytes, so limit words and repack mask accordingly.
	usesFourBitWordCounts := encodedPhrase.Is4BitSize
	maxSupportedWordCount := 16
	if usesFourBitWordCounts {
		maxSupportedWordCount = 8
	}

	limitedWordSizes := encodedPhrase.WordsSize
	if len(limitedWordSizes) > maxSupportedWordCount {
		limitedWordSizes = limitedWordSizes[:maxSupportedWordCount]
	}

	limitedWordMask, maskPackingError := packSizes(limitedWordSizes, usesFourBitWordCounts)
	if maskPackingError != nil {
		limitedWordMask = []uint8{0}
	}
	if len(limitedWordMask) == 0 {
		limitedWordMask = []uint8{0}
	}
	if len(limitedWordMask) > 4 {
		limitedWordMask = limitedWordMask[:4]
	}

	// Keep stream and word counts aligned after word truncation.
	totalPhonemeCount := 0
	for _, currentWordSize := range limitedWordSizes {
		totalPhonemeCount += int(currentWordSize)
	}
	limitedTextEncoded := encodedPhrase.Content
	if len(limitedTextEncoded) > totalPhonemeCount {
		limitedTextEncoded = limitedTextEncoded[:totalPhonemeCount]
	}

	hasBrand := normalizedBrandID != 0
	encodedCategoryCount := uint8(len(normalizedCategoryIDs) - 1) // 0..3 => real count 1..4
	wordMaskByteCount := uint8(len(limitedWordMask) - 1)           // 0..3 => real bytes 1..4

	// Build bit flags:
	// bits 1-2: category count, bit 3: brand, bit 4: 4-bit mode, bits 5-6: mask bytes.
	headerFlags := encodedCategoryCount & 0x03
	if hasBrand {
		headerFlags |= 1 << 2
	}
	if usesFourBitWordCounts {
		headerFlags |= 1 << 3
	}
	headerFlags |= (wordMaskByteCount & 0x03) << 4

	return ProductEncoded{
		Head:          headerFlags,
		ProductID:     []uint8{uint8(normalizedProductID), uint8(normalizedProductID >> 8)},
		BrandID:       normalizedBrandID,
		CategoriesIDs: normalizedCategoryIDs,
		WordCount:     limitedWordMask,
		TextEncoded:   limitedTextEncoded,
	}
}

