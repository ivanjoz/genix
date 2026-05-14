package llm

import (
	"context"
	"encoding/json"
	"testing"

	"app/core"
)

// loadEnvForTest populates core.Env from credentials.json so NewClient
// can resolve OPENROUTER_KEY. core.PopulateVariables panics when the
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
	if core.Env == nil || core.Env.OPENROUTER_KEY == "" {
		t.Skip("OPENROUTER_KEY not set in credentials.json, skipping live test")
	}
}

// TestChatLive hits the real OpenRouter endpoint. Skipped unless the
// OpenRouter key is present in credentials.json so this can stay in the
// standard test suite. Run locally with:
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
