package agent

import (
	"context"

	"github.com/susugadx/xelyon-cli/internal/uiruntime"
)

func (a *Agent) requestPromptIO(ctx context.Context) uiruntime.PromptIO {
	promptIO := a.ui().PromptIO()
	promptIO.Context = ctx
	return promptIO
}

func (a *Agent) requestToolPromptContext(requestCtx context.Context) context.Context {
	return a.requestCancelOnlyContext(requestCtx)
}

// beginRequestPromptCancellationScope は request 中 prompt の明示キャンセル範囲を開始する。
// API deadline は引き継がず、Ctrl+C などの明示キャンセルだけを prompt に伝える。
func (a *Agent) beginRequestPromptCancellationScope(requestCtx context.Context) (context.Context, func()) {
	base := context.Background()
	if requestCtx != nil {
		base = context.WithoutCancel(requestCtx)
	}

	promptCtx, cancelPrompt := context.WithCancel(base)
	previousCancel := a.cancelFunc
	a.cancelFunc = cancelPrompt

	cleanup := func() {
		cancelPrompt()
		a.cancelFunc = previousCancel
	}
	return promptCtx, cleanup
}
