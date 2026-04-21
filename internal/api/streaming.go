package api

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/fatih/color"
	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/ui"
)

// 警告表示用の色
var yellow = color.New(color.FgYellow)

// AI発言ヘッダー表示用の色
var cyanBold = color.New(color.FgCyan, color.Bold)

// PrintAIHeader はAI発言開始時のヘッダーを表示する
// 全プロバイダーから共通で使用
func PrintAIHeader() {
	PrintAIHeaderWithContext(context.Background())
}

// StreamParser はストリーミングレスポンスの1行をパースする関数型
// 戻り値: (content string, done bool, err error)
//   - content: この行から抽出されたテキストコンテンツ
//   - done: ストリームの終了を示すフラグ
//   - err: パースエラー
type StreamParser func(line string) (content string, done bool, err error)

// ParseStreamingResponse は共通のストリーミングレスポンス処理
// コンテキストキャンセル、スピナー制御、エラーハンドリングを統一的に処理
// アイドルタイムアウト方式: データ受信がない状態がN秒続くとタイムアウト
// ツールJSON部分は内部で記録するが表示しない
func ParseStreamingResponse(ctx context.Context, resp *http.Response, spinner *ui.Spinner, parser StreamParser) (string, error) {
	cfg := config.FromContext(ctx)
	idleTimeout := time.Duration(cfg.Streaming.IdleTimeoutSeconds) * time.Second
	errOut := errorWriterFromContext(ctx)

	controller := NewStreamLoopController(resp.Body, StreamLoopOptions{
		IdleTimeout:         idleTimeout,
		AutoResetIdleOnLine: true,
	})
	defer controller.Stop()

	state := newStreamOutputState(ctx, spinner)

	for {
		event := controller.Next(ctx, nil)
		switch event.Type {
		case StreamLoopEventContextDone:
			return state.finalizeContextDone(errOut, event.Err)

		case StreamLoopEventIdleTimeout:
			// アイドルタイムアウト
			state.stopSpinner()
			return state.response(), fmt.Errorf("idle timeout: no data received for %v", idleTimeout)

		case StreamLoopEventScannerDone:
			return state.finalizeScannerDone(event.Err)

		case StreamLoopEventLine:
			done, err := state.processLine(event.Line, parser)
			if err != nil {
				return state.finalizeParserError(err)
			}
			if done {
				return state.finalizeDone()
			}
		}
	}
}
