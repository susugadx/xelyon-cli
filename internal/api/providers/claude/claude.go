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

// Provider はClaude (Anthropic) APIのプロバイダー実装
type Provider struct {
	api.BaseProvider
	mcpTools          []api.ToolDefinition // MCP ツール定義（Tool Use用）
	usageCallback     api.UsageCallback    // トークン使用量コールバック
	compactionEnabled bool                 // Compaction API を使用するか
	compactionTrigger int                  // トリガー閾値（トークン数）
}

// ContextManagement は Compaction API の設定
type ContextManagement struct {
	Edits []ContextEdit `json:"edits"`
}

// ContextEdit は context_management.edits の要素
type ContextEdit struct {
	Type    string          `json:"type"` // "compact_20260112"
	Trigger *CompactTrigger `json:"trigger,omitempty"`
}

// CompactTrigger は compaction のトリガー条件
type CompactTrigger struct {
	Type  string `json:"type"`  // "input_tokens"
	Value int    `json:"value"` // トークン数（最低 50000）
}

// New は新しいProviderを作成
func New(apiKey string) *Provider {
	cfg := config.GetGlobalConfig()
	return &Provider{
		BaseProvider:      api.NewBaseProvider("Claude", apiKey, defaultClaudeURL, "ANTHROPIC_API_URL"),
		compactionEnabled: cfg.Compression.ClaudeCompaction,
		compactionTrigger: cfg.Compression.CompactionTrigger,
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
	cfg := config.GetGlobalConfig()
	if !cfg.Compression.ClaudeCompaction {
		return false
	}
	model := api.GetDefaultModel("", "claude", "claude-sonnet-4-6")
	return isCompactionSupported(model)
}

// ThinkingConfig は Extended Thinking の設定
type ThinkingConfig struct {
	Type         string `json:"type"`          // "enabled"
	BudgetTokens int    `json:"budget_tokens"` // min 1024
}

type Request struct {
	Model    string             `json:"model"`
	Messages []AnthropicMessage `json:"messages"`
	// System can be either string (legacy) or []api.SystemBlock (prompt caching).
	System            interface{}        `json:"system,omitempty"`
	MaxTokens         int                `json:"max_tokens"`
	Stream            bool               `json:"stream"`
	Thinking          *ThinkingConfig    `json:"thinking,omitempty"`
	Tools             []ClaudeTool       `json:"tools,omitempty"`              // Tool Use用
	ContextManagement *ContextManagement `json:"context_management,omitempty"` // NEW
}

// LevelToBudgetTokens は api.LevelToBudgetTokens のエイリアス（後方互換）
func LevelToBudgetTokens(level string) int {
	return api.LevelToBudgetTokens(level)
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
	MaxTokens         int                `json:"max_tokens"`
	Stream            bool               `json:"stream"`
	Thinking          *ThinkingConfig    `json:"thinking,omitempty"`
	Tools             []ClaudeTool       `json:"tools,omitempty"`              // Tool Use用
	ContextManagement *ContextManagement `json:"context_management,omitempty"` // NEW
}

// Response は通常レスポンス
type Response struct {
	Content    []Content `json:"content"`
	StopReason string    `json:"stop_reason,omitempty"` // "end_turn", "tool_use" など
}

// requestResult はexecuteRequestの結果を格納
type requestResult struct {
	Response *http.Response
	Spinner  *ui.Spinner
}

// executeRequest はClaude API呼び出しの共通処理
// withImage: 画像付きリクエストの場合はtrue（スピナー表示に影響）
func (p *Provider) executeRequest(ctx context.Context, reqBody interface{}, withImage bool) (*requestResult, error) {
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

	cfg := config.GetGlobalConfig()
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
	// Compaction が有効な場合は beta ヘッダーを追加
	if p.compactionEnabled {
		betaHeaders = append(betaHeaders, "compact-2026-01-12")
	}
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
	return p.handleNonStreamingResponse(result.Response, result.Spinner)
}

// isCompactionSupported は Compaction API 対応モデルか判定
// 現時点では Opus 4.6 のみ
func isCompactionSupported(model string) bool {
	return strings.Contains(model, "opus-4-6") || strings.Contains(model, "opus-4-5") ||
		strings.Contains(model, "sonnet-4-6")
}

// ChatWithTools は Provider interface の実装（context対応）
func (p *Provider) ChatWithTools(ctx context.Context, systemPrompt string, history []api.Message, model string) (string, error) {
	// モデル名を設定（config優先、フォールバックはclaude-sonnet-4-6）
	model = api.GetDefaultModel(model, "claude", "claude-sonnet-4-6")

	// Anthropic Messages API 形式に変換（role:"tool" → role:"user"+tool_result 等）
	messages := ConvertToAnthropicMessages(history)

	cfg := config.GetGlobalConfig()

	// プロンプトキャッシュ: 安定区間+最新userにブレークポイント設定
	if cfg != nil && cfg.PromptCache.Enabled {
		SetMessageCacheBreakpoints(messages)
	}

	reqBody := Request{
		Model:     model,
		Messages:  messages,
		System:    api.BuildSystemField(systemPrompt),
		MaxTokens: api.GetMaxOutputTokens(ctx, "claude", model),
		Stream:    true,
	}

	// Compaction API（Opus 4.6 のみ）
	if p.compactionEnabled && isCompactionSupported(model) {
		trigger := p.compactionTrigger
		if trigger == 0 {
			trigger = 150000
		}
		reqBody.ContextManagement = &ContextManagement{
			Edits: []ContextEdit{
				{
					Type: "compact_20260112",
					Trigger: &CompactTrigger{
						Type:  "input_tokens",
						Value: trigger,
					},
				},
			},
		}
	}

	// Extended Thinking 適用
	if api.IsThinkingEnabled(ctx) {
		reqBody.Thinking = &ThinkingConfig{
			Type:         "enabled",
			BudgetTokens: LevelToBudgetTokens(cfg.Thinking.Level),
		}
	}

	// Tool Use: ツール定義を追加（環境変数で無効化可能）
	if os.Getenv("CLAUDE_FUNCTION_CALLING") != "0" {
		reqBody.Tools = GetCombinedClaudeToolsWithContext(ctx, p.mcpTools)
	}

	result, err := p.executeRequest(ctx, reqBody, false)
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
func (p *Provider) handleNonStreamingResponse(resp *http.Response, spinner *ui.Spinner) (string, error) {
	var result Response
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		spinner.Stop()
		return "", err
	}

	spinner.Stop()

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
		fmt.Println(content)
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
	model = api.GetDefaultModel(model, "claude", "claude-sonnet-4-6")

	// Anthropic Messages API 形式に変換（role:"tool" → role:"user"+tool_result 等）
	converted := ConvertToAnthropicMessages(history)

	// プロンプトキャッシュ: 履歴部分にブレークポイント設定
	// multimodalMessage（画像付き新規入力）は converted に含まれないため BP 対象外。
	// 画像ターンでは実質 BP が system+tools+履歴の3個になるが、
	// 次ターンで multimodalMessage も履歴に含まれキャッシュされる。
	cfg := config.GetGlobalConfig()
	if cfg != nil && cfg.PromptCache.Enabled {
		SetMessageCacheBreakpoints(converted)
	}

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
		System:    api.BuildSystemField(systemPrompt),
		MaxTokens: api.GetMaxOutputTokens(ctx, "claude", model),
		Stream:    true,
	}

	// Extended Thinking 適用
	if api.IsThinkingEnabled(ctx) {
		reqBody.Thinking = &ThinkingConfig{
			Type:         "enabled",
			BudgetTokens: LevelToBudgetTokens(cfg.Thinking.Level),
		}
	}

	// Tool Use: ツール定義を追加（環境変数で無効化可能）
	if os.Getenv("CLAUDE_FUNCTION_CALLING") != "0" {
		reqBody.Tools = GetCombinedClaudeToolsWithContext(ctx, p.mcpTools)
	}

	result, err := p.executeRequest(ctx, reqBody, true)
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
