package colbin

import (
	"reflect"
	"testing"
)

// Value mode covers the top-level types the ORM stores as complex (colType 9)
// blobs that aren't struct/[]struct: maps and *struct.

func TestValueModeStringMap(t *testing.T) {
	in := map[string]string{"class": "hero", "id": "top", "": "empty-key"}
	var out map[string]string
	data, err := Marshal(in)
	if err != nil {
		t.Fatal(err)
	}
	if err := Unmarshal(data, &out); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(in, out) {
		t.Fatalf("got %#v want %#v", out, in)
	}
}

func TestValueModeAnyMap(t *testing.T) {
	// The AstNode.Props shape: nested map[string]any round-tripped at top level.
	in := map[string]any{
		"count":  int64(5),
		"ratio":  0.5,
		"label":  "hero",
		"list":   []any{"a", int64(2), true},
		"nested": map[string]any{"deep": nil},
	}
	var out map[string]any
	data, err := Marshal(in)
	if err != nil {
		t.Fatal(err)
	}
	if err := Unmarshal(data, &out); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(in, out) {
		t.Fatalf("got %#v want %#v", out, in)
	}
}

func TestValueModePointerToStruct(t *testing.T) {
	// *struct: encode derefs, decode allocates through the **T chain (the ORM
	// hands Unmarshal a **ContentFields).
	type ContentFields struct {
		Title string `cb:"title"`
		Limit int32  `cb:"limit"`
	}
	in := &ContentFields{Title: "Hola", Limit: 9}
	data, err := Marshal(in)
	if err != nil {
		t.Fatal(err)
	}
	var out *ContentFields
	if err := Unmarshal(data, &out); err != nil {
		t.Fatal(err)
	}
	if out == nil || *out != *in {
		t.Fatalf("got %#v want %#v", out, in)
	}
}

func TestValueModeSliceOfPointerStruct(t *testing.T) {
	// []*struct: elem is *struct (nullable), so this is value mode, not records.
	type Row struct {
		N int32 `cb:"n"`
	}
	in := []*Row{{N: 1}, nil, {N: 3}}
	data, err := Marshal(in)
	if err != nil {
		t.Fatal(err)
	}
	var out []*Row
	if err := Unmarshal(data, &out); err != nil {
		t.Fatal(err)
	}
	if len(out) != 3 || out[0].N != 1 || out[1] != nil || out[2].N != 3 {
		t.Fatalf("got %#v", out)
	}
}

func TestRecordsModeStillWorks(t *testing.T) {
	// Guard: struct and []struct must stay on the columnar records path.
	if !topLevelIsRecords(reflect.TypeOf(ScalarRecord{})) {
		t.Error("struct should be records mode")
	}
	if !topLevelIsRecords(reflect.TypeOf([]ScalarRecord{})) {
		t.Error("[]struct should be records mode")
	}
	if topLevelIsRecords(reflect.TypeOf(map[string]any{})) {
		t.Error("map should be value mode")
	}
	if topLevelIsRecords(reflect.TypeOf([]*ScalarRecord{})) {
		t.Error("[]*struct should be value mode")
	}
}
