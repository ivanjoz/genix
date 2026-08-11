package knowledge

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/qdrant/go-client/qdrant"
)

const (
	defaultCandidateLimit = 25
	defaultResultLimit    = 8
	maximumCandidateLimit = 100
	maximumResultLimit    = 20
	maximumAllowedRoutes  = 500
)

var searchPayloadFields = []string{
	"document_id",
	"route",
	"module",
	"title",
	"section_id",
	"section_title",
	"section_type",
	"content",
	"context_header",
	"keywords",
	"status",
	"visibility",
	"documentation_path",
	"documentation_hash",
	"content_hash",
	"source_hash_digest",
	"part_index",
	"part_count",
	"previous_point_id",
	"next_point_id",
	"indexed_at",
}

// SearchOptions bounds retrieval and optionally narrows it to the user's current context.
type SearchOptions struct {
	CandidateLimit uint64
	ResultLimit    uint64
	Route          string
	Module         string
	Visibility     string
	AllowedRoutes  []string
}

// DocumentationResult gives the answering agent evidence and an exact navigation target.
type DocumentationResult struct {
	PointID           string   `json:"point_id"`
	CitationID        string   `json:"citation_id"`
	DocumentID        string   `json:"document_id"`
	Route             string   `json:"route"`
	Module            string   `json:"module"`
	PageTitle         string   `json:"page_title"`
	SectionID         string   `json:"section_id"`
	SectionTitle      string   `json:"section_title"`
	SectionType       string   `json:"section_type"`
	Content           string   `json:"content"`
	ContextHeader     string   `json:"context_header"`
	Keywords          []string `json:"keywords,omitempty"`
	Status            string   `json:"status"`
	Visibility        string   `json:"visibility"`
	DocumentationPath string   `json:"documentation_path"`
	DocumentationHash string   `json:"documentation_hash"`
	ContentHash       string   `json:"content_hash"`
	SourceHashDigest  string   `json:"source_hash_digest"`
	PartIndex         int      `json:"part_index"`
	PartCount         int      `json:"part_count"`
	PreviousPointID   string   `json:"previous_point_id,omitempty"`
	NextPointID       string   `json:"next_point_id,omitempty"`
	IndexedAt         int64    `json:"indexed_at"`
	Score             float32  `json:"score"`
}

// QueryEmbedder applies the model-specific retrieval instruction to one user question.
type QueryEmbedder interface {
	EmbedQuery(context.Context, string) ([]float32, error)
}

// HybridPointSearcher keeps query embedding independent from Qdrant transport in tests.
type HybridPointSearcher interface {
	HybridSearch(context.Context, string, []float32, SearchOptions) ([]DocumentationResult, error)
}

// Retriever orchestrates query embedding and hybrid Qdrant search.
type Retriever struct {
	embedder QueryEmbedder
	searcher HybridPointSearcher
}

func NewRetriever(embedder QueryEmbedder, searcher HybridPointSearcher) (*Retriever, error) {
	if embedder == nil {
		return nil, errors.New("documentation query embedder is required")
	}
	if searcher == nil {
		return nil, errors.New("documentation point searcher is required")
	}
	return &Retriever{embedder: embedder, searcher: searcher}, nil
}

// SearchDocumentation preserves the user question for BM25 and embeds its instructed form.
func (retriever *Retriever) SearchDocumentation(ctx context.Context, userQuestion string, options SearchOptions) ([]DocumentationResult, error) {
	if strings.TrimSpace(userQuestion) == "" {
		return nil, errors.New("documentation search question is empty")
	}
	denseVector, err := retriever.embedder.EmbedQuery(ctx, userQuestion)
	if err != nil {
		return nil, fmt.Errorf("embed documentation query: %w", err)
	}
	return retriever.searcher.HybridSearch(ctx, userQuestion, denseVector, options)
}

// HybridSearch fuses dense and BM25 ranks in Qdrant; scores never cross retriever scales.
func (store *Store) HybridSearch(ctx context.Context, userQuestion string, denseVector []float32, options SearchOptions) ([]DocumentationResult, error) {
	normalizedQuestion := strings.TrimSpace(userQuestion)
	if normalizedQuestion == "" {
		return nil, errors.New("documentation search question is empty")
	}
	if len(denseVector) != store.config.Dimensions {
		return nil, fmt.Errorf("query embedding has %d dimensions, want %d", len(denseVector), store.config.Dimensions)
	}
	if err := normalizeSearchOptions(&options); err != nil {
		return nil, err
	}

	searchFilter := documentationSearchFilter(options)
	request := &qdrant.QueryPoints{
		CollectionName: store.config.Collection,
		Prefetch: []*qdrant.PrefetchQuery{
			{
				Query:  qdrant.NewQueryDense(denseVector),
				Using:  qdrant.PtrOf(DenseVectorName),
				Filter: searchFilter,
				Limit:  qdrant.PtrOf(options.CandidateLimit),
			},
			{
				Query: qdrant.NewQueryDocument(&qdrant.Document{
					Text:  normalizedQuestion,
					Model: "qdrant/bm25",
					Options: qdrant.NewValueMap(map[string]any{
						"language":      "none",
						"tokenizer":     "multilingual",
						"ascii_folding": true,
					}),
				}),
				Using:  qdrant.PtrOf(LexicalVectorName),
				Filter: searchFilter,
				Limit:  qdrant.PtrOf(options.CandidateLimit),
			},
		},
		Query:       qdrant.NewQueryFusion(qdrant.Fusion_RRF),
		Limit:       qdrant.PtrOf(options.ResultLimit),
		WithPayload: qdrant.NewWithPayloadInclude(searchPayloadFields...),
		WithVectors: qdrant.NewWithVectors(false),
	}

	startedAt := time.Now()
	points, err := store.client.Query(ctx, request)
	if err != nil {
		return nil, fmt.Errorf("hybrid documentation query: %w", err)
	}
	results := make([]DocumentationResult, 0, len(points))
	pointIDs := make([]string, 0, len(points))
	for _, point := range points {
		result := documentationResult(point)
		results = append(results, result)
		pointIDs = append(pointIDs, result.PointID)
	}
	questionDigest := sha256.Sum256([]byte(normalizedQuestion))
	log.Printf("[agent.knowledge] hybrid_search collection=%s query_hash=%x candidates=%d limit=%d results=%d route=%q allowed_routes=%d module=%q visibility=%q point_ids=%s duration=%s",
		store.config.Collection, questionDigest[:6], options.CandidateLimit, options.ResultLimit, len(results),
		options.Route, len(options.AllowedRoutes), options.Module, options.Visibility, strings.Join(pointIDs, ","), time.Since(startedAt).Round(time.Millisecond))
	return results, nil
}

