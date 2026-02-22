package index_builder

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"hash/crc32"
	"os"
	"sort"
	"strconv"
	"strings"
	"unicode"
)

const (
	BinaryMagic   = "GIXIDX01"
	BinaryVersion = uint8(1)

	HeaderFlagDictionaryDelta = uint8(1 << 0)
	HeaderFlagShapeDelta      = uint8(1 << 1)
	HeaderFlagNumericCompact  = uint8(1 << 2)
)

const (
	textSectionDictionary uint8 = 1
	textSectionShapes     uint8 = 2
	textSectionContent    uint8 = 3
	textSectionAliases    uint8 = 4
	textSectionProductIDs uint8 = 5
)

const productIDUint32Sentinel uint16 = 0

type RecordInput struct {
	ID            int32
	CategoriesIDs []int32
	BrandID       int32
	Text          string
}

type BuildOptions struct {
	MaxWordsPerRecord   int32
	MaxSyllablesPerWord int32
	MaxDictionarySlots  int32
	ConnectorWords      []string
}

type BuildStats struct {
	InputRecords            int32
	EncodedRecords          int32
	FixedSyllablesCount     int32
	ComputedSyllablesCount  int32
	ExtractedSyllablesTotal int32
	MostUsedWordsTop20      string
	UniqueShapes            int32
	ShapeCoverageTop255Pct  float32
	DictionaryCount         int32
	DictionaryBytes         int32
	AliasBytes              int32
	ShapesBytes             int32
	ContentBytes            int32
	ProductIDsBytes         int32
	TotalBytes              int32
	ShapeDelta8Count        int32
	ShapeDelta16Count       int32
	ShapeDelta24Count       int32
}

func DefaultOptions() BuildOptions {
	return BuildOptions{
		MaxWordsPerRecord:   8,
		MaxSyllablesPerWord: 7,
		MaxDictionarySlots:  255,
		ConnectorWords:      []string{"de", "la", "con", "del", "para", "en", "al", "con", "sin"},
	}
}

type normalizedRecord struct {
	id     int32
	tokens []string
}

type encodedRecord struct {
	id      int32
	shape   []uint8
	content []uint8
	shapeV  uint32
}

