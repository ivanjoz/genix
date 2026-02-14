package main

import (
	"flag"
	"log"

	"app/libs/word_parser_v2"
)

func main() {
	inputPath := flag.String("input", "libs/word_parser_v1/productos.txt", "path to source product names text file")
	outputPath := flag.String("output", "libs/word_parser_v2/productos.idx", "path to output binary index file")
	totalSlots := flag.Int("slots", 255, "total dictionary slots")
	topFrequent := flag.Int("top", word_parser_v2.DefaultTopFrequentCount, "top frequent syllables to prioritize")
	fixedSlots := flag.Int("fixed-slots", 60, "max slots available to fixed generator before frequent fill")
	strategy := flag.String("strategy", "frequency", "sorting strategy: frequency, atomic_first, atomic_digraph, ratio_80_20, coverage_greedy")
	twoLetterOnly := flag.Bool("two-letter-only", true, "force splitter to use only 2-letter syllable candidates")
	flag.Parse()
	word_parser_v2.SetForceTwoLetterSyllableSplit(*twoLetterOnly)

	fixedConfig := word_parser_v2.DefaultFixedSyllableGeneratorConfig()
	fixedConfig.MaxSlots = *fixedSlots

	frequentConfig := word_parser_v2.DefaultFrequentSyllableGeneratorConfig()
	frequentConfig.TotalSlots = *totalSlots
	frequentConfig.TopFrequentCount = *topFrequent
	frequentConfig.Strategy = *strategy

	if buildError := word_parser_v2.BuildNamesBinaryIndexFromFile(*inputPath, *outputPath, fixedConfig, frequentConfig); buildError != nil {
		log.Fatalf("word_parser_v2: build failed: %v", buildError)
	}

	log.Printf("word_parser_v2: build completed output=%s", *outputPath)
}
