package knowledge

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"
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
	PointID           string   `json:"point_id" cb:"point_id,1"`
	CitationID        string   `json:"citation_id" cb:"citation_id,2"`
	DocumentID        string   `json:"document_id" cb:"document_id,3"`
	Route             string   `json:"route" cb:"route,4"`
	Module            string   `json:"module" cb:"module,5"`
	PageTitle         string   `json:"page_title" cb:"page_title,6"`
	SectionID         string   `json:"section_id" cb:"section_id,7"`
	SectionTitle      string   `json:"section_title" cb:"section_title,8"`
	SectionType       string   `json:"section_type" cb:"section_type,9"`
	Content           string   `json:"content" cb:"content,10"`
	ContextHeader     string   `json:"context_header" cb:"context_header,11"`
	Keywords          []string `json:"keywords" cb:"keywords,12"`
	Status            string   `json:"status" cb:"status,13"`
	Visibility        string   `json:"visibility" cb:"visibility,14"`
	DocumentationPath string   `json:"documentation_path" cb:"documentation_path,15"`
	DocumentationHash string   `json:"documentation_hash" cb:"documentation_hash,16"`
	ContentHash       string   `json:"content_hash" cb:"content_hash,17"`
	SourceHashDigest  string   `json:"source_hash_digest" cb:"source_hash_digest,18"`
	PartIndex         int      `json:"part_index" cb:"part_index,19"`
	PartCount         int      `json:"part_count" cb:"part_count,20"`
	PreviousPointID   string   `json:"previous_point_id" cb:"previous_point_id,21"`
	NextPointID       string   `json:"next_point_id" cb:"next_point_id,22"`
	IndexedAt         int64    `json:"indexed_at" cb:"indexed_at,23"`
	Score             float32  `json:"score" cb:"score,24"`
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
	request := hybridQueryRequest{
		Prefetch: []prefetchQuery{
			{Query: denseVector, Using: DenseVectorName, Filter: searchFilter, Limit: options.CandidateLimit},
			{
				Query:  bm25Document{Text: normalizedQuestion, Model: bm25Model, Options: bm25DocumentOptions},
				Using:  LexicalVectorName,
				Filter: searchFilter,
				Limit:  options.CandidateLimit,
			},
		},
		Query:       fusionQuery{Fusion: "rrf"},
		Limit:       options.ResultLimit,
		WithPayload: searchPayloadFields,
	}

	startedAt := time.Now()
	queryResult := struct {
		Points []scoredPoint `json:"points"`
	}{}
	if err := store.callQdrant(ctx, http.MethodPost, "/points/query", request, &queryResult); err != nil {
		return nil, fmt.Errorf("hybrid documentation query: %w", err)
	}
	results := make([]DocumentationResult, 0, len(queryResult.Points))
	pointIDs := make([]string, 0, len(queryResult.Points))
	for _, point := range queryResult.Points {
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

func documentationSearchFilter(options SearchOptions) pointFilter {
	conditions := []fieldCondition{
		{Key: "status", Match: matchCondition{Value: "implemented"}},
		{Key: "documentation_current", Match: matchCondition{Value: true}},
	}
	if options.Route != "" {
		conditions = append(conditions, fieldCondition{Key: "route", Match: matchCondition{Value: options.Route}})
	}
	if len(options.AllowedRoutes) > 0 {
		conditions = append(conditions, fieldCondition{Key: "route", Match: matchCondition{Any: options.AllowedRoutes}})
	}
	if options.Module != "" {
		conditions = append(conditions, fieldCondition{Key: "module", Match: matchCondition{Value: options.Module}})
	}
	if options.Visibility != "" {
		conditions = append(conditions, fieldCondition{Key: "visibility", Match: matchCondition{Value: options.Visibility}})
	}
	return pointFilter{Must: conditions}
}

func documentationResult(point scoredPoint) DocumentationResult {
	payload := point.Payload
	citationID := payload.DocumentID + "#" + payload.SectionID
	if payload.PartCount > 1 {
		citationID = fmt.Sprintf("%s:part-%d", citationID, payload.PartIndex+1)
	}
	return DocumentationResult{
		PointID:           point.ID,
		CitationID:        citationID,
		DocumentID:        payload.DocumentID,
		Route:             payload.Route,
		Module:            payload.Module,
		PageTitle:         payload.Title,
		SectionID:         payload.SectionID,
		SectionTitle:      payload.SectionTitle,
		SectionType:       payload.SectionType,
		Content:           payload.Content,
		ContextHeader:     payload.ContextHeader,
		Keywords:          payload.Keywords,
		Status:            payload.Status,
		Visibility:        payload.Visibility,
		DocumentationPath: payload.DocumentationPath,
		DocumentationHash: payload.DocumentationHash,
		ContentHash:       payload.ContentHash,
		SourceHashDigest:  payload.SourceHashDigest,
		PartIndex:         payload.PartIndex,
		PartCount:         payload.PartCount,
		PreviousPointID:   payload.PreviousPointID,
		NextPointID:       payload.NextPointID,
		IndexedAt:         payload.IndexedAt,
		Score:             point.Score,
	}
}

// prefetchQuery.Query is a dense vector ([]float32) or a bm25Document for server-side inference.
type prefetchQuery struct {
	Query  any         `json:"query"`
	Using  string      `json:"using"`
	Filter pointFilter `json:"filter"`
	Limit  uint64      `json:"limit"`
}

type fusionQuery struct {
	Fusion string `json:"fusion"`
}

type hybridQueryRequest struct {
	Prefetch    []prefetchQuery `json:"prefetch"`
	Query       fusionQuery     `json:"query"`
	Limit       uint64          `json:"limit"`
	WithPayload []string        `json:"with_payload"`
	WithVector  bool            `json:"with_vector"`
}

// documentationPayload mirrors searchPayloadFields as written by the ragdocs indexer.
type documentationPayload struct {
	DocumentID        string   `json:"document_id"`
	Route             string   `json:"route"`
	Module            string   `json:"module"`
	Title             string   `json:"title"`
	SectionID         string   `json:"section_id"`
	SectionTitle      string   `json:"section_title"`
	SectionType       string   `json:"section_type"`
	Content           string   `json:"content"`
	ContextHeader     string   `json:"context_header"`
	Keywords          []string `json:"keywords"`
	Status            string   `json:"status"`
	Visibility        string   `json:"visibility"`
	DocumentationPath string   `json:"documentation_path"`
	DocumentationHash string   `json:"documentation_hash"`
	ContentHash       string   `json:"content_hash"`
	SourceHashDigest  string   `json:"source_hash_digest"`
	PartIndex         int      `json:"part_index"`
	PartCount         int      `json:"part_count"`
	PreviousPointID   string   `json:"previous_point_id"`
	NextPointID       string   `json:"next_point_id"`
	IndexedAt         int64    `json:"indexed_at"`
}

type scoredPoint struct {
	ID      string               `json:"id"`
	Score   float32              `json:"score"`
	Payload documentationPayload `json:"payload"`
}
