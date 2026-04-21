package claudestream

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/ui"
)

// CancelMode は context cancel 時の部分応答返却方針。
type CancelMode int

const (
	// CancelModePartialAsSuccess は partial を成功として返す（error=nil）。
	CancelModePartialAsSuccess CancelMode = iota
	// CancelModePartialAsError は partial と context error を同時に返す。
	CancelModePartialAsError
)

// RunnerOptions は Claude 互換 SSE ランナーの挙動オプション。
type RunnerOptions struct {
	CancelMode        CancelMode
	WarnOnPartial     bool
	IgnoreDecodeError bool
	EnableIdleTimeout bool
}

// EventHandler はイベントごとの provider 差分処理を受け取る。
// 戻り値: (textDelta, done, err)
type EventHandler func(event StreamEvent, rawData string) (string, bool, error)

var cancelWarnColor = color.New(color.FgYellow)

// RunStreamingResponse は Claude 互換 SSE ストリームの共通読み取りループを実行する。
// provider 側は handler で差分イベント処理のみを持つ。
func RunStreamingResponse(
	ctx context.Context,
	resp *http.Response,
	spinner *ui.Spinner,
	handler EventHandler,
	opts RunnerOptions,
) (string, error) {
	if handler == nil {
		return "", fmt.Errorf("event handler is required")
	}

	stopSpinner := func() {
		if spinner != nil {
			spinner.Stop()
		}
	}
	defer stopSpinner()

	out := api.OutputWriterFromContext(ctx)
	errOut := api.ErrorWriterFromContext(ctx)
	streamAssistantText := api.ShouldStreamAssistantText(ctx)

	var fullResponse strings.Builder
	firstChunk := true
	contentNewlineEmitted := false

	var idleTimeout time.Duration
	if opts.EnableIdleTimeout {
		cfg := config.FromContext(ctx)
		idleTimeout = time.Duration(cfg.Streaming.IdleTimeoutSeconds) * time.Second
	}

	controller := api.NewStreamLoopController(resp.Body, api.StreamLoopOptions{
		IdleTimeout:         idleTimeout,
		AutoResetIdleOnLine: true,
	})
	defer controller.Stop()

	for {
		eventResult := controller.Next(ctx, nil)
		switch eventResult.Type {
		case api.StreamLoopEventContextDone:
			stopSpinner()
			partial := fullResponse.String()
			if partial != "" {
				if streamAssistantText && !firstChunk && !contentNewlineEmitted {
					_, _ = fmt.Fprintln(out)
				}
				if opts.WarnOnPartial {
					cancelWarnColor.Fprintln(errOut, "\n⚠️  Response interrupted. Partial result returned.")
				}
				if opts.CancelMode == CancelModePartialAsError {
					return partial, ctx.Err()
				}
				return partial, nil
			}
			if eventResult.Err != nil {
				return "", eventResult.Err
			}
			return "", ctx.Err()

		case api.StreamLoopEventIdleTimeout:
			stopSpinner()
			return fullResponse.String(), fmt.Errorf("idle timeout: no data received for %v", idleTimeout)

		case api.StreamLoopEventScannerDone:
			stopSpinner()
			if eventResult.Err != nil {
				return fullResponse.String(), fmt.Errorf("stream reading error: %w", eventResult.Err)
			}
			if streamAssistantText && !firstChunk && !contentNewlineEmitted {
				_, _ = fmt.Fprintln(out)
			}
			return fullResponse.String(), nil

		case api.StreamLoopEventLine:
			line := eventResult.Line
			if line == "" {
				continue
			}

			data, handled := ParseSSEDataLine(line)
			if !handled {
				continue
			}

			event, err := DecodeEvent(data)
			if err != nil {
				if opts.IgnoreDecodeError {
					continue
				}
				stopSpinner()
				if streamAssistantText && !firstChunk && !contentNewlineEmitted {
					_, _ = fmt.Fprintln(out)
				}
				return fullResponse.String(), err
			}

			textDelta, done, err := handler(event, data)
			if streamAssistantText && !firstChunk && spinner != nil && spinner.IsActive() && !contentNewlineEmitted {
				_, _ = fmt.Fprintln(out)
				contentNewlineEmitted = true
			}

			if err != nil {
				stopSpinner()
				if streamAssistantText && !firstChunk && !contentNewlineEmitted {
					_, _ = fmt.Fprintln(out)
				}
				return fullResponse.String(), err
			}

			if done {
				stopSpinner()
				if streamAssistantText && !firstChunk && !contentNewlineEmitted {
					_, _ = fmt.Fprintln(out)
				}
				return fullResponse.String(), nil
			}

			if textDelta == "" {
				continue
			}

			if firstChunk {
				stopSpinner()
				firstChunk = false
				api.PrintAIHeaderWithContext(ctx)
			}
			if streamAssistantText {
				_, _ = fmt.Fprint(out, textDelta)
				contentNewlineEmitted = false
			}
			fullResponse.WriteString(textDelta)
		}
	}
}
