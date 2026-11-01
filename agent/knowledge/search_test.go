package knowledge

import (
	"context"
	"net/http"
	"reflect"
	"testing"
)

type fakeQueryEmbedder struct {
	question string
	vector   []float32
}

func (embedder *fakeQueryEmbedder) EmbedQuery(_ context.Context, question string) ([]float32, error) {
	embedder.question = question
	return embedder.vector, nil
}

type fakeHybridSearcher struct {
	question string
	vector   []float32
	options  SearchOptions
	results  []DocumentationResult
}

func (searcher *fakeHybridSearcher) HybridSearch(_ context.Context, question string, vector []float32, options SearchOptions) ([]DocumentationResult, error) {
	searcher.question = question
	searcher.vector = vector
	searcher.options = options
	return searcher.results, nil
}

func TestRetrieverEmbedsAndSearchesOriginalSpanishQuestion(t *testing.T) {
	question := "¿Dónde hago el arqueo de caja?"
	embedder := &fakeQueryEmbedder{vector: []float32{0.1, 0.2, 0.3}}
	searcher := &fakeHybridSearcher{results: []DocumentationResult{{Route: "/finance/cash-banks"}}}
	retriever, err := NewRetriever(embedder, searcher)
	if err != nil {
		t.Fatal(err)
	}

	results, err := retriever.SearchDocumentation(context.Background(), question, SearchOptions{Module: "Finance"})
	if err != nil {
		t.Fatal(err)
	}
	if embedder.question != question || searcher.question != question {
		t.Fatalf("question was changed: embed=%q search=%q", embedder.question, searcher.question)
	}
	if !reflect.DeepEqual(searcher.vector, embedder.vector) || searcher.options.Module != "Finance" {
		t.Fatalf("retrieval inputs were not forwarded: %+v", searcher)
	}
	if len(results) != 1 || results[0].Route != "/finance/cash-banks" {
		t.Fatalf("unexpected results: %+v", results)
	}
}

func TestHybridSearchBuildsDenseBM25RRFRequestAndDecodesNavigation(t *testing.T) {
	pointID := "243fbf6f-0777-5f99-9d7a-dbbabc3ab32b"
	payload := map[string]any{
		"document_id":        "finance.cash-banks",
		"route":              "/finance/cash-banks",
		"module":             "Finance",
		"title":              "Cash & Banks (Cajas y Bancos)",
		"section_id":         "capability.reconcile",
		"section_title":      "Reconcile cash (Realizar arqueo)",
		"section_type":       "capability",
		"content":            "Use el arqueo para comparar el efectivo observado.",
		"context_header":     "Page: Cash & Banks\nRoute: /finance/cash-banks",
		"keywords":           []any{"arqueo", "cuadre"},
		"status":             "implemented",
		"visibility":         "tenant",
		"documentation_path": "frontend/routes/finance/cash-banks/DOCUMENTATION.md",
		"documentation_hash": "sha256:document",
		"content_hash":       "sha256:content",
		"source_hash_digest": "sha256:sources",
		"part_index":         int64(0),
		"part_count":         int64(1),
		"previous_point_id":  "",
		"next_point_id":      "",
		"indexed_at":         int64(123),
	}
	store, fake := newFakeQdrantStore(t, QdrantConfig{Collection: "docs_v1", Dimensions: 3}, map[string]any{
		"POST /points/query": map[string]any{"points": []any{map[string]any{
			"id": pointID, "version": 1, "score": 0.75, "payload": payload,
		}}},
	})
	question := "¿Dónde hago el arqueo de caja?"

	results, err := store.HybridSearch(context.Background(), question, []float32{0.1, 0.2, 0.3}, SearchOptions{
		CandidateLimit: 25,
		ResultLimit:    6,
		Module:         "Finance",
		Visibility:     "tenant",
	})
	if err != nil {
		t.Fatal(err)
	}
	request := fake.lastRequest(t, http.MethodPost, "/points/query").body
	if prefetches, _ := request["prefetch"].([]any); len(prefetches) != 2 {
		t.Fatalf("expected dense and lexical prefetches: %+v", request)
	}
	if jsonPath(request, "query", "fusion") != "rrf" {
		t.Fatalf("expected native RRF query: %+v", request["query"])
	}
	densePrefetch := jsonPath(request, "prefetch", 0)
	if jsonPath(densePrefetch, "using") != DenseVectorName || !reflect.DeepEqual(jsonPath(densePrefetch, "query"), []any{0.1, 0.2, 0.3}) {
		t.Fatalf("unexpected dense prefetch: %+v", densePrefetch)
	}
	lexicalPrefetch := jsonPath(request, "prefetch", 1)
	if jsonPath(lexicalPrefetch, "using") != LexicalVectorName || jsonPath(lexicalPrefetch, "query", "text") != question ||
		jsonPath(lexicalPrefetch, "query", "model") != "qdrant/bm25" {
		t.Fatalf("unexpected lexical prefetch: %+v", lexicalPrefetch)
	}
	denseFilter := jsonPath(densePrefetch, "filter", "must")
	if filterConditions, _ := denseFilter.([]any); len(filterConditions) != 4 || !reflect.DeepEqual(denseFilter, jsonPath(lexicalPrefetch, "filter", "must")) {
		t.Fatalf("prefetch filters differ or are incomplete: %+v", denseFilter)
	}
	if jsonPath(densePrefetch, "filter", "must", 1, "match", "value") != true {
		t.Fatalf("documentation_current filter must send an explicit true: %+v", denseFilter)
	}
	if request["limit"] != 6.0 || request["with_vector"] != false {
		t.Fatalf("unexpected result request: %+v", request)
	}

	if len(results) != 1 {
		t.Fatalf("unexpected result count: %d", len(results))
	}
	result := results[0]
	if result.PointID != pointID || result.Route != "/finance/cash-banks" || result.CitationID != "finance.cash-banks#capability.reconcile" {
		t.Fatalf("navigation metadata was not decoded: %+v", result)
	}
	if !reflect.DeepEqual(result.Keywords, []string{"arqueo", "cuadre"}) || result.Score != 0.75 {
		t.Fatalf("retrieval metadata was not decoded: %+v", result)
	}
}

func TestHybridSearchRejectsWrongEmbeddingDimension(t *testing.T) {
	store, fake := newFakeQdrantStore(t, QdrantConfig{Collection: "docs_v1", Dimensions: 4096}, nil)
	_, err := store.HybridSearch(context.Background(), "¿Cómo creo una caja?", []float32{0.1}, SearchOptions{})
	if err == nil || len(fake.requests) != 0 {
		t.Fatal("wrong query embedding dimensions must fail before Qdrant")
	}
}

func TestDocumentationSearchFilterUsesAllowedRouteMatchAny(t *testing.T) {
	options := SearchOptions{AllowedRoutes: []string{" /finance/cash-banks ", "/logistics/purchase-orders", "/finance/cash-banks"}}
	if err := normalizeSearchOptions(&options); err != nil {
		t.Fatal(err)
	}
	if len(options.AllowedRoutes) != 2 {
		t.Fatalf("allowed routes were not normalized: %+v", options.AllowedRoutes)
	}
	filter := documentationSearchFilter(options)
	if len(filter.Must) != 3 {
		t.Fatalf("expected status, current, and route filters: %+v", filter)
	}
	keywords := filter.Must[2].Match.Any
	if !reflect.DeepEqual(keywords, options.AllowedRoutes) {
		t.Fatalf("unexpected route match-any values: %+v", keywords)
	}
}
