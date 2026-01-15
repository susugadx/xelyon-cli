package agent

import (
	"fmt"
	"sync"

	"github.com/fatih/color"
)

// AgentState represents the current interaction state.
// This is used to make it obvious whether XELYON is waiting for input,
// waiting for approval, running, or aborted.
type AgentState string

const (
	StateRunning         AgentState = "running"
	StateWaitingInput    AgentState = "waiting_input"
	StateWaitingApproval AgentState = "waiting_approval"
	StateAborted         AgentState = "aborted"
	StateCompleted       AgentState = "completed"
)

type AgentStatus struct {
	State      AgentState
	ReasonEN   string
	ReasonJP   string
	NextEN     string
	NextJP     string
	LastUpdate int64 // unix nanos (debug/ordering; optional)
}

var (
	statusCyan   = color.New(color.FgCyan)
	statusGreen  = color.New(color.FgGreen)
	statusYellow = color.New(color.FgYellow)
	statusRed    = color.New(color.FgRed)
)

// statusHolder is embedded into Agent to keep state + mutex.
type statusHolder struct {
	mu     sync.RWMutex
	status AgentStatus
}

func (h *statusHolder) setStatus(s AgentStatus) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.status = s
}

func (h *statusHolder) getStatus() AgentStatus {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.status
}

func defaultStatus() AgentStatus {
	return AgentStatus{
		State:    StateWaitingInput,
		ReasonEN: "Ready for input",
		ReasonJP: "入力待ち",
		NextEN:   "Type your request or a command like /help",
		NextJP:   "リクエスト、または /help などのコマンドを入力",
	}
}

// SetStatus updates the current agent status.
func (a *Agent) SetStatus(state AgentState, reasonEN, reasonJP, nextEN, nextJP string) {
	holderFor(a).setStatus(AgentStatus{
		State:    state,
		ReasonEN: reasonEN,
		ReasonJP: reasonJP,
		NextEN:   nextEN,
		NextJP:   nextJP,
	})
}

// PrintStatusFooter prints a short, bilingual status line.
// This should be called right before showing the input prompt.
func (a *Agent) PrintStatusFooter() {
	s := a.status.getStatus()

	// Compact 2-line footer (EN/JP) to make state obvious even after long outputs.
	label := "Status"
	stateText := string(s.State)

	printer := statusCyan
	switch s.State {
	case StateWaitingInput, StateCompleted:
		printer = statusGreen
	case StateWaitingApproval:
		printer = statusYellow
	case StateAborted:
		printer = statusRed
	case StateRunning:
		printer = statusCyan
	}

	printer.Printf("\n[%s] %s | %s / %s\n", label, stateText, s.ReasonEN, s.ReasonJP)
	if s.NextEN != "" || s.NextJP != "" {
		fmt.Printf("  Next: %s / %s\n", s.NextEN, s.NextJP)
	}
}
