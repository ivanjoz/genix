package word_parser_v2

import (
	"bufio"
	"fmt"
	"log"
	"math"
	"os"
	"sort"
	"strings"
	"unicode"
)

const (
	// DefaultDictionarySlots is the total number of syllable slots expected by the index format.
	DefaultDictionarySlots = 244
	// DefaultTopFrequentCount limits how many high-frequency syllables are prioritized first.
	DefaultTopFrequentCount = 200
	// CoverageGreedyCandidatePool limits the candidate set size used by the coverage-aware selector.
	CoverageGreedyCandidatePool = 400
)

// SyllableFrequency stores one syllable and how many times it appeared in product text.
type SyllableFrequency struct {
	Syllable string
	Count    int
}

// FixedSyllableGeneratorConfig controls deterministic generation of fixed syllable slots.
type FixedSyllableGeneratorConfig struct {
	PreassignedSlots              map[uint16][]string
	Vowels                        []string
	Consonants                    []string
	ConnectorTokens               []string
	VowelCombinationPatterns      [][]string
	EnableTwoSyllableCombinations bool
	MaxSlots                      int
}

// FrequentSyllableGeneratorConfig controls extraction of frequent syllables from product names.
type FrequentSyllableGeneratorConfig struct {
	TopFrequentCount int
	TotalSlots       int
	Strategy         string
}

var forceTwoLetterSyllableSplit bool

// SetForceTwoLetterSyllableSplit toggles splitter candidate extraction to 2-letter only.
func SetForceTwoLetterSyllableSplit(enabled bool) {
	forceTwoLetterSyllableSplit = enabled
}

// GeneratedDictionary contains both the fixed and text-derived sections.
type GeneratedDictionary struct {
	FixedSlots           []string
	FrequentSlots        []string
	CombinedSlots        []string
	FirstFrequentSlot    int
	TopFrequentSyllables []SyllableFrequency
}

// GenerateDictionaryFromProductNames runs the full pipeline: fixed section first, then frequent section.
func GenerateDictionaryFromProductNames(
	productNames []string,
	fixedConfig FixedSyllableGeneratorConfig,
	frequentConfig FrequentSyllableGeneratorConfig,
) (*GeneratedDictionary, error) {
	log.Printf(
		"word_parser_v2: starting full dictionary generation products=%d fixed_max_slots=%d total_slots=%d top=%d",
		len(productNames),
		fixedConfig.MaxSlots,
		frequentConfig.TotalSlots,
		frequentConfig.TopFrequentCount,
	)

	fixedSlots, reservedSyllables, _, fixedGenerationError := GenerateFixedSyllableSlotsDetailed(fixedConfig)
	if fixedGenerationError != nil {
		return nil, fixedGenerationError
	}

	return GenerateFrequentSyllableSlotsWithReserved(productNames, fixedSlots, reservedSyllables, frequentConfig)
}

// LoadProductNamesFromFile reads one product name per line, skipping empty lines.
func LoadProductNamesFromFile(productNamesFilePath string) ([]string, error) {
	sourceFile, openFileError := os.Open(productNamesFilePath)
	if openFileError != nil {
		return nil, fmt.Errorf("open product names file: %w", openFileError)
	}
	defer sourceFile.Close()

	productNames := make([]string, 0, 12000)
	lineScanner := bufio.NewScanner(sourceFile)
	lineScanner.Buffer(make([]byte, 0, 1024), 1024*1024)
	for lineScanner.Scan() {
		currentProductName := strings.TrimSpace(lineScanner.Text())
		if currentProductName == "" {
			continue
		}
		productNames = append(productNames, currentProductName)
	}
	if scanError := lineScanner.Err(); scanError != nil {
		return nil, fmt.Errorf("scan product names file: %w", scanError)
	}

	log.Printf("word_parser_v2: loaded product names file=%s count=%d", productNamesFilePath, len(productNames))
	return productNames, nil
}

