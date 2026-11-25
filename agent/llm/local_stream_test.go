package llm

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"app/core"
)

// The stream below is the shape llama-server sends with return_progress and timings_per_token:
// prompt progress first, then reasoning, then a tool call split in pieces.
const localTestStream = `data: {"id":"c1","created":1,"model":"m","choices":[{"index":0,"delta":{"role":"assistant","content":null}}],"prompt_progress":{"total":100,"cache":20,"processed":0,"time_ms":0}}

data: {"id":"c1","created":1,"model":"m","choices":[{"index":0,"delta":{"role":"assistant","content":null}}],"prompt_progress":{"total":100,"cache":20,"processed":80,"time_ms":210}}

data: {"id":"c1","created":1,"model":"m","choices":[{"index":0,"delta":{"reasoning_content":"pienso"}}],"timings":{"cache_n":20,"prompt_n":80,"prompt_ms":210.5,"predicted_n":1,"predicted_per_second":40}}

data: {"id":"c1","created":1,"model":"m","choices":[{"index":0,"delta":{"tool_calls":[{"index":0,"id":"call1","type":"function","function":{"name":"navigate","arguments":"{\"route\":"}}]}}],"timings":{"cache_n":20,"prompt_n":80,"prompt_ms":210.5,"predicted_n":5,"predicted_per_second":41}}

data: {"id":"c1","created":1,"model":"m","choices":[{"index":0,"delta":{"tool_calls":[{"index":0,"function":{"arguments":"\"/ventas\"}"}}]}}],"timings":{"cache_n":20,"prompt_n":80,"prompt_ms":210.5,"predicted_n":9,"predicted_per_second":42}}

data: {"id":"c1","created":1,"model":"m","choices":[{"index":0,"finish_reason":"tool_calls","delta":{}}],"timings":{"cache_n":20,"prompt_n":80,"prompt_ms":210.5,"predicted_n":10,"predicted_per_second":42}}

data: [DONE]

`

func TestLocalProviderAssemblesTheStreamAndReportsProgress(t *testing.T) {
	llamaServer := httptest.NewServer(http.HandlerFunc(func(responseWriter http.ResponseWriter, request *http.Request) {
		if authorization := request.Header.Get("Authorization"); strings.Contains(authorization, "openrouter-key") {
			t.Errorf("Authorization = %q: the local provider must carry no key", authorization)
		}
		requestBody := map[string]any{}
		_ = json.NewDecoder(request.Body).Decode(&requestBody)
		if requestBody["stream"] != true || requestBody["return_progress"] != true || requestBody["timings_per_token"] != true {
			t.Errorf("request did not ask for the progress stream: %v", requestBody)
		}
		responseWriter.Header().Set("Content-Type", "text/event-stream")
		_, _ = responseWriter.Write([]byte(localTestStream))
	}))
	defer llamaServer.Close()

	previousEnv := core.Env
	t.Cleanup(func() { core.Env = previousEnv })
	core.Env = &core.EnvStruct{OPENROUTER_KEY: "openrouter-key", LOCAL_MODEL_URL: llamaServer.URL + "/v1/", MODELS: []core.ModelEntry{{ID: "genix-bot", Provider: ProviderLocal}}}
	llamaServer.Config.Handler = http.StripPrefix("/v1", llamaServer.Config.Handler)

	client, err := NewClientForProvider(ProviderLocal, "genix-bot")
	if err != nil {
		t.Fatal(err)
	}
	reports := []LocalProgress{}
	ctx := WithLocalProgress(context.Background(), func(progress LocalProgress) { reports = append(reports, progress) })
	response, err := client.Chat(ctx, ChatRequest{Messages: []Message{{Role: "user", Content: "ve a ventas"}}})
	if err != nil {
		t.Fatal(err)
	}

	toolCalls := response.Choices[0].Message.ToolCalls
	if len(toolCalls) != 1 || toolCalls[0].ID != "call1" || toolCalls[0].Function.Name != "navigate" ||
		toolCalls[0].Function.Arguments != `{"route":"/ventas"}` {
		t.Fatalf("tool calls = %+v", toolCalls)
	}
	if response.Choices[0].FinishReason != "tool_calls" || response.Usage.PromptTokens != 100 || response.Usage.CompletionTokens != 10 {
		t.Fatalf("finish=%q usage=%+v", response.Choices[0].FinishReason, response.Usage)
	}
	phases := []string{}
	for _, report := range reports {
		phases = append(phases, report.Phase)
	}
	if len(phases) != 3 || phases[0] != "prefill" || phases[1] != "thinking" || phases[2] != "answer" {
		t.Fatalf("reported phases = %v, want [prefill thinking answer] (one per phase inside the interval)", phases)
	}
	if reports[1].PromptProcessed != 100 || reports[1].PromptMs != 210.5 || reports[2].TokensPerSecond != 41 {
		t.Fatalf("reports = %+v", reports)
	}
}