func BuildIndex(records []RecordInput, options BuildOptions) (*ProductosIndexBuild, error) {
	if len(records) == 0 {
		return nil, fmt.Errorf("no records provided")
	}
	if options.MaxWordsPerRecord <= 0 {
		options.MaxWordsPerRecord = 8
	}
	if options.MaxSyllablesPerWord <= 0 {
		options.MaxSyllablesPerWord = 7
	}
	if options.MaxDictionarySlots <= 0 || options.MaxDictionarySlots > 255 {
		options.MaxDictionarySlots = 255
	}

	connectorSet := make(map[string]struct{}, len(options.ConnectorWords))
	for _, connector := range options.ConnectorWords {
		normalizedConnector := normalizeToken(connector)
		if normalizedConnector == "" {
			continue
		}
		connectorSet[normalizedConnector] = struct{}{}
	}

	normalized := make([]normalizedRecord, 0, len(records))
	for _, inputRecord := range records {
		tokens := normalizeAndFilterTokens(inputRecord.Text, connectorSet)
		if len(tokens) == 0 {
			continue
		}
		if int32(len(tokens)) > options.MaxWordsPerRecord {
			tokens = tokens[:options.MaxWordsPerRecord]
		}
		normalized = append(normalized, normalizedRecord{id: inputRecord.ID, tokens: tokens})
	}
	if len(normalized) == 0 {
		return nil, fmt.Errorf("all records empty after normalization")
	}

	aliasToCanonical, fixedCanonical := defaultFixedAliases()
	syllableFrequency := extractSyllableFrequency(normalized, aliasToCanonical)
	extractedSyllablesTotal := int32(0)
	for _, count := range syllableFrequency {
		extractedSyllablesTotal += int32(count)
	}
	dictionaryTokens := buildDictionaryTokens(fixedCanonical, syllableFrequency, int(options.MaxDictionarySlots))
	if len(dictionaryTokens) == 0 {
		return nil, fmt.Errorf("empty dictionary")
	}

	dictionaryIDByToken := make(map[string]uint8, len(dictionaryTokens))
	for dictionaryIndex, dictionaryToken := range dictionaryTokens {
		dictionaryIDByToken[dictionaryToken] = uint8(dictionaryIndex + 1)
	}
	for aliasToken, canonicalToken := range aliasToCanonical {
		if dictionaryID, exists := dictionaryIDByToken[canonicalToken]; exists {
			dictionaryIDByToken[aliasToken] = dictionaryID
		}
	}
	// Map compact numeric buckets to valid dictionary IDs so numbers always encode in one byte.
	for compactBucket := 1; compactBucket <= 244; compactBucket++ {
		compactNumeric := fmt.Sprintf("%03d", compactBucket)
		remappedDictionaryID := uint8(((compactBucket - 1) % len(dictionaryTokens)) + 1)
		dictionaryIDByToken[compactNumeric] = remappedDictionaryID
	}

	encoded := make([]encodedRecord, 0, len(normalized))
	for _, normalizedRecord := range normalized {
		recordShape := make([]uint8, 0, len(normalizedRecord.tokens))
		recordContent := make([]uint8, 0, 24)
		for _, token := range normalizedRecord.tokens {
			wordIDs := encodeWordToken(token, dictionaryIDByToken, aliasToCanonical)
			if len(wordIDs) == 0 {
				continue
			}
			if int32(len(wordIDs)) > options.MaxSyllablesPerWord {
				wordIDs = wordIDs[:options.MaxSyllablesPerWord]
			}
			recordShape = append(recordShape, uint8(len(wordIDs)))
			recordContent = append(recordContent, wordIDs...)
		}
		if len(recordShape) == 0 || len(recordContent) == 0 {
			continue
		}
		shapeValue, shapeErr := encodeShapeValue(recordShape)
		if shapeErr != nil {
			return nil, shapeErr
		}
		encoded = append(encoded, encodedRecord{id: normalizedRecord.id, shape: recordShape, content: recordContent, shapeV: shapeValue})
	}
	if len(encoded) == 0 {
		return nil, fmt.Errorf("no encodable records")
	}

	sort.SliceStable(encoded, func(leftIndex, rightIndex int) bool {
		if encoded[leftIndex].shapeV != encoded[rightIndex].shapeV {
			return encoded[leftIndex].shapeV < encoded[rightIndex].shapeV
		}
		leftShape := encoded[leftIndex].shape
		rightShape := encoded[rightIndex].shape
		minimumLen := len(leftShape)
		if len(rightShape) < minimumLen {
			minimumLen = len(rightShape)
		}
		for shapeIndex := 0; shapeIndex < minimumLen; shapeIndex++ {
			if leftShape[shapeIndex] != rightShape[shapeIndex] {
				return leftShape[shapeIndex] < rightShape[shapeIndex]
			}
		}
		if len(leftShape) != len(rightShape) {
			return len(leftShape) < len(rightShape)
		}
		return encoded[leftIndex].id < encoded[rightIndex].id
	})

	sortedIDs := make([]int32, 0, len(encoded))
	content := make([]byte, 0, 1024)
	sortedShapeValues := make([]uint32, 0, len(encoded))
	for _, encodedRecord := range encoded {
		sortedIDs = append(sortedIDs, encodedRecord.id)
		sortedShapeValues = append(sortedShapeValues, encodedRecord.shapeV)
		content = append(content, encodedRecord.content...)
	}
	shapeStream, d8, d16, d24, shapeErr := encodeShapeDeltaStream(sortedShapeValues)
	if shapeErr != nil {
		return nil, shapeErr
	}
	productIDsSection, productIDsErr := encodeProductIDsSection(sortedIDs)
	if productIDsErr != nil {
		return nil, productIDsErr
	}

	dictionarySection, dictionaryErr := encodeDictionarySection(dictionaryTokens)
	if dictionaryErr != nil {
		return nil, dictionaryErr
	}
	aliasSection, aliasCount, aliasSectionErr := encodeAliasSection(aliasToCanonical, dictionaryIDByToken)
	if aliasSectionErr != nil {
		return nil, aliasSectionErr
	}
	shapeFrequencyByValue := make(map[uint32]int32, len(encoded))
	for _, shapeValue := range sortedShapeValues {
		shapeFrequencyByValue[shapeValue]++
	}
	type shapeCounter struct {
		shapeValue uint32
		count      int32
	}
	shapeRows := make([]shapeCounter, 0, len(shapeFrequencyByValue))
	for shapeValue, count := range shapeFrequencyByValue {
		shapeRows = append(shapeRows, shapeCounter{shapeValue: shapeValue, count: count})
	}
	sort.SliceStable(shapeRows, func(leftIndex, rightIndex int) bool {
		if shapeRows[leftIndex].count != shapeRows[rightIndex].count {
			return shapeRows[leftIndex].count > shapeRows[rightIndex].count
		}
		return shapeRows[leftIndex].shapeValue < shapeRows[rightIndex].shapeValue
	})
	recordsInTop255 := int32(0)
	maxRows := 255
	if len(shapeRows) < maxRows {
		maxRows = len(shapeRows)
	}
	for rowIndex := 0; rowIndex < maxRows; rowIndex++ {
		recordsInTop255 += shapeRows[rowIndex].count
	}
	shapeCoverageTop255Pct := float32(0)
	if len(encoded) > 0 {
		shapeCoverageTop255Pct = 100.0 * float32(recordsInTop255) / float32(len(encoded))
	}

	result := &ProductosIndexBuild{
		SortedIDs:         sortedIDs,
		ProductIDsSection: productIDsSection,
		Shapes:            shapeStream,
		Content:           content,
		// Caller can overwrite this timestamp when persisting the final .idx payload.
		BuildSunixTime:    0,
		HeaderFlags:       HeaderFlagShapeDelta | HeaderFlagNumericCompact,
		DictionaryTokens:  dictionaryTokens,
		DictionarySection: dictionarySection,
		AliasSection:      aliasSection,
		AliasCount:        int32(aliasCount),
		Stats: BuildStats{
			InputRecords:            int32(len(records)),
			EncodedRecords:          int32(len(encoded)),
			FixedSyllablesCount:     int32(len(fixedCanonical)),
			ComputedSyllablesCount:  int32(len(dictionaryTokens) - len(fixedCanonical)),
			ExtractedSyllablesTotal: extractedSyllablesTotal,
			MostUsedWordsTop20:      formatTopFrequencies(syllableFrequency, 20),
			UniqueShapes:            int32(len(shapeFrequencyByValue)),
			ShapeCoverageTop255Pct:  shapeCoverageTop255Pct,
			DictionaryCount:         int32(len(dictionaryTokens)),
			DictionaryBytes:         int32(len(dictionarySection)),
			AliasBytes:              int32(len(aliasSection)),
			ShapesBytes:             int32(len(shapeStream)),
			ContentBytes:            int32(len(content)),
			ProductIDsBytes:         int32(len(productIDsSection)),
			ShapeDelta8Count:        int32(d8),
			ShapeDelta16Count:       int32(d16),
			ShapeDelta24Count:       int32(d24),
		},
	}
	binaryPayload, marshalErr := result.MarshalBinary()
	if marshalErr != nil {
		return nil, marshalErr
	}
	result.Stats.TotalBytes = int32(len(binaryPayload))
	return result, nil
}

