package knowledge

import (
	"context"
	"reflect"
	"testing"

	"github.com/qdrant/go-client/qdrant"
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
	payload := qdrant.NewValueMap(map[string]any{
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
	})
	fakeClient := &fakeCollectionClient{queryResults: []*qdrant.ScoredPoint{{
		Id: qdrant.NewIDUUID(pointID), Payload: payload, Score: 0.75,
	}}}
	store := Store{client: fakeClient, config: QdrantConfig{Collection: "docs_v1", Dimensions: 3}}
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
	request := fakeClient.queryRequest
	if request == nil || len(request.GetPrefetch()) != 2 {
		t.Fatalf("expected dense and lexical prefetches: %+v", request)
	}
	if _, ok := request.GetQuery().GetVariant().(*qdrant.Query_Fusion); !ok || request.GetQuery().GetFusion() != qdrant.Fusion_RRF {
		t.Fatalf("expected native RRF query: %+v", request.GetQuery())
	}
	densePrefetch := request.GetPrefetch()[0]
	if densePrefetch.GetUsing() != DenseVectorName || !reflect.DeepEqual(densePrefetch.GetQuery().GetNearest().GetDense().GetData(), []float32{0.1, 0.2, 0.3}) {
		t.Fatalf("unexpected dense prefetch: %+v", densePrefetch)
	}
	lexicalPrefetch := request.GetPrefetch()[1]
	lexicalDocument := lexicalPrefetch.GetQuery().GetNearest().GetDocument()
	if lexicalPrefetch.GetUsing() != LexicalVectorName || lexicalDocument.GetText() != question || lexicalDocument.GetModel() != "qdrant/bm25" {
		t.Fatalf("unexpected lexical prefetch: %+v", lexicalPrefetch)
	}
	if len(densePrefetch.GetFilter().GetMust()) != 4 || densePrefetch.GetFilter() != lexicalPrefetch.GetFilter() {
		t.Fatalf("prefetch filters differ or are incomplete: %+v", densePrefetch.GetFilter())
	}
	if request.GetLimit() != 6 || request.GetWithVectors().GetEnable() {
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
	store := Store{client: &fakeCollectionClient{}, config: QdrantConfig{Collection: "docs_v1", Dimensions: 4096}}
	_, err := store.HybridSearch(context.Background(), "¿Cómo creo una caja?", []float32{0.1}, SearchOptions{})
	if err == nil {
		t.Fatal("wrong query embedding dimensions must fail before Qdrant")
	}
}
