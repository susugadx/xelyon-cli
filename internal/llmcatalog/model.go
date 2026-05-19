package llmcatalog

import "strings"

// ModelLimit はモデル名 prefix とトークン上限の組を表す。
type ModelLimit struct {
	Pattern string
	Limit   int
}

var knownModelMaxOutputTokens = map[string]int{
	"deepseek-chat":                         384000,
	"deepseek-coder":                        16384,
	"deepseek-reasoner":                     384000,
	"deepseek-v4-flash":                     384000,
	"deepseek-v4-pro":                       384000,
	"kimi-k2":                               32768,
	"kimi-k2.5":                             32768,
	"kimi-k2.6":                             32768,
	"kimi-k2-thinking":                      32768,
	"claude-sonnet-4-6":                     64000,
	"claude-sonnet-4.6":                     64000,
	"anthropic.claude-sonnet-4-6":           64000,
	"global.anthropic.claude-sonnet-4-6":    64000,
	"us.anthropic.claude-sonnet-4-6":        64000,
	"eu.anthropic.claude-sonnet-4-6":        64000,
	"au.anthropic.claude-sonnet-4-6":        64000,
	"claude-sonnet-4-5":                     64000,
	"claude-sonnet-4.5":                     64000,
	"claude-opus-4-7":                       128000,
	"claude-opus-4.7":                       128000,
	"global.anthropic.claude-opus-4-7-v1":   128000,
	"global.anthropic.claude-opus-4-7-v1:0": 128000,
	"claude-opus-4-6":                       128000,
	"claude-opus-4-5":                       64000,
	"gpt-5.5":                               128000,
	"gpt-5.5-2026-04-23":                    128000,
	"gpt-5.5-pro":                           128000,
	"gpt-5.5-pro-2026-04-23":                128000,
	"gpt-5.3-codex":                         128000,
	"gpt-5.2":                               16384,
	"gemini-2.5-flash":                      65536,
	"gemini-3.1-pro":                        65536,
	"gemini-3.1-pro-preview":                65536,
	"gemini-3.1-pro-preview-customtools":    65536,
}

var modelContextLimits = map[string]int{
	"claude-opus-4-7":            1000000,
	"claude-opus-4.7":            1000000,
	"claude-sonnet-4-6":          200000,
	"claude-sonnet-4.6":          200000,
	"claude-sonnet-4.5":          200000,
	"claude-opus-4-6":            200000,
	"claude-sonnet-4-20250514":   200000,
	"claude-sonnet-4-5-20250514": 200000,
	"claude-opus-4-20250514":     200000,
	"claude-3-5-sonnet-20241022": 200000,
	"claude-3-5-haiku-20241022":  200000,
	"claude-3-opus-20240229":     200000,
	"claude-3-sonnet-20240229":   200000,
	"claude-3-haiku-20240307":    200000,

	"global.anthropic.claude-sonnet-4-6":               200000,
	"us.anthropic.claude-sonnet-4-6":                   200000,
	"eu.anthropic.claude-sonnet-4-6":                   200000,
	"au.anthropic.claude-sonnet-4-6":                   200000,
	"global.anthropic.claude-opus-4-7-v1":              1000000,
	"global.anthropic.claude-opus-4-7-v1:0":            1000000,
	"anthropic.claude-sonnet-4-6":                      200000,
	"global.anthropic.claude-opus-4-5-20251101-v1:0":   200000,
	"us.anthropic.claude-sonnet-4-20250514-v1:0":       200000,
	"us.anthropic.claude-haiku-4-5-20251001-v1:0":      200000,
	"anthropic.claude-opus-4-20250514-v1:0":            200000,
	"global.anthropic.claude-sonnet-4-5-20250929-v1:0": 200000,

	"gpt-4.1":                1000000,
	"gpt-4.1-mini":           1000000,
	"gpt-4.1-nano":           1000000,
	"gpt-4o":                 128000,
	"gpt-4o-mini":            128000,
	"gpt-4-turbo":            128000,
	"gpt-4-turbo-preview":    128000,
	"gpt-4":                  8192,
	"gpt-4-32k":              32768,
	"gpt-3.5-turbo":          16385,
	"o1":                     200000,
	"o1-mini":                128000,
	"o1-preview":             128000,
	"o3-mini":                200000,
	"gpt-5":                  400000,
	"gpt-5.5":                1050000,
	"gpt-5.5-2026-04-23":     1050000,
	"gpt-5.5-pro":            1050000,
	"gpt-5.5-pro-2026-04-23": 1050000,
	"gpt-5.4":                1000000,
	"gpt-5.4-pro":            1000000,
	"gpt-5.4-mini":           400000,
	"gpt-5.4-nano":           400000,
	"gpt-5.3-codex":          400000,
	"gpt-5.1":                400000,
	"gpt-5.2":                400000,

	"gemini-3-pro-preview":               1000000,
	"gemini-3.1-pro":                     1000000,
	"gemini-3.1-pro-preview":             1000000,
	"gemini-3.1-pro-preview-customtools": 1000000,
	"gemini-2.0-flash":                   1000000,
	"gemini-2.0-flash-exp":               1000000,
	"gemini-2.5-flash":                   1000000,
	"gemini-2.5-pro":                     1000000,
	"gemini-2.5-pro-preview":             1000000,
	"gemini-1.5-pro":                     2000000,
	"gemini-1.5-flash":                   1000000,
	"gemini-exp-1206":                    2000000,
	"gemini-exp-1121":                    2000000,
	"gemini-pro":                         32768,

	"deepseek-chat":     1000000,
	"deepseek-coder":    64000,
	"deepseek-reasoner": 1000000,
	"deepseek-v4-flash": 1000000,
	"deepseek-v4-pro":   1000000,

	"kimi-k2.5":        256000,
	"kimi-k2.6":        256000,
	"kimi-k2-thinking": 256000,

	"llama-3.3-70b-versatile": 128000,
	"llama-3.1-70b-versatile": 128000,
	"llama-3.1-8b-instant":    128000,
	"mixtral-8x7b-32768":      32768,
	"gemma2-9b-it":            8192,

	"qwen2.5-coder:7b":  32768,
	"qwen2.5-coder:14b": 32768,
	"qwen2.5-coder:32b": 32768,
	"codellama:7b":      16384,
	"codellama:13b":     16384,
	"codellama:34b":     16384,
	"deepseek-coder-v2": 128000,
	"llama3:8b":         8192,
	"llama3:70b":        8192,
	"mistral:7b":        32768,

	"default": 100000,
}

