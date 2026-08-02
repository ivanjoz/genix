package serialize

import (
	"encoding/json"
	"strings"
	"testing"
)

// leadingArrayRecord puts a slice in the field position that every record fills,
// so the optimized order places it first and each header-0 record starts with an
// encoded array — the shape that used to collide with the skip block.
type leadingArrayRecord struct {
	Tags   []int32 `json:"tags"`
	ID     int32   `json:"id"`
	Name   string  `json:"name"`
	Detail string  `json:"detail,omitempty"`
}

func TestHeaderZeroSkipBlockIsUnambiguous(t *testing.T) {
	records := []leadingArrayRecord{
		// Empty (but non-nil) slices are not zero, so they encode as `[2]`, which is
		// byte-identical to a skip block naming field 2.
		{Tags: []int32{}, ID: 313, Name: "first", Detail: "only the first has this"},
		{Tags: []int32{}, ID: 316, Name: "second"},
		{Tags: []int32{2, 6}, ID: 317, Name: "third"},
		{Tags: []int32{}, ID: 323, Name: "fourth"},
	}

	encoded, err := Marshal(&records)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	var wire []json.RawMessage
	if err := json.Unmarshal(encoded, &wire); err != nil {
		t.Fatalf("wire unmarshal: %v", err)
	}
	var content []json.RawMessage
	if err := json.Unmarshal(wire[1], &content); err != nil {
		t.Fatalf("content unmarshal: %v", err)
	}
	// content[0] is the slice header; records follow. Every header-0 record must
	// carry a skip block so position 1 can never be read as a value.
	for i, record := range content[1:] {
		text := string(record)
		if !strings.HasPrefix(text, "[0,") {
			continue
		}
		if !strings.HasPrefix(text, "[0,[") {
			t.Errorf("record %d starts with a bare value, skip block missing: %s", i, text)
		}
	}

	var decoded []leadingArrayRecord
	if err := Unmarshal(encoded, &decoded); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if len(decoded) != len(records) {
		t.Fatalf("expected %d records, got %d", len(records), len(decoded))
	}
	for i, want := range records {
		got := decoded[i]
		if got.ID != want.ID || got.Name != want.Name || got.Detail != want.Detail {
			t.Errorf("record %d round-tripped as %+v, want %+v", i, got, want)
		}
		if len(got.Tags) != len(want.Tags) {
			t.Errorf("record %d tags round-tripped as %v, want %v", i, got.Tags, want.Tags)
		}
	}
}
