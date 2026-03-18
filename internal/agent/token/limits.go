package token

import (
	"encoding/json"
	"strings"
	"unicode"
)

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
	"gpt-5":               400000,
	"gpt-5.4":             1000000,
	"gpt-5.4-pro":         1000000,
	"gpt-5.4-mini":        400000,
	"gpt-5.4-nano":        400000,
	"gpt-5.1":             400000,
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
	"deepseek-chat":     128000,
	"deepseek-coder":    64000,
	"deepseek-reasoner": 128000,

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
		{"gpt-5.4-mini", 400000},
		{"gpt-5.4-nano", 400000},
		{"gpt-5.4", 1000000},
		{"gpt-5", 400000},
		{"o1", 200000},
		{"o3", 200000},
		{"gemini-3", 1000000},
		{"gemini-2", 1000000},
		{"gemini-1.5", 1000000},
		{"gemini-pro", 32768},
		{"deepseek-chat", 128000},
		{"deepseek-reasoner", 128000},
		{"deepseek-v3", 128000},
		{"deepseek-r1", 128000},
		{"deepseek-coder", 64000},
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

// EstimateTokenCount はテキストのトークン数を推定する。
func EstimateTokenCount(text string) int {
	return EstimateTokenCountForModel("", text)
}

// EstimateTokenCountForModel はモデル特性を考慮してテキストのトークン数を推定する。
// 日本語・記号・コード断片を過小評価しにくい安全側の見積もりを返す。
func EstimateTokenCountForModel(model string, text string) int {
	if len(text) == 0 {
		return 0
	}

	asciiChunk := asciiCharsPerToken(model)

	var asciiPunct int
	var asciiSpace int
	var nonASCII int
	var newlines int
	var asciiWordRun int
	var asciiWordTokens int

	flushASCIIWordRun := func() {
		if asciiWordRun == 0 {
			return
		}
		asciiWordTokens += estimateASCIIWordRunTokens(asciiWordRun, asciiChunk)
		asciiWordRun = 0
	}

	for _, r := range text {
		switch {
		case r == '\n':
			flushASCIIWordRun()
			newlines++
			asciiSpace++
		case r > unicode.MaxASCII:
			flushASCIIWordRun()
			nonASCII++
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			asciiWordRun++
		case unicode.IsSpace(r):
			flushASCIIWordRun()
			asciiSpace++
		default:
			flushASCIIWordRun()
			asciiPunct++
		}
	}
	flushASCIIWordRun()

	total := 0
	total += asciiWordTokens
	total += ceilDiv(asciiPunct, 2)
	total += ceilDiv(asciiSpace, 8)
	total += nonASCII

	// 改行やコード断片は BPE 分割が増えやすいため少し上乗せする。
	codeSignals := strings.Count(text, "{") + strings.Count(text, "}") +
		strings.Count(text, "(") + strings.Count(text, ")") +
		strings.Count(text, "[") + strings.Count(text, "]") +
		strings.Count(text, "=>") + strings.Count(text, "::")
	total += ceilDiv(newlines, 3)
	total += ceilDiv(codeSignals, 4)

	if total < 1 {
		return 1
	}
	return total
}

// EstimateStructuredValueTokenCount は構造化データを JSON 化してトークン数を推定する。
func EstimateStructuredValueTokenCount(value any) int {
	return EstimateStructuredValueTokenCountForModel("", value)
}

// EstimateStructuredValueTokenCountForModel はモデル特性を考慮して構造化データのトークン数を推定する。
func EstimateStructuredValueTokenCountForModel(model string, value any) int {
	if value == nil {
		return 0
	}
	jsonBytes, err := json.Marshal(value)
	if err != nil {
		return 0
	}
	return EstimateTokenCountForModel(model, string(jsonBytes))
}

func asciiCharsPerToken(model string) int {
	lm := strings.ToLower(model)
	switch {
	case strings.HasPrefix(lm, "gpt-5.4"), strings.HasPrefix(lm, "gpt-4.1"), strings.HasPrefix(lm, "gpt-4o"):
		return 4
	case strings.HasPrefix(lm, "claude"), strings.Contains(lm, "anthropic"):
		return 4
	case strings.HasPrefix(lm, "gemini"):
		return 4
	case strings.HasPrefix(lm, "deepseek"):
		return 4
	default:
		return 4
	}
}

func estimateASCIIWordRunTokens(runLen, asciiChunk int) int {
	switch {
	case runLen <= asciiChunk:
		return 1
	case runLen <= 32:
		return ceilDiv(runLen, asciiChunk)
	case runLen <= 128:
		return ceilDiv(runLen, asciiChunk-1)
	default:
		return ceilDiv(runLen, 2)
	}
}

func ceilDiv(n, d int) int {
	if n <= 0 {
		return 0
	}
	return (n + d - 1) / d
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
