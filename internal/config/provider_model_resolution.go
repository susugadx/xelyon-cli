package config

import "strings"

func defaultProviderModelConfig(provider string) (ProviderModelConfig, bool) {
	keys := ProviderModelLookupKeys(provider)
	if len(keys) == 0 {
		return ProviderModelConfig{}, false
	}

	defaults := DefaultConfig()
	merged := ProviderModelConfig{}
	found := false
	for i := len(keys) - 1; i >= 0; i-- {
		key := keys[i]
		if pm, ok := defaults.ProviderModels[key]; ok {
			merged = mergeProviderModelConfig(merged, pm)
			found = true
		}
	}
	return merged, found
}

func selectProviderModelKey(src map[string]ProviderModelConfig, provider string) (string, bool) {
	keys := ProviderModelLookupKeys(provider)
	if len(keys) == 0 || src == nil {
		return "", false
	}

	for _, key := range keys {
		if _, ok := src[key]; ok {
			return key, true
		}
	}

	return "", false
}

func (c *Config) explicitProviderModelSource() map[string]ProviderModelConfig {
	if c == nil {
		return nil
	}
	return c.providerModelsStore.explicitSource(c.effectiveProviderModels())
}

func (c *Config) explicitProviderModelSelection(provider string) (map[string]ProviderModelConfig, string, bool) {
	if c == nil {
		return nil, "", false
	}

	source := c.explicitProviderModelSource()
	key, ok := selectProviderModelKey(source, provider)
	if !ok {
		return source, "", false
	}
	return source, key, true
}

func (c *Config) rawExplicitProviderModelConfig(provider string) (ProviderModelConfig, bool) {
	source, selectedKey, ok := c.explicitProviderModelSelection(provider)
	if !ok {
		return ProviderModelConfig{}, false
	}
	pm, ok := source[selectedKey]
	if !ok {
		return ProviderModelConfig{}, false
	}
	return cloneProviderModelConfig(pm), true
}

func (c *Config) selectedProviderModelLookupKey(provider string) (string, bool) {
	if c == nil {
		return "", false
	}
	if _, key, ok := c.explicitProviderModelSelection(provider); ok {
		return key, true
	}
	return selectProviderModelKey(c.effectiveProviderModels(), provider)
}

// GetProviderDefaultModel は provider の実効デフォルトモデルを返す。
// provider_models の明示 override があればそれを使い、無ければ provider built-in default を返す。
func (c *Config) GetProviderDefaultModel(provider string) string {
	if c == nil {
		return ""
	}

	if pm, ok := c.GetProviderModelConfig(provider); ok {
		return pm.DefaultModel
	}

	return ""
}

// GetModelForProvider is kept as a compatibility alias for GetProviderDefaultModel.
func (c *Config) GetModelForProvider(provider string) string {
	return c.GetProviderDefaultModel(provider)
}

// GetEffectiveModelForProvider は merged provider config から実行時に使う既定モデルを返す。
// provider_models section の有無にかかわらず provider default を含む実効値を返す。
func (c *Config) GetEffectiveModelForProvider(provider string) string {
	if c == nil {
		return ""
	}
	if pm, ok := c.GetProviderModelConfig(provider); ok {
		return pm.DefaultModel
	}
	return ""
}

// GetSelectedModelForProvider は直接その provider を使うときのモデル解決を返す。
// 優先順位:
// 1. explicit provider_models.<provider>.default_model
// 2. global default_model (default provider のみ。provider と両立すると判断できる場合)
// 3. provider default
func (c *Config) GetSelectedModelForProvider(provider string) string {
	if c == nil {
		return ""
	}

	if pm, ok := c.rawExplicitProviderModelConfig(provider); ok && pm.DefaultModel != "" {
		return pm.DefaultModel
	}
	if SameProviderRuntimeIdentity(provider, c.DefaultProvider) && c.DefaultModel != "" {
		if c.configuredDefaultModelAppliesToProvider(provider, c.DefaultModel) {
			return c.DefaultModel
		}
	}
	if model := c.GetEffectiveModelForProvider(provider); model != "" {
		return model
	}
	return ""
}