func (buildResult *ProductosIndexBuild) MarshalBinary() ([]byte, error) {
	if buildResult == nil {
		return nil, fmt.Errorf("nil result")
	}
	if len(buildResult.DictionaryTokens) == 0 || len(buildResult.DictionaryTokens) > 255 {
		return nil, fmt.Errorf("invalid dictionary token count=%d", len(buildResult.DictionaryTokens))
	}
	if len(buildResult.SortedIDs) > int(^uint32(0)>>1) {
		return nil, fmt.Errorf("too many records=%d", len(buildResult.SortedIDs))
	}

	type textSectionDescriptor struct {
		sectionID uint8
		data      []byte
		itemCount uint32
	}
	sections := []textSectionDescriptor{
		// Explicit section table keeps decode cursor logic trivial and stable across versions.
		{sectionID: textSectionDictionary, data: buildResult.DictionarySection, itemCount: uint32(len(buildResult.DictionaryTokens))},
		{sectionID: textSectionShapes, data: buildResult.Shapes, itemCount: uint32(len(buildResult.SortedIDs))},
		{sectionID: textSectionContent, data: buildResult.Content, itemCount: uint32(len(buildResult.Content))},
		{sectionID: textSectionAliases, data: buildResult.AliasSection, itemCount: uint32(buildResult.AliasCount)},
		{sectionID: textSectionProductIDs, data: buildResult.ProductIDsSection, itemCount: uint32(len(buildResult.SortedIDs))},
	}

	const sectionEntrySize = 1 + 4 + 4 + 4 + 4 // section_id + offset + length + item_count + checksum_crc32
	// Header layout: magic + version + flags + record_count + build_sunix_time + dictionary_count + section_count + header_size.
	baseHeaderSize := len(BinaryMagic) + 1 + 1 + 4 + 4 + 1 + 1 + 2
	headerSize := baseHeaderSize + len(sections)*sectionEntrySize
	payload := make([]byte, 0, headerSize+len(buildResult.DictionarySection)+len(buildResult.Shapes)+len(buildResult.Content)+len(buildResult.ProductIDsSection))
	payload = append(payload, []byte(BinaryMagic)...)
	payload = append(payload, BinaryVersion)
	payload = append(payload, buildResult.HeaderFlags)

	var recordCountBytes [4]byte
	binary.LittleEndian.PutUint32(recordCountBytes[:], uint32(len(buildResult.SortedIDs)))
	payload = append(payload, recordCountBytes[:]...)
	var buildSunixTimeBytes [4]byte
	binary.LittleEndian.PutUint32(buildSunixTimeBytes[:], uint32(buildResult.BuildSunixTime))
	payload = append(payload, buildSunixTimeBytes[:]...)

	payload = append(payload, uint8(len(buildResult.DictionaryTokens)))
	payload = append(payload, uint8(len(sections)))
	if headerSize > 65535 {
		return nil, fmt.Errorf("text header too large bytes=%d", headerSize)
	}
	var headerSizeBytes [2]byte
	binary.LittleEndian.PutUint16(headerSizeBytes[:], uint16(headerSize))
	payload = append(payload, headerSizeBytes[:]...)

	currentOffset := uint32(headerSize)
	for _, section := range sections {
		if len(section.data) > int(^uint32(0)) {
			return nil, fmt.Errorf("text section overflows uint32 section=%d", section.sectionID)
		}
		payload = append(payload, section.sectionID)
		var offsetBytes [4]byte
		binary.LittleEndian.PutUint32(offsetBytes[:], currentOffset)
		payload = append(payload, offsetBytes[:]...)
		var lengthBytes [4]byte
		binary.LittleEndian.PutUint32(lengthBytes[:], uint32(len(section.data)))
		payload = append(payload, lengthBytes[:]...)
		var countBytes [4]byte
		binary.LittleEndian.PutUint32(countBytes[:], section.itemCount)
		payload = append(payload, countBytes[:]...)
		var checksumBytes [4]byte
		binary.LittleEndian.PutUint32(checksumBytes[:], crc32.ChecksumIEEE(section.data))
		payload = append(payload, checksumBytes[:]...)
		currentOffset += uint32(len(section.data))
	}

	for _, section := range sections {
		payload = append(payload, section.data...)
	}
	return payload, nil
}

