package knowledge

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

const testAPIKey = "test-api-key"

// recordedRequest is one call the fake Qdrant received, with its JSON body decoded generically
// so tests assert on the exact wire shape Qdrant will see.
type recordedRequest struct {
	method string
	path   string
	body   map[string]any
}

// fakeQdrant answers REST calls from canned results keyed by "METHOD /path" (query string included).
type fakeQdrant struct {
	results  map[string]any
	requests []recordedRequest
}

func newFakeQdrantStore(t *testing.T, config QdrantConfig, results map[string]any) (*Store, *fakeQdrant) {
	fake := &fakeQdrant{results: results}
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Header.Get("api-key") != testAPIKey {
			t.Errorf("%s %s sent api-key %q", request.Method, request.URL, request.Header.Get("api-key"))
		}
		requestPath := strings.TrimPrefix(request.URL.RequestURI(), "/collections/"+config.Collection)
		recorded := recordedRequest{method: request.Method, path: requestPath}
		if requestBytes, _ := io.ReadAll(request.Body); len(requestBytes) > 0 {
			if err := json.Unmarshal(requestBytes, &recorded.body); err != nil {
				t.Errorf("%s %s sent invalid JSON: %v", request.Method, requestPath, err)
			}
		}
		fake.requests = append(fake.requests, recorded)
		json.NewEncoder(writer).Encode(map[string]any{"status": "ok", "result": fake.results[request.Method+" "+requestPath]})
	}))
	t.Cleanup(server.Close)
	config.APIKey = testAPIKey
	return &Store{httpClient: server.Client(), baseURL: server.URL, config: config}, fake
}

// lastRequest returns the most recent call to method+path, failing the test when there was none.
func (fake *fakeQdrant) lastRequest(t *testing.T, method, path string) recordedRequest {
	t.Helper()
	for index := len(fake.requests) - 1; index >= 0; index-- {
		if fake.requests[index].method == method && fake.requests[index].path == path {
			return fake.requests[index]
		}
	}
	t.Fatalf("no %s %s request was sent; got %+v", method, path, fake.requests)
	return recordedRequest{}
}

// jsonPath walks nested maps and arrays: jsonPath(body, "prefetch", 0, "using").
func jsonPath(value any, keys ...any) any {
	for _, key := range keys {
		switch typedKey := key.(type) {
		case string:
			object, _ := value.(map[string]any)
			value = object[typedKey]
		case int:
			list, _ := value.([]any)
			if typedKey >= len(list) {
				return nil
			}
			value = list[typedKey]
		}
	}
	return value
}

func TestEnsureCollectionCreatesDenseAndBM25Schema(t *testing.T) {
	store, fake := newFakeQdrantStore(t, QdrantConfig{Collection: "docs_v1", Dimensions: 4096}, map[string]any{
		"GET /exists": map[string]any{"exists": false},
	})

	if err := store.EnsureCollection(context.Background()); err != nil {
		t.Fatal(err)
	}
	created := fake.lastRequest(t, http.MethodPut, "").body
	if jsonPath(created, "vectors", DenseVectorName, "size") != 4096.0 || jsonPath(created, "vectors", DenseVectorName, "distance") != "Cosine" {
		t.Fatalf("unexpected dense config: %+v", created["vectors"])
	}
	if jsonPath(created, "sparse_vectors", LexicalVectorName, "modifier") != "idf" {
		t.Fatalf("lexical modifier = %v, want idf", jsonPath(created, "sparse_vectors", LexicalVectorName, "modifier"))
	}
	if jsonPath(created, "metadata", "schema_version") != float64(CollectionSchema) {
		t.Fatalf("unexpected collection metadata: %+v", created["metadata"])
	}

	indexRequests := []recordedRequest{}
	for _, request := range fake.requests {
		if request.method == http.MethodPut && request.path == "/index?wait=true" {
			indexRequests = append(indexRequests, request)
		}
	}
	if len(indexRequests) != len(keywordPayloadFields)+len(booleanPayloadFields)+1 {
		t.Fatalf("created %d payload indexes, want %d", len(indexRequests), len(keywordPayloadFields)+len(booleanPayloadFields)+1)
	}
	if jsonPath(indexRequests[0].body, "field_schema") != "keyword" {
		t.Fatalf("unexpected keyword index: %+v", indexRequests[0].body)
	}
	contentIndex := indexRequests[len(indexRequests)-1].body
	if contentIndex["field_name"] != "content" || jsonPath(contentIndex, "field_schema", "type") != "text" ||
		jsonPath(contentIndex, "field_schema", "tokenizer") != "multilingual" ||
		jsonPath(contentIndex, "field_schema", "lowercase") != true ||
		jsonPath(contentIndex, "field_schema", "phrase_matching") != true ||
		jsonPath(contentIndex, "field_schema", "ascii_folding") != true {
		t.Fatalf("unexpected content index: %+v", contentIndex)
	}
}

func TestEnsureCollectionRejectsWrongPayloadIndexType(t *testing.T) {
	store, _ := newFakeQdrantStore(t, QdrantConfig{Collection: "docs_v1", Dimensions: 4096}, map[string]any{
		"GET /exists": map[string]any{"exists": true},
		"GET ": map[string]any{
			"config": map[string]any{"params": collectionRequest(QdrantConfig{Dimensions: 4096})},
			"payload_schema": map[string]any{"route": map[string]any{"data_type": "text"}},
		},
	})

	err := store.EnsureCollection(context.Background())
	if err == nil || !strings.Contains(err.Error(), `"route" is text, want keyword`) {
		t.Fatalf("expected payload type mismatch, got %v", err)
	}
}

func TestValidateCollectionSchemaRejectsDimensionMismatch(t *testing.T) {
	createdRequest := collectionRequest(QdrantConfig{Collection: "docs_v1", Dimensions: 2048})
	collection := collectionInfo{}
	collection.Config.Params.Vectors = createdRequest.Vectors
	collection.Config.Params.SparseVectors = createdRequest.SparseVectors

	err := validateCollectionSchema(collection, 4096)
	if err == nil || !strings.Contains(err.Error(), "2048 dimensions, want 4096") {
		t.Fatalf("expected dimension mismatch, got %v", err)
	}
}

func TestQdrantErrorStatusIsReported(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.WriteHeader(http.StatusNotFound)
		writer.Write([]byte(`{"status":{"error":"Collection docs_v1 doesn't exist"},"time":0}`))
	}))
	defer server.Close()
	store := &Store{httpClient: server.Client(), baseURL: server.URL, config: QdrantConfig{Collection: "docs_v1", Dimensions: 3}}

	err := store.ValidateExistingCollection(context.Background())
	if err == nil || !strings.Contains(err.Error(), "HTTP 404: Collection docs_v1 doesn't exist") {
		t.Fatalf("expected Qdrant error message, got %v", err)
	}
}
