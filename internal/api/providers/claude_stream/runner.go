package claudestream

import (
	"bufio"
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

	out := api.OutputWriterFromContext(ctx)
	errOut := api.ErrorWriterFromContext(ctx)
	streamAssistantText := api.ShouldStreamAssistantText(ctx)

	var fullResponse strings.Builder
	firstChunk := true
	contentNewlineEmitted := false

	type scanResult struct {
		line string
		err  error
		done bool
	}
	lineCh := make(chan scanResult)

	go func() {
		defer close(lineCh)
		scanner := bufio.NewScanner(resp.Body)
		buf := make([]byte, 0, 10*1024*1024)
		scanner.Buffer(buf, 10*1024*1024)
		for scanner.Scan() {
			lineCh <- scanResult{line: scanner.Text()}
		}
		if err := scanner.Err(); err != nil {
			lineCh <- scanResult{err: err, done: true}
		} else {
			lineCh <- scanResult{done: true}
		}
	}()

	var idleTimer *time.Timer
	var idleTimeout time.Duration
	if opts.EnableIdleTimeout {
		cfg := config.FromContext(ctx)
		idleTimeout = time.Duration(cfg.Streaming.IdleTimeoutSeconds) * time.Second
		idleTimer = time.NewTimer(idleTimeout)
		defer idleTimer.Stop()
	}

	resetIdleTimer := func() {
		if idleTimer == nil {
			return
		}
		if !idleTimer.Stop() {
			select {
			case <-idleTimer.C:
			default:
			}
		}
		idleTimer.Reset(idleTimeout)
	}

	for {
		var idleTimerCh <-chan time.Time
		if idleTimer != nil {
			idleTimerCh = idleTimer.C
		}

		select {
		case <-ctx.Done():
			spinner.Stop()
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
			return "", ctx.Err()

		case <-idleTimerCh:
			spinner.Stop()
			return fullResponse.String(), fmt.Errorf("idle timeout: no data received for %v", idleTimeout)

		case result, ok := <-lineCh:
			if !ok {
				spinner.Stop()
				if streamAssistantText && !firstChunk && !contentNewlineEmitted {
					_, _ = fmt.Fprintln(out)
				}
				return fullResponse.String(), nil
			}

			resetIdleTimer()

			if result.done {
				spinner.Stop()
				if result.err != nil {
					return fullResponse.String(), fmt.Errorf("stream reading error: %w", result.err)
				}
				if streamAssistantText && !firstChunk && !contentNewlineEmitted {
					_, _ = fmt.Fprintln(out)
				}
				return fullResponse.String(), nil
			}

			if result.line == "" {
				continue
			}

			data, handled := ParseSSEDataLine(result.line)
			if !handled {
				continue
			}

			event, err := DecodeEvent(data)
			if err != nil {
				if opts.IgnoreDecodeError {
					continue
				}
				spinner.Stop()
				if streamAssistantText && !firstChunk && !contentNewlineEmitted {
					_, _ = fmt.Fprintln(out)
				}
				return fullResponse.String(), err
			}

			textDelta, done, err := handler(event, data)
			if streamAssistantText && !firstChunk && spinner.IsActive() && !contentNewlineEmitted {
				_, _ = fmt.Fprintln(out)
				contentNewlineEmitted = true
			}

			if err != nil {
				spinner.Stop()
				if streamAssistantText && !firstChunk && !contentNewlineEmitted {
					_, _ = fmt.Fprintln(out)
				}
				return fullResponse.String(), err
			}

			if done {
				spinner.Stop()
				if streamAssistantText && !firstChunk && !contentNewlineEmitted {
					_, _ = fmt.Fprintln(out)
				}
				return fullResponse.String(), nil
			}

			if textDelta == "" {
				continue
			}

			if firstChunk {
				spinner.Stop()
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
