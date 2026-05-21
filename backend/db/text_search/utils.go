// Package text_search is a thin HTTP client for the SeekStorm lexical-only
// product index. ScyllaDB stays the source of truth for products; SeekStorm
// only stores {search_text, product_id} per the design captured in
// cloud/text-searh/seekstorm_product_search_setup.md.
//
// The REST API is keyed by numeric index_id (one per client). Callers are
// expected to resolve client_id -> index_id from ScyllaDB before invoking
// PostIndexRecord — this package does not cache that mapping.
package text_search

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"app/core"
)

// httpTimeout — SeekStorm is colocated on the same host, so a short timeout
// catches stalls quickly. Bump per-call via context if a bulk indexer needs
// more headroom.
const httpTimeout = 3 * time.Second

// seekstormClient wraps the SeekStorm HTTP API. One per process; *http.Client
// is safe for concurrent use, so we share a singleton via getClient.
type seekstormClient struct {
	BaseURL string
	APIKey  string
	HTTP    *http.Client
}

var (
	clientOnce sync.Once
	clientInst *seekstormClient
	clientErr  error
)

// getClient lazily builds the singleton, failing fast when SEEKSTORM_URL or
// SEEKSTORM_API_KEY are missing in credentials.json. We don't validate at
// process start because not every backend deployment uses search.
func getClient() (*seekstormClient, error) {
	clientOnce.Do(func() {
		if core.Env == nil {
			clientErr = errors.New("core.Env not populated; call core.PopulateVariables before text_search")
			return
		}
		baseURL := strings.TrimRight(core.Env.SEEKSTORM_URL, "/")
		apiKey := core.Env.SEEKSTORM_API_KEY
		if baseURL == "" || apiKey == "" {
			clientErr = errors.New("SEEKSTORM_URL or SEEKSTORM_API_KEY not set in credentials.json")
			return
		}
		clientInst = &seekstormClient{
			BaseURL: baseURL,
			APIKey:  apiKey,
			HTTP:    &http.Client{Timeout: httpTimeout},
		}
	})
	return clientInst, clientErr
}

// doJSON sends a JSON request and optionally decodes the JSON response.
// Non-2xx replies surface the upstream body (truncated) so schema/auth
// errors are immediately readable in logs without extra plumbing.
func (c *seekstormClient) doJSON(ctx context.Context, method, path string, in any, out any) error {
	var body io.Reader
	if in != nil {
		raw, err := json.Marshal(in)
		if err != nil {
			return fmt.Errorf("marshal %s: %w", path, err)
		}
		body = bytes.NewReader(raw)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.BaseURL+path, body)
	if err != nil {
		return err
	}
	req.Header.Set("apikey", c.APIKey)
	if in != nil {
		req.Header.Set("content-type", "application/json")
	}
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return fmt.Errorf("seekstorm http %s %s: %w", method, path, err)
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("seekstorm read %s %s: %w", method, path, err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("seekstorm %s %s: status=%d body=%s", method, path, resp.StatusCode, truncate(string(respBody), 500))
	}
	if out != nil && len(respBody) > 0 {
		if err := json.Unmarshal(respBody, out); err != nil {
			return fmt.Errorf("seekstorm decode %s: %w (body=%s)", path, err, truncate(string(respBody), 200))
		}
	}
	return nil
}

// schemaField mirrors one entry of the SeekStorm index schema JSON.
type schemaField struct {
	Field        string `json:"field"`
	FieldType    string `json:"field_type"`
	Store        bool   `json:"store"`
	IndexLexical bool   `json:"index_lexical"`
}

type createIndexRequest struct {
	IndexName  string        `json:"index_name"`
	Similarity string        `json:"similarity"`
	Tokenizer  string        `json:"tokenizer"`
	Schema     []schemaField `json:"schema"`
}

// createIndexResponse — SeekStorm versions differ on whether the returned
// id field is "index_id" or "id"; we accept both and use whichever is set.
type createIndexResponse struct {
	IndexID *int64 `json:"index_id,omitempty"`
	ID      *int64 `json:"id,omitempty"`
}

// CreateIndex provisions one lexical-only product index named indexName with
// the minimal {search_text, product_id} schema from §6 of the setup doc.
// Returns the numeric index_id assigned by SeekStorm — persist it in
// ScyllaDB (client_id -> index_id) so queries can address it later.
func CreateIndex(ctx context.Context, indexName string) (int64, error) {
	client, err := getClient()
	if err != nil {
		return 0, err
	}
	body := createIndexRequest{
		IndexName:  indexName,
		Similarity: "Bm25f",
		Tokenizer:  "UnicodeAlphanumeric",
		Schema: []schemaField{
			// search_text is indexed but not stored — ScyllaDB has the
			// original product text, so keeping it out of SeekStorm saves
			// disk and removes a redundant copy.
			{Field: "search_text", FieldType: "Text", Store: false, IndexLexical: true},
			// product_id is stored so query results carry the ScyllaDB key
			// back to the caller without an extra lookup table.
			{Field: "product_id", FieldType: "U64", Store: true, IndexLexical: false},
		},
	}
	var resp createIndexResponse
	if err := client.doJSON(ctx, http.MethodPost, "/api/v1/index", body, &resp); err != nil {
		return 0, err
	}
	var indexID int64
	switch {
	case resp.IndexID != nil:
		indexID = *resp.IndexID
	case resp.ID != nil:
		indexID = *resp.ID
	default:
		return 0, fmt.Errorf("seekstorm CreateIndex %q: response missing index_id/id", indexName)
	}
	core.Log("seekstorm.CreateIndex name::", indexName, " id::", indexID)
	return indexID, nil
}

// indexDocument is the row shape posted to /api/v1/index/{id}/doc. The
// product_id field doubles as the SeekStorm document key per §9, so updates
// are idempotent when the same productID is reposted.
type indexDocument struct {
	SearchText string `json:"search_text"`
	ProductID  int64  `json:"product_id"`
}

// PostIndexRecord upserts one product row into the lexical index identified
// by indexID. The caller is responsible for resolving client_id -> indexID
// from ScyllaDB; this function only talks to SeekStorm.
//
// SearchText should already be normalized (lower-case, single-space, no
// noise punctuation) — see §7.1 of the setup doc.
func PostIndexRecord(ctx context.Context, indexID int64, productID int64, searchText string) error {
	client, err := getClient()
	if err != nil {
		return err
	}
	path := fmt.Sprintf("/api/v1/index/%d/doc", indexID)
	return client.doJSON(ctx, http.MethodPost, path, indexDocument{
		SearchText: searchText,
		ProductID:  productID,
	}, nil)
}

// truncate keeps error messages bounded so a large SeekStorm body doesn't
// flood logs on a single failure.
func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
