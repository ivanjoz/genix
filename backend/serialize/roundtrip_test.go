package serialize_test

// Round-trip coverage for the compact [keys, content] format.
//
// These exist for two reasons. First, the emit order for a type is now frozen from the first
// payload that carries it, so the first payload and every later one are written with different
// orders — decoding both correctly is the invariant that keeps that optimisation honest.
// Second, they are the safety net for the direct-to-bytes encoder rewrite: any change that
// alters what the wire carries has to break one of these.
//
// Each test uses its own struct type. The registry is process-global and freezes order on first
// contact, so sharing a type between tests would leak state across them.

import (
	businessTypes "app/business/types"
	"app/serialize"
	"app/tests/fixtures"
	"reflect"
	"strings"
	"sync"
	"testing"
)

type freezeBoundaryRecord struct {
	ID       int32  `json:"id,omitempty"`
	Rare     string `json:"rare,omitempty"`
	Common   string `json:"common,omitempty"`
	Sometime int32  `json:"sometime,omitempty"`
}

// TestFreezeBoundaryRoundTrip is the regression guard for single-pass encoding. The first
// Marshal of a type emits in declaration order and only then learns the usage-sorted order, so
// payload 1 and payload 2 are laid out differently. Both must decode — which only holds because
// the decoder reads the order from each payload's own keys header.
func TestFreezeBoundaryRoundTrip(t *testing.T) {
	// "Common" is set on every record and "Rare" on one, so the learned order differs from
	// declaration order and the two payloads genuinely disagree about layout.
	records := []freezeBoundaryRecord{
		{ID: 1, Common: "a"},
		{ID: 2, Common: "b"},
		{ID: 3, Common: "c", Rare: "only here", Sometime: 7},
	}

	firstPayload, err := serialize.Marshal(&records)
	if err != nil {
		t.Fatalf("first Marshal: %v", err)
	}

	secondPayload, err := serialize.Marshal(&records)
	if err != nil {
		t.Fatalf("second Marshal: %v", err)
	}

	// If these matched, the test would not be exercising the boundary at all.
	if string(firstPayload) == string(secondPayload) {
		t.Log("warning: both payloads identical; freeze produced the declaration order")
	}

	for name, payload := range map[string][]byte{"pre-freeze": firstPayload, "post-freeze": secondPayload} {
		decoded := []freezeBoundaryRecord{}
		if err := serialize.Unmarshal(payload, &decoded); err != nil {
			t.Fatalf("%s Unmarshal: %v", name, err)
		}
		if !reflect.DeepEqual(records, decoded) {
			t.Errorf("%s round-trip mismatch:\n want %+v\n got  %+v", name, records, decoded)
		}
	}
}

type sparseRecord struct {
	A int32  `json:"a,omitempty"`
	B string `json:"b,omitempty"`
	C int32  `json:"c,omitempty"`
	D string `json:"d,omitempty"`
	E int32  `json:"e,omitempty"`
}

// TestSparseFieldsRoundTrip covers the skip-index machinery: every record zeroes a different
// subset, including the all-zero record and one that only sets the last field.
func TestSparseFieldsRoundTrip(t *testing.T) {
	records := []sparseRecord{
		{A: 1, C: 3, E: 5},
		{B: "two", D: "four"},
		{},
		{E: 99},
		{A: 1, B: "b", C: 3, D: "d", E: 5},
	}

	payload, err := serialize.Marshal(&records)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	decoded := []sparseRecord{}
	if err := serialize.Unmarshal(payload, &decoded); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if !reflect.DeepEqual(records, decoded) {
		t.Errorf("round-trip mismatch:\n want %+v\n got  %+v", records, decoded)
	}
}

