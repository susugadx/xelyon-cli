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
	if strings.TrimSpace(r.InvocationCWD) != "" {
		return
	}
	r.InvocationCWD = resolveRuntimeInvocationCWD()
}

func (r *AgentRuntime) effectiveInvocationCWD() string {
	if r != nil {
		if cwd := strings.TrimSpace(r.InvocationCWD); cwd != "" {
			return cwd
		}
	}
	if cwd := strings.TrimSpace(resolveRuntimeInvocationCWD()); cwd != "" {
		return cwd
	}
	return ""
}
