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
	toolCalls := openaicompatstream.NewToolCallCollector()
	var lastUsage *api.Usage

	parser := func(line string) (string, bool, error) {
		data, done, handled := openaicompatstream.ParseSSEDataLine(line)
		if !handled {
			return "", false, nil
		}
		if done {
			return "", true, nil
		}

		chunk, err := openaicompatstream.DecodeChunk(data)
		if err != nil {
			// 既存挙動を維持: 破損チャンクは無視して継続
			return "", false, nil
		}

		usage, err := openaicompatstream.DecodeStandardUsage(chunk.Usage)
		if err != nil {
			return "", false, nil
		}
		if usage != nil {
			lastUsage = usage
		}

		if len(chunk.Choices) == 0 {
			return "", false, nil
		}

		choice := chunk.Choices[0]
		toolCalls.Append(choice.Delta.ToolCalls, func(toolName string) {
			if !spinner.IsActive() {
				spinner.Start(ui.SpinnerMessageForTool(toolName))
			}
		})
		return choice.Delta.Content, false, nil
	}

	content, err := api.ParseStreamingResponse(ctx, resp, spinner, parser)
	if err != nil {
		return "", err
	}

	if p.usageCallback != nil && lastUsage != nil {
		p.usageCallback(*lastUsage)
	}

	toolCallsOutput := openaicompatstream.BuildToolCallJSON(
		toolCalls.ToOpenAIToolCalls(),
		openai.ConvertToolCallToToolJSON,
	)
	if toolCallsOutput != "" {
		return content + toolCallsOutput, nil
	}
	return content, nil
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
		cachedTokens := 0
		if apiResp.Usage.PromptTokensDetails != nil {
			cachedTokens = apiResp.Usage.PromptTokensDetails.CachedTokens
		}
		p.usageCallback(api.Usage{
			InputTokens:       apiResp.Usage.PromptTokens,
			OutputTokens:      apiResp.Usage.CompletionTokens,
			CachedInputTokens: cachedTokens,
		})
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
