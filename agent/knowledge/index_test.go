package knowledge

import (
	"context"
	"testing"
)

func TestUpsertDocumentationPointUsesDenseAndServerBM25(t *testing.T) {
	fakeClient := &fakeCollectionClient{}
	store := Store{client: fakeClient, config: QdrantConfig{Collection: "docs_v1"}}

	err := store.UpsertDocumentationPoints(context.Background(), []IndexPoint{{
		ID:          "243fbf6f-0777-5f99-9d7a-dbbabc3ab32b",
		DenseVector: []float32{0.1, 0.2, 0.3},
		LexicalText: "cuadre de caja",
		Payload:     map[string]any{"document_id": "finance.cash-banks"},
	}})
	if err != nil {
		t.Fatal(err)
	}
	if fakeClient.upsertRequest == nil || !fakeClient.upsertRequest.GetWait() {
		t.Fatal("expected synchronous upsert request")
	}
	point := fakeClient.upsertRequest.GetPoints()[0]
	vectors := point.GetVectors().GetVectors().GetVectors()
	if len(vectors[DenseVectorName].GetDense().GetData()) != 3 {
		t.Fatal("dense vector was not included")
	}
	lexicalDocument := vectors[LexicalVectorName].GetDocument()
	if lexicalDocument.GetModel() != "qdrant/bm25" || lexicalDocument.GetText() != "cuadre de caja" {
		t.Fatalf("unexpected lexical document: %+v", lexicalDocument)
	}
	if lexicalDocument.GetOptions()["language"].GetStringValue() != "none" ||
		lexicalDocument.GetOptions()["tokenizer"].GetStringValue() != "multilingual" {
		t.Fatalf("unexpected BM25 options: %+v", lexicalDocument.GetOptions())
	}
}

func TestReplacePayloadDoesNotSendVectors(t *testing.T) {
	fakeClient := &fakeCollectionClient{}
	store := Store{client: fakeClient, config: QdrantConfig{Collection: "docs_v1"}}

	if err := store.ReplacePointPayload(context.Background(), "243fbf6f-0777-5f99-9d7a-dbbabc3ab32b", map[string]any{"content_hash": "sha256:test"}); err != nil {
		t.Fatal(err)
	}
	if len(fakeClient.payloadRequests) != 1 || !fakeClient.payloadRequests[0].GetWait() {
		t.Fatal("expected one synchronous payload-only request")
	}
}