// DefaultFixedSyllableGeneratorConfig provides an opinionated baseline that remains editable.
func DefaultFixedSyllableGeneratorConfig() FixedSyllableGeneratorConfig {
	return FixedSyllableGeneratorConfig{
		PreassignedSlots: map[uint16][]string{
			1:   {"1"},
			2:   {"2"},
			3:   {"3", "y"},
			4:   {"4", "-"},
			5:   {"5"},
			6:   {"6", "z"},
			7:   {"7", "x"},
			8:   {"8", "w"},
			9:   {"9"},
			10: {"g", "gr", "grs"},
			11: {"ml","mm", "mg", "mgs", "m3","m2"},
			12: {"k", "kg", "kgs","kilos","kilo","kilogramos"},
			13: {"ud", "un", "uns","unidad","unidades"},
			14: {"c", "cm", "cm2", "cm3", "cl"},
			15: {"l", "lt", "lts","litro","litos"},
			16: {"r", "rr"},
			17: {"q", "qu"},
			18: {"p", "pack","paq","paquete"},
			/* 
			19: {"se", "ce", "xe"},
			20: {"si", "ci", "xi"},
			21: {"so", "xo"},
			22: {"sa", "xa"},
			23: {"su", "xu"},
			24: {"ca", "ka", "qa"},
			25: {"co", "ko", "qo"},
			26: {"cu", "ku"},
			27: {"va", "ba"},
			28: {"ve", "be"},
			29: {"vi", "bi"},
			30: {"vo", "bo"},
			31: {"vu", "bu"},
			*/
		},
		Vowels:                   []string{"a", "e", "i", "o", "u"},
		Consonants:               []string{"b", "c", "d", "f", "g", "h", "j", "k", "l", "m", "n", "ñ", "p", "q", "r", "s", "t", "v"},
		ConnectorTokens:          []string{"de", "la", "con", "para", "en"},
		VowelCombinationPatterns: [][]string{
			/*
				{"b*", "v*"},
				{"q*", "qu*"},
				{"r*", "rr*"},
			*/
			//	{"k-"}, {"d*"}, {"f*"}, {"g*", "gu*"}, {"h*"},
			//	{"j*"}, {"l*"}, {"m*"}, {"n*"}, {"ñ-"}, {"p*"},
			//	{"q*", "qu*"}, {"ch*"}, {"ll*"}, {"-l"}, {"-n"}, {"r*", "rr*"},

		},
		EnableTwoSyllableCombinations: false,
		MaxSlots:                      DefaultDictionarySlots,
	}
}

// DefaultFrequentSyllableGeneratorConfig provides defaults for the second stage.
func DefaultFrequentSyllableGeneratorConfig() FrequentSyllableGeneratorConfig {
	return FrequentSyllableGeneratorConfig{
		TopFrequentCount: DefaultTopFrequentCount,
		TotalSlots:       DefaultDictionarySlots,
	}
}

// GenerateFixedSyllableSlots builds the deterministic fixed section and returns occupied slots count.
func GenerateFixedSyllableSlots(config FixedSyllableGeneratorConfig) ([]string, error) {
	fixedSlots, _, _, generationError := GenerateFixedSyllableSlotsDetailed(config)
	return fixedSlots, generationError
}

