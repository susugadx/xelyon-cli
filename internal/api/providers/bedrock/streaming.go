package bedrock

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime/types"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/api/providers/claude"
	"github.com/susugadx/xelyon-cli/internal/ui"
)

// toolUseAccumulator はストリーミング中の tool_use を蓄積する
// claude パッケージの同名型は unexported のためローカルに定義
type toolUseAccumulator struct {
	ID    string
	Name  string
	Input strings.Builder
}

// handleEventStream は AWS SDK イベントストリームを処理する
func (p *Provider) handleEventStream(ctx context.Context, output *bedrockruntime.InvokeModelWithResponseStreamOutput, spinner *ui.Spinner) (string, error) {
	toolUses := make(map[int]*toolUseAccumulator)
	compactionBlocks := make(map[int]*strings.Builder)
	var toolCallsOutput strings.Builder
	var fullResponse strings.Builder
	var compactionOutput strings.Builder
	var lastUsage *api.Usage
	firstChunk := true

	stream := output.GetStream()
	defer stream.Close()

	events := stream.Events()
	for {
		select {
		case <-ctx.Done():
			spinner.Stop()
			// 部分的な結果を返す
			content := fullResponse.String()
			if compactionOutput.Len() > 0 {
				content = "[COMPACTION]\n" + compactionOutput.String() + "\n[/COMPACTION]\n" + content
			}
			if toolCallsOutput.Len() > 0 {
				if content != "" {
					return content + toolCallsOutput.String(), ctx.Err()
				}
				return toolCallsOutput.String(), ctx.Err()
			}
			return content, ctx.Err()

		case event, ok := <-events:
			if !ok {
				// ストリーム終了
				spinner.Stop()
				if err := stream.Err(); err != nil {
					return "", fmt.Errorf("bedrock stream error: %w", err)
				}

				content := fullResponse.String()
				if compactionOutput.Len() > 0 {
					content = "[COMPACTION]\n" + compactionOutput.String() + "\n[/COMPACTION]\n" + content
				}

				// usage コールバックを呼び出し
				if lastUsage != nil && p.usageCallback != nil {
					p.usageCallback(*lastUsage)
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

			switch v := event.(type) {
			case *types.ResponseStreamMemberChunk:
				text, done := p.processChunk(v.Value.Bytes, toolUses, &toolCallsOutput, &lastUsage, compactionBlocks, &compactionOutput)
				if text != "" {
					if firstChunk {
						spinner.Stop()
						firstChunk = false
						api.PrintAIHeader()
					}
					fmt.Print(text)
					fullResponse.WriteString(text)
				}
				if done {
					spinner.Stop()

					content := fullResponse.String()
					if compactionOutput.Len() > 0 {
						content = "[COMPACTION]\n" + compactionOutput.String() + "\n[/COMPACTION]\n" + content
					}

					// usage コールバックを呼び出し
					if lastUsage != nil && p.usageCallback != nil {
						p.usageCallback(*lastUsage)
					}

					if toolCallsOutput.Len() > 0 {
						if content != "" {
							return content + toolCallsOutput.String(), nil
						}
						return toolCallsOutput.String(), nil
					}
					return content, nil
				}
			}
		}
	}
}

// processChunk は Bedrock チャンクの JSON ペイロードを処理する
// イベント JSON は Claude SSE の data フィールドと同じ形式
func (p *Provider) processChunk(data []byte, toolUses map[int]*toolUseAccumulator, toolCallsOutput *strings.Builder, lastUsage **api.Usage, compactionBlocks map[int]*strings.Builder, compactionOutput *strings.Builder) (text string, done bool) {
	var event claude.StreamEvent
	if err := json.Unmarshal(data, &event); err != nil {
		return "", false
	}

	switch event.Type {
	case "message_start":
		// message_start イベントには message.usage に input_tokens が含まれる
		var msgStart struct {
			Message struct {
				Usage claude.StreamUsage `json:"usage"`
			} `json:"message"`
		}
		if err := json.Unmarshal(data, &msgStart); err == nil {
			usage := msgStart.Message.Usage
			if *lastUsage == nil {
				*lastUsage = &api.Usage{}
			}
			(*lastUsage).InputTokens = usage.InputTokens
			(*lastUsage).CachedInputTokens = usage.CacheReadInputTokens
			(*lastUsage).CacheCreationTokens = usage.CacheCreationInputTokens
		}
		return "", false

	case "message_stop":
		return "", true

	case "content_block_start":
		if event.ContentBlock == nil {
			return "", false
		}
		// content_block のタイプに応じて処理を分岐
		switch event.ContentBlock.Type {
		case "tool_use":
			toolUses[event.Index] = &toolUseAccumulator{
				ID:   event.ContentBlock.ID,
				Name: event.ContentBlock.Name,
			}
		case "compaction":
			compactionBlocks[event.Index] = &strings.Builder{}
		}
		return "", false

	case "content_block_delta":
		if event.Delta == nil {
			return "", false
		}
		// compaction ブロックなら蓄積して非表示
		if acc, ok := compactionBlocks[event.Index]; ok {
			acc.WriteString(event.Delta.Text)
			return "", false
		}
		// テキストデルタ
		if event.Delta.Type == "text_delta" {
			return event.Delta.Text, false
		}
		// Tool Use の input を蓄積
		if event.Delta.Type == "input_json_delta" {
			if acc := toolUses[event.Index]; acc != nil {
				acc.Input.WriteString(event.Delta.PartialJSON)
			}
		}
		return "", false

	case "content_block_stop":
		// compaction ブロック完了処理
		if acc, ok := compactionBlocks[event.Index]; ok {
			compactionOutput.WriteString(acc.String())
			delete(compactionBlocks, event.Index)
		}
		// tool_use ブロックの完了 - この時点で変換
		if acc := toolUses[event.Index]; acc != nil {
			var input map[string]interface{}
			if err := json.Unmarshal([]byte(acc.Input.String()), &input); err == nil {
				if toolJSON, err := claude.ConvertToolUseToToolJSON(acc.ID, acc.Name, input); err == nil {
					toolCallsOutput.WriteString(toolJSON)
				}
			}
		}
		return "", false

	case "message_delta":
		// usage 情報を記録（output_tokens は message_delta に含まれる）
		if event.Usage != nil {
			if *lastUsage == nil {
				*lastUsage = &api.Usage{}
			}
			(*lastUsage).OutputTokens = event.Usage.OutputTokens
			// message_delta に input_tokens がある場合も反映
			if event.Usage.InputTokens > 0 {
				(*lastUsage).InputTokens = event.Usage.InputTokens
			}
			if event.Usage.CacheReadInputTokens > 0 {
				(*lastUsage).CachedInputTokens = event.Usage.CacheReadInputTokens
			}
			if event.Usage.CacheCreationInputTokens > 0 {
				(*lastUsage).CacheCreationTokens = event.Usage.CacheCreationInputTokens
			}
		}
		return "", false
	}

	return "", false
}
