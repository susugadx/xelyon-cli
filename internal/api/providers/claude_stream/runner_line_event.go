package claudestream

type lineProcessResult struct {
	textDelta string
	done      bool
	err       error
	decodeErr bool
	skip      bool
}

func (s *runnerOutputState) processLineEvent(line string, handler EventHandler, ignoreDecodeError bool) lineProcessResult {
	if line == "" {
		return lineProcessResult{skip: true}
	}

	data, handled := ParseSSEDataLine(line)
	if !handled {
		return lineProcessResult{skip: true}
	}

	event, err := DecodeEvent(data)
	if err != nil {
		if ignoreDecodeError {
			return lineProcessResult{skip: true}
		}
		return lineProcessResult{err: err, decodeErr: true}
	}

	textDelta, done, handlerErr := handler(event, data)
	if handlerErr != nil {
		return lineProcessResult{err: handlerErr}
	}

	return lineProcessResult{
		textDelta: textDelta,
		done:      done,
	}
}