// GenerateFixedSyllableSlotsDetailed returns ordered fixed slots plus a complete reserved-syllable list.
func GenerateFixedSyllableSlotsDetailed(config FixedSyllableGeneratorConfig) ([]string, []string, map[string]uint16, error) {
	if config.MaxSlots <= 0 {
		return nil, nil, nil, fmt.Errorf("max slots must be > 0")
	}

	highestPreassignedSlot := config.MaxSlots
	for preassignedSlotID := range config.PreassignedSlots {
		if int(preassignedSlotID) > highestPreassignedSlot {
			highestPreassignedSlot = int(preassignedSlotID)
		}
	}
	// Keep all preassigned slots addressable even if fixed max is lower.
	internalFixedCapacity := highestPreassignedSlot
	orderedFixedSlots := make([]string, internalFixedCapacity)
	usedSyllablesByAlias := make(map[string]struct{}, internalFixedCapacity*2)

	registerAlias := func(token string) (string, bool) {
		normalizedToken := normalizeToken(token)
		if normalizedToken == "" {
			return "", false
		}
		if _, alreadyUsed := usedSyllablesByAlias[normalizedToken]; alreadyUsed {
			return normalizedToken, false
		}
		usedSyllablesByAlias[normalizedToken] = struct{}{}
		return normalizedToken, true
	}

	setSlotByIndex := func(slotIndex int, aliases []string) {
		if slotIndex < 0 || slotIndex >= len(orderedFixedSlots) {
			return
		}
		if orderedFixedSlots[slotIndex] != "" {
			return
		}
		for _, alias := range aliases {
			normalizedAlias, _ := registerAlias(alias)
			if normalizedAlias == "" {
				continue
			}
			if orderedFixedSlots[slotIndex] == "" {
				orderedFixedSlots[slotIndex] = normalizedAlias
			}
		}
	}

	for slotID, aliases := range config.PreassignedSlots {
		setSlotByIndex(int(slotID-1), aliases)
	}

	appendCandidateToNextFreeSlot := func(candidate string) bool {
		normalizedCandidate, inserted := registerAlias(candidate)
		if normalizedCandidate == "" || !inserted {
			return false
		}
		for slotIndex := 0; slotIndex < len(orderedFixedSlots); slotIndex++ {
			if orderedFixedSlots[slotIndex] != "" {
				continue
			}
			orderedFixedSlots[slotIndex] = normalizedCandidate
			return true
		}
		return false
	}

	for _, vowelToken := range config.Vowels {
		appendCandidateToNextFreeSlot(vowelToken)
	}
	for _, consonantToken := range config.Consonants {
		appendCandidateToNextFreeSlot(consonantToken)
	}
	for _, patternGroup := range config.VowelCombinationPatterns {
		for _, pattern := range patternGroup {
			for _, expandedCombination := range expandPatternWithVowels(pattern, config.Vowels) {
				appendCandidateToNextFreeSlot(expandedCombination)
			}
		}
	}

	if config.EnableTwoSyllableCombinations {
		// Two-syllable combinations are derived from already generated fixed syllables only.
		doubleCombinationSeeds := collectShortFixedSeeds(orderedFixedSlots)
		for _, leftSyllable := range doubleCombinationSeeds {
			for _, rightSyllable := range doubleCombinationSeeds {
				if allFixedSlotsPopulated(orderedFixedSlots) {
					break
				}
				appendCandidateToNextFreeSlot(leftSyllable + rightSyllable)
			}
			if allFixedSlotsPopulated(orderedFixedSlots) {
				break
			}
		}
	}

	compactFixedSlots := make([]string, 0, config.MaxSlots)
	originalIndexToCompactSlot := make(map[int]uint16, config.MaxSlots)
	for _, slotSyllable := range orderedFixedSlots {
		if slotSyllable != "" {
			compactFixedSlots = append(compactFixedSlots, slotSyllable)
		}
	}
	compactSlotCounter := uint16(1)
	for originalSlotIndex, slotSyllable := range orderedFixedSlots {
		if slotSyllable == "" {
			continue
		}
		originalIndexToCompactSlot[originalSlotIndex] = compactSlotCounter
		compactSlotCounter++
	}

	aliasToCompactSlot := make(map[string]uint16, len(usedSyllablesByAlias))
	for originalSlotIndex, slotSyllable := range orderedFixedSlots {
		if slotSyllable == "" {
			continue
		}
		compactSlotID := originalIndexToCompactSlot[originalSlotIndex]
		aliasToCompactSlot[slotSyllable] = compactSlotID
	}
	for preassignedSlotID, aliases := range config.PreassignedSlots {
		originalSlotIndex := int(preassignedSlotID - 1)
		compactSlotID, exists := originalIndexToCompactSlot[originalSlotIndex]
		if !exists {
			continue
		}
		for _, alias := range aliases {
			normalizedAlias := normalizeToken(alias)
			if normalizedAlias == "" {
				continue
			}
			aliasToCompactSlot[normalizedAlias] = compactSlotID
		}
	}

	reservedSyllables := make([]string, 0, len(usedSyllablesByAlias))
	for alias := range usedSyllablesByAlias {
		reservedSyllables = append(reservedSyllables, alias)
	}
	sort.Strings(reservedSyllables)

	log.Printf(
		"word_parser_v2: fixed syllable generation completed populated_slots=%d max_slots=%d reserved_aliases=%d",
		len(compactFixedSlots),
		config.MaxSlots,
		len(reservedSyllables),
	)
	return compactFixedSlots, reservedSyllables, aliasToCompactSlot, nil
}

