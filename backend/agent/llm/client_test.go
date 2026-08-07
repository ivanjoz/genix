package llm

import (
	"context"
	"encoding/json"
	"testing"

	"app/core"
)

// TestAdaptRequestToProvider locks in the reasoning/routing translation, which
// is otherwise silent when wrong: a stray `reasoning` object sent to Meta (or a
// `reasoning_effort` sent to OpenRouter) is an unknown parameter, and mapping a
// disabled budget to "none" makes Muse Spark answer HTTP 400.
func TestAdaptRequestToProvider(t *testing.T) {
	disabled := false
	routing := pinnedOpenRouterUpstream("deepseek")

	cases := []struct {
		name            string
		provider        string
		request         ChatRequest
		expectedEffort  string
		expectReasoning bool
		expectRouting   bool
	}{
		{
			name:           "meta collapses effort into the flat field",
			provider:       ProviderMeta,
			request:        ChatRequest{Reasoning: &ReasoningOptions{Effort: "medium", Exclude: true}, Routing: routing},
			expectedEffort: "medium",
		},
		{
			// The page builder's subagents disable reasoning outright; Meta has no
			// accepted "off", so this must land on the cheapest budget instead.
			name:           "meta maps disabled reasoning to minimal, never none",
			provider:       ProviderMeta,
			request:        ChatRequest{Reasoning: &ReasoningOptions{Enabled: &disabled}},
			expectedEffort: metaMinimalEffort,
		},
		{
			name:     "meta leaves the field out when no budget is set",
			provider: ProviderMeta,
			request:  ChatRequest{},
		},
		{
			name:            "openrouter keeps the nested object and drops the flat field",
			provider:        ProviderOpenRouter,
			request:         ChatRequest{Reasoning: &ReasoningOptions{Effort: "low"}, ReasoningEffort: "low", Routing: routing},
			expectReasoning: true,
			expectRouting:   true,
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			request := testCase.request
			adaptRequestToProvider(testCase.provider, &request)

			if request.ReasoningEffort != testCase.expectedEffort {
				t.Errorf("ReasoningEffort = %q, want %q", request.ReasoningEffort, testCase.expectedEffort)
			}
			if (request.Reasoning != nil) != testCase.expectReasoning {
				t.Errorf("Reasoning present = %v, want %v", request.Reasoning != nil, testCase.expectReasoning)
			}
			if (request.Routing != nil) != testCase.expectRouting {
				t.Errorf("Routing present = %v, want %v", request.Routing != nil, testCase.expectRouting)
			}
		})
	}
}

// loadEnvForTest populates core.Env from credentials.json so NewClient can
// resolve the active provider's key. core.PopulateVariables panics when the
// file isn't found — we recover and skip so `go test ./...` from a
// directory without access stays green. To run the live test, invoke
// from a path where credentials.json is discoverable (project root /
// backend dir), or set GENIX_CREDENTIALS_FILE.
func loadEnvForTest(t *testing.T) {
	t.Helper()
	if core.Env == nil {
		defer func() {
			if r := recover(); r != nil {
				t.Skipf("credentials.json not available: %v", r)
			}
		}()
		core.PopulateVariables()
	}
	if core.Env == nil || apiKeyForProvider(ActiveProvider()) == "" {
		t.Skipf("no API key for MODEL_PROVIDER=%s in credentials.json, skipping live test", ActiveProvider())
	}
	t.Logf("provider=%s model=%s", ActiveProvider(), DefaultModelID())
}

