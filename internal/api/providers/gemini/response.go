package gemini

import (
	"context"
	"errors"
	"net/http"
	"os"
	"time"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/ui"
)

// ErrThinkingTimeout はthought は流れているが actionable output が進まない場合のタイムアウトエラー
type ErrThinkingTimeout struct {
	Message string
}

// Error はエラーメッセージを返す
func (e *ErrThinkingTimeout) Error() string {
	return e.Message
}

// isThinkingTimeoutError はエラーがErrThinkingTimeoutかどうかを判定する
func isThinkingTimeoutError(err error) bool {
	var target *ErrThinkingTimeout
	return errors.As(err, &target)
}

// ErrIdleTimeout はSSE ストリームで有効な data を受信できない場合の transport idle timeout エラー
type ErrIdleTimeout struct {
	Message string
}

func (e *ErrIdleTimeout) Error() string {
	return e.Message
}

// ErrResponseStartTimeout はHTTPレスポンスヘッダー受信前のタイムアウトエラー
// Google側でリクエスト処理が詰まり、SSEストリームが開始されない場合に発生
type ErrResponseStartTimeout struct {
	Message string
}

// Error はエラーメッセージを返す
func (e *ErrResponseStartTimeout) Error() string {
	return e.Message
}

// handleSSEResponse は streamGenerateContent?alt=sse の SSE ストリームを処理する
// thinkingMsg はSSEストリーム開始時にスピナーを切り替えるメッセージ（空なら切り替えなし）
func (p *Provider) handleSSEResponse(ctx context.Context, resp *http.Response, spinner *ui.Spinner, thinkingMsg string) (string, error) {
	debug := os.Getenv("XELYON_DEBUG_GEMINI") == "1"
	state := newSSEInterpretState(ctx, spinner, thinkingMsg, debug)
	// どの return 経路でもスピナー停止を保証し、描画崩れを防ぐ。
	defer state.stopSpinner()

	cfg := config.FromContext(ctx)
	transportIdleTimeout := time.Duration(cfg.Streaming.IdleTimeoutSeconds) * time.Second
	loopPolicy := newSSELoopPolicy(ctx, transportIdleTimeout)
	controller := api.NewStreamLoopController(resp.Body, api.StreamLoopOptions{
		IdleTimeout:         transportIdleTimeout,
		AutoResetIdleOnLine: false, // Gemini は「有効な data chunk」を受信したときだけ idle をリセットする
	})
	defer controller.Stop()

	// thinking/progress タイマー: actionable output を受信せず thought のみが続く場合のタイムアウト
	thinkingTimeout := time.Duration(cfg.Streaming.ThinkingTimeoutSeconds) * time.Second
	if thinkingTimeout <= 0 {
		thinkingTimeout = 300 * time.Second // フォールバック
	}
	thinkingTimer := time.NewTimer(thinkingTimeout)
	defer thinkingTimer.Stop()

loop:
	for {
		eventResult := controller.Next(ctx, thinkingTimer.C)
		switch eventResult.Type {
		case api.StreamLoopEventContextDone:
			transition := loopPolicy.resolveContextDone(state.response(), eventResult.Err)
			if transition.terminate {
				return transition.response, transition.err
			}

		case api.StreamLoopEventIdleTimeout:
			transition := loopPolicy.resolveIdleTimeout(state.response())
			return transition.response, transition.err

		case api.StreamLoopEventExternal:
			if err := state.handleThinkingTimeout(thinkingTimeout); err != nil {
				return state.response(), err
			}
			thinkingTimer.Reset(thinkingTimeout)

		case api.StreamLoopEventScannerDone:
			transition := loopPolicy.resolveScannerDone(eventResult.Err)
			if transition.terminate {
				return transition.response, transition.err
			}
			if transition.continueToDone {
				break loop
			}

		case api.StreamLoopEventLine:
			data, handled := parseGeminiSSEDataLine(eventResult.Line)
			if !handled {
				continue
			}

			chunk, err := decodeGeminiSSEChunk(data)
			if err != nil {
				state.debugf("[DEBUG Gemini SSE] Failed to unmarshal chunk: %v\n", err)
				continue
			}
			controller.ResetIdleTimer()
			state.processChunk(ctx, chunk, thinkingTimer, thinkingTimeout)
		}
	}

	return state.finalize(p)
}