// GenerateFrequentSyllableSlots fills remaining slots with frequent syllables extracted from product names.
func GenerateFrequentSyllableSlots(productNames []string, fixedSlots []string, config FrequentSyllableGeneratorConfig) (*GeneratedDictionary, error) {
	return GenerateFrequentSyllableSlotsWithReserved(productNames, fixedSlots, nil, config)
}

// GenerateFrequentSyllableSlotsWithReserved skips syllables already reserved by preassigned slot aliases.
func GenerateFrequentSyllableSlotsWithReserved(
	productNames []string,
	fixedSlots []string,
	reservedSyllables []string,
	config FrequentSyllableGeneratorConfig,
) (*GeneratedDictionary, error) {
	if config.TotalSlots <= 0 {
		return nil, fmt.Errorf("total slots must be > 0")
	}
	if config.TopFrequentCount <= 0 {
		return nil, fmt.Errorf("top frequent count must be > 0")
	}
	if len(fixedSlots) > config.TotalSlots {
		return nil, fmt.Errorf("fixed slots (%d) exceed total slots (%d)", len(fixedSlots), config.TotalSlots)
	}

	usedSyllables := make(map[string]struct{}, config.TotalSlots*2)
	for _, reservedSyllable := range reservedSyllables {
		normalizedReservedSyllable := normalizeToken(reservedSyllable)
		if normalizedReservedSyllable == "" {
			continue
		}
		usedSyllables[normalizedReservedSyllable] = struct{}{}
	}
	for _, fixedSyllable := range fixedSlots {
		normalizedFixedSyllable := normalizeToken(fixedSyllable)
		if normalizedFixedSyllable == "" {
			continue
		}
		usedSyllables[normalizedFixedSyllable] = struct{}{}
	}

	frequencyBySyllable := make(map[string]int, 512)
	for _, productName := range productNames {
		normalizedName := normalizeText(productName)
		if normalizedName == "" {
			continue
		}
		for _, word := range strings.Fields(normalizedName) {
			for _, extractedSyllable := range splitWordIntoSyllables(word) {
				if extractedSyllable == "" {
					continue
				}
				frequencyBySyllable[extractedSyllable]++
			}
		}
	}

	sortedFrequencies := sortSyllableFrequencies(frequencyBySyllable)

	if config.Strategy == "frequency" || config.Strategy == "ratio_80_20" || config.Strategy == "coverage_greedy" {
		// No re-sort needed, sortSyllableFrequencies uses count desc.
	} else if config.Strategy == "atomic_digraph" {
		// Re-sort to prioritize atomic coverage (length <= 2) and structural digraphs (ch, ll, etc.)
		sort.SliceStable(sortedFrequencies, func(i, j int) bool {
			sylI := sortedFrequencies[i].Syllable
			sylJ := sortedFrequencies[j].Syllable
			lenI := len([]rune(sylI))
			lenJ := len([]rune(sylJ))

			isShortI := lenI <= 2
			isShortJ := lenJ <= 2

			// Prioritize 3-letter structural digraphs common in Spanish
			isDigraphI := strings.HasPrefix(sylI, "ch") || strings.HasPrefix(sylI, "ll") ||
				strings.HasPrefix(sylI, "rr") || strings.HasPrefix(sylI, "gu") || strings.HasPrefix(sylI, "qu")
			isDigraphJ := strings.HasPrefix(sylJ, "ch") || strings.HasPrefix(sylJ, "ll") ||
				strings.HasPrefix(sylJ, "rr") || strings.HasPrefix(sylJ, "gu") || strings.HasPrefix(sylJ, "qu")

			priorityI := isShortI || (lenI == 3 && isDigraphI)
			priorityJ := isShortJ || (lenJ == 3 && isDigraphJ)

			if priorityI != priorityJ {
				return priorityI
			}
			return false
		})
	} else {
		// Default: "atomic_first"
		// Prioritize atomic coverage (length <= 2) over pure frequency.
		// This proved superior to "atomic_digraph" (2263 vs 2345 shapes).
		sort.SliceStable(sortedFrequencies, func(i, j int) bool {
			lenI := len([]rune(sortedFrequencies[i].Syllable))
			lenJ := len([]rune(sortedFrequencies[j].Syllable))
			isShortI := lenI <= 2
			isShortJ := lenJ <= 2
			if isShortI != isShortJ {
				return isShortI
			}
			return false
		})
	}

	prioritizedFrequent := make([]SyllableFrequency, 0, config.TopFrequentCount)
	for _, current := range sortedFrequencies {
		if len(prioritizedFrequent) >= config.TopFrequentCount {
			break
		}
		if _, isFixed := usedSyllables[current.Syllable]; isFixed {
			continue
		}
		prioritizedFrequent = append(prioritizedFrequent, current)
	}

	availableFrequentSlots := config.TotalSlots - len(fixedSlots)
	frequentSlots := make([]string, 0, availableFrequentSlots)
	appendFrequentIfAvailable := func(syllable string) {
		if len(frequentSlots) >= availableFrequentSlots {
			return
		}
		if _, alreadyUsed := usedSyllables[syllable]; alreadyUsed {
			return
		}
		frequentSlots = append(frequentSlots, syllable)
		usedSyllables[syllable] = struct{}{}
	}

	if config.Strategy == "coverage_greedy" {
		coverageCandidates := make([]SyllableWordMapEntry, 0, CoverageGreedyCandidatePool)
		for _, current := range sortedFrequencies {
			if len(coverageCandidates) >= CoverageGreedyCandidatePool {
				break
			}
			if _, isFixed := usedSyllables[current.Syllable]; isFixed {
				continue
			}
			syllableLength := len([]rune(current.Syllable))
			if syllableLength < 2 || syllableLength > 3 {
				continue
			}
			coverageCandidates = append(coverageCandidates, SyllableWordMapEntry{
				Syllable: current.Syllable,
				Length:   syllableLength,
				Count:    current.Count,
			})
		}
		coverageSelection := BuildCoverageGreedyTopNSyllables(productNames, fixedSlots, coverageCandidates, availableFrequentSlots)
		for _, selectedSyllable := range coverageSelection {
			appendFrequentIfAvailable(selectedSyllable)
		}
	} else if config.Strategy == "ratio_80_20" {
		targetTwoLetterCount := int(math.Round(0.8 * float64(availableFrequentSlots)))
		targetThreeLetterCount := availableFrequentSlots - targetTwoLetterCount
		addedTwoLetterCount := 0
		addedThreeLetterCount := 0

		for _, current := range prioritizedFrequent {
			syllableLength := len([]rune(current.Syllable))
			if syllableLength == 2 && addedTwoLetterCount < targetTwoLetterCount {
				beforeCount := len(frequentSlots)
				appendFrequentIfAvailable(current.Syllable)
				if len(frequentSlots) > beforeCount {
					addedTwoLetterCount++
				}
			}
			if len(frequentSlots) >= availableFrequentSlots {
				break
			}
		}

		for _, current := range prioritizedFrequent {
			syllableLength := len([]rune(current.Syllable))
			if syllableLength == 3 && addedThreeLetterCount < targetThreeLetterCount {
				beforeCount := len(frequentSlots)
				appendFrequentIfAvailable(current.Syllable)
				if len(frequentSlots) > beforeCount {
					addedThreeLetterCount++
				}
			}
			if len(frequentSlots) >= availableFrequentSlots {
				break
			}
		}
	} else {
		for _, prioritized := range prioritizedFrequent {
			appendFrequentIfAvailable(prioritized.Syllable)
		}
	}
	for _, current := range sortedFrequencies {
		appendFrequentIfAvailable(current.Syllable)
		if len(frequentSlots) >= availableFrequentSlots {
			break
		}
	}

	combinedSlots := make([]string, 0, config.TotalSlots)
	combinedSlots = append(combinedSlots, fixedSlots...)
	combinedSlots = append(combinedSlots, frequentSlots...)

	log.Printf(
		"word_parser_v2: frequent syllable generation completed products=%d fixed_slots=%d frequent_slots=%d total=%d",
		len(productNames),
		len(fixedSlots),
		len(frequentSlots),
		len(combinedSlots),
	)

	return &GeneratedDictionary{
		FixedSlots:           append([]string(nil), fixedSlots...),
		FrequentSlots:        frequentSlots,
		CombinedSlots:        combinedSlots,
		FirstFrequentSlot:    len(fixedSlots),
		TopFrequentSyllables: prioritizedFrequent,
	}, nil
}

