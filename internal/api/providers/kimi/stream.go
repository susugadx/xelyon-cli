package kimi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/fatih/color"
	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/api/providers/openai"
	openaicompatstream "github.com/susugadx/xelyon-cli/internal/api/providers/openai_compat_stream"
	"github.com/susugadx/xelyon-cli/internal/ui"
)

// handleStreamingResponse はストリーミングレスポンスを処理する。
func (p *Provider) handleStreamingResponse(ctx context.Context, resp *http.Response, spinner *ui.Spinner) (string, error) {
	out := api.OutputWriterFromContext(ctx)
	dim := color.New(color.Faint)
	reasoningActive := false
	p.lastReasoningContent = ""

	streamResult, err := openaicompatstream.ParseSSEStream(ctx, resp, spinner, openaicompatstream.ParseSSEOptions{
		ValidateData: func(data string) error {
			if err := api.ValidateStreamResponse([]byte(data)); err != nil {
				return fmt.Errorf("invalid response structure: %w", err)
			}
			return nil
		},
		UsageDecoder: decodeKimiUsage,
		OnReasoningContent: func(content string, first bool) {
			if first {
				reasoningActive = true
				spinner.Stop()
				dim.Fprint(out, "💭 ")
			}
			dim.Fprint(out, content)
		},
		OnReasoningBoundary: func() {
			if !reasoningActive {
				return
			}
			_, _ = fmt.Fprintln(out)
			_, _ = fmt.Fprintln(out)
			reasoningActive = false
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

	p.lastReasoningContent = streamResult.ReasoningContent
	if streamResult.Usage != nil && p.usageCallback != nil {
		p.usageCallback(*streamResult.Usage)
	}

	return openaicompatstream.BuildContentWithToolCalls(
		streamResult.Content,
		streamResult.ToolCalls,
		openai.ConvertToolCallToToolJSON,
	), nil
}

func decodeKimiUsage(raw json.RawMessage) (*api.Usage, error) {
	if !openaicompatstream.HasUsagePayload(raw) {
		return nil, nil
	}

	var usagePayload struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		CachedTokens     int `json:"cached_tokens,omitempty"`
	}
	if err := json.Unmarshal(raw, &usagePayload); err != nil {
		return nil, err
	}

	return &api.Usage{
		InputTokens:       usagePayload.PromptTokens,
		OutputTokens:      usagePayload.CompletionTokens,
		CachedInputTokens: usagePayload.CachedTokens,
	}, nil
}
