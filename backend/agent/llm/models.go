package llm

import (
	"hash/fnv"
	"sort"
	"strconv"
)

// Registry of models we've validated for the in-app agent loop, with their
// per-model defaults. Each entry declares which provider serves it, because a
// model id is only routable through its own upstream — `muse-spark-1.2` means
// nothing to OpenRouter and `openai/gpt-5.6-luna` means nothing to Meta.
// Switching the active model is one of:
//
//  1. Change providers.model in config.toml (picks the upstream, and with
//     it the compile-time default in providerConfigs), or
//  2. Set agent.default_model in config.toml to one of the IDs below that
//     belongs to the active provider (runtime override).
//
// Any string is accepted by both upstreams — the registry just ensures we apply
// the right reasoning/routing knobs for known models. Unknown ids pass through
// with no per-model defaults applied.

// ModelConfig holds the request-level knobs we want to set differently per
// model. Add fields here as new per-model behaviour emerges.
type ModelConfig struct {
	// ID is the model id as the provider names it, e.g. "muse-spark-1.2" or
	// "deepseek/deepseek-v4-flash".
	ID string
	// Provider is the upstream that serves this model (ProviderMeta or
	// ProviderOpenRouter). ListModels only exposes entries matching the active
	// MODEL_PROVIDER, so the UI can't offer an unroutable model.
	Provider string
	// Reasoning is applied to every Chat() request for this model unless the
	// caller already set it. Nil means "model doesn't support reasoning
	// params" — don't send the field.
	Reasoning *ReasoningOptions
	// Routing pins request routing to a specific OpenRouter upstream for this
	// model. Nil means "no pin — let OpenRouter pick the cheapest / fastest
	// available". Use this when a specific upstream variant (quantization,
	// region, latency profile) is required. Meaningless for Meta models.
	Routing *OpenRouterRouting
	// Notes is free-form documentation for humans reading the registry.
	Notes string
	// Hash is the short base36 ID the frontend sends instead of the full model id.
	Hash string
	// IsDefault marks the model the backend falls back to when a request
	// carries no model hash. Derived in ListModels from DefaultModelID() —
	// don't set it in the registry literal. The frontend uses it to preselect
	// an entry in the model dropdown.
	IsDefault bool
}

// pinnedOpenRouterUpstream builds a Routing config that pins to exactly one
// OpenRouter upstream (no fallbacks). Use when a specific variant matters
// (e.g. an FP8 quantization or a regional endpoint).
func pinnedOpenRouterUpstream(name string) *OpenRouterRouting {
	allowFallbacks := false
	return &OpenRouterRouting{Order: []string{name}, AllowFallbacks: &allowFallbacks}
}

// Models is the curated registry. Order is not significant; entries are
// alphabetical by ID for easy diff reading.
var Models = map[string]ModelConfig{
	// ── Meta Model API (MODEL_PROVIDER=meta) ────────────────────────────────
	// Muse Spark is a reasoning family with a 1M-token context window. Effort
	// is set but Exclude/MaxTokens are deliberately omitted: Meta's flat
	// `reasoning_effort` has no equivalent for either, and it doesn't fold the
	// trace into `content`, so excluding it is already the default behaviour.
	"muse-spark-1.1": {
		ID:        "muse-spark-1.1",
		Provider:  ProviderMeta,
		Reasoning: &ReasoningOptions{Effort: "medium"},
		Notes:     "Muse Spark 1.1 — first Muse release, 1M context. Reasoning effort=medium. Cheaper/older than 1.2.",
	},
	"muse-spark-1.2": {
		ID:        "muse-spark-1.2",
		Provider:  ProviderMeta,
		Reasoning: &ReasoningOptions{Effort: "medium"},
		Notes:     "Muse Spark 1.2 standard tier, 1M context. Reasoning effort=medium — enough for multi-step tool sequences without the latency of high.",
	},
	"muse-spark-1.2-contributor": {
		ID:       "muse-spark-1.2-contributor",
		Provider: ProviderMeta,
		// Default agent model when MODEL_PROVIDER=meta (see providerConfigs in
		// client.go). Same weights as 1.2 on the contributor tier.
		Reasoning: &ReasoningOptions{Effort: "medium"},
		Notes:     "Default agent model on Meta. Muse Spark 1.2 contributor tier, 1M context. Reasoning effort=medium.",
	},

	// ── OpenRouter (MODEL_PROVIDER=openrouter) ──────────────────────────────
	"deepseek/deepseek-v4-flash": {
		ID:       "deepseek/deepseek-v4-flash",
		Provider: ProviderOpenRouter,
		// Reasoning model. Cap the thinking budget to "low" — most agent
		// turns are simple navigation/inspection and don't need a long
		// chain-of-thought. Exclude=true hides the trace from the response
		// so it doesn't bloat subsequent iterations' prompts.
		Reasoning: &ReasoningOptions{Effort: "low", Exclude: true},
		Routing:   pinnedOpenRouterUpstream("deepseek"),
		Notes:     "Reasoning model; effort=low keeps the loop snappy. Pinned to the first-party DeepSeek upstream.",
	},
	"openai/gpt-5.6-luna": {
		ID:       "openai/gpt-5.6-luna",
		Provider: ProviderOpenRouter,
		// Default agent model when MODEL_PROVIDER=openrouter. Reasoning-capable
		// and honors the effort knob — "medium" is the balance point: enough
		// deliberation for multi-step tool sequences without the latency of
		// "high". Exclude=true keeps the trace out of the response so it doesn't
		// bloat the next iteration's prompt.
		Reasoning: &ReasoningOptions{Effort: "medium", Exclude: true},
		Notes:     "Default agent model on OpenRouter. Reasoning effort=medium, trace excluded. No upstream pin — let OpenRouter route.",
	},
	"stepfun/step-3.5-flash": {
		ID:       "stepfun/step-3.5-flash",
		Provider: ProviderOpenRouter,
		// Non-reasoning model — sending `reasoning` would be a no-op at best
		// and an error at worst. Leave nil so the field is omitted.
		Reasoning: nil,
		Routing:   pinnedOpenRouterUpstream("deepinfra/fp8"),
		Notes:     "Non-reasoning, fast model. Pinned to the DeepInfra FP8 upstream.",
	},
	"tencent/hy3-preview": {
		ID:       "tencent/hy3-preview",
		Provider: ProviderOpenRouter,
		// Reasoning is configurable (disabled/low/high) and ACTUALLY honored —
		// unlike DeepSeek V4, which ignores effort:low. Default to low+exclude so
		// callers that don't override (e.g. the chat loop) stay snappy; the page
		// builder sets effort per call.
		Reasoning: &ReasoningOptions{Effort: "low", Exclude: true},
		Notes:     "Cheap ($0.063/$0.21), honors disabled/low/high reasoning. Upstream routing rejects tool_choice=\"required\"; stick to \"auto\".",
	},
}