func allFixedSlotsPopulated(fixedSlots []string) bool {
	for _, fixedSlot := range fixedSlots {
		if fixedSlot == "" {
			return false
		}
	}
	return true
}

func collectShortFixedSeeds(fixedSlots []string) []string {
	shortSeeds := make([]string, 0, len(fixedSlots))
	for _, fixedSlot := range fixedSlots {
		if fixedSlot == "" {
			continue
		}
		if len([]rune(fixedSlot)) <= 3 {
			shortSeeds = append(shortSeeds, fixedSlot)
		}
	}
	return shortSeeds
}

func expandPatternWithVowels(pattern string, vowels []string) []string {
	normalizedPattern := normalizePatternInstruction(pattern)
	if normalizedPattern == "" {
		return nil
	}

	if len(vowels) == 0 {
		vowels = []string{"a", "e", "i", "o", "u"}
	}
	expandedCombinations := make([]string, 0, len(vowels))
	appendExpanded := func(candidate string) {
		normalizedCandidate := normalizeToken(candidate)
		if normalizedCandidate == "" {
			return
		}
		expandedCombinations = append(expandedCombinations, normalizedCandidate)
	}

	buildWithVowels := func(template string, skipLastVowel bool) {
		for vowelIndex, vowel := range vowels {
			if skipLastVowel && vowelIndex == len(vowels)-1 {
				continue
			}
			appendExpanded(strings.ReplaceAll(template, "*", vowel))
		}
	}

	if strings.HasPrefix(normalizedPattern, "-") && len(normalizedPattern) > 1 {
		buildWithVowels("*"+normalizedPattern[1:], false)
		return expandedCombinations
	}
	if strings.HasSuffix(normalizedPattern, "-") && len(normalizedPattern) > 1 {
		buildWithVowels(normalizedPattern[:len(normalizedPattern)-1]+"*", true)
		return expandedCombinations
	}
	if strings.Contains(normalizedPattern, "*") {
		buildWithVowels(normalizedPattern, false)
		return expandedCombinations
	}

	appendExpanded(normalizedPattern)
	return expandedCombinations
}

