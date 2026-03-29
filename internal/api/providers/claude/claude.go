package claude

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/ui"
)

func init() {
	api.RegisterProvider("claude", func(apiKey string) (api.Provider, error) {
		if apiKey == "" {
			return nil, fmt.Errorf("ANTHROPIC_API_KEY not set")
		}
		return New(apiKey), nil
	})
	// anthropic エイリアス
	api.RegisterProvider("anthropic", func(apiKey string) (api.Provider, error) {
		if apiKey == "" {
			return nil, fmt.Errorf("ANTHROPIC_API_KEY not set")
		}
		return New(apiKey), nil
	})
}

const defaultClaudeURL = "https://api.anthropic.com/v1/messages"

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
	mcpTools      []api.ToolDefinition // MCP ツール定義（Tool Use用）
	usageCallback api.UsageCallback    // トークン使用量コールバック
	runtimeConfig *config.Config
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
	return &Provider{
		BaseProvider: api.NewBaseProvider("Claude", apiKey, defaultClaudeURL, "ANTHROPIC_API_URL"),
	}
}

// SupportsImages は画像入力対応を返す
func (p *Provider) SupportsImages() bool {
	return true
}

// IsFunctionCallingEnabled は Function Calling が有効かを返す
func (p *Provider) IsFunctionCallingEnabled() bool {
	return true
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
// Opus 4.6 / Sonnet 4.6: type="adaptive"（budget_tokens 不要）
// それ以前: type="enabled" + budget_tokens
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
type Delta struct {
	Type        string `json:"type"`
	Text        string `json:"text,omitempty"`
	PartialJSON string `json:"partial_json,omitempty"` // tool_use の input (input_json_delta)
	StopReason  string `json:"stop_reason,omitempty"`  // message_delta 用
}

// ContentBlock はストリーミングのコンテンツブロック (content_block_start 用)
type ContentBlock struct {
	Type  string                 `json:"type"`            // "text" or "tool_use"
	ID    string                 `json:"id,omitempty"`    // tool_use 用
	Name  string                 `json:"name,omitempty"`  // tool_use 用
	Text  string                 `json:"text,omitempty"`  // text 用
	Input map[string]interface{} `json:"input,omitempty"` // tool_use 用（非ストリーミング）
}

// StreamUsage は Claude のトークン使用量
type StreamUsage struct {
	InputTokens              int `json:"input_tokens"`
	OutputTokens             int `json:"output_tokens"`
	CacheReadInputTokens     int `json:"cache_read_input_tokens,omitempty"`
	CacheCreationInputTokens int `json:"cache_creation_input_tokens,omitempty"`
}

// StreamEvent はストリームイベント
type StreamEvent struct {
	Type         string        `json:"type"`
	Index        int           `json:"index,omitempty"`
	ContentBlock *ContentBlock `json:"content_block,omitempty"` // content_block_start 用
	Delta        *Delta        `json:"delta,omitempty"`
	Usage        *StreamUsage  `json:"usage,omitempty"` // message_delta 用
}

// toolUseAccumulator はストリーミング中の tool_use を蓄積する
type toolUseAccumulator struct {
	ID    string
	Name  string
	Input strings.Builder // JSON文字列を蓄積
}

// Content はレスポンスのコンテンツ
type Content struct {
	Type  string                 `json:"type"`            // "text" or "tool_use"
	Text  string                 `json:"text,omitempty"`  // text 用
	ID    string                 `json:"id,omitempty"`    // tool_use 用
	Name  string                 `json:"name,omitempty"`  // tool_use 用
	Input map[string]interface{} `json:"input,omitempty"` // tool_use 用
}

// ContentPart はマルチモーダルコンテンツのパート
type ContentPart struct {
	Type   string       `json:"type"`             // "text" or "image"
	Text   string       `json:"text,omitempty"`   // type="text"の場合
	Source *ImageSource `json:"source,omitempty"` // type="image"の場合
}

// ImageSource は画像ソース
type ImageSource struct {
	Type      string `json:"type"`       // "base64"
	MediaType string `json:"media_type"` // "image/png", "image/jpeg" etc
	Data      string `json:"data"`       // Base64エンコードされたデータ
}

// MultimodalMessage はマルチモーダルメッセージ（画像含む）
type MultimodalMessage struct {
	Role    string        `json:"role"`
	Content []ContentPart `json:"content"`
}

// MultimodalRequest はマルチモーダルAPIリクエスト
type MultimodalRequest struct {
	Model    string        `json:"model"`
	Messages []interface{} `json:"messages"` // Message or MultimodalMessage
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

// Response は通常レスポンス
type Response struct {
	Content    []Content   `json:"content"`
	StopReason string      `json:"stop_reason,omitempty"` // "end_turn", "tool_use" など
	Usage      StreamUsage `json:"usage"`                 // トークン使用量（キャッシュ含む）
}

// requestResult はexecuteRequestの結果を格納
type requestResult struct {
	Response *http.Response
	Spinner  *ui.Spinner
}

// executeRequest はClaude API呼び出しの共通処理
// withImage: 画像付きリクエストの場合はtrue（スピナー表示に影響）
func (p *Provider) executeRequest(ctx context.Context, reqBody interface{}, contextManagement *ContextManagement, withImage bool) (*requestResult, error) {
	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", p.APIURL, bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", p.APIKey)

	cfg := config.FromContext(ctx)
	pCfg := cfg.ProviderModels["claude"]

	// Anthropic Version
	version := pCfg.AnthropicVersion
	if version == "" {
		version = "2023-06-01"
	}
	req.Header.Set("anthropic-version", version)

	// Anthropic Beta
	betaHeaders := make([]string, 0)
	if len(pCfg.AnthropicBeta) > 0 {
		betaHeaders = append(betaHeaders, pCfg.AnthropicBeta...)
	}
	betaHeaders = MergeAnthropicBetaHeaders(betaHeaders, contextManagement)
	if len(betaHeaders) > 0 {
		req.Header.Set("anthropic-beta", strings.Join(betaHeaders, ","))
	}

	spinner := api.StartThinkingSpinner(ctx, withImage, "")

	resp, err := p.ExecuteRequest(req)
	if err != nil {
		spinner.Stop()
		return nil, err
	}

	if resp.StatusCode != 200 {
		spinner.Stop()
		defer resp.Body.Close()

		if rateLimitErr := api.HandleRateLimit(resp); rateLimitErr != nil {
			return nil, rateLimitErr
		}
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("API error (%d): unable to read response", resp.StatusCode)
		}
		return nil, api.SanitizeErrorMessage(body, resp.StatusCode)
	}

	return &requestResult{Response: resp, Spinner: spinner}, nil
}

