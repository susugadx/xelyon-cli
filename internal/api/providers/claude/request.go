package claude

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/config"
)

type ThinkingConfig struct {
	Type         string `json:"type"`                    // "enabled" or "adaptive"
	BudgetTokens int    `json:"budget_tokens,omitempty"` // type="enabled" 時のみ（min 1024）
}

// OutputConfig は出力制御の設定（Claude 4.6 モデル用）
type OutputConfig struct {
	Effort string `json:"effort"` // low / medium / high / max
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
	ContextManagement *ContextManagement `json:"context_management,omitempty"` // NEW
}

// LevelToBudgetTokens は api.LevelToBudgetTokens のエイリアス（後方互換）
func LevelToBudgetTokens(level string) int {
	return api.LevelToBudgetTokens(level)
}

// IsAdaptiveThinkingModel は adaptive thinking を使用すべきモデルか判定する。
// Claude Opus 4.6 と Sonnet 4.6 が対象。
func IsAdaptiveThinkingModel(model string) bool {
	m := strings.ToLower(model)
	return strings.Contains(m, "claude-opus-4-6") ||
		strings.Contains(m, "claude-sonnet-4-6") ||
		strings.Contains(m, "claude-opus-4.6") ||
		strings.Contains(m, "claude-sonnet-4.6")
}

// levelToEffort は thinking level を Claude effort パラメータに変換する。
// xhigh は Opus 4.6 限定の max にマッピングする。
func levelToEffort(level, model string) string {
	switch level {
	case "low":
		return "low"
	case "medium":
		return "medium"
	case "high":
		return "high"
	case "xhigh":
		// max は Opus 4.6 のみ対応
		if strings.Contains(strings.ToLower(model), "opus") {
			return "max"
		}
		return "high"
	default:
		return "medium"
	}
}

// Delta はストリームの差分

func (p *Provider) ChatWithTools(ctx context.Context, systemPrompt string, history []api.Message, model string) (string, error) {
	// モデル名を設定（config優先、フォールバックはclaude-sonnet-4-6）
	model = api.GetDefaultModelWithContext(ctx, model, "claude", "claude-sonnet-4-6")

	// Anthropic Messages API 形式に変換（role:"tool" → role:"user"+tool_result 等）
	messages := ConvertToAnthropicMessages(history)

	// デバッグ: tool_use/tool_result の整合性チェック
	if os.Getenv("XELYON_DEBUG_CLAUDE") == "1" {
		errOut := api.ErrorWriterFromContext(ctx)
		fmt.Fprintf(errOut, "[DEBUG Claude] === History (%d messages) ===\n", len(history))
		for i, m := range history {
			tcIDs := make([]string, len(m.ToolCalls))
			for j, tc := range m.ToolCalls {
				tcIDs[j] = tc.ID
			}
			if len(tcIDs) > 0 {
				fmt.Fprintf(errOut, "[DEBUG Claude] history[%d] role=%s tool_calls=%v\n", i, m.Role, tcIDs)
			} else if m.ToolCallID != "" {
				fmt.Fprintf(errOut, "[DEBUG Claude] history[%d] role=%s tool_call_id=%s\n", i, m.Role, m.ToolCallID)
			} else {
				fmt.Fprintf(errOut, "[DEBUG Claude] history[%d] role=%s content_len=%d\n", i, m.Role, len(m.Content))
			}
		}
		fmt.Fprintf(errOut, "[DEBUG Claude] === Converted (%d messages) ===\n", len(messages))
		for i, m := range messages {
			var types []string
			for _, b := range m.Content {
				switch b.Type {
				case "tool_use":
					types = append(types, "tool_use:"+b.ID)
				case "tool_result":
					types = append(types, "tool_result:"+b.ToolUseID)
				default:
					types = append(types, b.Type)
				}
			}
			fmt.Fprintf(errOut, "[DEBUG Claude] messages[%d] role=%s content=%v\n", i, m.Role, types)
		}
		validateAnthropicToolPairs(messages, errOut)
	}

	cfg := config.ResolveContext(ctx, p.effectiveConfig())

	reqBody := Request{
		Model:     model,
		Messages:  messages,
		System:    api.BuildSystemFieldWithConfig(systemPrompt, cfg),
		MaxTokens: p.maxOutputTokens(ctx, model),
		Stream:    true,
	}
	if cfg != nil && cfg.PromptCache.Enabled {
		// Anthropic automatic caching advances the breakpoint with conversation growth.
		// Keep explicit breakpoints on system/tools, and let the request-level cache
		// capture the latest conversation prefix from the second turn onward.
		reqBody.CacheControl = api.NewCacheControlWithConfig(cfg)
	}

	reqBody.ContextManagement = buildContextManagementForModel(model, cfg.Compression)

	// Extended Thinking 適用
	if api.IsThinkingEnabled(ctx) {
		if IsAdaptiveThinkingModel(model) {
			reqBody.Thinking = &ThinkingConfig{
				Type: "adaptive",
			}
			reqBody.OutputConfig = &OutputConfig{
				Effort: levelToEffort(cfg.Thinking.Level, model),
			}
		} else {
			reqBody.Thinking = &ThinkingConfig{
				Type:         "enabled",
				BudgetTokens: LevelToBudgetTokens(cfg.Thinking.Level),
			}
		}
	}

	// Tool Use: ツール定義を追加（環境変数で無効化可能）
	if os.Getenv("CLAUDE_FUNCTION_CALLING") != "0" {
		reqBody.Tools = GetCombinedClaudeToolsWithContext(ctx, p.mcpTools)
	}

	result, err := p.executeRequest(ctx, reqBody, model, reqBody.ContextManagement, false)
	if err != nil {
		return "", err
	}

	return p.processResponse(ctx, result)
}

