package claude

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/ui"
)

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
