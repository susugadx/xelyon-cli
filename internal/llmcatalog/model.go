package llmcatalog

import "strings"

// ModelLimit はモデル名 prefix とトークン上限の組を表す。
type ModelLimit struct {
	Pattern string
	Limit   int
}

var knownModelMaxOutputTokens = map[string]int{
	"deepseek-chat":                      8192,
	"deepseek-reasoner":                  64000,
	"claude-sonnet-4-6":                  64000,
	"claude-sonnet-4-5":                  64000,
	"claude-opus-4-6":                    128000,
	"claude-opus-4-5":                    64000,
	"gpt-5.5":                            128000,
	"gpt-5.5-2026-04-23":                 128000,
	"gpt-5.5-pro":                        128000,
	"gpt-5.5-pro-2026-04-23":             128000,
	"gpt-5.2":                            16384,
	"gemini-2.5-flash":                   65536,
	"gemini-3.1-pro-preview":             65536,
	"gemini-3.1-pro-preview-customtools": 65536,
}

var modelContextLimits = map[string]int{
	"claude-sonnet-4-6":          200000,
	"claude-opus-4-6":            200000,
	"claude-sonnet-4-20250514":   200000,
	"claude-sonnet-4-5-20250514": 200000,
	"claude-opus-4-20250514":     200000,
	"claude-3-5-sonnet-20241022": 200000,
	"claude-3-5-haiku-20241022":  200000,
	"claude-3-opus-20240229":     200000,
	"claude-3-sonnet-20240229":   200000,
	"claude-3-haiku-20240307":    200000,

	"global.anthropic.claude-sonnet-4-6-v1":            200000,
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
	"gpt-5.1":                400000,
	"gpt-5.2":                400000,

	"gemini-3-pro-preview":               1000000,
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

	"deepseek-chat":     128000,
	"deepseek-coder":    64000,
	"deepseek-reasoner": 128000,

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

var modelContextLimitPrefixes = []ModelLimit{
	{Pattern: "us.anthropic.claude", Limit: 200000},
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
	{Pattern: "deepseek-chat", Limit: 128000},
	{Pattern: "deepseek-reasoner", Limit: 128000},
	{Pattern: "deepseek-v3", Limit: 128000},
	{Pattern: "deepseek-r1", Limit: 128000},
	{Pattern: "deepseek-coder", Limit: 64000},
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

// KnownMaxOutputTokens は既知モデルの最大出力トークン数を返す。
func KnownMaxOutputTokens(model string) (int, bool) {
	tokens, ok := knownModelMaxOutputTokens[model]
	return tokens, ok
}

// ModelContextLimit はモデルのコンテキスト上限を返す。
func ModelContextLimit(model string) int {
	if limit, ok := modelContextLimits[model]; ok {
		return limit
	}

	for _, rule := range modelContextLimitPrefixes {
		if strings.HasPrefix(model, rule.Pattern) {
			return rule.Limit
		}
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
	m := strings.ToLower(model)
	return strings.Contains(m, "claude-opus-4-6") ||
		strings.Contains(m, "claude-sonnet-4-6") ||
		strings.Contains(m, "claude-opus-4.6") ||
		strings.Contains(m, "claude-sonnet-4.6")
}

func isOpenAIReasoningModelName(model string) bool {
	return strings.HasPrefix(model, "o1") ||
		strings.HasPrefix(model, "o3") ||
		strings.HasPrefix(model, "o4")
}
