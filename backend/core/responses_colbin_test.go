package core

import (
	"encoding/json"
	"testing"

	"github.com/ivanjoz/colbin"
)

type colbinProbeRow struct {
	ID      int32  `json:"ID" cb:"ID,1"`
	Updated int32  `json:"upd" cb:"upd,2"`
	Name    string `json:"Name" cb:"Name,3"`
	Hidden  string `json:"-" cb:"-"`
}

// The negotiation is by Accept alone: no Accept, or a wildcard, keeps plain JSON.
func TestMakeResponseColbinNegotiation(t *testing.T) {
	rows := []colbinProbeRow{{ID: 1, Updated: 7, Name: "a", Hidden: "x"}, {ID: 2, Updated: 8, Name: "b"}}

	for _, accept := range []string{"", "*/*", "application/json"} {
		req := &HandlerArgs{Accept: accept}
		response := MakeResponse(req, &rows)
		if response.IsBinary || response.Headers["Content-Type"] == ColbinMediaType {
			t.Errorf("Accept %q: want plain JSON, got a colbin response", accept)
		}
		var decoded []map[string]any
		if err := json.Unmarshal(*response.Body, &decoded); err != nil || len(decoded) != 2 {
			t.Errorf("Accept %q: body is not the JSON rows: %v %s", accept, err, *response.Body)
		}
	}

	req := &HandlerArgs{Accept: ColbinMediaType + ", application/json"}
	response := MakeResponse(req, &rows)
	if !response.IsBinary || response.Headers["Content-Type"] != ColbinMediaType {
		t.Fatalf("want a binary colbin response, got binary=%v type=%q", response.IsBinary, response.Headers["Content-Type"])
	}
	// Self-describing: the message alone renders as JSON carrying the json-tag names.
	text, err := colbin.ToJSON(nil, *response.Body)
	if err != nil {
		t.Fatalf("ToJSON: %v", err)
	}
	var decoded []map[string]any
	if err := json.Unmarshal(text, &decoded); err != nil {
		t.Fatalf("rendered JSON does not parse: %v %s", err, text)
	}
	if len(decoded) != 2 || decoded[0]["upd"] != float64(7) || decoded[0]["Name"] != "a" {
		t.Errorf("unexpected rows: %s", text)
	}
	if _, leaked := decoded[0]["Hidden"]; leaked {
		t.Errorf("a cb:\"-\" field was encoded: %s", text)
	}
}

// A nil slice must still be a valid colbin body, as the JSON path answers "[]".
func TestMakeResponseColbinNilSlice(t *testing.T) {
	var rows []colbinProbeRow
	response := MakeResponse(&HandlerArgs{Accept: ColbinMediaType}, &rows)
	if response.Body == nil || len(response.Error) > 0 {
		t.Fatalf("nil slice failed: %+v", response)
	}
	text, err := colbin.ToJSON(nil, *response.Body)
	t.Logf("nil slice renders %q err=%v", text, err)
}
