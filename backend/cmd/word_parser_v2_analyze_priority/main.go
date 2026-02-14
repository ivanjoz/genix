package main

import (
	"encoding/csv"
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	"app/libs/word_parser_v2"
)

func main() {
	inputPath := flag.String("input", "libs/word_parser_v2/productos.txt", "path to product names file")
	topMap := flag.Int("top-map", 400, "top N syllables (2/3 letters) to build map")
	topReduced := flag.Int("top-reduced", 250, "reduced selected syllables count")
	twoLetterRatio := flag.Float64("two-letter-ratio", 0.8, "ratio of 2-letter syllables for reduced set")
	selector := flag.String("selector", "coverage_greedy", "selection algorithm for reduced set: coverage_greedy, ratio")
	outputCSV := flag.String("output-csv", "libs/word_parser_v2/top_syllable_word_map.csv", "output CSV path for syllable map")
	flag.Parse()

	productNames, loadError := word_parser_v2.LoadProductNamesFromFile(*inputPath)
	if loadError != nil {
		log.Fatalf("analysis: load products failed: %v", loadError)
	}

	lengthFilter := map[int]bool{2: true, 3: true}
	topEntries := word_parser_v2.BuildTopSyllableWordMap(productNames, *topMap, lengthFilter)
	if writeError := writeTopEntriesCSV(*outputCSV, topEntries); writeError != nil {
		log.Fatalf("analysis: write csv failed: %v", writeError)
	}

	selectedTop400 := make([]string, 0, len(topEntries))
	for _, entry := range topEntries {
		selectedTop400 = append(selectedTop400, entry.Syllable)
	}
	metricsTop400 := word_parser_v2.EvaluateSyllableSetCoverage(productNames, selectedTop400)

	var reducedSelection []string
	if *selector == "ratio" {
		reducedSelection = word_parser_v2.BuildPrioritizedTopNSyllables(topEntries, *topReduced, *twoLetterRatio)
	} else {
		reducedSelection = word_parser_v2.BuildCoverageGreedyTopNSyllables(productNames, nil, topEntries, *topReduced)
	}
	metricsReduced := word_parser_v2.EvaluateSyllableSetCoverage(productNames, reducedSelection)

	fmt.Printf("analysis: top-map=%d reduced=%d ratio_2letter=%.2f selector=%s\n", *topMap, *topReduced, *twoLetterRatio, *selector)
	fmt.Printf("analysis: map_csv=%s entries=%d\n", *outputCSV, len(topEntries))
	fmt.Printf("coverage_top%d: total_words=%d words_with_1letter=%d usage1=%d usage2=%d usage3=%d other=%d\n",
		*topMap,
		metricsTop400.TotalWords,
		metricsTop400.WordsWithSingleLetterParts,
		metricsTop400.SingleLetterUsage,
		metricsTop400.TwoLetterUsage,
		metricsTop400.ThreeLetterUsage,
		metricsTop400.OtherLengthUsage,
	)
	fmt.Printf("coverage_top%d_prioritized: total_words=%d words_with_1letter=%d usage1=%d usage2=%d usage3=%d other=%d\n",
		*topReduced,
		metricsReduced.TotalWords,
		metricsReduced.WordsWithSingleLetterParts,
		metricsReduced.SingleLetterUsage,
		metricsReduced.TwoLetterUsage,
		metricsReduced.ThreeLetterUsage,
		metricsReduced.OtherLengthUsage,
	)
}

func writeTopEntriesCSV(outputPath string, entries []word_parser_v2.SyllableWordMapEntry) error {
	fileHandle, createError := os.Create(outputPath)
	if createError != nil {
		return createError
	}
	defer fileHandle.Close()

	writer := csv.NewWriter(fileHandle)
	defer writer.Flush()

	if writeError := writer.Write([]string{"rank", "syllable", "length", "count", "example_words"}); writeError != nil {
		return writeError
	}
	for index, entry := range entries {
		row := []string{
			strconv.Itoa(index + 1),
			entry.Syllable,
			strconv.Itoa(entry.Length),
			strconv.Itoa(entry.Count),
			strings.Join(entry.Words, "|"),
		}
		if writeError := writer.Write(row); writeError != nil {
			return writeError
		}
	}
	return nil
}
