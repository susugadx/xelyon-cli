package agent

import (
	"context"
	"time"

	"github.com/susugadx/xelyon-cli/internal/ui"
)

func (a *Agent) requestPromptIO(ctx context.Context) ui.PromptIO {
	promptIO := a.ui().PromptIO()
	promptIO.Context = ctx
	return promptIO
}

func (a *Agent) requestToolPromptContext(requestCtx context.Context) context.Context {
	if requestCtx == nil {
		requestCtx = context.Background()
	}
	if a == nil || a.requestPromptCancelCtx == nil {
		return requestCtx
	}
	return requestPromptContext{
		values: context.WithoutCancel(requestCtx),
		cancel: a.requestPromptCancelCtx,
	}
}

type requestPromptContext struct {
	values context.Context
	cancel context.Context
}

func (c requestPromptContext) Deadline() (time.Time, bool) {
	return time.Time{}, false
}

func (c requestPromptContext) Done() <-chan struct{} {
	if c.cancel == nil {
		return nil
	}
	return c.cancel.Done()
}

func (c requestPromptContext) Err() error {
	if c.cancel == nil {
		return nil
	}
	return c.cancel.Err()
}

func (c requestPromptContext) Value(key any) any {
	if c.values == nil {
		return nil
	}
	return c.values.Value(key)
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
