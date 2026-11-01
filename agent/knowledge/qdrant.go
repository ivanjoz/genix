package knowledge

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"app/core"
)

const (
	DenseVectorName   = "dense"
	LexicalVectorName = "lexical"
	CollectionSchema  = 1
	bm25Model         = "qdrant/bm25"
)

var keywordPayloadFields = []string{
	"point_type",
	"document_id",
	"route",
	"module",
	"section_type",
	"status",
	"visibility",
	"access_resources",
	"documentation_path",
	"content_hash",
}

var booleanPayloadFields = []string{"documentation_current"}

// bm25DocumentOptions must match between indexing and querying, or the sparse vectors stop
// sharing a vocabulary and lexical recall silently collapses.
var bm25DocumentOptions = map[string]any{
	"language":      "none",
	"tokenizer":     "multilingual",
	"ascii_folding": true,
}

// QdrantConfig is explicit so tests and administrative commands do not depend on globals.
type QdrantConfig struct {
	Host       string
	HTTPPort   int
	APIKey     string
	UseTLS     bool
	Public     bool
	Collection string
	Dimensions int
}

// Store talks to Qdrant's REST API and validates the versioned RAG collection.
type Store struct {
	httpClient *http.Client
	baseURL    string
	config     QdrantConfig
}

func ConfigFromEnv() (QdrantConfig, error) {
	if core.Env == nil {
		return QdrantConfig{}, errors.New("core.Env is not initialized")
	}
	configured := QdrantConfig{
		Host:       core.Env.QDRANT_HOST,
		HTTPPort:   core.Env.QDRANT_HTTP_PORT,
		APIKey:     core.Env.INTERNAL_APIKEY,
		UseTLS:     core.Env.QDRANT_USE_TLS,
		Public:     core.Env.QDRANT_PUBLIC,
		Collection: core.Env.QDRANT_COLLECTION,
		Dimensions: core.Env.EMBEDDING_DIMENSIONS,
	}
	return configured, configured.validate()
}

func (config QdrantConfig) validate() error {
	if strings.TrimSpace(config.Host) == "" {
		return errors.New("qdrant.host is required when the RAG subsystem starts")
	}
	if config.HTTPPort <= 0 || config.HTTPPort > 65535 {
		return fmt.Errorf("qdrant.http_port %d is invalid", config.HTTPPort)
	}
	if strings.TrimSpace(config.APIKey) == "" {
		return errors.New("internal_apikey is required by the Qdrant client")
	}
	if strings.TrimSpace(config.Collection) == "" {
		return errors.New("qdrant.collection is required")
	}
	if config.Dimensions <= 0 {
		return errors.New("embedding_model.dimensions must be greater than zero")
	}
	return nil
}

func NewStore(config QdrantConfig) (*Store, error) {
	if err := config.validate(); err != nil {
		return nil, err
	}
	scheme := "http"
	if config.UseTLS {
		scheme = "https"
	} else if config.Public {
		log.Printf("[agent.knowledge] warning: public Qdrant target %s:%d is configured without TLS", config.Host, config.HTTPPort)
	}
	log.Printf("[agent.knowledge] qdrant_client_ready host=%s port=%d collection=%s tls=%t",
		config.Host, config.HTTPPort, config.Collection, config.UseTLS)
	return &Store{
		httpClient: &http.Client{Timeout: 60 * time.Second},
		baseURL:    fmt.Sprintf("%s://%s:%d", scheme, config.Host, config.HTTPPort),
		config:     config,
	}, nil
}

func NewStoreFromEnv() (*Store, error) {
	config, err := ConfigFromEnv()
	if err != nil {
		return nil, err
	}
	return NewStore(config)
}

