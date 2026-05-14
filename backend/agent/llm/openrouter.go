// Package llm is a thin HTTP client for OpenRouter's OpenAI-compatible
// /chat/completions endpoint. The agent loop in backend/agent uses it to
// drive the LLM that decides which page actions to invoke; the wire format
// and tool-calling semantics are exactly OpenAI's, so the same client works
// against any OpenRouter-routed model (we default to tencent/hy3-preview).
//
// Design rationale (see backend/agent/AGENTIC_LOOP_DESIGN.md):
//   - One Client per process. Construct at startup via NewClient so a
//     missing API key fails fast instead of mid-loop.
//   - The wire types live here (Message, Tool, ToolCall, ChatRequest,
//     ChatResponse) so callers don't have to redeclare them. They mirror
//     OpenAI's shapes 1:1 with JSON snake_case tags — keep them that way.
//   - Errors include the raw response body on non-2xx so we can debug
//     model-side rejections (bad tool schema, rate limits, etc.) without
//     adding extra logging at the call site.
package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"app/core"
)

const (
	openrouterEndpoint = "https://openrouter.ai/api/v1/chat/completions"
	defaultModel       = "deepseek/deepseek-v4-flash"
	requestTimeout     = 90 * time.Second
)

// Message is the OpenAI-compatible chat message used in `messages[]`.
// Content is empty when the assistant turn is purely tool calls;
// ToolCallID is set on `role: "tool"` messages to link the result back to
// the originating call.
type Message struct {
	Role       string     `json:"role"` // "system" | "user" | "assistant" | "tool"
	Content    string     `json:"content,omitempty"`
	ToolCalls  []ToolCall `json:"tool_calls,omitempty"`
	ToolCallID string     `json:"tool_call_id,omitempty"`
}

// Tool declares one callable function to the model. Parameters is a
// JSON-Schema object the model uses to shape the call's Arguments string.
type Tool struct {
	Type     string       `json:"type"` // always "function"
	Function ToolFunction `json:"function"`
}

type ToolFunction struct {
	Name        string         `json:"name"`
	Description string         `json:"description,omitempty"`
	Parameters  map[string]any `json:"parameters,omitempty"`
}

// ToolCall is the assistant's request to invoke one tool. Arguments stays
// as a JSON-encoded string (per OpenAI's contract) — the loop unmarshals it
// into the per-tool param struct when dispatching.
type ToolCall struct {
	ID       string           `json:"id"`
	Type     string           `json:"type"` // always "function"
	Function ToolCallFunction `json:"function"`
}

type ToolCallFunction struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

// ChatRequest is the body posted to /chat/completions. Only the fields the
// loop actually needs are surfaced — extras can be added when used.
type ChatRequest struct {
	Model    string    `json:"model"`
	Messages []Message `json:"messages"`
	Tools    []Tool    `json:"tools,omitempty"`
	// ToolChoice accepts OpenAI's "auto" | "none". Earlier tests with
	// tencent/hy3-preview's provider routing returned HTTP 404 for "required";
	// the current default (deepseek/deepseek-v4-flash) hasn't been validated
	// for "required" either — keep using "auto" and let the system prompt
	// discipline the model into ending the turn via the `finish` tool.
	ToolChoice  string   `json:"tool_choice,omitempty"`
	Temperature *float32 `json:"temperature,omitempty"`
	// Reasoning controls the thinking budget on reasoning-capable models
	// (DeepSeek v4 Flash, o-series, etc). Non-reasoning models ignore the
	// field. See https://openrouter.ai/docs/use-cases/reasoning-tokens —
	// the chat loop sets `effort: "low"` to keep latency and cost down,
	// since most turns are simple navigation/inspection calls.
	Reasoning *ReasoningOptions `json:"reasoning,omitempty"`
	// Provider pins or biases provider routing for the chosen model. Used to
	// prefer a specific backend (e.g. `atlas-cloud/fp8` for DeepSeek v4 Flash)
	// when latency or quantization matters. See
	// https://openrouter.ai/docs/features/provider-routing.
	Provider *ProviderOptions `json:"provider,omitempty"`
}

// ProviderOptions mirrors the subset of OpenRouter's `provider` parameter
// we actually use. Order ranks acceptable providers (first preferred);
// AllowFallbacks=false combined with a single-entry Order pins routing to
// exactly that provider — useful for reproducible benchmarks. Nil → leave
// the field out and let OpenRouter pick.
type ProviderOptions struct {
	Order          []string `json:"order,omitempty"`
	AllowFallbacks *bool    `json:"allow_fallbacks,omitempty"`
}

