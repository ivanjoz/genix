// Package llm is a thin HTTP client for the OpenAI-compatible
// /chat/completions endpoint of the two upstreams the in-app agent can talk
// to: Meta's Model API (api.meta.ai) and OpenRouter. Which one is called is
// decided by providers.model in config.toml. The agent loop in
// backend/agent uses this to drive the LLM that decides which page actions to
// invoke.
//
// Both providers speak OpenAI's wire format for messages, tools and
// tool-calling, so the types here (Message, Tool, ToolCall, ChatRequest,
// ChatResponse) serve both unchanged — keep them mirroring OpenAI's shapes 1:1
// with JSON snake_case tags. What differs is only:
//
//   - the endpoint and the analytics headers, and
//   - how the thinking budget is expressed: Meta takes a flat
//     `reasoning_effort` string, OpenRouter takes a nested `reasoning` object
//     plus its own `provider` routing block.
//
// Chat() translates the internal ReasoningOptions into whichever shape the
// active provider expects, so callers never branch on the provider.
//
// Design rationale (see backend/agent/AGENTIC_LOOP_DESIGN.md):
//   - One Client per process. Construct at startup via NewClient so a missing
//     API key fails fast instead of mid-loop.
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
	"strings"
	"time"

	"app/core"
)

// Provider ids accepted in MODEL_PROVIDER.
const (
	ProviderMeta       = "meta"
	ProviderOpenRouter = "openrouter"
)

const (
	requestTimeout = 90 * time.Second
	// metaMinimalEffort is the cheapest reasoning budget Muse Spark accepts.
	// Meta's API also documents "none", but Muse Spark rejects it with HTTP 400
	// — so "reasoning disabled" maps here instead. See metaReasoningEffort.
	metaMinimalEffort = "minimal"
)

// providerConfig holds everything that differs between the two upstreams.
// Everything else about the request is identical OpenAI-compatible JSON.
type providerConfig struct {
	Endpoint string
	// KeyName is the config.toml field carrying the bearer token. Only used
	// to name the missing variable in NewClient's error.
	KeyName string
	// AnalyticsHeaders are provider-specific headers that don't affect routing
	// or pricing (OpenRouter dashboard attribution). Meta needs none.
	AnalyticsHeaders map[string]string
}

var providerConfigs = map[string]providerConfig{
	ProviderMeta: {
		Endpoint: "https://api.meta.ai/v1/chat/completions",
		KeyName:  "agent.meta_key",
	},
	ProviderOpenRouter: {
		Endpoint:         "https://openrouter.ai/api/v1/chat/completions",
		KeyName:          "agent.openrouter_key",
		AnalyticsHeaders: map[string]string{"HTTP-Referer": "https://genix.app", "X-Title": "Genix"},
	},
}

// ActiveProvider resolves providers.model from config.toml. Blank means
// OpenRouter — the historical default, so a deployment that never set the flag
// keeps working untouched. An unrecognized value is logged instead of silently
// accepted, because otherwise the only symptom would be a confusing
// "agent.openrouter_key not set".
func ActiveProvider() string {
	if core.Env == nil {
		return ProviderOpenRouter
	}
	switch provider := strings.ToLower(strings.TrimSpace(core.Env.MODEL_PROVIDER)); provider {
	case ProviderMeta, ProviderOpenRouter:
		return provider
	case "":
		return ProviderOpenRouter
	default:
		core.Log("llm.ActiveProvider unknown MODEL_PROVIDER::", provider, " using::", ProviderOpenRouter)
		return ProviderOpenRouter
	}
}