func normalizePatternInstruction(pattern string) string {
	var normalizedBuilder strings.Builder
	normalizedBuilder.Grow(len(pattern))
	for _, currentRune := range strings.ToLower(pattern) {
		switch currentRune {
		case 'á':
			currentRune = 'a'
		case 'é':
			currentRune = 'e'
		case 'í':
			currentRune = 'i'
		case 'ó':
			currentRune = 'o'
		case 'ú', 'ü':
			currentRune = 'u'
		}
		if unicode.IsLetter(currentRune) || unicode.IsDigit(currentRune) || currentRune == '*' || currentRune == '-' {
			normalizedBuilder.WriteRune(currentRune)
		}
	}
	return normalizedBuilder.String()
}

func normalizeText(text string) string {
	var normalizedBuilder strings.Builder
	normalizedBuilder.Grow(len(text))

	for _, currentRune := range strings.ToLower(text) {
		switch currentRune {
		case 'á', 'à', 'â', 'ä', 'ã', 'å', 'ª':
			currentRune = 'a'
		case 'é', 'è', 'ê', 'ë':
			currentRune = 'e'
		case 'í', 'ì', 'î', 'ï':
			currentRune = 'i'
		case 'ó', 'ò', 'ô', 'ö', 'õ', 'º':
			currentRune = 'o'
		case 'ú', 'ü':
			currentRune = 'u'
		case 'ñ':
			currentRune = 'n'
		}

		// Keep only normalized ASCII letters, digits 0-9, and spaces.
		if (currentRune >= 'a' && currentRune <= 'z') || (currentRune >= '0' && currentRune <= '9') || unicode.IsSpace(currentRune) {
			normalizedBuilder.WriteRune(currentRune)
			continue
		}
		// Drop unsupported punctuation/symbols inline instead of inserting separators.
	}

	return strings.Join(strings.Fields(normalizedBuilder.String()), " ")
}

