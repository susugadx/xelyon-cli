package gemini

import (
	"context"
	"fmt"
	"time"
)

type sseLoopTransition struct {
	terminate      bool
	response       string
	err            error
	continueToDone bool
}

type sseLoopPolicy struct {
	ctx                  context.Context
	transportIdleTimeout time.Duration
}

func newSSELoopPolicy(ctx context.Context, transportIdleTimeout time.Duration) sseLoopPolicy {
	return sseLoopPolicy{
		ctx:                  ctx,
		transportIdleTimeout: transportIdleTimeout,
	}
}

func (p sseLoopPolicy) resolveContextDone(partial string, eventErr error) sseLoopTransition {
	if partial != "" {
		return sseLoopTransition{
			terminate: true,
			response:  partial,
		}
	}
	if eventErr != nil {
		return sseLoopTransition{
			terminate: true,
			err:       eventErr,
		}
	}
	return sseLoopTransition{
		terminate: true,
		err:       p.ctx.Err(),
	}
}

func (p sseLoopPolicy) resolveIdleTimeout(partial string) sseLoopTransition {
	return sseLoopTransition{
		terminate: true,
		response:  partial,
		err:       &ErrIdleTimeout{Message: fmt.Sprintf("transport idle timeout: no valid SSE data received for %v", p.transportIdleTimeout)},
	}
}

func (p sseLoopPolicy) resolveScannerDone(scanErr error) sseLoopTransition {
	if scanErr != nil {
		return sseLoopTransition{
			terminate: true,
			err:       fmt.Errorf("SSE scan error: %w", scanErr),
		}
	}
	return sseLoopTransition{
		continueToDone: true,
	}
}
