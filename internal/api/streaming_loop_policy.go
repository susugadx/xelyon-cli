package api

import (
	"fmt"
	"io"
	"time"
)

type parseStreamingLoopTransition struct {
	handled  bool
	response string
	err      error
}

type parseStreamingLoopPolicy struct {
	state       *streamOutputState
	errOut      io.Writer
	idleTimeout time.Duration
}

func newParseStreamingLoopPolicy(state *streamOutputState, errOut io.Writer, idleTimeout time.Duration) parseStreamingLoopPolicy {
	return parseStreamingLoopPolicy{
		state:       state,
		errOut:      errOut,
		idleTimeout: idleTimeout,
	}
}

func (p parseStreamingLoopPolicy) resolve(event StreamLoopEvent) parseStreamingLoopTransition {
	switch event.Type {
	case StreamLoopEventContextDone:
		response, err := p.state.finalizeContextDone(p.errOut, event.Err)
		return parseStreamingLoopTransition{
			handled:  true,
			response: response,
			err:      err,
		}
	case StreamLoopEventIdleTimeout:
		p.state.stopSpinner()
		return parseStreamingLoopTransition{
			handled:  true,
			response: p.state.response(),
			err:      fmt.Errorf("idle timeout: no data received for %v", p.idleTimeout),
		}
	case StreamLoopEventScannerDone:
		response, err := p.state.finalizeScannerDone(event.Err)
		return parseStreamingLoopTransition{
			handled:  true,
			response: response,
			err:      err,
		}
	default:
		return parseStreamingLoopTransition{}
	}
}
