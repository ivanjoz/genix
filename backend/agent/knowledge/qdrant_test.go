package knowledge

import (
	"context"
	"strings"
	"testing"

	"github.com/qdrant/go-client/qdrant"
)

type fakeCollectionClient struct {
	exists         bool
	created        *qdrant.CreateCollection
	createdIndexes []*qdrant.CreateFieldIndexCollection
	collectionInfo *qdrant.CollectionInfo
}

func (client *fakeCollectionClient) CollectionExists(context.Context, string) (bool, error) {
	return client.exists, nil
}

func (client *fakeCollectionClient) CreateCollection(_ context.Context, request *qdrant.CreateCollection) error {
	client.created = request
	return nil
}

func (client *fakeCollectionClient) GetCollectionInfo(context.Context, string) (*qdrant.CollectionInfo, error) {
	return client.collectionInfo, nil
}

func (client *fakeCollectionClient) CreateFieldIndex(_ context.Context, request *qdrant.CreateFieldIndexCollection) (*qdrant.UpdateResult, error) {
	client.createdIndexes = append(client.createdIndexes, request)
	return &qdrant.UpdateResult{}, nil
}

func (client *fakeCollectionClient) Close() error { return nil }

func TestEnsureCollectionCreatesDenseAndBM25Schema(t *testing.T) {
	configured := QdrantConfig{Collection: "docs_v1", Dimensions: 4096}
	fakeClient := &fakeCollectionClient{}
	store := Store{client: fakeClient, config: configured}

	if err := store.EnsureCollection(context.Background()); err != nil {
		t.Fatal(err)
	}
	if fakeClient.created == nil {
		t.Fatal("collection was not created")
	}
	denseConfig := fakeClient.created.GetVectorsConfig().GetParamsMap().GetMap()[DenseVectorName]
	if denseConfig.GetSize() != 4096 || denseConfig.GetDistance() != qdrant.Distance_Cosine {
		t.Fatalf("unexpected dense config: %+v", denseConfig)
	}
	lexicalConfig := fakeClient.created.GetSparseVectorsConfig().GetMap()[LexicalVectorName]
	if lexicalConfig.GetModifier() != qdrant.Modifier_Idf {
		t.Fatalf("lexical modifier = %s, want IDF", lexicalConfig.GetModifier())
	}
	if len(fakeClient.createdIndexes) != len(keywordPayloadFields)+1 {
		t.Fatalf("created %d payload indexes, want %d", len(fakeClient.createdIndexes), len(keywordPayloadFields)+1)
	}
	contentIndex := fakeClient.createdIndexes[len(fakeClient.createdIndexes)-1]
	textParams := contentIndex.GetFieldIndexParams().GetTextIndexParams()
	if contentIndex.GetFieldName() != "content" || textParams.GetTokenizer() != qdrant.TokenizerType_Multilingual ||
		!textParams.GetLowercase() || !textParams.GetPhraseMatching() || !textParams.GetAsciiFolding() {
		t.Fatalf("unexpected content index: %+v", contentIndex)
	}
}

func TestValidateCollectionSchemaRejectsDimensionMismatch(t *testing.T) {
	request := collectionRequest(QdrantConfig{Collection: "docs_v1", Dimensions: 2048})
	collectionInfo := &qdrant.CollectionInfo{Config: &qdrant.CollectionConfig{
		Params: &qdrant.CollectionParams{
			VectorsConfig:       request.VectorsConfig,
			SparseVectorsConfig: request.SparseVectorsConfig,
		},
	}}

	err := validateCollectionSchema(collectionInfo, 4096)
	if err == nil || !strings.Contains(err.Error(), "2048 dimensions, want 4096") {
		t.Fatalf("expected dimension mismatch, got %v", err)
	}
}
