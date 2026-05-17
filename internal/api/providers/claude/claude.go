package claude

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/config"
)

func init() {
	api.RegisterProvider("claude", func(apiKey string) (api.Provider, error) {
		if apiKey == "" {
			return nil, fmt.Errorf("%s not set", anthropicAPIKeyEnv)
		}
		return newProvider(apiKey, "claude"), nil
	})
	// anthropic エイリアス
	api.RegisterProvider("anthropic", func(apiKey string) (api.Provider, error) {
		if apiKey == "" {
			return nil, fmt.Errorf("%s not set", anthropicAPIKeyEnv)
		}
		return newProvider(apiKey, "anthropic"), nil
	})
}

const (
	claudeMessagesEndpointPath = "/v1/messages"
	defaultClaudeURL           = "https://api.anthropic.com" + claudeMessagesEndpointPath
	defaultClaudeModel         = "claude-sonnet-4-6"
	anthropicAPIKeyEnv         = "ANTHROPIC_API_KEY"
	anthropicAPIURLEnv         = "ANTHROPIC_API_URL"
	claudeFunctionCallEnv      = "CLAUDE_FUNCTION_CALLING"
	defaultAnthropicVersion    = "2023-06-01"
)

const (
	compactEditType             = "compact_20260112"
	clearToolUsesEditType       = "clear_tool_uses_20250919"
	compactBetaHeader           = "compact-2026-01-12"
	contextManagementBetaHeader = "context-management-2025-06-27"
	defaultCompactionTrigger    = 150000
	defaultClearToolUsesTrigger = 80000
	minimumContextEditTrigger   = 50000
)

// Provider はClaude (Anthropic) APIのプロバイダー実装
type Provider struct {
	api.BaseProvider
	mcpTools          []api.ToolDefinition // MCP ツール定義（Tool Use用）
	toolChoice        *string              // tool_choice 強制用
	usageCallback     api.UsageCallback    // トークン使用量コールバック
	runtimeConfig     *config.Config
	configKey         string
	lastContentBlocks []api.AnthropicContentBlock
}

// ContextManagement は Claude Context Management API の設定
type ContextManagement struct {
	Edits []ContextEdit `json:"edits"`
}

// ContextEdit は context_management.edits の要素
type ContextEdit struct {
	Type            string          `json:"type"` // "compact_20260112" or "clear_tool_uses_20250919"
	Trigger         *CompactTrigger `json:"trigger,omitempty"`
	ClearToolInputs *bool           `json:"clear_tool_inputs,omitempty"` // clear_tool_uses 用
}

// CompactTrigger は context edit のトリガー条件
type CompactTrigger struct {
	Type  string `json:"type"`  // "input_tokens"
	Value int    `json:"value"` // トークン数（最低 50000）
}

// BuildContextManagement は Claude 系プロバイダー向けの context_management を構築する。
func BuildContextManagement(compression config.CompressionConfig, compactionSupported bool) *ContextManagement {
	edits := make([]ContextEdit, 0, 2)
	if compression.ClearToolUses {
		edit := ContextEdit{
			Type: clearToolUsesEditType,
			Trigger: &CompactTrigger{
				Type:  "input_tokens",
				Value: normalizeContextEditTrigger(compression.ClearToolUsesTrigger, defaultClearToolUsesTrigger),
			},
		}
		if compression.ClearToolInputs {
			clearToolInputs := true
			edit.ClearToolInputs = &clearToolInputs
		}
		edits = append(edits, edit)
	}

	if compression.ClaudeCompaction && compactionSupported {
		edits = append(edits, ContextEdit{
			Type: compactEditType,
			Trigger: &CompactTrigger{
				Type:  "input_tokens",
				Value: normalizeContextEditTrigger(compression.CompactionTrigger, defaultCompactionTrigger),
			},
		})
	}

	if len(edits) == 0 {
		return nil
	}

	return &ContextManagement{Edits: edits}
}

// MergeAnthropicBetaHeaders は Claude Context Management に必要な beta ヘッダーを重複なく追加する。
func MergeAnthropicBetaHeaders(headers []string, contextManagement *ContextManagement) []string {
	merged := append([]string{}, headers...)
	if contextManagement == nil {
		return merged
	}

	for _, edit := range contextManagement.Edits {
		switch edit.Type {
		case compactEditType:
			merged = appendUniqueStrings(merged, compactBetaHeader)
		case clearToolUsesEditType:
			merged = appendUniqueStrings(merged, contextManagementBetaHeader)
		}
	}
	return merged
}

func normalizeContextEditTrigger(value, defaultValue int) int {
	if value == 0 {
		return defaultValue
	}
	if value < minimumContextEditTrigger {
		return minimumContextEditTrigger
	}
	return value
}

func appendUniqueStrings(headers []string, extras ...string) []string {
	for _, extra := range extras {
		found := false
		for _, header := range headers {
			if header == extra {
				found = true
				break
			}
		}
		if !found {
			headers = append(headers, extra)
		}
	}
	return headers
}

// New は新しいProviderを作成
func New(apiKey string) *Provider {
	return newProvider(apiKey, "claude")
}

