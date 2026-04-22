package claudestream

import (
	"time"

	"github.com/susugadx/xelyon-cli/internal/api"
)

type runnerLoopTransition struct {
	handled  bool
	response string
	err      error
}

type runnerLoopPolicy struct {
	state       *runnerOutputState
	handler     EventHandler
	opts        RunnerOptions
	idleTimeout time.Duration
}

func newRunnerLoopPolicy(state *runnerOutputState, handler EventHandler, opts RunnerOptions, idleTimeout time.Duration) runnerLoopPolicy {
	return runnerLoopPolicy{
		state:       state,
		handler:     handler,
		opts:        opts,
		idleTimeout: idleTimeout,
	}
}

func (p runnerLoopPolicy) resolve(eventResult api.StreamLoopEvent) runnerLoopTransition {
	switch eventResult.Type {
	case api.StreamLoopEventContextDone:
		response, err := p.state.finalizeContextDone(p.opts, eventResult.Err)
		return runnerLoopTransition{
			handled:  true,
			response: response,
			err:      err,
		}
	case api.StreamLoopEventIdleTimeout:
		response, err := p.state.finalizeIdleTimeout(p.idleTimeout)
		return runnerLoopTransition{
			handled:  true,
			response: response,
			err:      err,
		}
	case api.StreamLoopEventScannerDone:
		response, err := p.state.finalizeScannerDone(eventResult.Err)
		return runnerLoopTransition{
			handled:  true,
			response: response,
			err:      err,
		}
	case api.StreamLoopEventLine:
		return p.resolveLine(eventResult.Line)
	default:
		return runnerLoopTransition{}
	}
}

func (p runnerLoopPolicy) resolveLine(line string) runnerLoopTransition {
	lineResult := p.state.processLineEvent(line, p.handler, p.opts.IgnoreDecodeError)
	if lineResult.skip {
		return runnerLoopTransition{}
	}
	p.state.syncNewlineForActiveSpinner()

	if lineResult.err != nil {
		if lineResult.decodeErr {
			response, err := p.state.finalizeDecodeError(lineResult.err)
			return runnerLoopTransition{
				handled:  true,
				response: response,
				err:      err,
			}
		}

		response, err := p.state.finalizeHandlerError(lineResult.err)
		return runnerLoopTransition{
			handled:  true,
			response: response,
			err:      err,
		}
	}

	if lineResult.done {
		response, err := p.state.finalizeDone()
		return runnerLoopTransition{
			handled:  true,
			response: response,
			err:      err,
		}
	}

	p.state.appendTextDelta(lineResult.textDelta)
	return runnerLoopTransition{}
}