func normalizeToken(token string) string {
	normalizedToken := normalizeText(token)
	normalizedToken = strings.ReplaceAll(normalizedToken, " ", "")
	return normalizedToken
}

func sortSyllableFrequencies(raw map[string]int) []SyllableFrequency {
	sorted := make([]SyllableFrequency, 0, len(raw))
	for syllable, count := range raw {
		sorted = append(sorted, SyllableFrequency{Syllable: syllable, Count: count})
	}

	sort.Slice(sorted, func(leftIndex, rightIndex int) bool {
		left := sorted[leftIndex]
		right := sorted[rightIndex]
		if left.Count == right.Count {
			return left.Syllable < right.Syllable
		}
		return left.Count > right.Count
	})

	return sorted
}

func splitWordIntoSyllables(word string) []string {
	normalizedWord := normalizeToken(word)
	if normalizedWord == "" {
		return nil
	}

	type splitState struct {
		coveredChars  int
		syllableCount int
		skippedChars  int
		nextIndex     int
		syllable      string
		hasPath       bool
	}

	wordLength := len(normalizedWord)
	bestStates := make([]splitState, wordLength+1)
	bestStates[wordLength] = splitState{hasPath: true}

	isBetterState := func(candidate splitState, current splitState) bool {
		if !current.hasPath {
			return true
		}
		if candidate.coveredChars != current.coveredChars {
			return candidate.coveredChars > current.coveredChars
		}
		if candidate.skippedChars != current.skippedChars {
			return candidate.skippedChars < current.skippedChars
		}
		if candidate.syllableCount != current.syllableCount {
			return candidate.syllableCount < current.syllableCount
		}
		return candidate.nextIndex > current.nextIndex
	}

	for index := wordLength - 1; index >= 0; index-- {
		followingState := bestStates[index+1]
		skipCandidate := splitState{
			coveredChars:  followingState.coveredChars,
			syllableCount: followingState.syllableCount,
			skippedChars:  followingState.skippedChars + 1,
			nextIndex:     index + 1,
			syllable:      "",
			hasPath:       true,
		}
		bestStates[index] = skipCandidate

		for _, candidateSyllable := range collectSyllableCandidatesAt(normalizedWord, index, forceTwoLetterSyllableSplit) {
			nextIndex := index + len(candidateSyllable)
			followingState := bestStates[nextIndex]
			candidateState := splitState{
				coveredChars:  followingState.coveredChars + len(candidateSyllable),
				syllableCount: followingState.syllableCount + 1,
				skippedChars:  followingState.skippedChars,
				nextIndex:     nextIndex,
				syllable:      candidateSyllable,
				hasPath:       true,
			}
			if isBetterState(candidateState, bestStates[index]) {
				bestStates[index] = candidateState
			}
		}
	}

	extracted := make([]string, 0, bestStates[0].syllableCount)
	for index := 0; index < wordLength; {
		currentState := bestStates[index]
		if !currentState.hasPath || currentState.nextIndex <= index {
			break
		}
		if currentState.syllable != "" {
			extracted = append(extracted, currentState.syllable)
		} else if currentState.nextIndex == index+1 {
			skippedSingleChar := normalizedWord[index : index+1]
			// Global rule: zero is never emitted as a syllable token.
			if skippedSingleChar == "0" {
				index = currentState.nextIndex
				continue
			}
			// Preserve skipped single letters except trailing plural-like "s" in long words.
			omitTrailingPluralS := wordLength > 5 && index == wordLength-1 && skippedSingleChar == "s"
			if !omitTrailingPluralS {
				extracted = append(extracted, skippedSingleChar)
			}
		}
		index = currentState.nextIndex
	}
	return extracted
}