// ResolveModelForProvider is kept as a compatibility alias for GetSelectedModelForProvider.
func (c *Config) ResolveModelForProvider(provider string) string {
	return c.GetSelectedModelForProvider(provider)
}

// ValidateModelForProvider は任意のモデル名を受け付ける（後方互換のため残す）
// 注: v0.16.0以降、モデル名の検証は行わない
func (c *Config) ValidateModelForProvider(provider, model string) bool {
	_, ok := c.GetProviderModelConfig(provider)
	return ok
}

// GetProviderModelConfig は provider_models を alias fallback 付きで取得する。
// exact requested key を最優先し、存在しない場合のみ sibling alias へ fallback する。
func (c *Config) GetProviderModelConfig(provider string) (ProviderModelConfig, bool) {
	selectedKey, ok := c.selectedProviderModelLookupKey(provider)
	if !ok {
		return ProviderModelConfig{}, false
	}

	merged := ProviderModelConfig{}
	if pm, ok := defaultProviderModelConfig(selectedKey); ok {
		merged = pm
	}
	if c != nil {
		if pm, ok := c.providerModelsStore.rawConfig(selectedKey); ok {
			return mergeProviderModelConfig(merged, pm), true
		}
		if pm, ok := c.effectiveProviderModelConfig(selectedKey); ok {
			return mergeProviderModelConfig(merged, pm), true
		}
	}

	return merged, true
}

// GetExplicitProviderModelConfig は明示設定された provider_models entry を返す。
// exact requested key を最優先し、存在しない場合のみ sibling alias へ fallback する。
func (c *Config) GetExplicitProviderModelConfig(provider string) (ProviderModelConfig, bool) {
	source, selectedKey, ok := c.explicitProviderModelSelection(provider)
	if !ok {
		return ProviderModelConfig{}, false
	}

	merged := ProviderModelConfig{}
	if pm, ok := defaultProviderModelConfig(selectedKey); ok {
		merged = pm
	}
	if pm, ok := source[selectedKey]; ok {
		return mergeProviderModelConfig(merged, pm), true
	}
	return merged, true
}

func providerConfigCandidates() []string {
	seen := map[string]bool{}
	candidates := make([]string, 0, len(ValidProviders))
	for _, provider := range ValidProviders {
		normalized := NormalizeProviderName(provider)
		if normalized == "" || seen[normalized] {
			continue
		}
		seen[normalized] = true
		candidates = append(candidates, normalized)
	}
	return candidates
}

func orderedProviderConfigCandidates(defaultProvider string) []string {
	seen := map[string]bool{}
	ordered := make([]string, 0, len(ValidProviders))

	for _, candidate := range ProviderModelLookupKeys(defaultProvider) {
		if candidate == "" || seen[candidate] {
			continue
		}
		seen[candidate] = true
		ordered = append(ordered, candidate)
	}

	for _, candidate := range providerConfigCandidates() {
		if seen[candidate] {
			continue
		}
		seen[candidate] = true
		ordered = append(ordered, candidate)
	}

	return ordered
}

// FindProviderByDefaultModel returns the first provider whose resolved default model matches model.
func (c *Config) FindProviderByDefaultModel(model string) string {
	if c == nil {
		return ""
	}
	for _, provider := range orderedProviderConfigCandidates(c.DefaultProvider) {
		if c.GetProviderDefaultModel(provider) == model {
			return provider
		}
	}
	return ""
}

// FindProviderBySelectedModel returns the first provider whose selected-model resolution matches model.
// The configured default provider spelling is checked first, but sibling aliases remain searchable afterward.
func (c *Config) FindProviderBySelectedModel(model string) string {
	if c == nil {
		return ""
	}

	model = strings.TrimSpace(model)
	if model == "" {
		return ""
	}

	for _, providerName := range orderedProviderConfigCandidates(c.DefaultProvider) {
		if c.GetSelectedModelForProvider(providerName) == model {
			return providerName
		}
	}

	return ""
}