// callQdrant sends one REST request and decodes the "result" field of Qdrant's response envelope.
// path is relative to the collection; result may be nil when only success matters.
func (store *Store) callQdrant(ctx context.Context, method, path string, requestBody, result any) error {
	var bodyReader io.Reader
	if requestBody != nil {
		encodedBody, err := json.Marshal(requestBody)
		if err != nil {
			return fmt.Errorf("encode Qdrant request: %w", err)
		}
		bodyReader = bytes.NewReader(encodedBody)
	}
	request, err := http.NewRequestWithContext(ctx, method, store.baseURL+"/collections/"+store.config.Collection+path, bodyReader)
	if err != nil {
		return err
	}
	request.Header.Set("api-key", store.config.APIKey)
	if requestBody != nil {
		request.Header.Set("Content-Type", "application/json")
	}

	response, err := store.httpClient.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	responseBytes, err := io.ReadAll(response.Body)
	if err != nil {
		return fmt.Errorf("read Qdrant response: %w", err)
	}

	envelope := struct {
		Result json.RawMessage `json:"result"`
		Status json.RawMessage `json:"status"`
	}{}
	if response.StatusCode < 200 || response.StatusCode > 299 {
		// Errors come back as {"status":{"error":"..."}}; fall back to the raw body when they don't.
		errorStatus := struct {
			Error string `json:"error"`
		}{}
		if json.Unmarshal(responseBytes, &envelope) == nil && json.Unmarshal(envelope.Status, &errorStatus) == nil && errorStatus.Error != "" {
			return fmt.Errorf("Qdrant %s %s: HTTP %d: %s", method, path, response.StatusCode, errorStatus.Error)
		}
		return fmt.Errorf("Qdrant %s %s: HTTP %d: %s", method, path, response.StatusCode, strings.TrimSpace(string(responseBytes)))
	}
	if result == nil {
		return nil
	}
	if err := json.Unmarshal(responseBytes, &envelope); err != nil {
		return fmt.Errorf("decode Qdrant response envelope: %w", err)
	}
	if err := json.Unmarshal(envelope.Result, result); err != nil {
		return fmt.Errorf("decode Qdrant %s %s result: %w", method, path, err)
	}
	return nil
}

func (store *Store) CollectionExists(ctx context.Context) (bool, error) {
	existsResult := struct {
		Exists bool `json:"exists"`
	}{}
	if err := store.callQdrant(ctx, http.MethodGet, "/exists", nil, &existsResult); err != nil {
		return false, fmt.Errorf("check Qdrant collection %q: %w", store.config.Collection, err)
	}
	return existsResult.Exists, nil
}

// ValidateExistingCollection is read-only and is used by dry-run before any proposed writes.
func (store *Store) ValidateExistingCollection(ctx context.Context) error {
	_, err := store.getValidatedCollection(ctx)
	return err
}

// EnsureCollection creates the versioned schema once and rejects incompatible reuse.
func (store *Store) EnsureCollection(ctx context.Context) error {
	exists, err := store.CollectionExists(ctx)
	if err != nil {
		return err
	}
	if !exists {
		if err := store.callQdrant(ctx, http.MethodPut, "", collectionRequest(store.config), nil); err != nil {
			return fmt.Errorf("create Qdrant collection %q: %w", store.config.Collection, err)
		}
		log.Printf("[agent.knowledge] collection_created name=%s dimensions=%d schema=%d",
			store.config.Collection, store.config.Dimensions, CollectionSchema)
		return store.ensurePayloadIndexes(ctx, nil)
	}

	collection, err := store.getValidatedCollection(ctx)
	if err != nil {
		return err
	}
	log.Printf("[agent.knowledge] collection_valid name=%s dimensions=%d schema=%d",
		store.config.Collection, store.config.Dimensions, CollectionSchema)
	return store.ensurePayloadIndexes(ctx, collection.PayloadSchema)
}

func (store *Store) getValidatedCollection(ctx context.Context) (collectionInfo, error) {
	collection := collectionInfo{}
	if err := store.callQdrant(ctx, http.MethodGet, "", nil, &collection); err != nil {
		return collection, fmt.Errorf("inspect Qdrant collection %q: %w", store.config.Collection, err)
	}
	if err := validateCollectionSchema(collection, store.config.Dimensions); err != nil {
		return collection, fmt.Errorf("Qdrant collection %q is incompatible: %w", store.config.Collection, err)
	}
	return collection, nil
}

// ensurePayloadIndexes creates every missing index and rejects an existing one of the wrong type.
func (store *Store) ensurePayloadIndexes(ctx context.Context, existingSchema map[string]payloadSchemaInfo) error {
	requiredIndexes := []payloadIndexRequest{}
	for _, fieldName := range keywordPayloadFields {
		requiredIndexes = append(requiredIndexes, payloadIndexRequest{FieldName: fieldName, FieldSchema: "keyword"})
	}
	for _, fieldName := range booleanPayloadFields {
		requiredIndexes = append(requiredIndexes, payloadIndexRequest{FieldName: fieldName, FieldSchema: "bool"})
	}
	requiredIndexes = append(requiredIndexes, contentIndexRequest())

	for _, indexRequest := range requiredIndexes {
		dataType := indexDataType(indexRequest)
		if existing, exists := existingSchema[indexRequest.FieldName]; exists {
			if existing.DataType != dataType {
				return fmt.Errorf("payload field %q is %s, want %s", indexRequest.FieldName, existing.DataType, dataType)
			}
			continue
		}
		if err := store.callQdrant(ctx, http.MethodPut, "/index?wait=true", indexRequest, nil); err != nil {
			return fmt.Errorf("create %s payload index %q: %w", dataType, indexRequest.FieldName, err)
		}
		log.Printf("[agent.knowledge] payload_index_created collection=%s field=%s type=%s",
			store.config.Collection, indexRequest.FieldName, dataType)
	}
	return nil
}

