package token

import "strings"

// modelTokenLimits はモデルごとのコンテキストウィンドウサイズ（トークン数）
// 注: 出力トークンを考慮して、実際の上限より少なめに設定
var modelTokenLimits = map[string]int{
	// === Claude ===
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

	// === Bedrock (Claude models) ===
	"global.anthropic.claude-sonnet-4-6-v1":            200000,
	"anthropic.claude-sonnet-4-6":                      200000,
	"global.anthropic.claude-opus-4-5-20251101-v1:0":   200000,
	"us.anthropic.claude-sonnet-4-20250514-v1:0":       200000,
	"us.anthropic.claude-haiku-4-5-20251001-v1:0":      200000,
	"anthropic.claude-opus-4-20250514-v1:0":            200000,
	"global.anthropic.claude-sonnet-4-5-20250929-v1:0": 200000,

	// === OpenAI ===
	"gpt-4.1":             1000000,
	"gpt-4.1-mini":        1000000,
	"gpt-4.1-nano":        1000000,
	"gpt-4o":              128000,
	"gpt-4o-mini":         128000,
	"gpt-4-turbo":         128000,
	"gpt-4-turbo-preview": 128000,
	"gpt-4":               8192,
	"gpt-4-32k":           32768,
	"gpt-3.5-turbo":       16385,
	"o1":                  200000,
	"o1-mini":             128000,
	"o1-preview":          128000,
	"o3-mini":             200000,
	"gpt-5":               200000,
	"gpt-5.4":             1000000,
	"gpt-5.4-pro":         1000000,
	"gpt-5.1":             196000,
	"gpt-5.2":             400000,

	// === Gemini ===
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

	// === DeepSeek ===
	"deepseek-chat":     64000,
	"deepseek-coder":    64000,
	"deepseek-reasoner": 64000,

	// === Groq ===
	"llama-3.3-70b-versatile": 128000,
	"llama-3.1-70b-versatile": 128000,
	"llama-3.1-8b-instant":    128000,
	"mixtral-8x7b-32768":      32768,
	"gemma2-9b-it":            8192,

	// === Ollama (local) - typical defaults ===
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

	// === Default fallback ===
	"default": 100000,
}

// GetModelTokenLimit はモデルのトークン上限を取得
func GetModelTokenLimit(model string) int {
	if limit, ok := modelTokenLimits[model]; ok {
		return limit
	}

	// プレフィックスマッチング（バージョン違い対応）
	prefixes := []struct {
		prefix string
		limit  int
	}{
		{"us.anthropic.claude", 200000},
		{"anthropic.claude", 200000},
		{"global.anthropic.claude", 200000},
		{"claude-", 200000},
		{"gpt-4.1", 1000000},
		{"gpt-4o", 128000},
		{"gpt-4", 8192},
		{"gpt-3.5", 16385},
		{"gpt-5.4", 1000000},
		{"gpt-5", 200000},
		{"o1", 200000},
		{"o3", 200000},
		{"gemini-3", 1000000},
		{"gemini-2", 1000000},
		{"gemini-1.5", 1000000},
		{"gemini-pro", 32768},
		{"deepseek", 64000},
		{"llama-3", 128000},
		{"llama3", 8192},
		{"qwen", 32768},
		{"mixtral", 32768},
		{"codellama", 16384},
	}

	for _, p := range prefixes {
		if len(model) >= len(p.prefix) && model[:len(p.prefix)] == p.prefix {
			return p.limit
		}
	}

	return modelTokenLimits["default"]
}

// EstimateTokenCount はテキストのトークン数を推定
// 英語: 約4文字 = 1トークン
// 日本語: 約1-2文字 = 1トークン
// 混在を考慮して約2.5文字 = 1トークン
func EstimateTokenCount(text string) int {
	if len(text) == 0 {
		return 0
	}
	// 概算: 2.5文字 = 1トークン
	return (len(text)*10 + 24) / 25 // 四捨五入相当
}

// IsTokenLimitError はエラーがトークン上限エラーかどうかを判定
func IsTokenLimitError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	// OpenAI
	if strings.Contains(msg, "input tokens exceed") {
		return true
	}
	// Generic patterns (best-effort)
	if strings.Contains(msg, "context length") || strings.Contains(msg, "maximum context") {
		return true
	}
	return false
}