func (buildResult *ProductosIndexBuild) WriteBinaryFile(outputPath string) error {
	payload, marshalErr := buildResult.MarshalBinary()
	if marshalErr != nil {
		return marshalErr
	}
	if writeErr := os.WriteFile(outputPath, payload, 0o644); writeErr != nil {
		return fmt.Errorf("write index file: %w", writeErr)
	}
	return nil
}

// ReadBuildSunixTimeFromHeader reads the build_sunix_time int32 directly from the
// text header prefix without decoding full sections.
func ReadBuildSunixTimeFromHeader(indexBytes []byte) (int32, error) {
	const headerPrefixSize = len(BinaryMagic) + 1 + 1 + 4 + 4 // magic + version + flags + record_count + build_sunix_time
	if len(indexBytes) < headerPrefixSize {
		return 0, fmt.Errorf("index too small bytes=%d", len(indexBytes))
	}
	if string(indexBytes[:len(BinaryMagic)]) != BinaryMagic {
		return 0, fmt.Errorf("invalid text magic")
	}
	if indexBytes[len(BinaryMagic)] != BinaryVersion {
		return 0, fmt.Errorf("unsupported text version=%d (expected=%d)", indexBytes[len(BinaryMagic)], BinaryVersion)
	}
	buildSunixOffset := len(BinaryMagic) + 1 + 1 + 4
	buildSunixTime := int32(binary.LittleEndian.Uint32(indexBytes[buildSunixOffset : buildSunixOffset+4]))
	return buildSunixTime, nil
}