func newProvider(apiKey, configKey string) *Provider {
	return &Provider{
		BaseProvider: api.NewBaseProvider("Claude", apiKey, defaultClaudeURL, anthropicAPIURLEnv),
		configKey:    config.NormalizeProviderName(configKey),
	}
}

func (p *Provider) configLookupKey() string {
	if p != nil && p.configKey != "" {
		return p.configKey
	}
	return "claude"
}

func (p *Provider) ProviderConfigKey() string {
	return p.configLookupKey()
}

func (p *Provider) SetProviderConfigKey(key string) {
	if p == nil {
		return
	}
	p.configKey = config.NormalizeProviderName(key)
}

func (p *Provider) maxOutputTokens(ctx context.Context, model string) int {
	return api.GetMaxOutputTokens(ctx, p.configLookupKey(), model)
}

// LastAnthropicThinkingBlocks は最後の API 呼び出しで返された thinking blocks を返す。
func (p *Provider) LastAnthropicThinkingBlocks() []api.AnthropicThinkingBlock {
	if p == nil {
		return nil
	}
	return api.AnthropicThinkingBlocksFromContentBlocks(p.lastContentBlocks)
}

// LastAnthropicContentBlocks は最後の API 呼び出しで返された assistant content blocks を順序付きで返す。
func (p *Provider) LastAnthropicContentBlocks() []api.AnthropicContentBlock {
	if p == nil || len(p.lastContentBlocks) == 0 {
		return nil
	}
	return api.CloneAnthropicContentBlocks(p.lastContentBlocks)
}

// SupportsImages は画像入力対応を返す
func (p *Provider) SupportsImages() bool {
	return true
}

// IsFunctionCallingEnabled は Function Calling が有効かを返す
func (p *Provider) IsFunctionCallingEnabled() bool {
	return claudeFunctionCallingEnabled()
}

func claudeFunctionCallingEnabled() bool {
	return os.Getenv(claudeFunctionCallEnv) != "0"
}

// SupportsClaudeCompaction は Claude Compaction 対応を返す
func (p *Provider) SupportsClaudeCompaction() bool {
	return p.supportsClaudeCompactionWithConfig(p.effectiveConfig(), "")
}

// SupportsClaudeCompactionWithContext は request context とモデルを使って Claude Compaction 対応可否を返す。
func (p *Provider) SupportsClaudeCompactionWithContext(ctx context.Context, model string) bool {
	cfg := p.effectiveConfig()
	if ctxCfg, ok := config.LookupContext(ctx); ok {
		cfg = ctxCfg
	}
	return p.supportsClaudeCompactionWithConfig(cfg, model)
}

// SetRuntimeConfig は provider が参照する runtime 設定を差し替える。
func (p *Provider) SetRuntimeConfig(cfg *config.Config) {
	p.runtimeConfig = cfg
}

// ThinkingConfig は Extended Thinking の設定
// Opus 4.7 / Opus 4.6 / Sonnet 4.6: type="adaptive"（budget_tokens 不要）
// それ以前: type="enabled" + budget_tokens

// IsCompactionSupportedModel は Claude Compaction API 対応モデルか判定する。
func IsCompactionSupportedModel(model string) bool {
	m := strings.ToLower(strings.TrimSpace(model))
	return strings.Contains(m, "opus-4-7") || strings.Contains(m, "opus-4.7") ||
		strings.Contains(m, "opus-4-6") || strings.Contains(m, "opus-4.6") ||
		strings.Contains(m, "opus-4-5") || strings.Contains(m, "opus-4.5") ||
		strings.Contains(m, "sonnet-4-6") || strings.Contains(m, "sonnet-4.6")
}

func isCompactionSupported(model string) bool {
	return IsCompactionSupportedModel(model)
}

func (p *Provider) effectiveConfig() *config.Config {
	if p != nil && p.runtimeConfig != nil {
		return p.runtimeConfig
	}
	return config.DefaultConfig()
}

func (p *Provider) supportsClaudeCompactionWithConfig(cfg *config.Config, model string) bool {
	if cfg == nil || !cfg.Compression.ClaudeCompaction {
		return false
	}
	providerKey := p.configLookupKey()
	if model == "" {
		model = cfg.GetEffectiveModelForProvider(providerKey)
	}
	if model == "" {
		model = defaultClaudeModel
	}
	return isCompactionSupported(cfg.ModelCatalogName(providerKey, model))
}

func buildContextManagementForModel(model string, compression config.CompressionConfig) *ContextManagement {
	return BuildContextManagement(compression, isCompactionSupported(model))
}

// ChatWithTools は Provider interface の実装（context対応）

func (p *Provider) SetMCPTools(tools []api.ToolDefinition) {
	p.mcpTools = tools
}

// SetToolChoice は tool_choice を設定する。
func (p *Provider) SetToolChoice(name string) {
	p.toolChoice = &name
}

// ClearToolChoice は tool_choice をクリアする。
func (p *Provider) ClearToolChoice() {
	p.toolChoice = nil
}

// SetUsageCallback は使用量レポートのコールバックを設定する
func (p *Provider) SetUsageCallback(callback api.UsageCallback) {
	p.usageCallback = callback
}
