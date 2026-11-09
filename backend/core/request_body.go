package core

import (
	"encoding/json"
	"strings"

	"github.com/ivanjoz/colbin"
)

// Request bodies in colbin.
//
// The browser may send a write as a self-describing colbin message (`marshal(...).standalone`
// in the colbin npm package) with `Content-Type: application/vnd.colbin`, the same media type the
// responses use. The schema travels inside the message and was inferred by the browser, so
// colbin.Unmarshal matches it to a Go struct by field name and converts the column types.
//
// Two ways to read it, and a handler needs neither unless it wants the faster one:
//   - NormalizeBody, run once by mainHandler after the caller is authenticated, rewrites a colbin
//     body as the JSON text it stands for, so every handler that still does
//     json.Unmarshal([]byte(*req.Body), &x) keeps working.
//   - UnmarshalBody decodes straight into the target (colbin or JSON by Content-Type) with no JSON
//     text in between. A handler that moves to it should call it instead of json.Unmarshal.

// BodyIsColbin reports whether the request body is a colbin message.
func (req *HandlerArgs) BodyIsColbin() bool {
	return strings.Contains(req.ContentType, ColbinMediaType)
}

// UnmarshalBody decodes the request body into target, whichever form it arrived in.
func (req *HandlerArgs) UnmarshalBody(target any) error {
	if req.Body == nil {
		return json.Unmarshal(nil, target)
	}
	if req.BodyIsColbin() {
		return colbin.Unmarshal([]byte(*req.Body), target)
	}
	return json.Unmarshal([]byte(*req.Body), target)
}

// NormalizeBody replaces a colbin body with its JSON text and clears the marker, so code that
// reads *req.Body as JSON does not have to know. Integers past 2^53 come out as the exact digits
// they were. A body that is not JSON-typed is left alone.
func (req *HandlerArgs) NormalizeBody() error {
	if req.Body == nil || !req.BodyIsColbin() {
		return nil
	}
	text, err := colbin.ToJSON(nil, []byte(*req.Body))
	if err != nil {
		return err
	}
	body := string(text)
	req.Body = &body
	req.ContentType = "application/json"
	return nil
}
