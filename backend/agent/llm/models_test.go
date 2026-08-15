package llm

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"app/core"
)

func withModelRegistryTestEnv(t *testing.T) {
	t.Helper()
	previousEnv := core.Env
	previousProviderConfigs := providerConfigs
	t.Cleanup(func() {
		core.Env = previousEnv
		providerConfigs = previousProviderConfigs
	})
	core.Env = &core.EnvStruct{
		MODEL_PROVIDER: ProviderMeta,
		META_KEY:       "meta-key",
		OPENROUTER_KEY: "openrouter-key",
		DEFAULT_MODEL:  "muse-spark-1.2-contributor",
		MODELS: []core.ModelEntry{
			{ID: "muse-spark-1.2-contributor", Provider: ProviderMeta, Reasoning: &core.ModelReasoning{Effort: "medium"}},
			{ID: "poolside/laguna-s-2.1", Provider: ProviderOpenRouter, Reasoning: &core.ModelReasoning{Effort: "medium", Exclude: true}},
			{ID: "deepseek/deepseek-v4-flash-0731", Provider: ProviderOpenRouter, Reasoning: &core.ModelReasoning{Effort: "low", Exclude: true}},
		},
	}
	providerConfigs = map[string]providerConfig{
		ProviderMeta:       previousProviderConfigs[ProviderMeta],
		ProviderOpenRouter: previousProviderConfigs[ProviderOpenRouter],
	}
}

func TestListModelsIncludesEveryConfiguredProvider(t *testing.T) {
	withModelRegistryTestEnv(t)

	models := ListModels()
	if len(models) != 3 {
		t.Fatalf("ListModels returned %d models, want 3: %+v", len(models), models)
	}
	if models[0].Provider != ProviderMeta || models[1].Provider != ProviderOpenRouter || models[2].Provider != ProviderOpenRouter {
		t.Fatalf("ListModels providers = [%q %q %q], want [meta openrouter openrouter]", models[0].Provider, models[1].Provider, models[2].Provider)
	}
	if !models[0].IsDefault {
		t.Fatalf("default model %q was not flagged", models[0].ID)
	}
}

func TestChatRoutesSelectedModelToItsConfiguredProvider(t *testing.T) {
	withModelRegistryTestEnv(t)

	openRouterServer := httptest.NewServer(http.HandlerFunc(func(responseWriter http.ResponseWriter, request *http.Request) {
		if authorization := request.Header.Get("Authorization"); authorization != "Bearer openrouter-key" {
			t.Errorf("Authorization = %q, want OpenRouter key", authorization)
		}
		var requestBody ChatRequest
		if err := json.NewDecoder(request.Body).Decode(&requestBody); err != nil {
			t.Errorf("decode request: %v", err)
			responseWriter.WriteHeader(http.StatusBadRequest)
			return
		}
		if requestBody.Model != "poolside/laguna-s-2.1" {
			t.Errorf("model = %q, want poolside/laguna-s-2.1", requestBody.Model)
		}
		if requestBody.Reasoning == nil || requestBody.Reasoning.Effort != "medium" {
			t.Errorf("OpenRouter reasoning = %+v, want medium registry default", requestBody.Reasoning)
		}
		responseWriter.Header().Set("Content-Type", "application/json")
		_, _ = responseWriter.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"ok"},"finish_reason":"stop"}],"usage":{"total_tokens":1}}`))
	}))
	defer openRouterServer.Close()

	openRouterConfig := providerConfigs[ProviderOpenRouter]
	openRouterConfig.Endpoint = openRouterServer.URL
	providerConfigs[ProviderOpenRouter] = openRouterConfig

	client, err := NewClient()
	if err != nil {
		t.Fatal(err)
	}
	if client.Provider != ProviderMeta || !client.RouteByModel {
		t.Fatalf("client startup provider=%q route_by_model=%t, want meta/true", client.Provider, client.RouteByModel)
	}
	response, err := client.Chat(context.Background(), ChatRequest{
		Model:    "poolside/laguna-s-2.1",
		Messages: []Message{{Role: "user", Content: "test"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if response.Choices[0].Message.Content != "ok" {
		t.Fatalf("response content = %q, want ok", response.Choices[0].Message.Content)
	}
}
