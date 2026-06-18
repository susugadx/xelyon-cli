package claudestream

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/fatih/color"
	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/uiruntime"
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
	spinner *uiruntime.Spinner,
	handler EventHandler,
	opts RunnerOptions,
) (string, error) {
	if handler == nil {
		return "", fmt.Errorf("event handler is required")
	}

	state := newRunnerOutputState(ctx, spinner)
	defer state.stopSpinner()

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
	loopPolicy := newRunnerLoopPolicy(state, handler, opts, idleTimeout)

	for {
		eventResult := controller.Next(ctx, nil)
		transition := loopPolicy.resolve(eventResult)
		if transition.handled {
			return transition.response, transition.err
		}
	}
}
