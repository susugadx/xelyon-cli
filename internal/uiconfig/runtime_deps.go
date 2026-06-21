package uiconfig

import "github.com/susugadx/xelyon-cli/internal/uiruntime"

type Runtime = uiruntime.Runtime
type PromptIO = uiruntime.PromptIO

const (
	colorCyan  = "\033[36m"
	colorGreen = "\033[32m"
	colorDim   = "\033[2m"
	colorReset = "\033[0m"
)

func DefaultRuntime() *Runtime {
	return uiruntime.DefaultRuntime()
}

func runtimeOrDefault(runtime *Runtime) *Runtime {
	if runtime == nil {
		return uiruntime.DefaultRuntime()
	}
	return runtime
}

func StripBracketedPaste(s string) string {
	return uiruntime.StripBracketedPaste(s)
}
