package api

import (
	"context"
	"io"

	"github.com/susugadx/xelyon-cli/internal/ui"
)

func uiRuntimeFromContext(ctx context.Context) *ui.Runtime {
	return ui.RuntimeFromContext(ctx)
}

// RuntimeFromContext は request context に紐づく UI runtime を返す。
func RuntimeFromContext(ctx context.Context) *ui.Runtime {
	return uiRuntimeFromContext(ctx)
}

// RuntimeOrDefault は nil のとき process 互換の default runtime を返す。
func RuntimeOrDefault(runtime *ui.Runtime) *ui.Runtime {
	if runtime != nil {
		return runtime
	}
	return ui.DefaultRuntime()
}

func outputWriterFromContext(ctx context.Context) io.Writer {
	return OutputWriterFromContext(ctx)
}

func errorWriterFromContext(ctx context.Context) io.Writer {
	return ErrorWriterFromContext(ctx)
}

// OutputWriterFromContext は request context に紐づく標準出力先を返す。
func OutputWriterFromContext(ctx context.Context) io.Writer {
	return uiRuntimeFromContext(ctx).Output()
}

// ErrorWriterFromContext は request context に紐づく標準エラー出力先を返す。
func ErrorWriterFromContext(ctx context.Context) io.Writer {
	return uiRuntimeFromContext(ctx).ErrorOutput()
}

// SpinnerFromContext は request context に紐づく spinner を返す。
func SpinnerFromContext(ctx context.Context) *ui.Spinner {
	return uiRuntimeFromContext(ctx).CurrentSpinner()
}

// StopSpinnerAndResetTerminal は request context に紐づく spinner を停止して terminal 状態を戻す。
func StopSpinnerAndResetTerminal(ctx context.Context) {
	runtime := uiRuntimeFromContext(ctx)
	runtime.StopSpinner()
	runtime.ResetTerminalState()
}

// PrintAIHeaderWithContext は request context に紐づく出力先へ AI 発言ヘッダーを表示する。
func PrintAIHeaderWithContext(ctx context.Context) {
	cyanBold.Fprint(outputWriterFromContext(ctx), "\n💬 ")
}
