package main

import (
	"flag"
	"fmt"
	"log"

	"app/libs/index_builder"
)

func main() {
	inputPath := flag.String("input", "libs/index_builder/productos.txt", "path to input text file (one record per line)")
	outputPath := flag.String("output", "libs/index_builder/productos.idx", "path to output idx file")
	maxWords := flag.Int("max-words", 8, "max words per record")
	maxSyllablesPerWord := flag.Int("max-syllables-per-word", 7, "max syllables per word")
	maxSlots := flag.Int("slots", 255, "max dictionary slots")
	flag.Parse()

	records, loadErr := index_builder.LoadRecordsFromTextFile(*inputPath)
	if loadErr != nil {
		log.Fatalf("index_builder: load failed: %v", loadErr)
	}

	options := index_builder.DefaultOptions()
	options.MaxWordsPerRecord = int32(*maxWords)
	options.MaxSyllablesPerWord = int32(*maxSyllablesPerWord)
	options.MaxDictionarySlots = int32(*maxSlots)

	result, buildErr := index_builder.Build(records, options)
	if buildErr != nil {
		log.Fatalf("index_builder: build failed: %v", buildErr)
	}
	if writeErr := result.WriteBinaryFile(*outputPath); writeErr != nil {
		log.Fatalf("index_builder: write failed: %v", writeErr)
	}

	fmt.Printf("Index Summary\n")
	fmt.Printf("input_records: %d\n", result.Stats.InputRecords)
	fmt.Printf("encoded_records: %d\n", result.Stats.EncodedRecords)
	fmt.Printf("fixed_syllables_count: %d\n", result.Stats.FixedSyllablesCount)
	fmt.Printf("computed_syllables_count: %d\n", result.Stats.ComputedSyllablesCount)
	fmt.Printf("most_used_words_top20: %s\n", result.Stats.MostUsedWordsTop20)
	fmt.Printf("shape_delta_counts: d8=%d d16=%d d24=%d\n", result.Stats.ShapeDelta8Count, result.Stats.ShapeDelta16Count, result.Stats.ShapeDelta24Count)
	fmt.Printf("dictionary_bytes: %d\n", result.Stats.DictionaryBytes)
	fmt.Printf("extracted_syllables_total: %d\n", result.Stats.ExtractedSyllablesTotal)
	fmt.Printf("shape_coverage_top255_pct: %.2f\n", result.Stats.ShapeCoverageTop255Pct)
	fmt.Printf("unique_shapes: %d\n", result.Stats.UniqueShapes)
	fmt.Printf("shape_size_bytes: %d\n", result.Stats.ShapesBytes)
	fmt.Printf("text_content_size_bytes: %d\n", result.Stats.ContentBytes)
	fmt.Printf("total_size_bytes: %d\n", result.Stats.TotalBytes)
	fmt.Printf("output: %s\n", *outputPath)
}
