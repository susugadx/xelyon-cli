package ui

import "io"

type configPromptContext struct {
	runtime  *Runtime
	promptIO PromptIO
	out      io.Writer
}

func newConfigPromptContext(runtime *Runtime) configPromptContext {
	rt := runtimeOrDefault(runtime)
	rt.StopSpinner()
	rt.ResetTerminalState()
	promptIO := rt.PromptIO()
	return configPromptContext{
		runtime:  rt,
		promptIO: promptIO,
		out:      promptIO.Out,
	}
}
