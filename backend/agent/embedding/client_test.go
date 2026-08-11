package embedding

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestDocumentAndQueryEmbeddingContracts(t *testing.T) {
	receivedInputs := make([][]string, 0, 2)
	testServer := httptest.NewServer(http.HandlerFunc(func(responseWriter http.ResponseWriter, request *http.Request) {
		if request.Header.Get("Authorization") != "Bearer test-key" {
			t.Errorf("Authorization = %q", request.Header.Get("Authorization"))
		}
		decodedRequest := embeddingRequest{}
		if err := json.NewDecoder(request.Body).Decode(&decodedRequest); err != nil {
			t.Fatal(err)
		}
		receivedInputs = append(receivedInputs, decodedRequest.Input)
		if decodedRequest.Model != "test-model" || decodedRequest.Dimensions != 3 || decodedRequest.EncodingFormat != "float" {
			t.Errorf("unexpected request: %+v", decodedRequest)
		}

		responseWriter.Header().Set("Content-Type", "application/json")
		responseWriter.Write([]byte(`{"data":[{"index":0,"embedding":[0.1,0.2,0.3]}],"usage":{"prompt_tokens":7}}`))
	}))
	defer testServer.Close()

	client, err := NewClient(ClientConfig{
		APIKey:     "test-key",
		Model:      "test-model",
		Dimensions: 3,
		Endpoint:   testServer.URL,
		HTTPClient: testServer.Client(),
	})
	if err != nil {
		t.Fatal(err)
	}

	documentText := "A cash register (caja) records the available balance (saldo)."
	if _, err := client.EmbedDocuments(context.Background(), []string{documentText}); err != nil {
		t.Fatal(err)
	}
	if _, err := client.EmbedQuery(context.Background(), "  ¿Dónde   hago un cuadre?  "); err != nil {
		t.Fatal(err)
	}

	if receivedInputs[0][0] != documentText {
		t.Fatalf("document input was modified: %q", receivedInputs[0][0])
	}
	if strings.HasPrefix(receivedInputs[0][0], "Instruct:") {
		t.Fatal("document input must not carry the query instruction")
	}
	expectedQuery := queryInstruction + "¿Dónde hago un cuadre?"
	if receivedInputs[1][0] != expectedQuery {
		t.Fatalf("query input = %q, want %q", receivedInputs[1][0], expectedQuery)
	}
}

func TestEmbeddingResponseDimensionIsValidated(t *testing.T) {
	testServer := httptest.NewServer(http.HandlerFunc(func(responseWriter http.ResponseWriter, request *http.Request) {
		responseWriter.Header().Set("Content-Type", "application/json")
		responseWriter.Write([]byte(`{"data":[{"index":0,"embedding":[0.1,0.2]}]}`))
	}))
	defer testServer.Close()

	client, err := NewClient(ClientConfig{
		APIKey: "test-key", Model: "test-model", Dimensions: 3,
		Endpoint: testServer.URL, HTTPClient: testServer.Client(),
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = client.EmbedDocuments(context.Background(), []string{"document"})
	if err == nil || !strings.Contains(err.Error(), "has 2 dimensions, want 3") {
		t.Fatalf("expected dimension error, got %v", err)
	}
}
