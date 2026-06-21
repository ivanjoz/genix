package llm

import (
	"hash/fnv"
	"sort"
	"strconv"
)

// Registry of OpenRouter models we've validated for the in-app agent loop,
// with their per-model defaults. Switching the active model is one of:
//
//   1. Edit `defaultModel` in openrouter.go (compile-time default), or
//   2. Set OPENROUTER_MODEL in credentials.json to one of the IDs below
//      (runtime override; takes precedence over `defaultModel`).
//
// Any string is accepted by OpenRouter — the registry just ensures we apply
// the right reasoning/temperature knobs for known models. Unknown ids
// pass through with no per-model defaults applied.

// ModelConfig holds the request-level knobs we want to set differently per
// model. Add fields here as new per-model behaviour emerges.
type ModelConfig struct {
	// ID is the OpenRouter slug, e.g. "deepseek/deepseek-v4-flash".
	ID string
	// Reasoning is applied to every Chat() request for this model unless the
	// caller already set it. Nil means "model doesn't support reasoning
	// params" — don't send the field.
	Reasoning *ReasoningOptions
	// Provider pins request routing to a specific OpenRouter provider for
	// this model. Nil means "no pin — let OpenRouter pick the cheapest /
	// fastest available". Use this when a specific provider variant
	// (quantization, region, latency profile) is required.
	Provider *ProviderOptions
	// Notes is free-form documentation for humans reading the registry.
	Notes string
	// Hash is the short base36 ID the frontend sends instead of the full model slug.
	Hash string
}

// pinnedProvider builds a Provider config that pins routing to exactly one
// provider (no fallbacks). Use when a specific provider variant matters
// (e.g. an FP8 quantization or a regional endpoint).
func pinnedProvider(name string) *ProviderOptions {
	allowFallbacks := false
	return &ProviderOptions{Order: []string{name}, AllowFallbacks: &allowFallbacks}
}

// Models is the curated registry. Order is not significant; entries are
// alphabetical by ID for easy diff reading.
var Models = map[string]ModelConfig{
	"deepseek/deepseek-v4-flash": {
		ID: "deepseek/deepseek-v4-flash",
		// Reasoning model. Cap the thinking budget to "low" — most agent
		// turns are simple navigation/inspection and don't need a long
		// chain-of-thought. Exclude=true hides the trace from the response
		// so it doesn't bloat subsequent iterations' prompts.
		Reasoning: &ReasoningOptions{Effort: "low", Exclude: true},
		// FP8 variant on atlas-cloud — picked for consistent low latency.
		Provider: pinnedProvider("deepseek"),
		Notes:    "Reasoning model; effort=low keeps the loop snappy. Pinned to atlas-cloud/fp8.",
	},
	"stepfun/step-3.5-flash": {
		ID: "stepfun/step-3.5-flash",
		// Non-reasoning model — sending `reasoning` would be a no-op at best
		// and an error at worst. Leave nil so the field is omitted.
		Reasoning: nil,
		Provider:  pinnedProvider("deepinfra/fp8"),
		Notes:     "Non-reasoning, fast model. Pinned to first-party StepFun provider.",
	},
	"tencent/hy3-preview": {
		ID: "tencent/hy3-preview",
		// Reasoning is configurable (disabled/low/high) and ACTUALLY honored —
		// unlike DeepSeek V4, which ignores effort:low. Default to low+exclude so
		// callers that don't override (e.g. the chat loop) stay snappy; the page
		// builder pins this model and sets effort per call.
		Reasoning: &ReasoningOptions{Effort: "low", Exclude: true},
		Notes:     "Cheap ($0.063/$0.21), honors disabled/low/high reasoning. Used by the page builder. Provider routing rejects tool_choice=\"required\"; stick to \"auto\".",
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

// ListModels returns a stable, hash-hydrated view for API responses.
func ListModels() []ModelConfig {
	ids := make([]string, 0, len(Models))
	for id := range Models {
		ids = append(ids, id)
	}
	sort.Strings(ids)

	models := make([]ModelConfig, 0, len(ids))
	for _, id := range ids {
		cfg := Models[id]
		cfg.Hash = ModelIDHash(cfg.ID)
		models = append(models, cfg)
	}
	return models
}

// LookupModelHash resolves the frontend's short model key back to an ID.
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
// the request goes to OpenRouter exactly as the caller built it.
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
	if req.Reasoning == nil && cfg.Reasoning != nil {
		copied := *cfg.Reasoning
		req.Reasoning = &copied
	}
	if req.Provider == nil && cfg.Provider != nil {
		copied := *cfg.Provider
		if cfg.Provider.Order != nil {
			copied.Order = append([]string(nil), cfg.Provider.Order...)
		}
		req.Provider = &copied
	}
}
