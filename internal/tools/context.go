package tools

import (
	"io"
	"os"

	"github.com/susugadx/xelyon-cli/internal/tools/common"
	"github.com/susugadx/xelyon-cli/internal/ui"
)

// ExecutionContext はツール実行時の周辺コンテキストを保持する。
// web_search などが現在のプロバイダー/モデルや対話 I/O を参照するために使用する。
// 各実行経路が明示的に組み立てて注入し、process-global 状態には依存しない。
type ExecutionContext struct {
	ProviderName string
	Model        string
	Stdin        io.Reader
	Stdout       io.Writer
	Stderr       io.Writer
	PromptReader *ui.MultilineReader
}

// DefaultExecutionContext は標準入出力を使う実行コンテキストを返す。
func DefaultExecutionContext() ExecutionContext {
	return normalizeExecutionContext(ExecutionContext{})
}

// Output は common.Output へ変換する。
func (ctx ExecutionContext) Output() common.Output {
	normalized := normalizeExecutionContext(ctx)
	return common.NewOutput(normalized.Stdout, normalized.Stderr)
}

// PromptIO は対話 UI 用の入出力コンテキストへ変換する。
func (ctx ExecutionContext) PromptIO() ui.PromptIO {
	normalized := normalizeExecutionContext(ctx)
	return ui.NewPromptIO(normalized.Stdin, normalized.Stdout, normalized.Stderr, normalized.PromptReader)
}

func normalizeExecutionContext(ctx ExecutionContext) ExecutionContext {
	if ctx.Stdin == nil {
		ctx.Stdin = os.Stdin
	}
	if ctx.Stdout == nil {
		ctx.Stdout = os.Stdout
	}
	if ctx.Stderr == nil {
		ctx.Stderr = os.Stderr
	}
	return ctx
}
