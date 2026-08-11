package llm

import (
	"hash/fnv"
	"strconv"
	"strings"

	"app/core"
)

// The registry of models the in-app agent may use lives in config.toml, in the
// [[models]] array table — nothing here is hardcoded, so adding a model is a
// config edit and a restart. Each entry names the provider that serves it,
// because a model id is only routable through its own upstream:
// `muse-spark-1.2-contributor` means nothing to OpenRouter and
// `deepseek/deepseek-v4-flash-0731` means nothing to Meta.
//
// Which model a turn uses is, in order:
//
//  1. the model hash the frontend sends (the picker, resolved by LookupModelHash),
//  2. agent.default_model in config.toml, or
//  3. the first [[models]] entry the active provider serves — file order decides.
//
// Any string is accepted by both upstreams, so an id outside the registry still
// goes through; it just gets no per-model reasoning/routing defaults applied.

// ModelConfig holds the request-level knobs we want to set differently per
// model. Add fields here as new per-model behaviour emerges.
type ModelConfig struct {
	// ID is the model id as the provider names it, e.g.
	// "muse-spark-1.2-contributor" or "deepseek/deepseek-v4-flash-0731".
	ID string
	// Provider is the upstream that serves this model (ProviderMeta or
	// ProviderOpenRouter). ListModels only exposes entries matching the active
	// MODEL_PROVIDER, so the UI can't offer an unroutable model.
	Provider string
	// Reasoning is applied to every Chat() request for this model unless the
	// caller already set it. Nil means "model doesn't support reasoning
	// params" — don't send the field.
	Reasoning *ReasoningOptions
	// Routing biases (sort) or pins (order) which OpenRouter upstream serves this
	// model. Nil = let OpenRouter choose. Meaningless for Meta models, which the
	// provider serves itself.
	Routing *OpenRouterRouting
	// Hash is the short base36 ID the frontend sends instead of the full model id.
	Hash string
	// IsDefault marks the model the backend falls back to when a request
	// carries no model hash. Derived in ListModels from DefaultModelID(), never
	// read from config. The frontend uses it to preselect an entry in the model
	// dropdown.
	IsDefault bool
}

// configuredModels reads the [[models]] entries of config.toml and returns the
// ones the active provider can actually serve, in file order. Filtering here
// (rather than in the UI) means a user can never pick a model id the configured
// key would reject — and the frontend already falls back to the default hash
// when a previously stored selection is no longer in the list.
//
// Rebuilt on each call instead of cached: the list has a handful of entries, and
// a package-level cache would freeze whatever core.Env held the first time it
// was read — which in tests is before PopulateVariables has run.
func configuredModels() []ModelConfig {
	if core.Env == nil {
		return nil
	}
	activeProvider := ActiveProvider()
	models := make([]ModelConfig, 0, len(core.Env.MODELS))
	for _, entry := range core.Env.MODELS {
		// An entry with no provider is an OpenRouter model — same rule as a blank
		// providers.model, so the common case needs one key less per entry.
		provider := strings.ToLower(strings.TrimSpace(entry.Provider))
		if provider == "" {
			provider = ProviderOpenRouter
		}
		if provider != activeProvider {
			continue
		}
		models = append(models, ModelConfig{
			ID:        strings.TrimSpace(entry.ID),
			Provider:  provider,
			Reasoning: reasoningFromConfig(entry.Reasoning),
			Routing:   routingFromConfig(entry.Routing),
			Hash:      ModelIDHash(strings.TrimSpace(entry.ID)),
		})
	}
	return models
}

// reasoningFromConfig / routingFromConfig translate the config shapes into the
// wire shapes. They are separate types on purpose: core owns the file's schema
// and llm owns the provider contract, so neither package's tags leak into the
// other. Nil in, nil out — an omitted block means "don't send the parameter".
func reasoningFromConfig(configured *core.ModelReasoning) *ReasoningOptions {
	if configured == nil {
		return nil
	}
	return &ReasoningOptions{
		Effort:    configured.Effort,
		MaxTokens: configured.MaxTokens,
		Exclude:   configured.Exclude,
		Enabled:   configured.Enabled,
	}
}

func routingFromConfig(configured *core.ModelRouting) *OpenRouterRouting {
	if configured == nil {
		return nil
	}
	return &OpenRouterRouting{
		Order:          configured.Order,
		Sort:           configured.Sort,
		AllowFallbacks: configured.AllowFallbacks,
	}
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

// ListModels is the view the models API returns: the active provider's entries
// in config.toml order, hash-hydrated and with the default flagged so the UI
// preselects the same model the backend would have used anyway.
func ListModels() []ModelConfig {
	models := configuredModels()
	activeDefault := DefaultModelID()
	for index := range models {
		models[index].IsDefault = models[index].ID == activeDefault
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

// LookupModel returns the [[models]] entry for id, or a zero-value config when
// the active provider has no entry for it. Zero-value means "no per-model
// defaults applied" — the request goes upstream exactly as the caller built it.
func LookupModel(id string) ModelConfig {
	for _, cfg := range configuredModels() {
		if cfg.ID == id {
			return cfg
		}
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
