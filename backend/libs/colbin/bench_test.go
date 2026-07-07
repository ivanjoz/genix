package colbin

import (
	"math/rand"
	"reflect"
	"testing"

	"app/libs/cbor"
)

// --- random data generators (deterministic via the passed *rand.Rand) ---

func randString(rng *rand.Rand, maxLen int) string {
	const alphabet = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789 "
	n := rng.Intn(maxLen + 1)
	b := make([]byte, n)
	for i := range b {
		b[i] = alphabet[rng.Intn(len(alphabet))]
	}
	return string(b)
}

// randScalarRecords builds ERP-ish rows: IDs drift upward by small steps (the
// case FOR+bit-packing is built for), plus small ages, occasional negatives,
// short names and floats.
func randScalarRecords(n int, rng *rand.Rand) []ScalarRecord {
	recs := make([]ScalarRecord, n)
	userID := int64(100000 + rng.Intn(1000))
	for i := range recs {
		userID += int64(rng.Intn(20)) // monotonic-ish -> tiny deltas
		recs[i] = ScalarRecord{
			UserID:    userID,
			CompanyID: int32(1 + rng.Intn(8)), // few distinct values
			Age:       int16(rng.Intn(100)),
			Balance:   int32(rng.Intn(4000) - 2000), // signed column
			Active:    rng.Intn(2) == 0,
			Name:      randString(rng, 16),
			Weight:    float32(rng.Intn(20000)) / 100,
			Score:     rng.Float64() * 100,
		}
	}
	return recs
}

func randTag(rng *rand.Rand) Tag {
	return Tag{Key: randString(rng, 8), Weight: int32(rng.Intn(200) - 50)}
}

func randNestedRecords(n int, rng *rand.Rand) []NestedRecord {
	recs := make([]NestedRecord, n)
	base := int64(500)
	// zero-length slices are left nil: colbin does not distinguish nil from an
	// empty slice, so the generated data mirrors that (else DeepEqual would fail).
	for i := range recs {
		base += int64(rng.Intn(10))
		var scores []int32
		for j, m := 0, rng.Intn(7); j < m; j++ {
			scores = append(scores, int32(rng.Intn(1000)))
		}
		var labels []string
		for j, m := 0, rng.Intn(4); j < m; j++ {
			labels = append(labels, randString(rng, 10))
		}
		var tags []Tag
		for j, m := 0, rng.Intn(5); j < m; j++ {
			tags = append(tags, randTag(rng))
		}
		var grid [][]int32
		for j, m := 0, rng.Intn(4); j < m; j++ {
			var row []int32
			for k, mm := 0, rng.Intn(5); k < mm; k++ {
				row = append(row, int32(rng.Intn(500)))
			}
			grid = append(grid, row)
		}
		recs[i] = NestedRecord{
			ID: base, Scores: scores, Labels: labels,
			Meta: randTag(rng), Tags: tags, Grid: grid,
		}
	}
	return recs
}

// TestRandomRoundTrip guards the benchmark data path: random values (incl. large
// deltas and negatives) must survive a colbin round-trip exactly.
func TestRandomRoundTrip(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	scalars := randScalarRecords(500, rng)
	data, err := Marshal(scalars)
	if err != nil {
		t.Fatal(err)
	}
	var gotScalars []ScalarRecord
	if err := Unmarshal(data, &gotScalars); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(scalars, gotScalars) {
		t.Fatal("scalar random round-trip mismatch")
	}

	nested := randNestedRecords(500, rng)
	data, err = Marshal(nested)
	if err != nil {
		t.Fatal(err)
	}
	var gotNested []NestedRecord
	if err := Unmarshal(data, &gotNested); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(nested, gotNested) {
		t.Fatal("nested random round-trip mismatch")
	}
}

// TestSizeReport logs colbin vs CBOR payload size on a 1000-row batch.
func TestSizeReport(t *testing.T) {
	rng := rand.New(rand.NewSource(7))
	scalars := randScalarRecords(1000, rng)
	col, _ := Marshal(scalars)
	cb, _ := cbor.Marshal(scalars)
	t.Logf("scalar x1000: colbin=%d B  cbor=%d B  (%.2fx smaller)",
		len(col), len(cb), float64(len(cb))/float64(len(col)))

	nested := randNestedRecords(1000, rng)
	col, _ = Marshal(nested)
	cb, _ = cbor.Marshal(nested)
	t.Logf("nested x1000: colbin=%d B  cbor=%d B  (%.2fx smaller)",
		len(col), len(cb), float64(len(cb))/float64(len(col)))
}

// --- benchmarks: throughput reported via b.SetBytes (encoded size) ---

const benchN = 1000

func BenchmarkColbinEncodeScalar(b *testing.B) {
	recs := randScalarRecords(benchN, rand.New(rand.NewSource(1)))
	out, _ := Marshal(recs)
	b.SetBytes(int64(len(out)))
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		out, _ = Marshal(recs)
	}
	_ = out
}

func BenchmarkCBOREncodeScalar(b *testing.B) {
	recs := randScalarRecords(benchN, rand.New(rand.NewSource(1)))
	out, _ := cbor.Marshal(recs)
	b.SetBytes(int64(len(out)))
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		out, _ = cbor.Marshal(recs)
	}
	_ = out
}

func BenchmarkColbinDecodeScalar(b *testing.B) {
	recs := randScalarRecords(benchN, rand.New(rand.NewSource(1)))
	data, _ := Marshal(recs)
	b.SetBytes(int64(len(data)))
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var out []ScalarRecord
		_ = Unmarshal(data, &out)
	}
}

func BenchmarkCBORDecodeScalar(b *testing.B) {
	recs := randScalarRecords(benchN, rand.New(rand.NewSource(1)))
	data, _ := cbor.Marshal(recs)
	b.SetBytes(int64(len(data)))
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var out []ScalarRecord
		_ = cbor.Unmarshal(data, &out)
	}
}

func BenchmarkColbinEncodeNested(b *testing.B) {
	recs := randNestedRecords(benchN, rand.New(rand.NewSource(1)))
	out, _ := Marshal(recs)
	b.SetBytes(int64(len(out)))
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		out, _ = Marshal(recs)
	}
	_ = out
}

func BenchmarkColbinDecodeNested(b *testing.B) {
	recs := randNestedRecords(benchN, rand.New(rand.NewSource(1)))
	data, _ := Marshal(recs)
	b.SetBytes(int64(len(data)))
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var out []NestedRecord
		_ = Unmarshal(data, &out)
	}
}