func (c *Config) selectedModelOwnerWithinRuntimeIdentity(provider, model string) string {
	if c == nil {
		return ""
	}

	model = strings.TrimSpace(model)
	if model == "" {
		return ""
	}

	for _, candidate := range ProviderModelLookupKeys(provider) {
		if c.GetSelectedModelForProvider(candidate) == model {
			return candidate
		}
	}

	return ""
}

// RuntimeProviderConfigKey returns the config-side provider key that should own runtime settings
// for a provider/model pair. This keeps model-adjacent settings aligned with selected-model ownership.
func (c *Config) RuntimeProviderConfigKey(provider, model string) string {
	if c == nil {
		return FallbackProviderConfigKey(provider, "")
	}
	if owner := c.selectedModelOwnerWithinRuntimeIdentity(provider, model); owner != "" {
		return owner
	}
	if owner := c.FindProviderBySelectedModel(model); owner != "" && SameProviderRuntimeIdentity(owner, provider) {
		return owner
	}
	return FallbackProviderConfigKey(provider, c.DefaultProvider)
}

// ResolveProviderForModel resolves the runtime provider that should own model.
// Resolution prefers config-selected models, then provider defaults, then name-based inference.
func (c *Config) ResolveProviderForModel(currentProvider, model string) string {
	currentProvider = CanonicalProviderName(currentProvider)
	model = strings.TrimSpace(model)
	if model == "" {
		return currentProvider
	}

	if providerName := c.FindProviderBySelectedModel(model); providerName != "" {
		return CanonicalProviderName(providerName)
	}
	if providerName := c.FindProviderByDefaultModel(model); providerName != "" {
		return CanonicalProviderName(providerName)
	}
	if inferred := InferProviderFromModel(model); inferred != "" {
		return CanonicalProviderName(inferred)
	}
	return currentProvider
}

func (c *Config) configuredDefaultModelAppliesToProvider(provider, model string) bool {
	if model == "" {
		return false
	}

	if c != nil {
		matchedAnyProvider := false
		for _, candidate := range providerConfigCandidates() {
			pm, ok := c.GetProviderModelConfig(candidate)
			if !ok || pm.DefaultModel != model {
				continue
			}
			matchedAnyProvider = true
			if SameProviderRuntimeIdentity(provider, candidate) {
				return true
			}
		}
		if matchedAnyProvider {
			return false
		}
	}

	if providerTreatsConfiguredDefaultModelAsOpaque(provider, model) {
		return true
	}

	inferredProvider := InferProviderFromModel(model)
	if inferredProvider == "" {
		return true
	}
	return SameProviderRuntimeIdentity(provider, inferredProvider)
}

// InferProviderFromModel はモデル名から provider を推定する。
// 推定できない場合は空文字を返す。
func InferProviderFromModel(model string) string {
	normalized := strings.ToLower(strings.TrimSpace(model))
	if normalized == "" {
		return ""
	}

	switch {
	case strings.HasPrefix(normalized, "gpt-"),
		normalized == "codex-mini",
		normalized == "codex",
		isOpenAIReasoningModelName(normalized):
		return "openai"
	case strings.HasPrefix(normalized, "gemini"):
		return "gemini"
	case strings.HasPrefix(normalized, "claude"):
		return "claude"
	case strings.HasPrefix(normalized, "deepseek"):
		return "deepseek"
	case strings.HasPrefix(normalized, "global.anthropic."):
		return "bedrock"
	case strings.Contains(normalized, "/"):
		return "openrouter"
	default:
		return ""
	}
}

func isOpenAIReasoningModelName(model string) bool {
	switch {
	case strings.HasPrefix(model, "o1"),
		strings.HasPrefix(model, "o3"),
		strings.HasPrefix(model, "o4"):
		return true
	default:
		return false
	}
}

func providerTreatsConfiguredDefaultModelAsOpaque(provider, model string) bool {
	switch CanonicalProviderName(provider) {
	case "ollama":
		return true
	case "groq":
		return strings.Contains(strings.TrimSpace(model), "/")
	default:
		return false
	}
}
