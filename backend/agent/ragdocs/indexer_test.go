package ragdocs

import (
	"context"
	"testing"
	"time"

	"app/agent/knowledge"
)

type fakeEmbedder struct {
	inputs []string
}

func (embedder *fakeEmbedder) EmbedDocuments(_ context.Context, inputs []string) ([][]float32, error) {
	embedder.inputs = append(embedder.inputs, inputs...)
	vectors := make([][]float32, len(inputs))
	for inputIndex := range inputs {
		vectors[inputIndex] = []float32{float32(inputIndex), 0.2, 0.3}
	}
	return vectors, nil
}

type fakePointStore struct {
	existing       []knowledge.ExistingPoint
	upserted       []knowledge.IndexPoint
	payloadUpdates []string
	deleted        []string
}

func (store *fakePointStore) ListDocumentPoints(context.Context, string) ([]knowledge.ExistingPoint, error) {
	return store.existing, nil
}

func (store *fakePointStore) UpsertDocumentationPoints(_ context.Context, points []knowledge.IndexPoint) error {
	store.upserted = append(store.upserted, points...)
	return nil
}

func (store *fakePointStore) ReplacePointPayload(_ context.Context, pointID string, _ map[string]any) error {
	store.payloadUpdates = append(store.payloadUpdates, pointID)
	return nil
}

func (store *fakePointStore) DeleteDocumentationPoints(_ context.Context, pointIDs []string) error {
	store.deleted = append(store.deleted, pointIDs...)
	return nil
}

func testDocumentAndChunks(t *testing.T) (*Document, []Chunk, IndexOptions) {
	t.Helper()
	document := &Document{
		RepositoryPath: "frontend/routes/finance/example/DOCUMENTATION.md",
		Frontmatter:    Frontmatter{Schema: 1, PageID: "finance.example", Route: "/finance/example", Title: "Example", Status: "implemented", Visibility: "tenant"},
		Sections: []Section{
			{ID: "page-purpose", Title: "Page purpose", Type: "purpose", Markdown: "## Page purpose\n\nPurpose."},
			{ID: "capability.create", Title: "Create", Type: "capability", Markdown: "## Create\n\nCreate something."},
		},
		FileHash:          "sha256:file",
		DocumentationHash: "sha256:document",
		SourceHashDigest:  "sha256:sources",
	}
	chunks, err := BuildChunks(document)
	if err != nil {
		t.Fatal(err)
	}
	options := IndexOptions{BatchSize: 8, IndexedAt: time.Unix(123, 0), Model: "model", Dimensions: 3}
	return document, chunks, options
}

func TestIndexDocumentSkipsCompletedExactFile(t *testing.T) {
	document, chunks, options := testDocumentAndChunks(t)
	contractHash := IndexContractHash(options.Model, options.Dimensions)
	store := &fakePointStore{existing: []knowledge.ExistingPoint{
		{
			ID: chunks[0].PointID, DocumentationFileHash: document.FileHash,
			SourceHashDigest: document.SourceHashDigest, IndexContractHash: contractHash,
			IndexState: "complete", DocumentChunkCount: len(chunks),
		},
		{ID: chunks[1].PointID},
	}}
	embedder := &fakeEmbedder{}

	result, err := IndexDocument(context.Background(), store, embedder, document, chunks, options)
	if err != nil {
		t.Fatal(err)
	}
	if !result.Skipped || len(embedder.inputs) != 0 || len(store.upserted) != 0 {
		t.Fatalf("unexpected exact-file result: %+v", result)
	}
}

func TestIndexDocumentDoesNotSkipWhenAStoredChunkIsMissing(t *testing.T) {
	document, chunks, options := testDocumentAndChunks(t)
	contractHash := IndexContractHash(options.Model, options.Dimensions)
	store := &fakePointStore{existing: []knowledge.ExistingPoint{{
		ID: chunks[0].PointID, DocumentationFileHash: document.FileHash,
		SourceHashDigest: document.SourceHashDigest, IndexContractHash: contractHash,
		IndexState: "complete", DocumentChunkCount: len(chunks),
	}}}

	result, err := IndexDocument(context.Background(), store, &fakeEmbedder{}, document, chunks, options)
	if err != nil {
		t.Fatal(err)
	}
	if result.Skipped {
		t.Fatal("document with a missing stored chunk must be repaired")
	}
}

func TestIndexDocumentReusesUnchangedVectorsAndCommitsMarkerLast(t *testing.T) {
	document, chunks, options := testDocumentAndChunks(t)
	contractHash := IndexContractHash(options.Model, options.Dimensions)
	store := &fakePointStore{existing: []knowledge.ExistingPoint{
		{ID: chunks[0].PointID, ContentHash: chunks[0].ContentHash, DocumentationFileHash: "sha256:old", IndexContractHash: contractHash, IndexState: "complete"},
		{ID: chunks[1].PointID, ContentHash: chunks[1].ContentHash, IndexContractHash: contractHash},
		{ID: "deleted-point"},
	}}
	embedder := &fakeEmbedder{}

	result, err := IndexDocument(context.Background(), store, embedder, document, chunks, options)
	if err != nil {
		t.Fatal(err)
	}
	if result.Embedded != 0 || len(embedder.inputs) != 0 {
		t.Fatalf("unchanged chunks were embedded: %+v", result)
	}
	if len(store.payloadUpdates) != 2 || store.payloadUpdates[1] != chunks[0].PointID {
		t.Fatalf("marker was not committed last: %+v", store.payloadUpdates)
	}
	if len(store.deleted) != 1 || store.deleted[0] != "deleted-point" {
		t.Fatalf("obsolete point was not deleted: %+v", store.deleted)
	}
}
