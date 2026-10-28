// Package harness is the only package that drives Eino's runner: the chat model, the checkpoint
// store, the tool-error middleware and the turn runner. The agent's tools and prompts live in the
// agent package; this one knows nothing about pages, sales or the browser.
package harness

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"sync"

	"app/core"
	"genix-agent/llm"

	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/components/model"
)

// The chat models are cached per model id: in Lambda the container outlives the invocation, and
// rebuilding the client on every turn would throw its connection pool away.
var (
	chatModelCache      = map[string]model.ToolCallingChatModel{}
	chatModelCacheMutex sync.Mutex
)

// ResolveModelID turns the hash the chat widget sends into a [[models]] id. An empty hash is the
// configured default.
func ResolveModelID(modelHash string) (string, error) {
	if modelHash = strings.TrimSpace(modelHash); modelHash == "" {
		if modelID := llm.DefaultModelID(); modelID != "" {
			return modelID, nil
		}
		return "", fmt.Errorf("no [[models]] entry is configured")
	}
	modelConfig, found := llm.LookupModelHash(modelHash)
	if !found {
		return "", fmt.Errorf("modelo de agente no válido: %s", modelHash)
	}
	return modelConfig.ID, nil
}

// ChatModel returns the Eino chat model of one [[models]] entry. Both providers speak OpenAI's
// protocol; what differs is carried the way llm.Client carried it: OpenRouter takes the nested
// `reasoning` and `provider` objects as extra body fields, Meta a flat `reasoning_effort`.
func ChatModel(ctx context.Context, modelID string) (model.ToolCallingChatModel, error) {
	chatModelCacheMutex.Lock()
	defer chatModelCacheMutex.Unlock()
	if cached, found := chatModelCache[modelID]; found {
		return cached, nil
	}

	modelConfig := llm.LookupModel(modelID)
	provider := modelConfig.Provider
	if provider == "" {
		provider = llm.ActiveProvider()
	}
	baseURL, apiKey, headers, err := llm.ProviderConnection(provider)
	if err != nil {
		return nil, fmt.Errorf("model %q: %w", modelID, err)
	}

	config := &openai.ChatModelConfig{
		APIKey:     apiKey,
		BaseURL:    baseURL,
		Model:      modelID,
		Timeout:    llm.RequestTimeout,
		HTTPClient: &http.Client{Timeout: llm.RequestTimeout, Transport: headerTransport{headers: headers}},
	}
	if provider == llm.ProviderMeta {
		if modelConfig.Reasoning != nil {
			config.ReasoningEffort = openai.ReasoningEffortLevel(llm.MetaReasoningEffort(*modelConfig.Reasoning))
		}
	} else {
		extraFields := map[string]any{}
		if modelConfig.Reasoning != nil {
			extraFields["reasoning"] = modelConfig.Reasoning
		}
		if modelConfig.Routing != nil {
			extraFields["provider"] = modelConfig.Routing
		}
		if len(extraFields) > 0 {
			config.ExtraFields = extraFields
		}
	}

	chatModel, err := openai.NewChatModel(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("build chat model %q: %w", modelID, err)
	}
	core.Log("agent.harness chat model ready::", modelID, " provider::", provider)
	chatModelCache[modelID] = chatModel
	return chatModel, nil
}

// headerTransport adds the provider's attribution headers (OpenRouter's dashboard) to every call.
type headerTransport struct{ headers map[string]string }

func (transport headerTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	for name, value := range transport.headers {
		request.Header.Set(name, value)
	}
	return http.DefaultTransport.RoundTrip(request)
}
