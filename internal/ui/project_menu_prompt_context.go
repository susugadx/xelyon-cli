package ui

import "io"

type projectPromptContext struct {
	runtime  *Runtime
	promptIO PromptIO
	out      io.Writer
}

func newProjectPromptContext(runtime *Runtime) projectPromptContext {
	rt := runtimeOrDefault(runtime)
	rt.StopSpinner()
	rt.ResetTerminalState()
	promptIO := rt.PromptIO()
	return projectPromptContext{
		runtime:  rt,
		promptIO: promptIO,
		out:      promptIO.Out,
	}
}
