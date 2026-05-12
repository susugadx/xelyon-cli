package agent

import (
	"os"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/ledger"
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

func (r *AgentRuntime) ensureTaskLedger() {
	if r == nil {
		return
	}
	cwd := strings.TrimSpace(r.InvocationCWD)
	if cwd == "" {
		cwd = resolveRuntimeInvocationCWD()
		r.InvocationCWD = cwd
	}
	if r.TaskLedger != nil && r.TaskLedger != r.managedTaskLedger {
		r.managedTaskLedger = nil
		r.taskLedgerInvocationCWD = ""
		return
	}
	if r.TaskLedger != nil && r.taskLedgerInvocationCWD == cwd {
		return
	}
	r.TaskLedger = ledger.NewStoreForInvocationCWD(cwd)
	r.managedTaskLedger = r.TaskLedger
	r.taskLedgerInvocationCWD = cwd
}
