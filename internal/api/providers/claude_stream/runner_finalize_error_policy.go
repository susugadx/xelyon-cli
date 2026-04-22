package claudestream

import (
	"fmt"
	"time"
)

type runnerFinalizeErrorResolution struct {
	err                  error
	printTrailingNewline bool
}

type runnerFinalizeErrorPolicy struct{}

func newRunnerFinalizeErrorPolicy() runnerFinalizeErrorPolicy {
	return runnerFinalizeErrorPolicy{}
}

func (p runnerFinalizeErrorPolicy) resolveWithTrailingNewline(err error) runnerFinalizeErrorResolution {
	return runnerFinalizeErrorResolution{
		err:                  err,
		printTrailingNewline: true,
	}
}

func (p runnerFinalizeErrorPolicy) resolveIdleTimeout(idleTimeout time.Duration) runnerFinalizeErrorResolution {
	return runnerFinalizeErrorResolution{
		err: fmt.Errorf("idle timeout: no data received for %v", idleTimeout),
	}
}

func (p runnerFinalizeErrorPolicy) resolveScannerDone(scanErr error) runnerFinalizeErrorResolution {
	if scanErr != nil {
		return runnerFinalizeErrorResolution{
			err: fmt.Errorf("stream reading error: %w", scanErr),
		}
	}
	return runnerFinalizeErrorResolution{
		printTrailingNewline: true,
	}
}

func (p runnerFinalizeErrorPolicy) resolveDecodeError(err error) runnerFinalizeErrorResolution {
	return p.resolveWithTrailingNewline(err)
}

func (p runnerFinalizeErrorPolicy) resolveHandlerError(err error) runnerFinalizeErrorResolution {
	return p.resolveWithTrailingNewline(err)
}

func (p runnerFinalizeErrorPolicy) resolveDone() runnerFinalizeErrorResolution {
	return p.resolveWithTrailingNewline(nil)
}
