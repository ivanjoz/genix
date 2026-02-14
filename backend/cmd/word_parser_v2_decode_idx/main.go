package main

import (
	"flag"
	"fmt"
	"log"
	"sort"
	"strings"
	"time"

	"app/libs/word_parser_v2"
)

func main() {
	inputPath := flag.String("input", "libs/word_parser_v2/productos.idx", "path to binary index file")
	sampleCount := flag.Int("sample", 50, "number of random decoded records to print")
	seed := flag.Int64("seed", 0, "random seed (0 means current time)")
	printDictionary := flag.Bool("print-dictionary", false, "print final dictionary slots and exit")
	printDictionaryAliases := flag.Bool("print-dictionary-aliases", false, "print final dictionary with alias groups and exit")
	flag.Parse()

	if *printDictionary || *printDictionaryAliases {
		finalDictionaryTokens, decodeDictionaryError := word_parser_v2.DecodeBinaryIndexDictionary(*inputPath)
		if decodeDictionaryError != nil {
			log.Fatalf("word_parser_v2: decode dictionary failed: %v", decodeDictionaryError)
		}
		fmt.Printf("word_parser_v2: final dictionary count=%d\n", len(finalDictionaryTokens))

		if *printDictionaryAliases {
			aliasGroupByToken := make(map[string][]string, 64)
			for _, preassignedAliases := range word_parser_v2.DefaultFixedSyllableGeneratorConfig().PreassignedSlots {
				normalizedAliases := make([]string, 0, len(preassignedAliases))
				seenAlias := make(map[string]struct{}, len(preassignedAliases))
				for _, aliasToken := range preassignedAliases {
					normalizedAlias := strings.TrimSpace(strings.ToLower(aliasToken))
					if normalizedAlias == "" {
						continue
					}
					if _, exists := seenAlias[normalizedAlias]; exists {
						continue
					}
					seenAlias[normalizedAlias] = struct{}{}
					normalizedAliases = append(normalizedAliases, normalizedAlias)
				}
				sort.Strings(normalizedAliases)
				for _, aliasToken := range normalizedAliases {
					aliasGroupByToken[aliasToken] = normalizedAliases
				}
			}
			for dictionaryIndex, dictionaryToken := range finalDictionaryTokens {
				slotID := dictionaryIndex + 1
				aliasGroup, hasAliases := aliasGroupByToken[dictionaryToken]
				if !hasAliases || len(aliasGroup) == 0 {
					fmt.Printf("[%03d] %s\n", slotID, dictionaryToken)
					continue
				}
				fmt.Printf("[%03d] %s\n", slotID, strings.Join(aliasGroup, ", "))
			}
			return
		}

		for dictionaryIndex, dictionaryToken := range finalDictionaryTokens {
			fmt.Printf("[%03d] %s\n", dictionaryIndex+1, dictionaryToken)
		}
		return
	}

	decodedRecords, summary, decodeError := word_parser_v2.DecodeBinaryIndexFile(*inputPath)
	if decodeError != nil {
		log.Fatalf("word_parser_v2: decode failed: %v", decodeError)
	}

	effectiveSeed := *seed
	if effectiveSeed == 0 {
		effectiveSeed = time.Now().UnixNano()
	}
	sampledRecords := word_parser_v2.SampleRandomDecodedRecords(decodedRecords, *sampleCount, effectiveSeed)

	fmt.Printf("word_parser_v2: decoded index summary\n")
	fmt.Printf("  records=%d dictionary_count=%d dictionary_bytes=%d\n", summary.RecordCount, summary.DictionaryCount, summary.DictionaryBytes)
	fmt.Printf("  shapes_class0=%d shapes_class1=%d shape_table_bytes=%d shape_usage_bytes=%d\n", summary.ShapeCountClass0, summary.ShapeCountClass1, summary.ShapeDictionaryBytes, summary.ShapeUsageBytes)
	fmt.Printf("  content_bytes=%d shape_delta_class0=%t shape_delta_class1=%t\n", summary.ContentBytes, summary.ShapeDeltaClass0, summary.ShapeDeltaClass1)
	fmt.Printf("  shape_delta_counts: d8=%d d16=%d d24=%d\n", summary.ShapeDelta8Count, summary.ShapeDelta16Count, summary.ShapeDelta24Count)
	fmt.Printf("  shape_delta_bits: d8=%d d16=%d d24=%d\n", summary.ShapeDelta8Bits, summary.ShapeDelta16Bits, summary.ShapeDelta24Bits)
	if summary.ShapeProgress5Pct != "" {
		fmt.Printf("  shape_delta_progress_5pct: %s\n", summary.ShapeProgress5Pct)
	}
	fmt.Printf("  sample_count=%d seed=%d\n", len(sampledRecords), effectiveSeed)
	fmt.Println()

	for _, sampledRecord := range sampledRecords {
		fmt.Printf("[%06d] class=%d shape_id=%d shape=%v text=%s\n", sampledRecord.RecordIndex, sampledRecord.ShapeClass, sampledRecord.ShapeLocalID, sampledRecord.WordSizes, sampledRecord.DecodedPhrase)
	}
}
