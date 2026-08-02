package serialize_test

// Stage 0 baseline for PLAN_RESPONSE_SERIALIZATION_MEMORY.md.
//
// Marshal currently walks the object graph twice (marshal.go:40 analysis pass, marshal.go:52
// emit pass), builds a throwaway []any tree on the first pass, and then hands the second tree
// to sonic. These benchmarks pin B/op and allocs/op so each stage of the plan can be measured
// against a real number instead of an estimate.
//
// Run: go test ./serialize/ -bench 'Products' -benchmem -run '^$'

import (
	"app/serialize"
	"app/tests/fixtures"
	"testing"
)

var productSizes = []struct {
	name  string
	count int
}{
	{"100", 100},
	{"1000", 1000},
	{"5000", 5000},
}

func BenchmarkMarshalProducts(b *testing.B) {
	for _, size := range productSizes {
		products := fixtures.MakeProducts(size.count)

		// Report the payload size once so B/op can be read as a multiple of the output.
		encoded, err := serialize.Marshal(&products)
		if err != nil {
			b.Fatalf("Marshal returned error: %v", err)
		}
		payloadBytes := len(encoded)

		b.Run(size.name, func(b *testing.B) {
			b.ReportAllocs()
			b.SetBytes(int64(payloadBytes))
			b.ResetTimer()

			for b.Loop() {
				out, err := serialize.Marshal(&products)
				if err != nil {
					b.Fatalf("Marshal returned error: %v", err)
				}
				if len(out) == 0 {
					b.Fatal("Marshal returned an empty payload")
				}
			}
		})
	}
}
