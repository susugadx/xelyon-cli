package api

import (
	"context"
	"io"

	"github.com/susugadx/xelyon-cli/internal/ui"
)

func uiRuntimeFromContext(ctx context.Context) *ui.Runtime {
	return ui.RuntimeFromContext(ctx)
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

// PrintAIHeaderWithContext は request context に紐づく出力先へ AI 発言ヘッダーを表示する。
func PrintAIHeaderWithContext(ctx context.Context) {
	cyanBold.Fprint(outputWriterFromContext(ctx), "\n💬 ")
}
