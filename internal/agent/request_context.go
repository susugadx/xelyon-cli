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
