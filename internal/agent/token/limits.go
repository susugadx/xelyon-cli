package token

import (
	"encoding/json"
	"strings"
	"unicode"

	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/llmcatalog"
)

// GetModelTokenLimit はモデルのトークン上限を取得
func GetModelTokenLimit(model string) int {
	return llmcatalog.ModelContextLimit(model)
}

// GetKnownModelTokenLimit は既知モデルのトークン上限を取得する。
func GetKnownModelTokenLimit(model string) (int, bool) {
	return llmcatalog.KnownModelContextLimit(model)
}

// GetModelTokenLimitForConfig は catalog_model 設定を考慮してモデルのトークン上限を取得する。
func GetModelTokenLimitForConfig(cfg *config.Config, provider, model string) int {
	if cfg != nil {
		model = cfg.ModelCatalogName(provider, model)
	}
	return GetModelTokenLimit(model)
}

// GetKnownModelTokenLimitForConfig は catalog_model 設定を考慮して既知モデルのトークン上限を取得する。
func GetKnownModelTokenLimitForConfig(cfg *config.Config, provider, model string) (int, bool) {
	if cfg != nil {
		model = cfg.ModelCatalogName(provider, model)
	}
	return GetKnownModelTokenLimit(model)
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