// handleStreamingResponse はストリーミングレスポンスを処理

func (p *Provider) ChatWithImage(ctx context.Context, systemPrompt string, history []api.Message, userMessage string, image *api.ImageData, model string) (string, error) {
	// 画像がない場合は通常のChatWithToolsを使用
	if image == nil || image.Base64 == "" {
		history = append(history, api.Message{Role: "user", Content: userMessage})
		return p.ChatWithTools(ctx, systemPrompt, history, model)
	}

	// モデル名を設定（config優先、フォールバックはclaude-sonnet-4-6）
	model = api.GetDefaultModelWithContext(ctx, model, "claude", "claude-sonnet-4-6")

	// Anthropic Messages API 形式に変換（role:"tool" → role:"user"+tool_result 等）
	converted := ConvertToAnthropicMessages(history)

	cfg := config.ResolveContext(ctx, p.effectiveConfig())

	var messages []interface{}
	for _, msg := range converted {
		messages = append(messages, msg)
	}

	// 画像付きユーザーメッセージを追加
	multimodalMessage := MultimodalMessage{
		Role: "user",
		Content: []ContentPart{
			{
				Type: "image",
				Source: &ImageSource{
					Type:      "base64",
					MediaType: image.MediaType,
					Data:      image.Base64,
				},
			},
			{
				Type: "text",
				Text: userMessage,
			},
		},
	}
	messages = append(messages, multimodalMessage)

	reqBody := MultimodalRequest{
		Model:     model,
		Messages:  messages,
		System:    api.BuildSystemFieldWithConfig(systemPrompt, cfg),
		MaxTokens: p.maxOutputTokens(ctx, model),
		Stream:    true,
	}
	if cfg != nil && cfg.PromptCache.Enabled {
		reqBody.CacheControl = api.NewCacheControlWithConfig(cfg)
	}

	reqBody.ContextManagement = buildContextManagementForModel(model, cfg.Compression)

	// Extended Thinking 適用
	if api.IsThinkingEnabled(ctx) {
		if IsAdaptiveThinkingModel(model) {
			reqBody.Thinking = &ThinkingConfig{
				Type: "adaptive",
			}
			reqBody.OutputConfig = &OutputConfig{
				Effort: levelToEffort(cfg.Thinking.Level, model),
			}
		} else {
			reqBody.Thinking = &ThinkingConfig{
				Type:         "enabled",
				BudgetTokens: LevelToBudgetTokens(cfg.Thinking.Level),
			}
		}
	}

	// Tool Use: ツール定義を追加（環境変数で無効化可能）
	if os.Getenv("CLAUDE_FUNCTION_CALLING") != "0" {
		reqBody.Tools = GetCombinedClaudeToolsWithContext(ctx, p.mcpTools)
	}

	result, err := p.executeRequest(ctx, reqBody, model, reqBody.ContextManagement, true)
	if err != nil {
		return "", err
	}

	return p.processResponse(ctx, result)
}

// SetMCPTools は MCP ツール定義を設定する（Tool Use用）