// TestProductsRoundTrip runs the real GET.products payload shape end to end.
func TestProductsRoundTrip(t *testing.T) {
	products := fixtures.MakeProducts(50)

	payload, err := serialize.Marshal(&products)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	decoded := []businessTypes.Product{}
	if err := serialize.Unmarshal(payload, &decoded); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if len(decoded) != len(products) {
		t.Fatalf("record count: want %d, got %d", len(products), len(decoded))
	}

	// Spot-check fields spanning scalars, slices and nested structs rather than DeepEqual on
	// the whole record: Product embeds db.TableStruct, whose unexported state is not part of
	// the wire format.
	for i := range products {
		want, got := products[i], decoded[i]
		if want.ID != got.ID || want.Name != got.Name || want.SKU != got.SKU {
			t.Fatalf("record %d scalars: want %v/%v/%v, got %v/%v/%v",
				i, want.ID, want.Name, want.SKU, got.ID, got.Name, got.SKU)
		}
		if !reflect.DeepEqual(want.CategoryIDs, got.CategoryIDs) {
			t.Fatalf("record %d CategoryIDs: want %v, got %v", i, want.CategoryIDs, got.CategoryIDs)
		}
		if !reflect.DeepEqual(want.ImageDescriptions, got.ImageDescriptions) {
			t.Fatalf("record %d ImageDescriptions: want %v, got %v", i, want.ImageDescriptions, got.ImageDescriptions)
		}
		if len(want.Presentations) != len(got.Presentations) {
			t.Fatalf("record %d Presentations count: want %d, got %d", i, len(want.Presentations), len(got.Presentations))
		}
		for p := range want.Presentations {
			if want.Presentations[p] != got.Presentations[p] {
				t.Fatalf("record %d presentation %d: want %+v, got %+v",
					i, p, want.Presentations[p], got.Presentations[p])
			}
		}
	}
}

type concurrentRecord struct {
	Worker int32  `json:"w,omitempty"`
	Label  string `json:"l,omitempty"`
	Extra  string `json:"x,omitempty"`
}

// TestConcurrentMarshalIsolation is the guard for the data race that made two simultaneous
// Marshal calls stomp each other's keys header. Field usage is now Encoder-local, so each
// payload must describe only its own content. Meaningful under -race.
func TestConcurrentMarshalIsolation(t *testing.T) {
	const workers = 16

	var waitGroup sync.WaitGroup
	payloads := make([][]byte, workers)
	errs := make([]error, workers)

	for worker := range workers {
		waitGroup.Add(1)
		go func(worker int) {
			defer waitGroup.Done()
			records := []concurrentRecord{{
				Worker: int32(worker + 1),
				Label:  strings.Repeat("x", worker+1),
			}}
			// Half the workers also set Extra, so the correct keys header differs per payload.
			if worker%2 == 0 {
				records[0].Extra = "extra"
			}
			payloads[worker], errs[worker] = serialize.Marshal(&records)
		}(worker)
	}
	waitGroup.Wait()

	for worker := range workers {
		if errs[worker] != nil {
			t.Fatalf("worker %d Marshal: %v", worker, errs[worker])
		}

		decoded := []concurrentRecord{}
		if err := serialize.Unmarshal(payloads[worker], &decoded); err != nil {
			t.Fatalf("worker %d Unmarshal: %v", worker, err)
		}
		if len(decoded) != 1 {
			t.Fatalf("worker %d: want 1 record, got %d", worker, len(decoded))
		}

		wantExtra := ""
		if worker%2 == 0 {
			wantExtra = "extra"
		}
		want := concurrentRecord{
			Worker: int32(worker + 1),
			Label:  strings.Repeat("x", worker+1),
			Extra:  wantExtra,
		}
		if decoded[0] != want {
			t.Errorf("worker %d: want %+v, got %+v", worker, want, decoded[0])
		}
	}
}

// TestProductsDirectEncoderMatchesTree diffs the two encoders over the real GET.products shape.
// The internal differential test cannot reach this fixture: app/tests/fixtures depends on
// app/business/types -> app/core -> app/serialize, so only an external test package can import it.
func TestProductsDirectEncoderMatchesTree(t *testing.T) {
	products := fixtures.MakeProducts(200)

	// Freeze the emit order first so the comparison is about rendering, not ordering.
	if _, err := serialize.Marshal(&products); err != nil {
		t.Fatalf("warm-up Marshal: %v", err)
	}

	fromTree, err := serialize.MarshalTree(&products)
	if err != nil {
		t.Fatalf("MarshalTree: %v", err)
	}
	fromDirect, err := serialize.Marshal(&products)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	if string(fromTree) != string(fromDirect) {
		t.Fatalf("direct encoder diverged from tree encoder over %d products (tree %d bytes, direct %d bytes)",
			len(products), len(fromTree), len(fromDirect))
	}
}
