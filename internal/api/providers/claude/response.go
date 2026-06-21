package claude

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/api"
	claudestream "github.com/susugadx/xelyon-cli/internal/api/providers/claude_stream"
	"github.com/susugadx/xelyon-cli/internal/uiruntime"
	"github.com/susugadx/xelyon-cli/internal/uitoolview"
)

// Claude 互換ストリーム型は claude_stream owner に集約し、この package からは alias を公開する。
type Delta = claudestream.Delta
type ContentBlock = claudestream.ContentBlock
type StreamUsage = claudestream.StreamUsage
type StreamEvent = claudestream.StreamEvent

// Content はレスポンスのコンテンツ
type Content struct {
	Type      string                 `json:"type"`                // "text" or "tool_use"
	Text      string                 `json:"text,omitempty"`      // text 用
	Thinking  string                 `json:"thinking,omitempty"`  // thinking 用
	Signature string                 `json:"signature,omitempty"` // thinking 用
	Data      string                 `json:"data,omitempty"`      // redacted_thinking 用
	ID        string                 `json:"id,omitempty"`        // tool_use 用
	Name      string                 `json:"name,omitempty"`      // tool_use 用
	Input     map[string]interface{} `json:"input,omitempty"`     // tool_use 用
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
	ToolChoice        *ClaudeToolChoice  `json:"tool_choice,omitempty"`        // Tool Use 強制用
	ContextManagement *ContextManagement `json:"context_management,omitempty"` // NEW
}

// Response は通常レスポンス
type Response struct {
	Content    []Content   `json:"content"`
	StopReason string      `json:"stop_reason,omitempty"` // "end_turn", "tool_use" など
	Usage      StreamUsage `json:"usage"`                 // トークン使用量（キャッシュ含む）
}

// requestResult はexecuteRequestの結果を格納

func (p *Provider) handleStreamingResponse(ctx context.Context, resp *http.Response, spinner *uiruntime.Spinner) (string, error) {
	toolUses := claudestream.NewToolUseCollector()
	var toolCallsOutput strings.Builder

	compaction := claudestream.NewCompactionCollector()
	contentBlocks := claudestream.NewContentBlockCollector()

	// usage 情報を追跡
	var lastUsage *api.Usage

	// Claude固有のイベント処理
	handler := func(event claudestream.StreamEvent, data string) (string, bool, error) {
		switch event.Type {
		case "message_start":
			// message_start は input_tokens の権威的ソース。
			if usage, err := claudestream.DecodeMessageStartUsage(data); err == nil {
				lastUsage = usage
			}
			return "", false, nil

		case "message_stop":
			return "", true, nil

		case "content_block_start":
			contentBlocks.Start(event.Index, event.ContentBlock)
			claudestream.HandleContentBlockStart(event, toolUses, compaction)
			return "", false, nil

		case "content_block_delta":
			contentBlocks.AppendDelta(event.Index, event.Delta)
			return claudestream.HandleContentBlockDelta(event, toolUses, compaction, func(toolName string) {
				// スピナーを再表示（引数生成中）
				if !spinner.IsActive() {
					spinner.Start(uitoolview.SpinnerMessageForTool(toolName))
				}
			}), false, nil

		case "content_block_stop":
			contentBlocks.Stop(event.Index)
			if toolJSON := claudestream.HandleContentBlockStop(event, toolUses, compaction, ConvertToolUseToToolJSON); toolJSON != "" {
				toolCallsOutput.WriteString(toolJSON)
			}
			return "", false, nil

		case "message_delta":
			// message_start 欠損や Web Search 由来の更新をフォールバックで反映する。
			lastUsage = claudestream.UpdateUsageFromMessageDelta(lastUsage, event.Usage, true)
			return "", false, nil
		}

		return "", false, nil
	}

	content, err := claudestream.RunStreamingResponse(ctx, resp, spinner, handler, claudestream.RunnerOptions{
		CancelMode:        claudestream.CancelModePartialAsSuccess,
		WarnOnPartial:     true,
		IgnoreDecodeError: false,
		EnableIdleTimeout: true,
	})
	if err != nil {
		return "", err
	}
	p.lastContentBlocks = contentBlocks.Blocks()

	// usage コールバックを呼び出し
	if lastUsage != nil && p.usageCallback != nil {
		p.usageCallback(*lastUsage)
	}

	// compaction が発生した場合、レスポンスの先頭にマーカーを付加
	compactionOutput := compaction.Output()
	if compactionOutput != "" {
		content = "[COMPACTION]\n" + compactionOutput + "\n[/COMPACTION]\n" + content
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
func (p *Provider) handleNonStreamingResponse(ctx context.Context, resp *http.Response, spinner *uiruntime.Spinner) (string, error) {
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
	var contentBlocks []api.AnthropicContentBlock

	for _, c := range result.Content {
		switch c.Type {
		case "text":
			textContent.WriteString(c.Text)
			if c.Text != "" {
				contentBlocks = append(contentBlocks, api.AnthropicContentBlock{
					Type: "text",
					Text: c.Text,
				})
			}
		case "thinking":
			contentBlocks = append(contentBlocks, api.AnthropicContentBlock{
				Type:      "thinking",
				Thinking:  c.Thinking,
				Signature: c.Signature,
			})
		case "redacted_thinking":
			contentBlocks = append(contentBlocks, api.AnthropicContentBlock{
				Type: "redacted_thinking",
				Data: c.Data,
			})
		case "tool_use":
			contentBlocks = append(contentBlocks, api.AnthropicContentBlock{
				Type:  "tool_use",
				ID:    c.ID,
				Name:  c.Name,
				Input: cloneContentInput(c.Input),
			})
			if toolJSON, err := ConvertToolUseToToolJSON(c.ID, c.Name, c.Input); err == nil {
				toolCallsOutput.WriteString(toolJSON)
			}
		}
	}
	p.lastContentBlocks = contentBlocks

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

func cloneContentInput(src map[string]interface{}) map[string]any {
	if len(src) == 0 {
		return nil
	}
	dst := make(map[string]any, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}

// ChatWithImage は画像付きメッセージで会話を行う
