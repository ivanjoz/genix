package colbin

import (
	"reflect"
	"testing"
)

// AstNodeLike mirrors the webpage AstNode shape that motivated ftAny: a struct
// carrying map[string]any / []any fields alongside plain columnar fields.
type AstNodeLike struct {
	TagName    string            `cb:"tag"`
	Order      int32             `cb:"order"`
	Props      map[string]any    `cb:"props"`
	Attributes map[string]string `cb:"attrs"`
	Extra      any               `cb:"extra"`
}

// roundTripAny marshals and unmarshals a value, failing on any error.
func roundTripAny[T any](t *testing.T, in T) T {
	t.Helper()
	data, err := Marshal(in)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	var out T
	if err := Unmarshal(data, &out); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	return out
}

func TestAnyScalarKinds(t *testing.T) {
	// A record whose `any` field cycles through every canonical concrete type.
	type Box struct {
		V any `cb:"v"`
	}
	// Values are compared after normalization to colbin's canonical decode set:
	// signed->int64, unsigned->uint64, any float->float64.
	cases := []struct {
		in   any
		want any
	}{
		{nil, nil},
		{true, true},
		{false, false},
		{int(-7), int64(-7)},
		{int32(42), int64(42)},
		{uint16(9), uint64(9)},
		{float32(1.5), float64(1.5)},
		{float64(3.25), 3.25},
		{"hola", "hola"},
		{[]byte{1, 2, 3}, []byte{1, 2, 3}},
	}
	for _, c := range cases {
		out := roundTripAny(t, []Box{{V: c.in}})
		if !reflect.DeepEqual(out[0].V, c.want) {
			t.Errorf("in %#v (%T): got %#v (%T), want %#v (%T)",
				c.in, c.in, out[0].V, out[0].V, c.want, c.want)
		}
	}
}

func TestAnyNestedMapAndSlice(t *testing.T) {
	// Deeply nested map/slice of any — the JSON-ish shape webpage Props hold.
	node := AstNodeLike{
		TagName: "section",
		Order:   3,
		Props: map[string]any{
			"count":   int64(5),
			"ratio":   0.75,
			"label":   "Hero",
			"enabled": true,
			"tags":    []any{"a", "b", int64(3)},
			"nested": map[string]any{
				"deep": []any{true, nil, "x"},
			},
		},
		Attributes: map[string]string{"class": "hero", "id": "top"},
		Extra:      []any{int64(1), map[string]any{"k": "v"}},
	}
	out := roundTripAny(t, []AstNodeLike{node})
	if !reflect.DeepEqual(out[0], node) {
		t.Fatalf("round-trip mismatch:\n got  %#v\n want %#v", out[0], node)
	}
}

func TestAnyEmptyAndNilMaps(t *testing.T) {
	// nil map/slice/any must survive; empty map decodes back as empty map.
	type Box struct {
		M map[string]any `cb:"m"`
		S []any          `cb:"s"`
		V any            `cb:"v"`
	}
	in := []Box{
		{M: nil, S: nil, V: nil},
		{M: map[string]any{}, S: []any{}, V: "set"},
	}
	out := roundTripAny(t, in)
	if out[0].M != nil || out[0].S != nil || out[0].V != nil {
		t.Errorf("row0 should stay nil, got %#v", out[0])
	}
	if out[1].V != "set" {
		t.Errorf("row1.V = %#v, want \"set\"", out[1].V)
	}
	// Empty (non-nil) map collapses to nil like all colbin empty maps — documented.
}

func TestAnyMultiRecordColumnar(t *testing.T) {
	// Several records so the ftAny column carries a genuine sequence of values.
	recs := make([]AstNodeLike, 20)
	for i := range recs {
		recs[i] = AstNodeLike{
			TagName: "div",
			Order:   int32(i),
			Props:   map[string]any{"i": int64(i), "half": float64(i) / 2},
			Extra:   []any{"row", int64(i)},
		}
	}
	out := roundTripAny(t, recs)
	if !reflect.DeepEqual(out, recs) {
		t.Fatalf("multi-record round-trip mismatch")
	}
}

func TestAnyUnsupportedTypeErrors(t *testing.T) {
	// A struct inside an interface{} has no self-describing form -> Marshal errors
	// (recovered from the internal panic) rather than corrupting the stream.
	type Inner struct{ X int }
	type Box struct {
		V any `cb:"v"`
	}
	if _, err := Marshal([]Box{{V: Inner{X: 1}}}); err == nil {
		t.Fatal("expected error encoding a struct inside any, got nil")
	}
	// Non-string map key is likewise rejected.
	if _, err := Marshal([]Box{{V: map[int]any{1: "x"}}}); err == nil {
		t.Fatal("expected error encoding a non-string-keyed map inside any, got nil")
	}
}
