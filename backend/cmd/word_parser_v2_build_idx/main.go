package main

import (
	"flag"
	"log"

	"app/libs/word_parser_v2"
)

func main() {
	inputPath := flag.String("input", "libs/word_parser_v1/productos.txt", "path to source product names text file")
	outputPath := flag.String("output", "libs/word_parser_v2/productos.idx", "path to output binary index file")
	totalSlots := flag.Int("slots", word_parser_v2.DefaultDictionarySlots, "total dictionary slots")
	topFrequent := flag.Int("top", word_parser_v2.DefaultTopFrequentCount, "top frequent syllables to prioritize")
	fixedSlots := flag.Int("fixed-slots", word_parser_v2.DefaultDictionarySlots, "max slots available to fixed generator before frequent fill")
	flag.Parse()

	fixedConfig := word_parser_v2.DefaultFixedSyllableGeneratorConfig()
	fixedConfig.MaxSlots = *fixedSlots

	frequentConfig := word_parser_v2.DefaultFrequentSyllableGeneratorConfig()
	frequentConfig.TotalSlots = *totalSlots
	frequentConfig.TopFrequentCount = *topFrequent

	if buildError := word_parser_v2.BuildNamesBinaryIndexFromFile(*inputPath, *outputPath, fixedConfig, frequentConfig); buildError != nil {
		log.Fatalf("word_parser_v2: build failed: %v", buildError)
	}

	log.Printf("word_parser_v2: build completed output=%s", *outputPath)
}
