package knowledge

import (
	"context"
	"net/http"
	"testing"
)

func TestUpsertDocumentationPointUsesDenseAndServerBM25(t *testing.T) {
	store, fake := newFakeQdrantStore(t, QdrantConfig{Collection: "docs_v1"}, nil)

	err := store.UpsertDocumentationPoints(context.Background(), []IndexPoint{{
		ID:          "243fbf6f-0777-5f99-9d7a-dbbabc3ab32b",
		DenseVector: []float32{0.1, 0.2, 0.3},
		LexicalText: "cuadre de caja",
		Payload:     map[string]any{"document_id": "finance.cash-banks"},
	}})
	if err != nil {
		t.Fatal(err)
	}
	// wait=true makes the upsert synchronous.
	point := jsonPath(fake.lastRequest(t, http.MethodPut, "/points?wait=true").body, "points", 0)
	if jsonPath(point, "id") != "243fbf6f-0777-5f99-9d7a-dbbabc3ab32b" || jsonPath(point, "payload", "document_id") != "finance.cash-banks" {
		t.Fatalf("unexpected point: %+v", point)
	}
	if denseVector, _ := jsonPath(point, "vector", DenseVectorName).([]any); len(denseVector) != 3 {
		t.Fatal("dense vector was not included")
	}
	lexicalDocument := jsonPath(point, "vector", LexicalVectorName)
	if jsonPath(lexicalDocument, "model") != "qdrant/bm25" || jsonPath(lexicalDocument, "text") != "cuadre de caja" {
		t.Fatalf("unexpected lexical document: %+v", lexicalDocument)
	}
	if jsonPath(lexicalDocument, "options", "language") != "none" || jsonPath(lexicalDocument, "options", "tokenizer") != "multilingual" {
		t.Fatalf("unexpected BM25 options: %+v", jsonPath(lexicalDocument, "options"))
	}
}

func TestReplacePayloadDoesNotSendVectors(t *testing.T) {
	store, fake := newFakeQdrantStore(t, QdrantConfig{Collection: "docs_v1"}, nil)

	if err := store.ReplacePointPayload(context.Background(), "243fbf6f-0777-5f99-9d7a-dbbabc3ab32b", map[string]any{"content_hash": "sha256:test"}); err != nil {
		t.Fatal(err)
	}
	if len(fake.requests) != 1 {
		t.Fatalf("expected one payload-only request, got %+v", fake.requests)
	}
	// PUT (not POST) is the overwrite, and wait=true makes it synchronous.
	body := fake.lastRequest(t, http.MethodPut, "/points/payload?wait=true").body
	if jsonPath(body, "points", 0) != "243fbf6f-0777-5f99-9d7a-dbbabc3ab32b" || jsonPath(body, "payload", "content_hash") != "sha256:test" || body["vector"] != nil {
		t.Fatalf("unexpected payload request: %+v", body)
	}
}

func TestListDocumentPointsDecodesPayloadOnlyScroll(t *testing.T) {
	store, fake := newFakeQdrantStore(t, QdrantConfig{Collection: "docs_v1"}, map[string]any{
		"POST /points/scroll": map[string]any{
			"points": []any{map[string]any{
				"id":      "243fbf6f-0777-5f99-9d7a-dbbabc3ab32b",
				"payload": map[string]any{"content_hash": "sha256:content", "document_chunk_count": 4},
			}},
			"next_page_offset": nil,
		},
	})

	points, err := store.ListDocumentPoints(context.Background(), "frontend/routes/finance/cash-banks/DOCUMENTATION.md")
	if err != nil {
		t.Fatal(err)
	}
	if len(points) != 1 || points[0].ID != "243fbf6f-0777-5f99-9d7a-dbbabc3ab32b" || points[0].ContentHash != "sha256:content" || points[0].DocumentChunkCount != 4 {
		t.Fatalf("unexpected points: %+v", points)
	}
	body := fake.lastRequest(t, http.MethodPost, "/points/scroll").body
	if jsonPath(body, "filter", "must", 0, "key") != "documentation_path" || body["with_vector"] != false {
		t.Fatalf("unexpected scroll request: %+v", body)
	}
}