// processResponse はレスポンス処理（ストリーミング/非ストリーミング）
func (p *Provider) processResponse(ctx context.Context, result *requestResult) (string, error) {
	defer result.Response.Body.Close()

	contentType := result.Response.Header.Get("Content-Type")
	if strings.Contains(contentType, "text/event-stream") {
		return p.handleStreamingResponse(ctx, result.Response, result.Spinner)
	}
	return p.handleNonStreamingResponse(ctx, result.Response, result.Spinner)
}

// isCompactionSupported は Compaction API 対応モデルか判定
// 現時点では Opus 4.6 のみ
func isCompactionSupported(model string) bool {
	return strings.Contains(model, "opus-4-6") || strings.Contains(model, "opus-4-5") ||
		strings.Contains(model, "sonnet-4-6")
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
	if model == "" {
		model = cfg.GetModelForProvider("claude")
	}
	if model == "" {
		model = "claude-sonnet-4-6"
	}
	return isCompactionSupported(model)
}

func buildContextManagementForModel(model string, compression config.CompressionConfig) *ContextManagement {
	return BuildContextManagement(compression, isCompactionSupported(model))
}

// ChatWithTools は Provider interface の実装（context対応）
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
		MaxTokens: api.GetMaxOutputTokens(ctx, "claude", model),
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

	result, err := p.executeRequest(ctx, reqBody, reqBody.ContextManagement, false)
	if err != nil {
		return "", err
	}

	return p.processResponse(ctx, result)
}

