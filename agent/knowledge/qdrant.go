package knowledge

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"

	"app/core"

	"github.com/qdrant/go-client/qdrant"
)

const (
	DenseVectorName   = "dense"
	LexicalVectorName = "lexical"
	CollectionSchema  = 1
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

// QdrantConfig is explicit so tests and administrative commands do not depend on globals.
type QdrantConfig struct {
	Host       string
	GRPCPort   int
	APIKey     string
	UseTLS     bool
	Public     bool
	Collection string
	Dimensions int
}

type collectionClient interface {
	CollectionExists(context.Context, string) (bool, error)
	CreateCollection(context.Context, *qdrant.CreateCollection) error
	GetCollectionInfo(context.Context, string) (*qdrant.CollectionInfo, error)
	CreateFieldIndex(context.Context, *qdrant.CreateFieldIndexCollection) (*qdrant.UpdateResult, error)
	Query(context.Context, *qdrant.QueryPoints) ([]*qdrant.ScoredPoint, error)
	ScrollAndOffset(context.Context, *qdrant.ScrollPoints) ([]*qdrant.RetrievedPoint, *qdrant.PointId, error)
	Upsert(context.Context, *qdrant.UpsertPoints) (*qdrant.UpdateResult, error)
	OverwritePayload(context.Context, *qdrant.SetPayloadPoints) (*qdrant.UpdateResult, error)
	Delete(context.Context, *qdrant.DeletePoints) (*qdrant.UpdateResult, error)
	Close() error
}

// Store owns the Qdrant connection and validates the versioned RAG collection.
type Store struct {
	client collectionClient
	config QdrantConfig
}

func ConfigFromEnv() (QdrantConfig, error) {
	if core.Env == nil {
		return QdrantConfig{}, errors.New("core.Env is not initialized")
	}
	configured := QdrantConfig{
		Host:       core.Env.QDRANT_HOST,
		GRPCPort:   core.Env.QDRANT_GRPC_PORT,
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
	if config.GRPCPort <= 0 || config.GRPCPort > 65535 {
		return fmt.Errorf("qdrant.grpc_port %d is invalid", config.GRPCPort)
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
	if config.Public && !config.UseTLS {
		log.Printf("[agent.knowledge] warning: public Qdrant gRPC target %s:%d is configured without TLS", config.Host, config.GRPCPort)
	}
	client, err := qdrant.NewClient(&qdrant.Config{
		Host:     config.Host,
		Port:     config.GRPCPort,
		APIKey:   config.APIKey,
		UseTLS:   config.UseTLS,
		PoolSize: 1,
	})
	if err != nil {
		return nil, fmt.Errorf("connect to Qdrant %s:%d: %w", config.Host, config.GRPCPort, err)
	}
	log.Printf("[agent.knowledge] qdrant_client_ready host=%s port=%d collection=%s tls=%t",
		config.Host, config.GRPCPort, config.Collection, config.UseTLS)
	return &Store{client: client, config: config}, nil
}

func NewStoreFromEnv() (*Store, error) {
	config, err := ConfigFromEnv()
	if err != nil {
		return nil, err
	}
	return NewStore(config)
}

func (store *Store) Close() error {
	return store.client.Close()
}

func (store *Store) CollectionExists(ctx context.Context) (bool, error) {
	exists, err := store.client.CollectionExists(ctx, store.config.Collection)
	if err != nil {
		return false, fmt.Errorf("check Qdrant collection %q: %w", store.config.Collection, err)
	}
	return exists, nil
}

// ValidateExistingCollection is read-only and is used by dry-run before any proposed writes.
func (store *Store) ValidateExistingCollection(ctx context.Context) error {
	collectionInfo, err := store.client.GetCollectionInfo(ctx, store.config.Collection)
	if err != nil {
		return fmt.Errorf("inspect Qdrant collection %q: %w", store.config.Collection, err)
	}
	if err := validateCollectionSchema(collectionInfo, store.config.Dimensions); err != nil {
		return fmt.Errorf("Qdrant collection %q is incompatible: %w", store.config.Collection, err)
	}
	return nil
}

// EnsureCollection creates the versioned schema once and rejects incompatible reuse.
func (store *Store) EnsureCollection(ctx context.Context) error {
	exists, err := store.client.CollectionExists(ctx, store.config.Collection)
	if err != nil {
		return fmt.Errorf("check Qdrant collection %q: %w", store.config.Collection, err)
	}
	if !exists {
		if err := store.client.CreateCollection(ctx, collectionRequest(store.config)); err != nil {
			return fmt.Errorf("create Qdrant collection %q: %w", store.config.Collection, err)
		}
		log.Printf("[agent.knowledge] collection_created name=%s dimensions=%d schema=%d",
			store.config.Collection, store.config.Dimensions, CollectionSchema)
		return store.ensurePayloadIndexes(ctx, nil)
	}

	collectionInfo, err := store.client.GetCollectionInfo(ctx, store.config.Collection)
	if err != nil {
		return fmt.Errorf("inspect Qdrant collection %q: %w", store.config.Collection, err)
	}
	if err := validateCollectionSchema(collectionInfo, store.config.Dimensions); err != nil {
		return fmt.Errorf("Qdrant collection %q is incompatible: %w", store.config.Collection, err)
	}
	log.Printf("[agent.knowledge] collection_valid name=%s dimensions=%d schema=%d",
		store.config.Collection, store.config.Dimensions, CollectionSchema)
	return store.ensurePayloadIndexes(ctx, collectionInfo.GetPayloadSchema())
}

func (store *Store) ensurePayloadIndexes(ctx context.Context, existingSchema map[string]*qdrant.PayloadSchemaInfo) error {
	for _, fieldName := range keywordPayloadFields {
		if existing, exists := existingSchema[fieldName]; exists {
			if existing.GetDataType() != qdrant.PayloadSchemaType_Keyword {
				return fmt.Errorf("payload field %q is %s, want keyword", fieldName, existing.GetDataType())
			}
			continue
		}
		if _, err := store.client.CreateFieldIndex(ctx, keywordIndexRequest(store.config.Collection, fieldName)); err != nil {
			return fmt.Errorf("create keyword payload index %q: %w", fieldName, err)
		}
		log.Printf("[agent.knowledge] payload_index_created collection=%s field=%s type=keyword",
			store.config.Collection, fieldName)
	}
	for _, fieldName := range booleanPayloadFields {
		if existing, exists := existingSchema[fieldName]; exists {
			if existing.GetDataType() != qdrant.PayloadSchemaType_Bool {
				return fmt.Errorf("payload field %q is %s, want bool", fieldName, existing.GetDataType())
			}
			continue
		}
		if _, err := store.client.CreateFieldIndex(ctx, booleanIndexRequest(store.config.Collection, fieldName)); err != nil {
			return fmt.Errorf("create bool payload index %q: %w", fieldName, err)
		}
		log.Printf("[agent.knowledge] payload_index_created collection=%s field=%s type=bool",
			store.config.Collection, fieldName)
	}

	if existing, exists := existingSchema["content"]; exists {
		if existing.GetDataType() != qdrant.PayloadSchemaType_Text {
			return fmt.Errorf("payload field %q is %s, want text", "content", existing.GetDataType())
		}
		return nil
	}
	if _, err := store.client.CreateFieldIndex(ctx, contentIndexRequest(store.config.Collection)); err != nil {
		return fmt.Errorf("create text payload index %q: %w", "content", err)
	}
	log.Printf("[agent.knowledge] payload_index_created collection=%s field=content type=text", store.config.Collection)
	return nil
}

func keywordIndexRequest(collectionName, fieldName string) *qdrant.CreateFieldIndexCollection {
	return &qdrant.CreateFieldIndexCollection{
		CollectionName: collectionName,
		FieldName:      fieldName,
		FieldType:      qdrant.FieldType_FieldTypeKeyword.Enum(),
		Wait:           qdrant.PtrOf(true),
	}
}

func booleanIndexRequest(collectionName, fieldName string) *qdrant.CreateFieldIndexCollection {
	return &qdrant.CreateFieldIndexCollection{
		CollectionName: collectionName,
		FieldName:      fieldName,
		FieldType:      qdrant.FieldType_FieldTypeBool.Enum(),
		Wait:           qdrant.PtrOf(true),
	}
}

func contentIndexRequest(collectionName string) *qdrant.CreateFieldIndexCollection {
	return &qdrant.CreateFieldIndexCollection{
		CollectionName: collectionName,
		FieldName:      "content",
		FieldType:      qdrant.FieldType_FieldTypeText.Enum(),
		Wait:           qdrant.PtrOf(true),
		FieldIndexParams: qdrant.NewPayloadIndexParamsText(&qdrant.TextIndexParams{
			Tokenizer:      qdrant.TokenizerType_Multilingual,
			Lowercase:      qdrant.PtrOf(true),
			PhraseMatching: qdrant.PtrOf(true),
			AsciiFolding:   qdrant.PtrOf(true),
		}),
	}
}

func collectionRequest(config QdrantConfig) *qdrant.CreateCollection {
	return &qdrant.CreateCollection{
		CollectionName: config.Collection,
		VectorsConfig: qdrant.NewVectorsConfigMap(map[string]*qdrant.VectorParams{
			DenseVectorName: {
				Size:     uint64(config.Dimensions),
				Distance: qdrant.Distance_Cosine,
			},
		}),
		SparseVectorsConfig: qdrant.NewSparseVectorsConfig(map[string]*qdrant.SparseVectorParams{
			// Server-side qdrant/bm25 supplies TF/length weights; Qdrant applies corpus IDF.
			LexicalVectorName: {Modifier: qdrant.Modifier_Idf.Enum()},
		}),
		Metadata: qdrant.NewValueMap(map[string]any{
			"schema_version": int64(CollectionSchema),
		}),
	}
}

func validateCollectionSchema(collectionInfo *qdrant.CollectionInfo, expectedDimensions int) error {
	params := collectionInfo.GetConfig().GetParams()
	denseVectors := params.GetVectorsConfig().GetParamsMap().GetMap()
	denseConfig, exists := denseVectors[DenseVectorName]
	if !exists {
		return fmt.Errorf("missing named vector %q", DenseVectorName)
	}
	if denseConfig.GetSize() != uint64(expectedDimensions) {
		return fmt.Errorf("dense vector has %d dimensions, want %d", denseConfig.GetSize(), expectedDimensions)
	}
	if denseConfig.GetDistance() != qdrant.Distance_Cosine {
		return fmt.Errorf("dense vector uses %s distance, want cosine", denseConfig.GetDistance())
	}

	lexicalConfig, exists := params.GetSparseVectorsConfig().GetMap()[LexicalVectorName]
	if !exists {
		return fmt.Errorf("missing sparse vector %q", LexicalVectorName)
	}
	if lexicalConfig.GetModifier() != qdrant.Modifier_Idf {
		return fmt.Errorf("sparse vector %q must use the IDF modifier", LexicalVectorName)
	}
	return nil
}
