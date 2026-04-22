package claudestream

type contextDoneResolution struct {
	err                  error
	printTrailingNewline bool
	warnPartial          bool
}

type runnerContextDonePolicy struct {
	opts RunnerOptions
}

func newRunnerContextDonePolicy(opts RunnerOptions) runnerContextDonePolicy {
	return runnerContextDonePolicy{opts: opts}
}

func (p runnerContextDonePolicy) resolve(partial string, eventErr, ctxErr error) contextDoneResolution {
	if partial != "" {
		resolution := contextDoneResolution{
			printTrailingNewline: true,
			warnPartial:          p.opts.WarnOnPartial,
		}
		if p.opts.CancelMode == CancelModePartialAsError {
			resolution.err = ctxErr
		}
		return resolution
	}

	if eventErr != nil {
		return contextDoneResolution{err: eventErr}
	}
	return contextDoneResolution{err: ctxErr}
}
