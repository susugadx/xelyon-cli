package agent

import (
	"context"
	"time"
)

func (a *Agent) beginChatRequestContext() (context.Context, func()) {
	return a.beginCancelableRequestContext(context.Background(), "request")
}

func requestContextErr(ctx context.Context) error {
	if ctx == nil {
		return nil
	}
	return ctx.Err()
}

// requestCancelOnlyContext は requestCtx の値を保ちつつ、API timeout/deadline は引き継がない。
// Ctrl+C などの明示キャンセルだけを request-scoped な後続処理へ伝える。
func (a *Agent) requestCancelOnlyContext(requestCtx context.Context) context.Context {
	if requestCtx == nil {
		requestCtx = context.Background()
	}
	if a == nil || a.requestPromptCancelCtx == nil {
		return requestCtx
	}
	return requestValueCancelContext{
		values: context.WithoutCancel(requestCtx),
		cancel: a.requestPromptCancelCtx,
	}
}

func (a *Agent) finalCheckParentContext(requestCtx context.Context) context.Context {
	return a.requestCancelOnlyContext(requestCtx)
}

type requestValueCancelContext struct {
	values context.Context
	cancel context.Context
}

func (c requestValueCancelContext) Deadline() (time.Time, bool) {
	return time.Time{}, false
}

func (c requestValueCancelContext) Done() <-chan struct{} {
	if c.cancel == nil {
		return nil
	}
	return c.cancel.Done()
}

func (c requestValueCancelContext) Err() error {
	if c.cancel == nil {
		return nil
	}
	return c.cancel.Err()
}

func (c requestValueCancelContext) Value(key any) any {
	if c.values == nil {
		return nil
	}
	return c.values.Value(key)
}

func (a *Agent) beginReviewRequestContext(ctx context.Context) (context.Context, func()) {
	return a.beginCancelableRequestContext(ctx, "review")
}

func (a *Agent) beginCancelableRequestContext(parent context.Context, operation string) (context.Context, func()) {
	if parent == nil {
		parent = context.Background()
	}
	if operation == "" {
		operation = "request"
	}

	cfg := a.cfg()
	timeout := time.Duration(cfg.APIRetry.Timeout) * time.Second
	interruptCtx, cancelInterrupt := context.WithCancel(parent)
	ctx, cancelTimeout := context.WithTimeout(interruptCtx, timeout)

	a.lastCancelReason = ""
	a.cancelFunc = cancelInterrupt
	a.requestCtx = ctx
	a.requestPromptCancelCtx = interruptCtx
	a.debugCancelf("%s started (timeout=%s, model=%s, provider=%s)", operation, timeout, a.CurrentModel, a.ProviderName)

	cleanup := func() {
		a.debugCancelf("%s finished (ctx_err=%v, cancel_reason=%q)", operation, ctx.Err(), a.lastCancelReason)
		cancelTimeout()
		cancelInterrupt()
		a.requestCtx = nil
		a.requestPromptCancelCtx = nil
		a.cancelFunc = nil
		a.lastCancelReason = ""
	}

	return ctx, cleanup
}
