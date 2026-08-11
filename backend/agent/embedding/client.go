package embedding

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"strings"
	"time"

	"app/core"
)

const (
	openRouterEmbeddingsEndpoint = "https://openrouter.ai/api/v1/embeddings"
	requestTimeout               = 60 * time.Second
	maxErrorBodyBytes            = 8 << 10
	queryInstruction             = "Instruct: Given a mostly Spanish question from a small-business ERP user, retrieve Genix documentation passages that answer what the software can do, required conditions, business rules, limitations, side effects, or where the user should navigate.\nQuery: "
)

// ClientConfig keeps the embeddings wire contract independent from chat completions.
type ClientConfig struct {
	APIKey     string
	Model      string
	Dimensions int
	Endpoint   string
	HTTPClient *http.Client
}

// Client sends dense embedding requests to OpenRouter's OpenAI-compatible endpoint.
type Client struct {
	apiKey     string
	model      string
	dimensions int
	endpoint   string
	httpClient *http.Client
}

type embeddingRequest struct {
	Model          string   `json:"model"`
	Input          []string `json:"input"`
	Dimensions     int      `json:"dimensions"`
	EncodingFormat string   `json:"encoding_format"`
}

type embeddingResponse struct {
	Data []struct {
		Index     int       `json:"index"`
		Embedding []float32 `json:"embedding"`
	} `json:"data"`
	Usage struct {
		PromptTokens int `json:"prompt_tokens"`
		TotalTokens  int `json:"total_tokens"`
	} `json:"usage"`
}

func NewClient(config ClientConfig) (*Client, error) {
	config.APIKey = strings.TrimSpace(config.APIKey)
	config.Model = strings.TrimSpace(config.Model)
	if config.APIKey == "" {
		return nil, errors.New("agent.openrouter_key is required for documentation embeddings")
	}
	if config.Model == "" {
		return nil, errors.New("embedding_model.id is required")
	}
	if config.Dimensions <= 0 {
		return nil, errors.New("embedding_model.dimensions must be greater than zero")
	}
	if strings.TrimSpace(config.Endpoint) == "" {
		config.Endpoint = openRouterEmbeddingsEndpoint
	}
	if config.HTTPClient == nil {
		config.HTTPClient = &http.Client{Timeout: requestTimeout}
	}
	return &Client{
		apiKey:     config.APIKey,
		model:      config.Model,
		dimensions: config.Dimensions,
		endpoint:   config.Endpoint,
		httpClient: config.HTTPClient,
	}, nil
}

// NewClientFromEnv validates RAG-specific settings independently from the chat provider.
func NewClientFromEnv() (*Client, error) {
	if core.Env == nil {
		return nil, errors.New("core.Env is not initialized")
	}
	if core.Env.EMBEDDING_PROVIDER != "openrouter" {
		return nil, fmt.Errorf("unsupported embedding_model.provider %q", core.Env.EMBEDDING_PROVIDER)
	}
	return NewClient(ClientConfig{
		APIKey:     core.Env.OPENROUTER_KEY,
		Model:      core.Env.EMBEDDING_MODEL_ID,
		Dimensions: core.Env.EMBEDDING_DIMENSIONS,
	})
}

// EmbedDocuments intentionally sends document text without a retrieval instruction.
func (client *Client) EmbedDocuments(ctx context.Context, documents []string) ([][]float32, error) {
	if len(documents) == 0 {
		return nil, errors.New("at least one document is required")
	}
	for documentIndex, document := range documents {
		if strings.TrimSpace(document) == "" {
			return nil, fmt.Errorf("document %d is empty", documentIndex)
		}
	}
	return client.embed(ctx, documents)
}

// EmbedQuery preserves the user's wording and adds the Qwen retrieval instruction only here.
func (client *Client) EmbedQuery(ctx context.Context, userQuestion string) ([]float32, error) {
	normalizedQuestion := strings.Join(strings.Fields(userQuestion), " ")
	if normalizedQuestion == "" {
		return nil, errors.New("query is empty")
	}
	vectors, err := client.embed(ctx, []string{queryInstruction + normalizedQuestion})
	if err != nil {
		return nil, err
	}
	return vectors[0], nil
}

func (client *Client) embed(ctx context.Context, inputs []string) ([][]float32, error) {
	requestBody, err := json.Marshal(embeddingRequest{
		Model:          client.model,
		Input:          inputs,
		Dimensions:     client.dimensions,
		EncodingFormat: "float",
	})
	if err != nil {
		return nil, fmt.Errorf("encode embedding request: %w", err)
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, client.endpoint, bytes.NewReader(requestBody))
	if err != nil {
		return nil, fmt.Errorf("create embedding request: %w", err)
	}
	request.Header.Set("Authorization", "Bearer "+client.apiKey)
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("HTTP-Referer", "https://genix.app")
	request.Header.Set("X-Title", "Genix")

	startedAt := time.Now()
	response, err := client.httpClient.Do(request)
	if err != nil {
		return nil, fmt.Errorf("request embeddings: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		errorBody, _ := io.ReadAll(io.LimitReader(response.Body, maxErrorBodyBytes))
		return nil, fmt.Errorf("embedding provider returned HTTP %d: %s", response.StatusCode, strings.TrimSpace(string(errorBody)))
	}

	decodedResponse := embeddingResponse{}
	if err := json.NewDecoder(response.Body).Decode(&decodedResponse); err != nil {
		return nil, fmt.Errorf("decode embedding response: %w", err)
	}
	if len(decodedResponse.Data) != len(inputs) {
		return nil, fmt.Errorf("embedding response count %d does not match input count %d", len(decodedResponse.Data), len(inputs))
	}

	vectors := make([][]float32, len(inputs))
	for _, embeddingData := range decodedResponse.Data {
		if embeddingData.Index < 0 || embeddingData.Index >= len(inputs) {
			return nil, fmt.Errorf("embedding response index %d is out of range", embeddingData.Index)
		}
		if vectors[embeddingData.Index] != nil {
			return nil, fmt.Errorf("embedding response repeats index %d", embeddingData.Index)
		}
		if len(embeddingData.Embedding) != client.dimensions {
			return nil, fmt.Errorf("embedding %d has %d dimensions, want %d", embeddingData.Index, len(embeddingData.Embedding), client.dimensions)
		}
		for vectorIndex, value := range embeddingData.Embedding {
			if math.IsNaN(float64(value)) || math.IsInf(float64(value), 0) {
				return nil, fmt.Errorf("embedding %d contains a non-finite value at dimension %d", embeddingData.Index, vectorIndex)
			}
		}
		vectors[embeddingData.Index] = embeddingData.Embedding
	}

	log.Printf("[agent.embedding] model=%s inputs=%d dimensions=%d prompt_tokens=%d duration=%s",
		client.model, len(inputs), client.dimensions, decodedResponse.Usage.PromptTokens, time.Since(startedAt).Round(time.Millisecond))
	return vectors, nil
}