// TestChatLive hits the real provider endpoint selected by MODEL_PROVIDER.
// Skipped unless that provider's key is present in credentials.json so this can
// stay in the standard test suite. Run locally with:
//
//	go test -v -run TestChatLive ./agent/llm/...
func TestChatLive(t *testing.T) {
	loadEnvForTest(t)
	c, err := NewClient()
	if err != nil {
		t.Fatal(err)
	}
	resp, err := c.Chat(context.Background(), ChatRequest{
		Messages: []Message{
			{Role: "system", Content: "You are a terse echo bot. Reply with exactly the word the user asks for, no punctuation."},
			{Role: "user", Content: "Say: pong"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("content=%q  finish=%q  tokens=%d in / %d out",
		resp.Choices[0].Message.Content,
		resp.Choices[0].FinishReason,
		resp.Usage.PromptTokens,
		resp.Usage.CompletionTokens,
	)
	if resp.Choices[0].Message.Content == "" {
		t.Fatalf("expected non-empty content, got: %+v", resp.Choices[0])
	}
}

// TestChatLiveToolCall verifies the tool-calling path: we declare one
// function and ask the model to invoke it. Skipped without the API key.
// This exercises the wire shape the loop will actually use (Tool /
// ToolCall / Arguments-as-JSON-string) so we catch shape bugs before the
// loop is built.
func TestChatLiveToolCall(t *testing.T) {
	loadEnvForTest(t)
	c, err := NewClient()
	if err != nil {
		t.Fatal(err)
	}
	resp, err := c.Chat(context.Background(), ChatRequest{
		Messages: []Message{
			{Role: "system", Content: "You always answer by calling the navigate tool with the given route. Never reply in plain text."},
			{Role: "user", Content: "Take me to /products"},
		},
		Tools: []Tool{
			{
				Type: "function",
				Function: ToolFunction{
					Name:        "navigate",
					Description: "Change the SPA route in the browser.",
					Parameters: map[string]any{
						"type": "object",
						"properties": map[string]any{
							"route": map[string]any{"type": "string", "description": "The SPA path to navigate to, e.g. /products"},
						},
						"required": []string{"route"},
					},
				},
			},
		},
		ToolChoice: "auto",
	})
	if err != nil {
		t.Fatal(err)
	}
	calls := resp.Choices[0].Message.ToolCalls
	if len(calls) == 0 {
		t.Fatalf("expected at least one tool call, got message=%+v finish=%q", resp.Choices[0].Message, resp.Choices[0].FinishReason)
	}
	call := calls[0]
	if call.Function.Name != "navigate" {
		t.Fatalf("expected navigate call, got %q", call.Function.Name)
	}
	// Arguments arrives as a JSON-encoded string per OpenAI's contract — the
	// loop will unmarshal this into a per-tool param struct. Round-trip
	// through json.Unmarshal here so the test fails if the wire shape ever
	// changes.
	var args struct {
		Route string `json:"route"`
	}
	if err := json.Unmarshal([]byte(call.Function.Arguments), &args); err != nil {
		t.Fatalf("decode tool args %q: %v", call.Function.Arguments, err)
	}
	t.Logf("tool=%s args=%+v finish=%q tokens=%d in / %d out",
		call.Function.Name, args, resp.Choices[0].FinishReason,
		resp.Usage.PromptTokens, resp.Usage.CompletionTokens)
}

// TestChatLiveDisabledReasoning exercises the budget the page builder's
// subagents use (ReasoningOptions{Enabled: false}) against the live provider.
// It exists because that budget is the one shape the two providers disagree
// on: it goes out as `reasoning: {enabled: false}` on OpenRouter but has to
// become `reasoning_effort: "minimal"` on Meta, since Muse Spark rejects the
// literal "none" with HTTP 400. A regression here would only surface as a 400
// deep inside a page-builder turn.
func TestChatLiveDisabledReasoning(t *testing.T) {
	loadEnvForTest(t)
	c, err := NewClient()
	if err != nil {
		t.Fatal(err)
	}
	reasoningDisabled := false
	resp, err := c.Chat(context.Background(), ChatRequest{
		Messages: []Message{
			{Role: "system", Content: "You are a terse echo bot. Reply with exactly the word the user asks for, no punctuation."},
			{Role: "user", Content: "Say: pong"},
		},
		Reasoning: &ReasoningOptions{Enabled: &reasoningDisabled},
	})
	if err != nil {
		t.Fatalf("disabled-reasoning request rejected by %s: %v", c.Provider, err)
	}
	if resp.Choices[0].Message.Content == "" {
		t.Fatalf("expected non-empty content, got: %+v", resp.Choices[0])
	}
	t.Logf("content=%q tokens=%d in / %d out", resp.Choices[0].Message.Content,
		resp.Usage.PromptTokens, resp.Usage.CompletionTokens)
}