// handleStreamingResponse はストリーミングレスポンスを処理
func (p *Provider) handleStreamingResponse(ctx context.Context, resp *http.Response, spinner *ui.Spinner) (string, error) {
	// Tool Use の蓄積用
	toolUses := make(map[int]*toolUseAccumulator)
	var toolCallsOutput strings.Builder

	// Compaction の蓄積用
	compactionBlocks := make(map[int]*strings.Builder)
	var compactionOutput strings.Builder

	// usage 情報を追跡
	var lastUsage *api.Usage

	// Claude固有のパース処理
	parser := func(line string) (string, bool, error) {
		if !strings.HasPrefix(line, "data: ") {
			return "", false, nil
		}

		data := strings.TrimPrefix(line, "data: ")
		var event StreamEvent
		if err := json.Unmarshal([]byte(data), &event); err != nil {
			return "", false, err
		}

		switch event.Type {
		case "message_start":
			// message_start は常に最初のイベント。input_tokens の権威的ソース。
			// message_delta には基本リクエストで output_tokens のみ含まれるため、
			// input_tokens は message_start から取得する。
			var msgStart struct {
				Message struct {
					Usage StreamUsage `json:"usage"`
				} `json:"message"`
			}
			if err := json.Unmarshal([]byte(data), &msgStart); err == nil {
				u := msgStart.Message.Usage
				lastUsage = &api.Usage{
					// InputTokens を正規化: API の input_tokens は非キャッシュ分のみ
					InputTokens:         u.InputTokens + u.CacheReadInputTokens + u.CacheCreationInputTokens,
					OutputTokens:        u.OutputTokens,
					CachedInputTokens:   u.CacheReadInputTokens,
					CacheCreationTokens: u.CacheCreationInputTokens,
				}
			}
			return "", false, nil

		case "message_stop":
			return "", true, nil

		case "content_block_start":
			if event.ContentBlock != nil {
				switch event.ContentBlock.Type {
				case "tool_use":
					toolUses[event.Index] = &toolUseAccumulator{
						ID:   event.ContentBlock.ID,
						Name: event.ContentBlock.Name,
					}
				case "compaction":
					// compaction ブロックの開始を記録
					compactionBlocks[event.Index] = &strings.Builder{}
				}
			}
			return "", false, nil

		case "content_block_delta":
			if event.Delta == nil {
				return "", false, nil
			}
			// テキストデルタ
			if event.Delta.Type == "text_delta" {
				// compaction ブロックの場合は蓄積（表示しない）
				if acc, ok := compactionBlocks[event.Index]; ok {
					acc.WriteString(event.Delta.Text)
					return "", false, nil
				}
				return event.Delta.Text, false, nil
			}
			// Tool Use の input を蓄積
			if event.Delta.Type == "input_json_delta" {
				if acc := toolUses[event.Index]; acc != nil {
					// スピナーを再表示（引数生成中）
					if !spinner.IsActive() {
						spinner.Start(ui.SpinnerMessageForTool(acc.Name))
					}
					acc.Input.WriteString(event.Delta.PartialJSON)
				}
			}
			return "", false, nil

		case "content_block_stop":
			// compaction ブロックの完了
			if acc, ok := compactionBlocks[event.Index]; ok {
				compactionOutput.WriteString(acc.String())
				delete(compactionBlocks, event.Index)
			}
			// tool_use ブロックの完了 - この時点で変換
			if acc := toolUses[event.Index]; acc != nil {
				var input map[string]interface{}
				if err := json.Unmarshal([]byte(acc.Input.String()), &input); err == nil {
					if toolJSON, err := ConvertToolUseToToolJSON(acc.ID, acc.Name, input); err == nil {
						toolCallsOutput.WriteString(toolJSON)
					}
				}
			}
			return "", false, nil

		case "message_delta":
			// usage 情報を記録（message_delta の usage は累積値）
			if event.Usage != nil {
				if lastUsage == nil {
					lastUsage = &api.Usage{}
				}
				lastUsage.OutputTokens = event.Usage.OutputTokens
				// フォールバック: message_start が欠損した場合 or Web Search で input_tokens が更新された場合
				if event.Usage.InputTokens > 0 {
					lastUsage.InputTokens = event.Usage.InputTokens + event.Usage.CacheReadInputTokens + event.Usage.CacheCreationInputTokens
				}
				if event.Usage.CacheReadInputTokens > 0 {
					lastUsage.CachedInputTokens = event.Usage.CacheReadInputTokens
				}
				if event.Usage.CacheCreationInputTokens > 0 {
					lastUsage.CacheCreationTokens = event.Usage.CacheCreationInputTokens
				}
			}
			return "", false, nil
		}

		return "", false, nil
	}

	content, err := api.ParseStreamingResponse(ctx, resp, spinner, parser)
	if err != nil {
		return "", err
	}

	// usage コールバックを呼び出し
	if lastUsage != nil && p.usageCallback != nil {
		p.usageCallback(*lastUsage)
	}

	// compaction が発生した場合、レスポンスの先頭にマーカーを付加
	if compactionOutput.Len() > 0 {
		content = "[COMPACTION]\n" + compactionOutput.String() + "\n[/COMPACTION]\n" + content
	}

	// Tool Use がある場合はそれを追加して返す
	if toolCallsOutput.Len() > 0 {
		if content != "" {
			return content + toolCallsOutput.String(), nil
		}
		return toolCallsOutput.String(), nil
	}
	return content, nil
}