var modelMaxOutputTokenPrefixes = []ModelLimit{
	{Pattern: "llama-4-scout", Limit: 8192},
	{Pattern: "deepseek-v4", Limit: 384000},
}

var modelContextLimitPrefixes = []ModelLimit{
	{Pattern: "us.anthropic.claude", Limit: 200000},
	{Pattern: "eu.anthropic.claude", Limit: 200000},
	{Pattern: "au.anthropic.claude", Limit: 200000},
	{Pattern: "anthropic.claude", Limit: 200000},
	{Pattern: "global.anthropic.claude", Limit: 200000},
	{Pattern: "claude-", Limit: 200000},
	{Pattern: "gpt-4.1", Limit: 1000000},
	{Pattern: "gpt-4o", Limit: 128000},
	{Pattern: "gpt-4", Limit: 8192},
	{Pattern: "gpt-3.5", Limit: 16385},
	{Pattern: "gpt-5.4-mini", Limit: 400000},
	{Pattern: "gpt-5.4-nano", Limit: 400000},
	{Pattern: "gpt-5.4", Limit: 1000000},
	{Pattern: "gpt-5.5-pro", Limit: 1050000},
	{Pattern: "gpt-5.5", Limit: 1050000},
	{Pattern: "gpt-5", Limit: 400000},
	{Pattern: "o1", Limit: 200000},
	{Pattern: "o3", Limit: 200000},
	{Pattern: "gemini-3", Limit: 1000000},
	{Pattern: "gemini-2", Limit: 1000000},
	{Pattern: "gemini-1.5", Limit: 1000000},
	{Pattern: "gemini-pro", Limit: 32768},
	{Pattern: "deepseek-chat", Limit: 1000000},
	{Pattern: "deepseek-reasoner", Limit: 1000000},
	{Pattern: "deepseek-v4", Limit: 1000000},
	{Pattern: "deepseek-v3", Limit: 128000},
	{Pattern: "deepseek-r1", Limit: 128000},
	{Pattern: "deepseek-coder", Limit: 64000},
	{Pattern: "llama-4-scout", Limit: 131072},
	{Pattern: "llama-3", Limit: 128000},
	{Pattern: "llama3", Limit: 8192},
	{Pattern: "qwen", Limit: 32768},
	{Pattern: "mixtral", Limit: 32768},
	{Pattern: "codellama", Limit: 16384},
}

// InferProviderFromModel はモデル名から provider を推定する。
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
	case strings.HasPrefix(normalized, "deepseek."):
		return "bedrock"
	case strings.HasPrefix(normalized, "deepseek"):
		return "deepseek"
	case strings.HasPrefix(normalized, "kimi-"):
		return "kimi"
	case IsBedrockModelID(normalized):
		return "bedrock"
	case strings.Contains(normalized, "/"):
		return "openrouter"
	default:
		return ""
	}
}