// apiKeyForProvider reads the bearer token of the given provider from core.Env.
// Only the active provider's key has to be present — the other can stay empty.
func apiKeyForProvider(provider string) string {
	if core.Env == nil {
		return ""
	}
	if provider == ProviderMeta {
		return core.Env.META_KEY
	}
	return core.Env.OPENROUTER_KEY
}

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
// loop actually needs are surfaced — extras can be added when used. The
// provider-specific fields are filled in by Chat() from Reasoning; callers set
// Reasoning (or nothing) and stay provider-agnostic.
type ChatRequest struct {
	Model    string    `json:"model"`
	Messages []Message `json:"messages"`
	Tools    []Tool    `json:"tools,omitempty"`
	// ToolChoice accepts OpenAI's "auto" | "none". Earlier tests found at least
	// one OpenRouter upstream answering HTTP 404 to "required", and no configured
	// model has been validated for it — keep using "auto" and let the system
	// prompt discipline the model into ending the turn via the `finish` tool.
	ToolChoice  string   `json:"tool_choice,omitempty"`
	Temperature *float32 `json:"temperature,omitempty"`
	// MaxTokens bounds mechanical subagents such as the global classifier.
	// Zero leaves the provider default unchanged for existing agent loops.
	MaxTokens int `json:"max_tokens,omitempty"`
	// Reasoning is the provider-agnostic thinking budget callers set. Chat()
	// translates it: sent as-is to OpenRouter, collapsed into ReasoningEffort
	// for Meta. Nil = don't constrain the budget.
	Reasoning *ReasoningOptions `json:"reasoning,omitempty"`
	// ReasoningEffort is Meta's flat thinking-budget field
	// ("minimal"|"low"|"medium"|"high"|"xhigh"). Derived from Reasoning by
	// Chat(); a caller that sets it directly wins, but then it only has an
	// effect when MODEL_PROVIDER=meta.
	ReasoningEffort string `json:"reasoning_effort,omitempty"`
	// Routing pins or biases OpenRouter's upstream selection for the chosen
	// model (e.g. `deepinfra/fp8`) when latency or quantization matters. See
	// https://openrouter.ai/docs/features/provider-routing. Dropped on Meta,
	// which serves its own models and has no routing concept.
	Routing *OpenRouterRouting `json:"provider,omitempty"`
}

// OpenRouterRouting mirrors the subset of OpenRouter's `provider` parameter we
// actually use — named for the vendor so it isn't confused with MODEL_PROVIDER,
// which picks the API we call. Sort ranks the model's upstreams by one criterion
// ("throughput" for the fastest-decoding endpoint, "price", "latency") and is
// the knob to reach for when any upstream will do but speed matters. Order
// instead names acceptable upstreams explicitly (first preferred), and with
// AllowFallbacks=false pins routing to exactly one — for when a specific
// quantization or region is required. Nil → let OpenRouter pick.
type OpenRouterRouting struct {
	Order          []string `json:"order,omitempty"`
	Sort           string   `json:"sort,omitempty"`
	AllowFallbacks *bool    `json:"allow_fallbacks,omitempty"`
}

// ReasoningOptions is the internal representation of a thinking budget. It
// mirrors OpenRouter's `reasoning` parameter, which is the richer of the two:
// either set `Effort` ("minimal"|"low"|"medium"|"high"|"xhigh") for coarse
// control, or `MaxTokens` for a hard cap. `Exclude: true` hides the reasoning
// trace from the response so it doesn't bloat the next prompt — recommended for
// tool-calling loops where the model's chain-of-thought isn't useful between
// iterations. On Meta only `Effort` and `Enabled` survive the translation (see
// metaReasoningEffort).
type ReasoningOptions struct {
	Effort    string `json:"effort,omitempty"`
	MaxTokens int    `json:"max_tokens,omitempty"`
	Exclude   bool   `json:"exclude,omitempty"`
	Enabled   *bool  `json:"enabled,omitempty"`
}

// ChatResponse is the /chat/completions reply. The loop only ever looks at
// Choices[0] and Usage, but we keep the full list in case multi-choice
// sampling becomes useful later. Both providers return this shape.
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

// Client holds the resolved provider, API key, default model, and reusable
// HTTP client. One instance per process is enough — Chat is safe for
// concurrent callers because *http.Client is.
type Client struct {
	Provider string
	APIKey   string
	Model    string
	HTTP     *http.Client
}

// DefaultModelID is the model used when a request carries no explicit model:
// agent.default_model from config.toml when set, otherwise the first [[models]]
// entry the active provider serves — so reordering the file changes the default
// and no model id is hardcoded in Go. Single source of truth: ListModels flags
// the matching entry with IsDefault so the UI preselects the same model.
func DefaultModelID() string {
	if core.Env != nil && core.Env.DEFAULT_MODEL != "" {
		return core.Env.DEFAULT_MODEL
	}
	if models := configuredModels(); len(models) > 0 {
		return models[0].ID
	}
	return ""
}

// NewClient resolves the active provider (providers.model), its API key
// (required) and agent.default_model (optional) from core.Env — same path the rest of
// the backend uses for config.toml. Failing here at startup is much
// friendlier than a 401 on the first user message.
func NewClient() (*Client, error) {
	if core.Env == nil {
		return nil, errors.New("core.Env not populated; call core.PopulateVariables before llm.NewClient")
	}
	provider := ActiveProvider()
	return NewClientForProvider(provider, DefaultModelID())
}