func collectSyllableCandidatesAt(normalizedWord string, index int, twoLetterOnly bool) []string {
	remaining := normalizedWord[index:]
	candidates := make([]string, 0, 5)

	if !twoLetterOnly {
		// Keep required 4-letter exceptions used by the existing dictionary tests.
		if strings.HasPrefix(remaining, "cion") {
			candidates = append(candidates, "cion")
		}
		if strings.HasPrefix(remaining, "sion") {
			candidates = append(candidates, "sion")
		}
		if len(remaining) >= 3 {
			candidate := remaining[:3]
			if matchesSyllablePattern(candidate) {
				candidates = append(candidates, candidate)
			}
		}
	}
	if len(remaining) >= 2 {
		candidate := remaining[:2]
		if matchesSyllablePattern(candidate) {
			candidates = append(candidates, candidate)
		}
	}
	return candidates
}

func matchesSyllablePattern(candidate string) bool {
	if candidate == "" {
		return false
	}

	runes := []rune(candidate)
	switch len(runes) {
	case 2:
		// Patterns: vowel+consonant (VC) or consonant+vowel (CV).
		return (isVowel(runes[0]) && isConsonant(runes[1])) || (isConsonant(runes[0]) && isVowel(runes[1]))
	case 3:
		// Pattern A: vowel + vowel + consonant (VVC).
		if isVowel(runes[0]) && isVowel(runes[1]) && isConsonant(runes[2]) {
			return true
		}
		// Pattern B: vowel + consonant + consonant (VCC).
		if isVowel(runes[0]) && isConsonant(runes[1]) && isConsonant(runes[2]) {
			return true
		}
		// Pattern C: consonant + consonant + vowel (CCV).
		if isConsonant(runes[0]) && isConsonant(runes[1]) && isVowel(runes[2]) {
			return true
		}
		// Pattern D: consonant + vowel + consonant (CVC).
		if isConsonant(runes[0]) && isVowel(runes[1]) && isConsonant(runes[2]) {
			return true
		}
		return false
	default:
		return false
	}
}

func isVowel(character rune) bool {
	switch character {
	case 'a', 'e', 'i', 'o', 'u':
		return true
	default:
		return false
	}
}

func isConsonant(character rune) bool {
	if !unicode.IsLetter(character) {
		return false
	}
	return !isVowel(character)
}