func indexDataType(indexRequest payloadIndexRequest) string {
	if textSchema, isText := indexRequest.FieldSchema.(textIndexSchema); isText {
		return textSchema.Type
	}
	return indexRequest.FieldSchema.(string)
}

func contentIndexRequest() payloadIndexRequest {
	return payloadIndexRequest{
		FieldName: "content",
		FieldSchema: textIndexSchema{
			Type:           "text",
			Tokenizer:      "multilingual",
			Lowercase:      true,
			PhraseMatching: true,
			AsciiFolding:   true,
		},
	}
}

func collectionRequest(config QdrantConfig) createCollectionRequest {
	return createCollectionRequest{
		Vectors: map[string]denseVectorParams{
			DenseVectorName: {Size: config.Dimensions, Distance: "Cosine"},
		},
		SparseVectors: map[string]sparseVectorParams{
			// Server-side qdrant/bm25 supplies TF/length weights; Qdrant applies corpus IDF.
			LexicalVectorName: {Modifier: "idf"},
		},
		Metadata: map[string]any{"schema_version": CollectionSchema},
	}
}

func validateCollectionSchema(collection collectionInfo, expectedDimensions int) error {
	params := collection.Config.Params
	denseConfig, exists := params.Vectors[DenseVectorName]
	if !exists {
		return fmt.Errorf("missing named vector %q", DenseVectorName)
	}
	if denseConfig.Size != expectedDimensions {
		return fmt.Errorf("dense vector has %d dimensions, want %d", denseConfig.Size, expectedDimensions)
	}
	if denseConfig.Distance != "Cosine" {
		return fmt.Errorf("dense vector uses %s distance, want Cosine", denseConfig.Distance)
	}

	lexicalConfig, exists := params.SparseVectors[LexicalVectorName]
	if !exists {
		return fmt.Errorf("missing sparse vector %q", LexicalVectorName)
	}
	if lexicalConfig.Modifier != "idf" {
		return fmt.Errorf("sparse vector %q must use the IDF modifier", LexicalVectorName)
	}
	return nil
}

// Qdrant REST wire types. Only the fields this package sends or reads are declared.

type denseVectorParams struct {
	Size     int    `json:"size"`
	Distance string `json:"distance"`
}

type sparseVectorParams struct {
	Modifier string `json:"modifier,omitempty"`
}

type createCollectionRequest struct {
	Vectors       map[string]denseVectorParams  `json:"vectors"`
	SparseVectors map[string]sparseVectorParams `json:"sparse_vectors"`
	Metadata      map[string]any                `json:"metadata"`
}

type payloadSchemaInfo struct {
	DataType string `json:"data_type"`
}

type collectionInfo struct {
	Config struct {
		Params struct {
			Vectors       map[string]denseVectorParams  `json:"vectors"`
			SparseVectors map[string]sparseVectorParams `json:"sparse_vectors"`
		} `json:"params"`
	} `json:"config"`
	PayloadSchema map[string]payloadSchemaInfo `json:"payload_schema"`
}

// payloadIndexRequest.FieldSchema is a bare type name ("keyword", "bool") or a textIndexSchema.
type payloadIndexRequest struct {
	FieldName   string `json:"field_name"`
	FieldSchema any    `json:"field_schema"`
}

type textIndexSchema struct {
	Type           string `json:"type"`
	Tokenizer      string `json:"tokenizer"`
	Lowercase      bool   `json:"lowercase"`
	PhraseMatching bool   `json:"phrase_matching"`
	AsciiFolding   bool   `json:"ascii_folding"`
}

// bm25Document asks Qdrant to infer the sparse BM25 vector from text server-side.
type bm25Document struct {
	Text    string         `json:"text"`
	Model   string         `json:"model"`
	Options map[string]any `json:"options"`
}

type matchCondition struct {
	Value any      `json:"value,omitempty"`
	Any   []string `json:"any,omitempty"`
}

type fieldCondition struct {
	Key   string         `json:"key"`
	Match matchCondition `json:"match"`
}

type pointFilter struct {
	Must []fieldCondition `json:"must"`
}