func normalizeSearchOptions(options *SearchOptions) error {
	if options.CandidateLimit == 0 {
		options.CandidateLimit = defaultCandidateLimit
	}
	if options.ResultLimit == 0 {
		options.ResultLimit = defaultResultLimit
	}
	if options.CandidateLimit > maximumCandidateLimit {
		return fmt.Errorf("candidate limit %d exceeds maximum %d", options.CandidateLimit, maximumCandidateLimit)
	}
	if options.ResultLimit > maximumResultLimit {
		return fmt.Errorf("result limit %d exceeds maximum %d", options.ResultLimit, maximumResultLimit)
	}
	if options.CandidateLimit < options.ResultLimit {
		options.CandidateLimit = options.ResultLimit
	}
	options.Route = strings.TrimSpace(options.Route)
	options.Module = strings.TrimSpace(options.Module)
	options.Visibility = strings.TrimSpace(options.Visibility)
	uniqueRoutes := make([]string, 0, len(options.AllowedRoutes))
	seenRoutes := map[string]bool{}
	for _, route := range options.AllowedRoutes {
		route = strings.TrimSpace(route)
		if route != "" && !seenRoutes[route] {
			seenRoutes[route] = true
			uniqueRoutes = append(uniqueRoutes, route)
		}
	}
	if len(uniqueRoutes) > maximumAllowedRoutes {
		return fmt.Errorf("allowed route count %d exceeds maximum %d", len(uniqueRoutes), maximumAllowedRoutes)
	}
	options.AllowedRoutes = uniqueRoutes
	return nil
}

func documentationSearchFilter(options SearchOptions) *qdrant.Filter {
	conditions := []*qdrant.Condition{
		qdrant.NewMatchKeyword("status", "implemented"),
		qdrant.NewMatchBool("documentation_current", true),
	}
	if options.Route != "" {
		conditions = append(conditions, qdrant.NewMatchKeyword("route", options.Route))
	}
	if len(options.AllowedRoutes) > 0 {
		conditions = append(conditions, qdrant.NewMatchKeywords("route", options.AllowedRoutes...))
	}
	if options.Module != "" {
		conditions = append(conditions, qdrant.NewMatchKeyword("module", options.Module))
	}
	if options.Visibility != "" {
		conditions = append(conditions, qdrant.NewMatchKeyword("visibility", options.Visibility))
	}
	return &qdrant.Filter{Must: conditions}
}

func documentationResult(point *qdrant.ScoredPoint) DocumentationResult {
	payload := point.GetPayload()
	partIndex := payloadInteger(payload, "part_index")
	partCount := payloadInteger(payload, "part_count")
	documentID := payloadString(payload, "document_id")
	sectionID := payloadString(payload, "section_id")
	citationID := documentID + "#" + sectionID
	if partCount > 1 {
		citationID = fmt.Sprintf("%s:part-%d", citationID, partIndex+1)
	}
	return DocumentationResult{
		PointID:           point.GetId().GetUuid(),
		CitationID:        citationID,
		DocumentID:        documentID,
		Route:             payloadString(payload, "route"),
		Module:            payloadString(payload, "module"),
		PageTitle:         payloadString(payload, "title"),
		SectionID:         sectionID,
		SectionTitle:      payloadString(payload, "section_title"),
		SectionType:       payloadString(payload, "section_type"),
		Content:           payloadString(payload, "content"),
		ContextHeader:     payloadString(payload, "context_header"),
		Keywords:          payloadStrings(payload, "keywords"),
		Status:            payloadString(payload, "status"),
		Visibility:        payloadString(payload, "visibility"),
		DocumentationPath: payloadString(payload, "documentation_path"),
		DocumentationHash: payloadString(payload, "documentation_hash"),
		ContentHash:       payloadString(payload, "content_hash"),
		SourceHashDigest:  payloadString(payload, "source_hash_digest"),
		PartIndex:         partIndex,
		PartCount:         partCount,
		PreviousPointID:   payloadString(payload, "previous_point_id"),
		NextPointID:       payloadString(payload, "next_point_id"),
		IndexedAt:         int64(payloadInteger(payload, "indexed_at")),
		Score:             point.GetScore(),
	}
}

func payloadStrings(payload map[string]*qdrant.Value, key string) []string {
	list := payload[key].GetListValue()
	if list == nil {
		return nil
	}
	values := make([]string, 0, len(list.GetValues()))
	for _, value := range list.GetValues() {
		if text := value.GetStringValue(); text != "" {
			values = append(values, text)
		}
	}
	return values
}
