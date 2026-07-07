package colbin

import (
	"reflect"
	"testing"

	"app/libs/cbor"
)

// Scalar demo record: ids, ages, a name, a flag, and two float weights — the
// shape of a typical DB row batch. Uses cb tags on a couple of fields.
type ScalarRecord struct {
	UserID    int64   `cb:"user_id"`
	CompanyID int32   `cb:"company_id"`
	Age       int16   `cb:"age"`
	Balance   int32   // negative allowed -> signed column
	Active    bool    `cb:"active"`
	Name      string  `cb:"name"`
	Weight    float32 `cb:"weight"`
	Score     float64 `cb:"score"`
}

func scalarSample() []ScalarRecord {
	return []ScalarRecord{
		{UserID: 100000, CompanyID: 12, Age: 34, Balance: -50, Active: true, Name: "Ivan", Weight: 71.5, Score: 9.81},
		{UserID: 100005, CompanyID: 12, Age: 0, Balance: 0, Active: false, Name: "Ana", Weight: 0, Score: 3.14},
		{UserID: 100011, CompanyID: 13, Age: 41, Balance: 200, Active: true, Name: "", Weight: 55.25, Score: 0},
		{UserID: 100050, CompanyID: 13, Age: 29, Balance: -1000, Active: false, Name: "José López", Weight: 88, Score: 2.5},
	}
}

func TestScalarRoundTrip(t *testing.T) {
	in := scalarSample()
	data, err := Marshal(in)
	if err != nil {
		t.Fatal(err)
	}
	var out []ScalarRecord
	if err := Unmarshal(data, &out); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(in, out) {
		t.Fatalf("round-trip mismatch:\n in=%+v\nout=%+v", in, out)
	}
}

func TestSingleStructRoundTrip(t *testing.T) {
	in := scalarSample()[0]
	data, err := Marshal(in)
	if err != nil {
		t.Fatal(err)
	}
	var out ScalarRecord
	if err := Unmarshal(data, &out); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(in, out) {
		t.Fatalf("single mismatch:\n in=%+v\nout=%+v", in, out)
	}
}

// Phase B: scalar arrays, a nested struct, and an array of structs.
type Tag struct {
	Key    string `cb:"key"`
	Weight int32  `cb:"weight"`
}

type NestedRecord struct {
	ID     int64     `cb:"id"`
	Scores []int32   `cb:"scores"` // scalar array (FOR-packed flattened)
	Labels []string  `cb:"labels"` // string array
	Meta   Tag       `cb:"meta"`   // nested struct (sub-table)
	Tags   []Tag     `cb:"tags"`   // array of structs (recursion)
	Grid   [][]int32 `cb:"grid"`   // nested arrays
}

func nestedSample() []NestedRecord {
	return []NestedRecord{
		{
			ID:     500,
			Scores: []int32{10, 12, 11, 15},
			Labels: []string{"alpha", "beta"},
			Meta:   Tag{Key: "root", Weight: 3},
			Tags:   []Tag{{Key: "x", Weight: 1}, {Key: "y", Weight: 2}},
			Grid:   [][]int32{{1, 2}, {3, 4, 5}},
		},
		{
			ID:     501,
			Scores: nil, // empty array stays nil
			Labels: []string{"gamma"},
			Meta:   Tag{Key: "", Weight: 0},
			Tags:   nil,
			Grid:   [][]int32{{9}},
		},
		{
			ID:     0,
			Scores: []int32{7},
			Labels: nil,
			Meta:   Tag{Key: "leaf", Weight: -4},
			Tags:   []Tag{{Key: "z", Weight: 99}},
			Grid:   nil,
		},
	}
}

func TestNestedRoundTrip(t *testing.T) {
	in := nestedSample()
	data, err := Marshal(in)
	if err != nil {
		t.Fatal(err)
	}
	var out []NestedRecord
	if err := Unmarshal(data, &out); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(in, out) {
		t.Fatalf("nested round-trip mismatch:\n in=%+v\nout=%+v", in, out)
	}
	cb, _ := cbor.Marshal(in)
	t.Logf("nested: colbin=%d bytes  cbor=%d bytes", len(data), len(cb))
}

// TestSizeVsCBOR is informational: reports colbin vs CBOR payload size on the batch.
func TestSizeVsCBOR(t *testing.T) {
	in := scalarSample()
	cb, err := colbin{}.marshalCBOR(in)
	if err != nil {
		t.Fatal(err)
	}
	col, err := Marshal(in)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("records=%d  colbin=%d bytes  cbor=%d bytes", len(in), len(col), len(cb))
}

// tiny shim so the test file has a single import surface for cbor.
type colbin struct{}

func (colbin) marshalCBOR(v any) ([]byte, error) { return cbor.Marshal(v) }
