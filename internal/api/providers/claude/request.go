package claude

import (
	"context"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/llmcatalog"
)

type ThinkingConfig struct {
	Type         string `json:"type"`                    // "enabled" or "adaptive"
	BudgetTokens int    `json:"budget_tokens,omitempty"` // type="enabled" 時のみ（min 1024）
}

// OutputConfig は出力制御の設定（Claude adaptive thinking モデル用）
type OutputConfig struct {
	Effort string `json:"effort"` // low / medium / high / xhigh / max
}

// ClaudeToolChoice は Anthropic Messages API の forced tool choice を表す。
type ClaudeToolChoice struct {
	Type string `json:"type"`
	Name string `json:"name"`
}

type Request struct {
	Model    string             `json:"model"`
	Messages []AnthropicMessage `json:"messages"`
	// System can be either string (legacy) or []api.SystemBlock (prompt caching).
	System            interface{}        `json:"system,omitempty"`
	CacheControl      *api.CacheControl  `json:"cache_control,omitempty"`
	MaxTokens         int                `json:"max_tokens"`
	Stream            bool               `json:"stream"`
	Thinking          *ThinkingConfig    `json:"thinking,omitempty"`
	OutputConfig      *OutputConfig      `json:"output_config,omitempty"`
	Tools             []ClaudeTool       `json:"tools,omitempty"`              // Tool Use用
	ToolChoice        *ClaudeToolChoice  `json:"tool_choice,omitempty"`        // Tool Use 強制用
	ContextManagement *ContextManagement `json:"context_management,omitempty"` // NEW
}

// LevelToBudgetTokens は api.LevelToBudgetTokens のエイリアス（後方互換）
func LevelToBudgetTokens(level string) int {
	return api.LevelToBudgetTokens(level)
}

// IsAdaptiveThinkingModel は adaptive thinking を使用すべきモデルか判定する。
func IsAdaptiveThinkingModel(model string) bool {
	return llmcatalog.IsAdaptiveClaudeThinkingModel(model)
}

// IsAlwaysOnThinkingModel は thinking を無効化できない Claude モデルか判定する。
func IsAlwaysOnThinkingModel(model string) bool {
	return llmcatalog.IsAlwaysOnClaudeThinkingModel(model)
}

// LevelToEffort は thinking level を Claude effort パラメータに変換する。
// xhigh は Opus 4.8 / 4.7 / Fable 5 では xhigh、Opus 4.6 では max、Sonnet 4.6 では high にする。
func LevelToEffort(level, model string) string {
	switch level {
	case "low":
		return "low"
	case "medium":
		return "medium"
	case "high":
		return "high"
	case "xhigh":
		if isClaudeOpusXHighEffortModel(model) {
			return "xhigh"
		}
		if isClaudeOpus46Model(model) {
			return "max"
		}
		return "high"
	default:
		return "medium"
	}
}

func levelToEffort(level, model string) string {
	return LevelToEffort(level, model)
}

func isClaudeOpusXHighEffortModel(model string) bool {
	m := strings.ToLower(model)
	return strings.Contains(m, "claude-opus-4-8") || strings.Contains(m, "claude-opus-4.8") ||
		strings.Contains(m, "claude-opus-4-7") || strings.Contains(m, "claude-opus-4.7") ||
		strings.Contains(m, "claude-fable-5")
}

func isClaudeOpus46Model(model string) bool {
	m := strings.ToLower(model)
	return strings.Contains(m, "claude-opus-4-6") || strings.Contains(m, "claude-opus-4.6")
}

// Delta はストリームの差分

func (p *Provider) ChatWithTools(ctx context.Context, systemPrompt string, history []api.Message, model string) (string, error) {
	p.lastContentBlocks = nil

	built := p.buildMessagesRequest(ctx, systemPrompt, history, model)
	result, err := p.executeRequest(ctx, built.Request, built.Model, built.Request.ContextManagement, false)
	if err != nil {
		return "", err
	}

	return p.processResponse(ctx, result)
}

// handleStreamingResponse はストリーミングレスポンスを処理

func (p *Provider) ChatWithImage(ctx context.Context, systemPrompt string, history []api.Message, userMessage string, image *api.ImageData, model string) (string, error) {
	history = append(history, api.NewUserMessageWithOptionalImage(userMessage, image))
	return p.ChatWithTools(ctx, systemPrompt, history, model)
}

// SetMCPTools は MCP ツール定義を設定する（Tool Use用）