func LoadRecordsFromTextFile(inputPath string) ([]RecordInput, error) {
	inputFile, openErr := os.Open(inputPath)
	if openErr != nil {
		return nil, fmt.Errorf("open input file: %w", openErr)
	}
	defer inputFile.Close()

	records := make([]RecordInput, 0, 12000)
	scanner := bufio.NewScanner(inputFile)
	scanner.Buffer(make([]byte, 0, 1024), 1024*1024)
	var recordID int32 = 1
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		records = append(records, RecordInput{ID: recordID, Text: line})
		recordID++
	}
	if scanErr := scanner.Err(); scanErr != nil {
		return nil, fmt.Errorf("scan input file: %w", scanErr)
	}
	return records, nil
}

func normalizeAndFilterTokens(rawText string, connectorSet map[string]struct{}) []string {
	normalizedText := normalizeText(rawText)
	if normalizedText == "" {
		return nil
	}
	rawTokens := strings.Fields(normalizedText)
	filtered := make([]string, 0, len(rawTokens))
	for tokenIndex := 0; tokenIndex < len(rawTokens); tokenIndex++ {
		token := rawTokens[tokenIndex]
		if token == "" {
			continue
		}
		if _, isConnector := connectorSet[token]; isConnector {
			continue
		}

		// Compact numeric prefix forms like "345g" into "n011g".
		if numberPrefix, suffix, splitOK := splitNumericPrefixToken(token); splitOK {
			compactNumeric := compactNumericToken(numberPrefix)
			if suffix != "" {
				token = compactNumeric + suffix
			} else {
				token = compactNumeric
			}
		} else if isNumericToken(token) {
			token = compactNumericToken(token)
			if tokenIndex+1 < len(rawTokens) {
				nextToken := rawTokens[tokenIndex+1]
				if len(nextToken) <= 3 && !isNumericToken(nextToken) {
					token = token + nextToken
					tokenIndex++
				}
			}
		}

		if utf8Len(token) == 1 {
			continue
		}
		filtered = append(filtered, token)
	}
	return filtered
}

func normalizeText(text string) string {
	var output strings.Builder
	output.Grow(len(text))
	for _, r := range strings.ToLower(text) {
		switch r {
		case 'á', 'à', 'â', 'ä', 'ã', 'å', 'ª':
			r = 'a'
		case 'é', 'è', 'ê', 'ë':
			r = 'e'
		case 'í', 'ì', 'î', 'ï':
			r = 'i'
		case 'ó', 'ò', 'ô', 'ö', 'õ', 'º':
			r = 'o'
		case 'ú', 'ü':
			r = 'u'
		case 'ñ':
			r = 'n'
		}
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || unicode.IsSpace(r) {
			output.WriteRune(r)
		}
	}
	return strings.Join(strings.Fields(output.String()), " ")
}

