package bedrock

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime/types"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/api/providers/claude"
	claudestream "github.com/susugadx/xelyon-cli/internal/api/providers/claude_stream"
	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/ui"
)

type bedrockStreamState struct {
	toolUses        *claudestream.ToolUseCollector
	compaction      *claudestream.CompactionCollector
	contentBlocks   *claudestream.ContentBlockCollector
	toolCallsOutput strings.Builder
	lastUsage       *api.Usage
	spinner         *ui.Spinner
}

func newBedrockStreamState(spinner *ui.Spinner) *bedrockStreamState {
	return &bedrockStreamState{
		toolUses:      claudestream.NewToolUseCollector(),
		compaction:    claudestream.NewCompactionCollector(),
		contentBlocks: claudestream.NewContentBlockCollector(),
		spinner:       spinner,
	}
}

func (s *bedrockStreamState) finalContent(content string) string {
	if s == nil {
		return content
	}
	if compactionOutput := s.compaction.Output(); compactionOutput != "" {
		content = "[COMPACTION]\n" + compactionOutput + "\n[/COMPACTION]\n" + content
	}
	if s.toolCallsOutput.Len() == 0 {
		return content
	}
	if content != "" {
		return content + s.toolCallsOutput.String()
	}
	return s.toolCallsOutput.String()
}

// handleEventStream は AWS SDK イベントストリームを処理する
func (p *Provider) handleEventStream(ctx context.Context, output *bedrockruntime.InvokeModelWithResponseStreamOutput, spinner *ui.Spinner) (string, error) {
	out := api.OutputWriterFromContext(ctx)
	state := newBedrockStreamState(spinner)
	var fullResponse strings.Builder
	firstChunk := true

	stream := output.GetStream()
	defer stream.Close()

	idleTimeout := bedrockStreamIdleTimeout(ctx)
	var idleTimer *time.Timer
	var idleTimerCh <-chan time.Time
	if idleTimeout > 0 {
		idleTimer = time.NewTimer(idleTimeout)
		defer stopBedrockIdleTimer(idleTimer)
		idleTimerCh = idleTimer.C
	}

	events := stream.Events()
	for {
		select {
		case <-ctx.Done():
			spinner.Stop()
			return state.finalContent(fullResponse.String()), ctx.Err()

		case <-idleTimerCh:
			spinner.Stop()
			return state.finalContent(fullResponse.String()), fmt.Errorf("idle timeout: no data received for %v", idleTimeout)

		case event, ok := <-events:
			if !ok {
				// ストリーム終了
				spinner.Stop()
				if err := stream.Err(); err != nil {
					return "", fmt.Errorf("bedrock stream error: %w", err)
				}

				// usage コールバックを呼び出し
				if state.lastUsage != nil && p.usageCallback != nil {
					p.usageCallback(*state.lastUsage)
				}
				p.lastContentBlocks = state.contentBlocks.Blocks()

				return state.finalContent(fullResponse.String()), nil
			}
			resetBedrockIdleTimer(idleTimer, idleTimeout)

			switch v := event.(type) {
			case *types.ResponseStreamMemberChunk:
				text, done := p.processChunk(v.Value.Bytes, state)
				if text != "" {
					if firstChunk {
						spinner.Stop()
						firstChunk = false
						api.PrintAIHeaderWithContext(ctx)
					}
					if api.ShouldStreamAssistantText(ctx) {
						_, _ = fmt.Fprint(out, text)
					}
					fullResponse.WriteString(text)
				}
				if done {
					spinner.Stop()

					// usage コールバックを呼び出し
					if state.lastUsage != nil && p.usageCallback != nil {
						p.usageCallback(*state.lastUsage)
					}
					p.lastContentBlocks = state.contentBlocks.Blocks()

					return state.finalContent(fullResponse.String()), nil
				}
			}
		}
	}
}

// processChunk は Bedrock チャンクの JSON ペイロードを処理する
// イベント JSON は Claude SSE の data フィールドと同じ形式
func (p *Provider) processChunk(data []byte, state *bedrockStreamState) (text string, done bool) {
	var event claude.StreamEvent
	if err := json.Unmarshal(data, &event); err != nil {
		return "", false
	}

	switch event.Type {
	case "message_start":
		if usage, err := claudestream.DecodeMessageStartUsage(string(data)); err == nil {
			state.lastUsage = usage
		}
		return "", false

	case "message_stop":
		return "", true

	case "content_block_start":
		if event.ContentBlock == nil {
			return "", false
		}
		claudestream.HandleContentBlockStart(event, state.toolUses, state.compaction)
		state.contentBlocks.Start(event.Index, event.ContentBlock)
		return "", false

	case "content_block_delta":
		if event.Delta == nil {
			return "", false
		}
		state.contentBlocks.AppendDelta(event.Index, event.Delta)
		return claudestream.HandleContentBlockDelta(event, state.toolUses, state.compaction, func(toolName string) {
			if state.spinner != nil {
				// スピナーを再表示（引数生成中）
				if !state.spinner.IsActive() {
					state.spinner.Start(ui.SpinnerMessageForTool(toolName))
				}
			}
		}), false

	case "content_block_stop":
		state.contentBlocks.Stop(event.Index)
		if toolJSON := claudestream.HandleContentBlockStop(event, state.toolUses, state.compaction, claude.ConvertToolUseToToolJSON); toolJSON != "" {
			state.toolCallsOutput.WriteString(toolJSON)
		}
		return "", false

	case "message_delta":
		// usage 情報を記録（output_tokens は message_delta に含まれる）
		state.lastUsage = claudestream.UpdateUsageFromMessageDelta(state.lastUsage, event.Usage, true)
		return "", false
	}

	return "", false
}

func bedrockStreamIdleTimeout(ctx context.Context) time.Duration {
	cfg := config.FromContext(ctx)
	if cfg == nil || cfg.Streaming.IdleTimeoutSeconds <= 0 {
		return 0
	}
	return time.Duration(cfg.Streaming.IdleTimeoutSeconds) * time.Second
}

func resetBedrockIdleTimer(timer *time.Timer, timeout time.Duration) {
	if timer == nil || timeout <= 0 {
		return
	}
	if !timer.Stop() {
		select {
		case <-timer.C:
		default:
		}
	}
	timer.Reset(timeout)
}

func stopBedrockIdleTimer(timer *time.Timer) {
	if timer == nil {
		return
	}
	if !timer.Stop() {
		select {
		case <-timer.C:
		default:
		}
	}
}
