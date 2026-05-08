package openrouter

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/api/providers/openai"
	openaicompatstream "github.com/susugadx/xelyon-cli/internal/api/providers/openai_compat_stream"
	"github.com/susugadx/xelyon-cli/internal/ui"
)

// handleStreamingResponse は OpenRouter の OpenAI 互換 SSE 処理を担う。
// request 構築やモデル分岐は openrouter.go 側が owner。
func (p *Provider) handleStreamingResponse(ctx context.Context, resp *http.Response, spinner *ui.Spinner) (string, error) {
	streamResult, err := openaicompatstream.ParseSSEStream(ctx, resp, spinner, openaicompatstream.ParseSSEOptions{
		OnChunkDecodeError: func(error) error {
			// 既存挙動を維持: 破損チャンクは無視して継続
			return nil
		},
		OnUsageDecodeError: func(error) error {
			return nil
		},
		OnToolCallArguments: func(toolName string) {
			if !spinner.IsActive() {
				spinner.Start(ui.SpinnerMessageForTool(toolName))
			}
		},
	})
	if err != nil {
		return "", err
	}

	if p.usageCallback != nil && streamResult.Usage != nil {
		p.usageCallback(*streamResult.Usage)
	}

	return openaicompatstream.BuildContentWithToolCalls(
		streamResult.Content,
		streamResult.ToolCalls,
		openai.ConvertToolCallToToolJSON,
	), nil
}

// handleNonStreamingResponse は OpenRouter の OpenAI 互換非ストリーミング処理を担う。
func (p *Provider) handleNonStreamingResponse(ctx context.Context, resp *http.Response, spinner *ui.Spinner) (string, error) {
	var apiResp struct {
		Choices []api.Choice        `json:"choices"`
		Usage   api.StreamUsageInfo `json:"usage"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		spinner.Stop()
		return "", err
	}
	spinner.Stop()

	if p.usageCallback != nil {
		p.usageCallback(apiResp.Usage.ToUsage())
	}

	if len(apiResp.Choices) > 0 {
		api.PrintAIHeaderWithContext(ctx)
		content := apiResp.Choices[0].Message.Content
		if api.ShouldStreamAssistantText(ctx) {
			_, _ = fmt.Fprintln(api.OutputWriterFromContext(ctx), content)
		}
		return content, nil
	}
	return "", nil
}