func normalizeToken(token string) string {
	return strings.ReplaceAll(normalizeText(token), " ", "")
}

func splitNumericPrefixToken(token string) (string, string, bool) {
	if token == "" {
		return "", "", false
	}
	splitIndex := 0
	for splitIndex < len(token) {
		b := token[splitIndex]
		if b < '0' || b > '9' {
			break
		}
		splitIndex++
	}
	if splitIndex == 0 || splitIndex >= len(token) {
		return "", "", false
	}
	for suffixIndex := splitIndex; suffixIndex < len(token); suffixIndex++ {
		b := token[suffixIndex]
		if b < 'a' || b > 'z' {
			return "", "", false
		}
	}
	return token[:splitIndex], token[splitIndex:], true
}

func compactNumericToken(rawNumber string) string {
	parsed, err := strconv.Atoi(rawNumber)
	if err != nil || parsed <= 0 {
		return "001"
	}
	bucket := ((parsed - 1) % 244) + 1
	return fmt.Sprintf("%03d", bucket)
}

func isNumericToken(token string) bool {
	if token == "" {
		return false
	}
	for _, r := range token {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

func splitCompactNumericToken(token string) (string, bool) {
	// Expected format: 3 digits + optional [a-z] suffix.
	if len(token) < 3 {
		return "", false
	}
	for digitIndex := 0; digitIndex < 3; digitIndex++ {
		currentByte := token[digitIndex]
		if currentByte < '0' || currentByte > '9' {
			return "", false
		}
	}
	for suffixIndex := 3; suffixIndex < len(token); suffixIndex++ {
		currentByte := token[suffixIndex]
		if currentByte < 'a' || currentByte > 'z' {
			return "", false
		}
	}
	return token[3:], true
}

func extractSyllableFrequency(records []normalizedRecord, aliasToCanonical map[string]string) map[string]int {
	frequency := make(map[string]int, 1024)
	for _, record := range records {
		for _, token := range record.tokens {
			if canonical, exists := aliasToCanonical[token]; exists {
				frequency[canonical]++
				continue
			}
			if compactSuffix, isCompactNumeric := splitCompactNumericToken(token); isCompactNumeric {
				// Never include numeric fragments (e.g. "75") in dictionary frequencies.
				if compactSuffix == "" {
					continue
				}
				if canonicalSuffix, hasCanonicalSuffix := aliasToCanonical[compactSuffix]; hasCanonicalSuffix {
					frequency[canonicalSuffix]++
					continue
				}
				for _, suffixSyllable := range splitTokenIntoSyllables(compactSuffix) {
					frequency[suffixSyllable]++
				}
				continue
			}
			for _, syllable := range splitTokenIntoSyllables(token) {
				frequency[syllable]++
			}
		}
	}
	return frequency
}

func buildDictionaryTokens(fixedCanonical []string, frequencyBySyllable map[string]int, maxSlots int) []string {
	tokens := make([]string, 0, maxSlots)
	seen := make(map[string]struct{}, maxSlots*2)
	appendToken := func(token string) {
		if len(tokens) >= maxSlots || token == "" {
			return
		}
		// Keep single-digit numeric canonicals, but block 2+ digit pure numbers from dictionary slots.
		if isNumericToken(token) && len(token) > 1 {
			return
		}
		if _, exists := seen[token]; exists {
			return
		}
		seen[token] = struct{}{}
		tokens = append(tokens, token)
	}
	for _, fixed := range fixedCanonical {
		appendToken(fixed)
	}

	type freqRow struct {
		token string
		count int
	}
	rows := make([]freqRow, 0, len(frequencyBySyllable))
	for token, count := range frequencyBySyllable {
		rows = append(rows, freqRow{token: token, count: count})
	}
	sort.SliceStable(rows, func(i, j int) bool {
		if rows[i].count != rows[j].count {
			return rows[i].count > rows[j].count
		}
		return rows[i].token < rows[j].token
	})
	for _, row := range rows {
		appendToken(row.token)
		if len(tokens) >= maxSlots {
			break
		}
	}
	return tokens
}

func formatTopFrequencies(frequencyByToken map[string]int, limit int) string {
	type row struct {
		token string
		count int
	}
	rows := make([]row, 0, len(frequencyByToken))
	for token, count := range frequencyByToken {
		rows = append(rows, row{token: token, count: count})
	}
	sort.SliceStable(rows, func(leftIndex, rightIndex int) bool {
		if rows[leftIndex].count != rows[rightIndex].count {
			return rows[leftIndex].count > rows[rightIndex].count
		}
		return rows[leftIndex].token < rows[rightIndex].token
	})
	if limit > len(rows) {
		limit = len(rows)
	}
	parts := make([]string, 0, limit)
	for rowIndex := 0; rowIndex < limit; rowIndex++ {
		parts = append(parts, fmt.Sprintf("%s:%d", rows[rowIndex].token, rows[rowIndex].count))
	}
	return strings.Join(parts, ", ")
}

func encodeWordToken(token string, dictionaryIDByToken map[string]uint8, aliasToCanonical map[string]string) []uint8 {
	if canonical, exists := aliasToCanonical[token]; exists {
		if dictionaryID, ok := dictionaryIDByToken[canonical]; ok {
			return []uint8{dictionaryID}
		}
	}

	// Compact numeric prefix token: XYZ + optional suffix.
	if suffix, isCompactNumeric := splitCompactNumericToken(token); isCompactNumeric {
		numericPart := token[:3]
		if dictionaryID, ok := dictionaryIDByToken[numericPart]; ok {
			encoded := []uint8{dictionaryID}
			if suffix != "" {
				if suffixID, suffixOK := dictionaryIDByToken[suffix]; suffixOK {
					encoded = append(encoded, suffixID)
				} else {
					for _, syllable := range splitTokenIntoSyllables(suffix) {
						if syllableID, exists := dictionaryIDByToken[syllable]; exists {
							encoded = append(encoded, syllableID)
						}
					}
				}
			}
			return encoded
		}
	}

	encoded := make([]uint8, 0, 8)
	for _, syllable := range splitTokenIntoSyllables(token) {
		if syllableID, exists := dictionaryIDByToken[syllable]; exists {
			encoded = append(encoded, syllableID)
			continue
		}
		for _, r := range syllable {
			single := string(r)
			if syllableID, exists := dictionaryIDByToken[single]; exists {
				encoded = append(encoded, syllableID)
			}
		}
	}
	return encoded
}

func encodeShapeValue(wordSizes []uint8) (uint32, error) {
	if len(wordSizes) == 0 || len(wordSizes) > 8 {
		return 0, fmt.Errorf("invalid shape word count=%d", len(wordSizes))
	}
	value := uint32(0)
	for wordIndex := 0; wordIndex < 8; wordIndex++ {
		code := uint8(0)
		if wordIndex < len(wordSizes) {
			if wordSizes[wordIndex] < 1 || wordSizes[wordIndex] > 7 {
				return 0, fmt.Errorf("invalid word syllable count=%d at index=%d", wordSizes[wordIndex], wordIndex)
			}
			code = wordSizes[wordIndex]
		}
		value = (value << 3) | uint32(code)
	}
	return value, nil
}

func encodeDictionarySection(tokens []string) ([]byte, error) {
	section := make([]byte, 0, len(tokens)*4)
	for _, token := range tokens {
		if token == "" || len(token) > 255 {
			return nil, fmt.Errorf("invalid dictionary token len=%d token=%q", len(token), token)
		}
		section = append(section, uint8(len(token)))
		section = append(section, token...)
	}
	return section, nil
}

func encodeAliasSection(aliasToCanonical map[string]string, dictionaryIDByToken map[string]uint8) ([]byte, int, error) {
	type aliasRow struct {
		aliasToken          string
		canonicalDictionary uint8
	}

	aliasRows := make([]aliasRow, 0, len(aliasToCanonical))
	for aliasToken, canonicalToken := range aliasToCanonical {
		if aliasToken == canonicalToken {
			continue
		}
		canonicalDictionaryID, exists := dictionaryIDByToken[canonicalToken]
		if !exists || canonicalDictionaryID == 0 {
			continue
		}
		aliasRows = append(aliasRows, aliasRow{
			aliasToken:          aliasToken,
			canonicalDictionary: canonicalDictionaryID,
		})
	}

	sort.SliceStable(aliasRows, func(leftIndex, rightIndex int) bool {
		return aliasRows[leftIndex].aliasToken < aliasRows[rightIndex].aliasToken
	})

	section := make([]byte, 0, len(aliasRows)*4)
	for _, alias := range aliasRows {
		if len(alias.aliasToken) == 0 || len(alias.aliasToken) > 255 {
			return nil, 0, fmt.Errorf("invalid alias token len=%d token=%q", len(alias.aliasToken), alias.aliasToken)
		}
		section = append(section, uint8(len(alias.aliasToken)))
		section = append(section, alias.aliasToken...)
		section = append(section, alias.canonicalDictionary)
	}
	return section, len(aliasRows), nil
}

func encodeProductIDsSection(sortedProductIDs []int32) ([]byte, error) {
	// Compact IDs as uint16 words; sentinel(0) means the next two words hold one uint32.
	sectionBytes := make([]byte, 0, len(sortedProductIDs)*2)
	appendWord := func(word uint16) {
		sectionBytes = append(sectionBytes, byte(word), byte(word>>8))
	}

	for recordIndex, currentProductID := range sortedProductIDs {
		if currentProductID <= 0 {
			return nil, fmt.Errorf("invalid product id at record=%d id=%d", recordIndex, currentProductID)
		}

		productIDAsUint32 := uint32(currentProductID)
		if productIDAsUint32 <= uint32(^uint16(0)) && productIDAsUint32 != uint32(productIDUint32Sentinel) {
			appendWord(uint16(productIDAsUint32))
			continue
		}

		appendWord(productIDUint32Sentinel)
		appendWord(uint16(productIDAsUint32 & 0xFFFF))
		appendWord(uint16(productIDAsUint32 >> 16))
	}
	return sectionBytes, nil
}

type bitWriter struct {
	bytes  []byte
	offset uint8
}

func newBitWriter() *bitWriter {
	return &bitWriter{bytes: make([]byte, 0, 256)}
}

func (w *bitWriter) writeBit(bit uint8) {
	if w.offset == 0 {
		w.bytes = append(w.bytes, 0)
	}
	if bit != 0 {
		idx := len(w.bytes) - 1
		w.bytes[idx] |= 1 << (7 - w.offset)
	}
	w.offset++
	if w.offset == 8 {
		w.offset = 0
	}
}

func (w *bitWriter) writeBits(value uint32, count int) {
	for bitIndex := count - 1; bitIndex >= 0; bitIndex-- {
		bit := uint8((value >> bitIndex) & 1)
		w.writeBit(bit)
	}
}

func encodeShapeDeltaStream(shapeValues []uint32) ([]byte, int, int, int, error) {
	if len(shapeValues) == 0 {
		return nil, 0, 0, 0, fmt.Errorf("empty shape stream")
	}
	writer := newBitWriter()
	previous := uint32(0)
	count8 := 0
	count16 := 0
	count24 := 0
	for _, current := range shapeValues {
		if current < previous {
			return nil, 0, 0, 0, fmt.Errorf("shape values must be sorted")
		}
		delta := current - previous
		if delta <= 255 {
			writer.writeBit(0)
			writer.writeBits(delta, 8)
			count8++
		} else if delta <= 65534 {
			writer.writeBit(1)
			writer.writeBits(delta, 16)
			count16++
		} else if delta <= 0xFFFFFF {
			writer.writeBit(1)
			writer.writeBits(0xFFFF, 16)
			writer.writeBits(delta, 24)
			count24++
		} else {
			return nil, 0, 0, 0, fmt.Errorf("shape delta over 24 bits=%d", delta)
		}
		previous = current
	}
	return writer.bytes, count8, count16, count24, nil
}

func utf8Len(value string) int {
	count := 0
	for range value {
		count++
	}
	return count
}