// ReasoningOptions mirrors OpenRouter's `reasoning` parameter. Either set
// `Effort` ("low"|"medium"|"high") for coarse control, or `MaxTokens` for a
// hard cap. `Exclude: true` hides the reasoning trace from the response so
// it doesn't bloat the next prompt — recommended for tool-calling loops
// where the model's chain-of-thought isn't useful between iterations.
type ReasoningOptions struct {
	Effort    string `json:"effort,omitempty"`
	MaxTokens int    `json:"max_tokens,omitempty"`
	Exclude   bool   `json:"exclude,omitempty"`
	Enabled   *bool  `json:"enabled,omitempty"`
}

// ChatResponse is the /chat/completions reply. The loop only ever looks at
// Choices[0] and Usage, but we keep the full list in case multi-choice
// sampling becomes useful later.
type ChatResponse struct {
	ID      string   `json:"id"`
	Choices []Choice `json:"choices"`
	Usage   Usage    `json:"usage"`
}

type Choice struct {
	Index        int     `json:"index"`
	Message      Message `json:"message"`
	FinishReason string  `json:"finish_reason"`
}

type Usage struct {
	PromptTokens     int32 `json:"prompt_tokens"`
	CompletionTokens int32 `json:"completion_tokens"`
	TotalTokens      int32 `json:"total_tokens"`
}

// Client holds the resolved API key, default model, and reusable HTTP
// client. One instance per process is enough — Chat is safe for concurrent
// callers because *http.Client is.
type Client struct {
	APIKey string
	Model  string
	HTTP   *http.Client
}

// NewClient resolves OPENROUTER_KEY (required) and OPENROUTER_MODEL
// (optional, defaults to tencent/hy3-preview) from core.Env — same path
// the rest of the backend uses for credentials.json. Failing here at
// startup is much friendlier than a 401 on the first user message.
func NewClient() (*Client, error) {
	if core.Env == nil {
		return nil, errors.New("core.Env not populated; call core.PopulateVariables before llm.NewClient")
	}
	key := core.Env.OPENROUTER_KEY
	if key == "" {
		return nil, errors.New("OPENROUTER_KEY not set in credentials.json")
	}
	model := core.Env.OPENROUTER_MODEL
	if model == "" {
		model = defaultModel
	}
	return &Client{
		APIKey: key,
		Model:  model,
		HTTP:   &http.Client{Timeout: requestTimeout},
	}, nil
}

// Chat performs one POST to /chat/completions. If req.Model is empty we
// fill it from c.Model so callers don't repeat the model on every turn.
// Non-2xx responses surface the raw body in the error so tool-schema or
// rate-limit problems are immediately readable.
func (c *Client) Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
	if req.Model == "" {
		req.Model = c.Model
	}
	// Fill in any per-model defaults the caller didn't set. Caller values
	// always win; unknown model IDs get no defaults (registry lookup is a
	// no-op for them).
	LookupModel(req.Model).applyDefaults(&req)
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", openrouterEndpoint, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.APIKey)
	// HTTP-Referer / X-Title are OpenRouter-specific analytics headers — they
	// don't affect routing or pricing, but show up in the dashboard.
	httpReq.Header.Set("HTTP-Referer", "https://genix.app")
	httpReq.Header.Set("X-Title", "Genix")

	core.Log("openrouter.Chat model::", req.Model, " messages::", len(req.Messages), " tools::", len(req.Tools))

	// Measure wall-clock from request send to body fully read so the TPS we
	// log later reflects what the user actually waits for, including TTFT.
	startedAt := time.Now()

	resp, err := c.HTTP.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("openrouter http: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("openrouter %d: %s", resp.StatusCode, truncate(string(respBody), 1000))
	}

	var out ChatResponse
	if err := json.Unmarshal(respBody, &out); err != nil {
		return nil, fmt.Errorf("decode response: %w (body=%s)", err, truncate(string(respBody), 500))
	}
	if len(out.Choices) == 0 {
		return nil, fmt.Errorf("openrouter: no choices in response (body=%s)", truncate(string(respBody), 500))
	}
	// Log in/out separately so it's obvious when the bloat is the prompt
	// (long get_page snapshots replayed every iteration) vs the completion
	// (long reasoning trace). TPS is completion-only — it's the metric you
	// want to compare across models, since input tokens are processed in
	// parallel at much higher rates than the decode loop.
	elapsed := time.Since(startedAt)
	tps := 0.0
	if seconds := elapsed.Seconds(); seconds > 0 {
		tps = float64(out.Usage.CompletionTokens) / seconds
	}
	core.Log("openrouter.Chat ok finish::", out.Choices[0].FinishReason,
		" in::", out.Usage.PromptTokens,
		" out::", out.Usage.CompletionTokens,
		" total::", out.Usage.TotalTokens,
		" elapsed::", elapsed.Round(time.Millisecond),
		fmt.Sprintf(" tps:: %.1f", tps))
	return &out, nil
}

// truncate keeps error messages bounded so a multi-KB upstream body doesn't
// flood logs when something goes wrong.
func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…(+" + fmt.Sprint(len(s)-n) + " bytes)"
}
