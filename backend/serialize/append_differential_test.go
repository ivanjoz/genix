package serialize

// The guarantee for the direct-to-bytes rewrite: for the same input, Marshal (append.go) must
// produce exactly what marshalTree (the retained original, in marshal_tree_oracle_test.go)
// produced. Byte-for-byte, because this payload is parsed by a hand-written decoder in the
// browser and any drift in the compact encoding is a silent client-side breakage.
//
// Both encoders read the emit order from the same registry, so every case marshals once up
// front to freeze the order before the two are compared — otherwise the first call would learn
// the order and the second would use it, and the diff would be about ordering rather than
// rendering.

import (
	"testing"
)

type diffScalars struct {
	Int32   int32   `json:"i,omitempty"`
	Int64   int64   `json:"i64,omitempty"`
	Uint16  uint16  `json:"u,omitempty"`
	Float32 float32 `json:"f32,omitempty"`
	Float64 float64 `json:"f64,omitempty"`
	Str     string  `json:"s,omitempty"`
	Flag    bool    `json:"b,omitempty"`
	Int8    int8    `json:"i8,omitempty"`
}

type diffNested struct {
	Name  string        `json:"n,omitempty"`
	Inner diffScalars   `json:"in,omitempty"`
	List  []diffScalars `json:"l,omitempty"`
	Nums  []int32       `json:"nums,omitempty"`
	Words []string      `json:"w,omitempty"`
}

// diffLeadingArray puts a slice first so the empty-skip-block rule is exercised.
type diffLeadingArray struct {
	First  []int32 `json:"f,omitempty"`
	Second int32   `json:"s,omitempty"`
}

type diffWithMap struct {
	ID     int32             `json:"id,omitempty"`
	Labels map[string]string `json:"lb,omitempty"`
	Counts map[int32]int32   `json:"ct,omitempty"`
}

type diffPointers struct {
	ID    int32        `json:"id,omitempty"`
	Ptr   *diffScalars `json:"p,omitempty"`
	NilP  *diffScalars `json:"np,omitempty"`
	Iface any          `json:"an,omitempty"`
}

type diffIgnored struct {
	Kept    int32  `json:"k,omitempty"`
	Skipped string `json:"-"`
	Also    int32  `json:"a,omitempty"`
}

func assertEncodersAgree(t *testing.T, name string, value any) {
	t.Helper()

	// Freeze the emit order so both encoders start from the same registry state.
	if _, err := Marshal(value); err != nil {
		t.Fatalf("%s: warm-up Marshal: %v", name, err)
	}

	fromTree, err := marshalTree(value)
	if err != nil {
		t.Fatalf("%s: marshalTree: %v", name, err)
	}

	fromDirect, err := Marshal(value)
	if err != nil {
		t.Fatalf("%s: Marshal: %v", name, err)
	}

	if string(fromTree) != string(fromDirect) {
		t.Errorf("%s: direct encoder diverged from tree encoder\n tree   %s\n direct %s",
			name, fromTree, fromDirect)
	}
}

func TestDirectEncoderMatchesTreeEncoder(t *testing.T) {
	scalars := []diffScalars{
		{Int32: 1, Str: "one", Flag: true},
		{},
		{Float32: 10.5, Float64: 1e-9, Int64: -9007199254740993},
		{Uint16: 65535, Int8: -128, Str: `quote " back \ slash`},
		{Str: "<p>html & entities</p>\ttab\nnewline\x01ctl"},
		{Float64: 1e21, Int32: -1},
	}

	nested := []diffNested{
		{Name: "with inner", Inner: diffScalars{Int32: 7, Str: "x"}},
		{Name: "with list", List: []diffScalars{{Int32: 1}, {Str: "two"}, {}}},
		{Nums: []int32{1, 2, 3}, Words: []string{"a", "", "c"}},
		{},
		{Name: "empty slices", List: []diffScalars{}, Nums: []int32{}},
	}

	leading := []diffLeadingArray{
		{First: []int32{9, 8}, Second: 3},
		{Second: 4},
		{First: []int32{1}},
		{},
	}

	maps := []diffWithMap{
		{ID: 1, Labels: map[string]string{"b": "two", "a": "one", "c": "three"}},
		{ID: 2, Counts: map[int32]int32{10: 1, 2: 2, 33: 3}},
		{ID: 3},
		{},
	}

	inner := diffScalars{Int32: 42, Str: "pointed at"}
	pointers := []diffPointers{
		{ID: 1, Ptr: &inner},
		{ID: 2},
		{ID: 3, Iface: "boxed string"},
		{ID: 4, Iface: []int32{1, 2}},
	}

	ignored := []diffIgnored{
		{Kept: 1, Skipped: "never emitted", Also: 2},
		{Skipped: "still never emitted"},
	}

	assertEncodersAgree(t, "scalars", &scalars)
	assertEncodersAgree(t, "nested", &nested)
	assertEncodersAgree(t, "leading-array", &leading)
	assertEncodersAgree(t, "maps", &maps)
	assertEncodersAgree(t, "pointers", &pointers)
	assertEncodersAgree(t, "ignored-fields", &ignored)

	// Non-slice roots and bare scalars go through the same entry point.
	assertEncodersAgree(t, "single-struct", &diffScalars{Int32: 5, Str: "solo"})
	assertEncodersAgree(t, "bare-int", 42)
	assertEncodersAgree(t, "bare-string", "hello")
	assertEncodersAgree(t, "bare-nil-slice", []diffScalars(nil))
	assertEncodersAgree(t, "empty-slice", []diffScalars{})
}
