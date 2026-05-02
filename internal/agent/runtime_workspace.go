package agent

import (
	"os"
	"strings"
)

func resolveRuntimeInvocationCWD() string {
	cwd, err := os.Getwd()
	if err != nil {
		return ""
	}
	return cwd
}

func (r *AgentRuntime) refreshInvocationCWD() {
	if r == nil {
		return
	}
	r.InvocationCWD = resolveRuntimeInvocationCWD()
}

func (r *AgentRuntime) effectiveInvocationCWD() string {
	if r == nil {
		return resolveRuntimeInvocationCWD()
	}
	cwd := strings.TrimSpace(resolveRuntimeInvocationCWD())
	if cwd != "" {
		r.InvocationCWD = cwd
		return cwd
	}
	cwd = strings.TrimSpace(r.InvocationCWD)
	if cwd != "" {
		return cwd
	}
	return ""
}