// ModelIDHash keeps the UI payload compact while preserving a deterministic
// mapping back to the validated registry IDs.
func ModelIDHash(id string) string {
	hash := fnv.New32a()
	_, _ = hash.Write([]byte(id))
	hashValue := int32(hash.Sum32() & 0x7fffffff)
	if hashValue == 0 {
		hashValue = 1
	}
	return strconv.FormatInt(int64(hashValue), 36)
}

// ListModels returns a stable, hash-hydrated view for API responses, limited to
// the models the active provider can actually serve. Filtering here (rather
// than in the UI) means a user can never pick a model id the configured key
// would reject — and the frontend already falls back to the default hash when a
// previously stored selection is no longer in the list.
func ListModels() []ModelConfig {
	activeProvider := ActiveProvider()
	ids := make([]string, 0, len(Models))
	for id, cfg := range Models {
		if cfg.Provider == activeProvider {
			ids = append(ids, id)
		}
	}
	sort.Strings(ids)

	activeDefault := DefaultModelID()
	models := make([]ModelConfig, 0, len(ids))
	for _, id := range ids {
		cfg := Models[id]
		cfg.Hash = ModelIDHash(cfg.ID)
		cfg.IsDefault = cfg.ID == activeDefault
		models = append(models, cfg)
	}
	return models
}

// LookupModelHash resolves the frontend's short model key back to an ID. Only
// the active provider's models resolve — a stale hash from another provider is
// reported as invalid, which is what we want: it can't be routed.
func LookupModelHash(hash string) (ModelConfig, bool) {
	for _, cfg := range ListModels() {
		if cfg.Hash == hash {
			return cfg, true
		}
	}
	return ModelConfig{}, false
}

// LookupModel returns the registry entry for id, or a zero-value config if
// the id isn't known. Zero-value means "no per-model defaults applied" —
// the request goes upstream exactly as the caller built it.
func LookupModel(id string) ModelConfig {
	if cfg, ok := Models[id]; ok {
		cfg.Hash = ModelIDHash(cfg.ID)
		return cfg
	}
	return ModelConfig{ID: id}
}

// applyDefaults fills in fields on req from cfg only when the caller left
// them unset. Caller-supplied values always win — the registry is the
// fallback, not a force-override. Every default is copied so a caller
// mutating the returned request can't poison the registry entry.
func (cfg ModelConfig) applyDefaults(req *ChatRequest) {
	if req.Reasoning == nil && req.ReasoningEffort == "" && cfg.Reasoning != nil {
		copied := *cfg.Reasoning
		req.Reasoning = &copied
	}
	if req.Routing == nil && cfg.Routing != nil {
		copied := *cfg.Routing
		if cfg.Routing.Order != nil {
			copied.Order = append([]string(nil), cfg.Routing.Order...)
		}
		req.Routing = &copied
	}
}