// KnownMaxOutputTokens は既知モデルの最大出力トークン数を返す。
func KnownMaxOutputTokens(model string) (int, bool) {
	for _, candidate := range modelLimitLookupCandidates(model) {
		tokens, ok := knownMaxOutputTokensForModel(candidate)
		if ok {
			return tokens, true
		}
	}
	return 0, false
}

// IsKnownModelName は組み込み catalog が model を直接知っているか返す。
func IsKnownModelName(model string) bool {
	for _, candidate := range modelLimitLookupCandidates(model) {
		if isKnownModelLimitName(candidate) {
			return true
		}
	}
	return false
}

// KnownModelContextLimit は既知モデルのコンテキスト上限を返す。
func KnownModelContextLimit(model string) (int, bool) {
	for _, candidate := range modelLimitLookupCandidates(model) {
		limit, ok := knownModelContextLimitForModel(candidate)
		if ok {
			return limit, true
		}
	}
	return 0, false
}

func modelLimitLookupCandidates(model string) []string {
	model = normalizeModelName(model)
	if model == "" || model == "default" {
		return nil
	}

	candidates := []string{model}
	if _, delegatedModel, ok := strings.Cut(model, "/"); ok {
		delegatedModel = strings.TrimSpace(delegatedModel)
		if delegatedModel != "" && delegatedModel != model {
			candidates = append(candidates, delegatedModel)
		}
	}
	return candidates
}

func knownMaxOutputTokensForModel(model string) (int, bool) {
	tokens, ok := knownModelMaxOutputTokens[model]
	if ok {
		return tokens, true
	}
	if isClaudeOpus47ModelName(model) {
		return 128000, true
	}
	if tokens, ok := knownBedrockMaxOutputTokens(model); ok {
		return tokens, true
	}
	for _, rule := range modelMaxOutputTokenPrefixes {
		if strings.HasPrefix(model, rule.Pattern) {
			return rule.Limit, true
		}
	}
	return 0, false
}

func knownModelContextLimitForModel(model string) (int, bool) {
	if limit, ok := modelContextLimits[model]; ok {
		return limit, true
	}
	if isClaudeOpus47ModelName(model) {
		return 1000000, true
	}

	for _, rule := range modelContextLimitPrefixes {
		if strings.HasPrefix(model, rule.Pattern) {
			return rule.Limit, true
		}
	}

	return 0, false
}

func isKnownModelLimitName(model string) bool {
	if model == "" || model == "default" {
		return false
	}
	if _, ok := knownMaxOutputTokensForModel(model); ok {
		return true
	}
	if _, ok := modelContextLimits[model]; ok {
		return true
	}
	if isClaudeOpus47ModelName(model) {
		return true
	}
	if _, ok := knownBedrockMaxOutputTokens(model); ok {
		return true
	}
	return false
}

// ModelContextLimit はモデルのコンテキスト上限を返す。
func ModelContextLimit(model string) int {
	if limit, ok := KnownModelContextLimit(model); ok {
		return limit
	}
	return modelContextLimits["default"]
}

// IsOpenAIResponsesModel はモデルが OpenAI Responses API を使うか返す。
func IsOpenAIResponsesModel(model string, extraModels []string) bool {
	model = strings.TrimSpace(model)
	if strings.HasPrefix(model, "gpt-5") ||
		strings.HasPrefix(model, "gpt-4o") ||
		strings.HasPrefix(model, "o1") ||
		strings.HasPrefix(model, "o3") ||
		strings.HasPrefix(model, "o4") {
		return true
	}

	for _, extra := range extraModels {
		if extra == model {
			return true
		}
	}
	return false
}

// IsAdaptiveClaudeThinkingModel は adaptive thinking を使う Claude モデルか返す。
func IsAdaptiveClaudeThinkingModel(model string) bool {
	m := normalizeModelName(model)
	return isClaudeOpus47ModelName(m) ||
		strings.Contains(m, "claude-opus-4-6") ||
		strings.Contains(m, "claude-sonnet-4-6") ||
		strings.Contains(m, "claude-opus-4.6") ||
		strings.Contains(m, "claude-sonnet-4.6")
}

func normalizeModelName(model string) string {
	return strings.ToLower(strings.TrimSpace(model))
}

func isClaudeOpus47ModelName(model string) bool {
	return strings.Contains(model, "claude-opus-4-7") ||
		strings.Contains(model, "claude-opus-4.7")
}

func isOpenAIReasoningModelName(model string) bool {
	return strings.HasPrefix(model, "o1") ||
		strings.HasPrefix(model, "o3") ||
		strings.HasPrefix(model, "o4")
}