// NewClientForProvider creates a fixed-provider client for system subagents such
// as the global classifier. It does not follow the chat UI's selected model.
func NewClientForProvider(provider, modelID string) (*Client, error) {
	provider = strings.ToLower(strings.TrimSpace(provider))
	if provider == "" {
		provider = ActiveProvider()
	}
	if _, known := providerConfigs[provider]; !known {
		return nil, fmt.Errorf("unsupported model provider %q", provider)
	}
	apiKey := apiKeyForProvider(provider)
	if apiKey == "" {
		return nil, fmt.Errorf("%s not set in config.toml (providers.model=%s)", providerConfigs[provider].KeyName, provider)
	}
	// A registry with no entry for the active provider leaves every request without
	// a model id, which the upstream answers with an opaque 400 — say so at startup,
	// where the fix (a [[models]] entry) is obvious.
	defaultModel := strings.TrimSpace(modelID)
	if defaultModel == "" {
		return nil, fmt.Errorf("no model configured for provider=%s", provider)
	}
	return &Client{
		Provider: provider,
		APIKey:   apiKey,
		Model:    defaultModel,
		HTTP:     &http.Client{Timeout: requestTimeout},
	}, nil
}

// adaptRequestToProvider rewrites the reasoning/routing knobs into the shape
// the target provider understands, so the rest of the codebase only ever deals
// with ReasoningOptions. Called after the per-model registry defaults are
// applied, so registry-supplied budgets get translated too.
func adaptRequestToProvider(provider string, req *ChatRequest) {
	if provider != ProviderMeta {
		// OpenRouter's API is the nested object; the flat field is Meta-only and
		// would be an unknown parameter here.
		req.ReasoningEffort = ""
		return
	}
	if req.ReasoningEffort == "" && req.Reasoning != nil {
		req.ReasoningEffort = metaReasoningEffort(*req.Reasoning)
	}
	// Neither knob exists on Meta's API: `reasoning` is superseded by the flat
	// field, and upstream routing is an OpenRouter-only concept.
	req.Reasoning = nil
	req.Routing = nil
}

// metaReasoningEffort collapses a ReasoningOptions onto Meta's flat
// `reasoning_effort` string. Enabled=false means "don't think", but Muse Spark
// answers HTTP 400 to "none", so it becomes the cheapest accepted budget
// instead. MaxTokens has no Meta equivalent and Exclude is unnecessary there —
// Meta doesn't fold the trace into `content` — so both are dropped. An empty
// Effort leaves the field out and lets the model use its own default.
func metaReasoningEffort(options ReasoningOptions) string {
	if options.Enabled != nil && !*options.Enabled {
		return metaMinimalEffort
	}
	return options.Effort
}

// Chat performs one POST to the active provider's /chat/completions. If
// req.Model is empty we fill it from c.Model so callers don't repeat the model
// on every turn. Non-2xx responses surface the raw body in the error so
// tool-schema or rate-limit problems are immediately readable.
func (c *Client) Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
	if req.Model == "" {
		req.Model = c.Model
	}
	// Fill in any per-model defaults the caller didn't set. Caller values
	// always win; unknown model IDs get no defaults (registry lookup is a
	// no-op for them).
	LookupModel(req.Model).applyDefaults(&req)
	config := providerConfigs[c.Provider]
	adaptRequestToProvider(c.Provider, &req)

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", config.Endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.APIKey)
	for name, value := range config.AnalyticsHeaders {
		httpReq.Header.Set(name, value)
	}

	core.Log("llm.Chat provider::", c.Provider, " model::", req.Model, " messages::", len(req.Messages), " tools::", len(req.Tools), " reasoning_effort::", req.ReasoningEffort)

	// Measure wall-clock from request send to body fully read so the TPS we
	// log later reflects what the user actually waits for, including TTFT.
	startedAt := time.Now()

	resp, err := c.HTTP.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("%s http: %w", c.Provider, err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("%s %d: %s", c.Provider, resp.StatusCode, truncate(string(respBody), 1000))
	}
	// Charge actual wire bytes only after a successful inference response exists.
	if err := core.ChargeInferenceUsage(ctx, len(body), len(respBody)); err != nil {
		return nil, fmt.Errorf("charge inference usage: %w", err)
	}

	var out ChatResponse
	if err := json.Unmarshal(respBody, &out); err != nil {
		return nil, fmt.Errorf("decode response: %w (body=%s)", err, truncate(string(respBody), 500))
	}
	if len(out.Choices) == 0 {
		return nil, fmt.Errorf("%s: no choices in response (body=%s)", c.Provider, truncate(string(respBody), 500))
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
	core.Log("llm.Chat ok provider::", c.Provider, " finish::", out.Choices[0].FinishReason,
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