// handleNonStreamingResponse は非ストリーミングレスポンスを処理（フォールバック）
func (p *Provider) handleNonStreamingResponse(ctx context.Context, resp *http.Response, spinner *ui.Spinner) (string, error) {
	var result Response
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		spinner.Stop()
		return "", err
	}

	spinner.Stop()

	// usage コールバック呼び出し（キャッシュヒット情報含む）
	if p.usageCallback != nil {
		u := result.Usage
		p.usageCallback(api.Usage{
			InputTokens:         u.InputTokens + u.CacheReadInputTokens + u.CacheCreationInputTokens,
			OutputTokens:        u.OutputTokens,
			CachedInputTokens:   u.CacheReadInputTokens,
			CacheCreationTokens: u.CacheCreationInputTokens,
		})
	}

	if len(result.Content) == 0 {
		return "", fmt.Errorf("no response from API")
	}

	// テキストと Tool Use を収集
	var textContent strings.Builder
	var toolCallsOutput strings.Builder

	for _, c := range result.Content {
		switch c.Type {
		case "text":
			textContent.WriteString(c.Text)
		case "tool_use":
			if toolJSON, err := ConvertToolUseToToolJSON(c.ID, c.Name, c.Input); err == nil {
				toolCallsOutput.WriteString(toolJSON)
			}
		}
	}

	content := textContent.String()
	if content != "" {
		if api.ShouldStreamAssistantText(ctx) {
			_, _ = fmt.Fprintln(api.OutputWriterFromContext(ctx), content)
		}
	}

	// Tool Use がある場合は追加
	if toolCallsOutput.Len() > 0 {
		if content != "" {
			return content + toolCallsOutput.String(), nil
		}
		return toolCallsOutput.String(), nil
	}
	return content, nil
}

// ChatWithImage は画像付きメッセージで会話を行う
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
		MaxTokens: api.GetMaxOutputTokens(ctx, "claude", model),
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

	result, err := p.executeRequest(ctx, reqBody, reqBody.ContextManagement, true)
	if err != nil {
		return "", err
	}

	return p.processResponse(ctx, result)
}

// SetMCPTools は MCP ツール定義を設定する（Tool Use用）
func (p *Provider) SetMCPTools(tools []api.ToolDefinition) {
	p.mcpTools = tools
}

// SetUsageCallback は使用量レポートのコールバックを設定する
func (p *Provider) SetUsageCallback(callback api.UsageCallback) {
	p.usageCallback = callback
}
