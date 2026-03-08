package agent

import (
	"fmt"
	"strings"
	"sync"

	"github.com/fatih/color"
	"github.com/susugadx/xelyon-cli/internal/agent/token"
	"github.com/susugadx/xelyon-cli/internal/ui"
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
	statusCyan  = color.New(color.FgCyan)
	statusGreen = color.New(color.FgGreen)
	statusDim   = color.New(color.Faint)
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
	a.statusRef().setStatus(AgentStatus{
		State:    state,
		ReasonEN: reasonEN,
		ReasonJP: reasonJP,
		NextEN:   nextEN,
		NextJP:   nextJP,
	})
}

func (a *Agent) statusRef() *statusHolder {
	if a == nil {
		holder := &statusHolder{status: defaultStatus()}
		return holder
	}
	if a.status.getStatus().State == "" {
		a.status.setStatus(defaultStatus())
	}
	return &a.status
}

// PrintStatusFooter prints a status bar with divider lines.
// Format: ● model │ Mode │ tokens │ ~$cost
// This should be called right before showing the input prompt.
func (a *Agent) PrintStatusFooter() {
	const dividerLine = "────────────────────────────────────────────"

	// Mode 表示
	modeText := "Normal"
	if a.PlanModeEnabled {
		modeText = "Plan"
	}

	// トークン使用量（API実測値・累計のみ）
	tokens := a.Stats.TotalTokens()
	tokenStr := FormatTokens(tokens)

	// コスト
	cost := a.Stats.EstimatedCost()

	// セパレータ（dim色）
	sep := statusDim.Sprint("│")

	// インジケーター（常に緑）
	indicator := statusGreen.Sprint("●")
	out := a.output()

	// 区切り線（dim色）
	_, _ = fmt.Fprintln(out)
	statusDim.Fprintln(out, dividerLine)

	// ステータス行: ● model │ Mode │ tokens │ ~$cost
	// Ollama の場合はコスト非表示
	providerLower := strings.ToLower(a.ProviderName)
	if providerLower == "ollama" {
		_, _ = fmt.Fprintf(out, "%s %s %s %s %s %s\n",
			indicator,
			statusCyan.Sprint(a.CurrentModel),
			sep,
			modeText,
			sep,
			tokenStr)
	} else {
		_, _ = fmt.Fprintf(out, "%s %s %s %s %s %s %s ~$%.3f\n",
			indicator,
			statusCyan.Sprint(a.CurrentModel),
			sep,
			modeText,
			sep,
			tokenStr,
			sep,
			cost)
	}

	// 下の区切り線
	statusDim.Fprintln(out, dividerLine)
}

// handleStatusCommand は現在の状態、直近リクエスト、セッション統計を表示する。
func handleStatusCommand(agent *Agent) bool {
	out := agent.output()
	printCommandHeaderToWriter(out, "Status / 状態")
	_, _ = fmt.Fprintln(out)

	modeText := "Normal"
	if agent.PlanModeEnabled {
		modeText = "Plan"
	}

	status := agent.statusRef().getStatus()
	currentTokens := agent.EstimateTokens()
	limit := token.GetModelTokenLimit(agent.CurrentModel)
	contextText := formatNumber(currentTokens)
	if limit > 0 {
		contextText = fmt.Sprintf("%s / %s (%.1f%%)", formatNumber(currentTokens), formatNumber(limit), float64(currentTokens)/float64(limit)*100)
	}

	statusTable := ui.NewTable().
		AddRow("State", string(status.State)).
		AddRow("Reason", status.ReasonEN).
		AddRow("Next", status.NextEN).
		AddRow("Mode", modeText).
		AddRow("Provider", agent.ProviderName).
		AddRow("Model", agent.CurrentModel).
		AddRow("Context", contextText)
	_, _ = fmt.Fprint(out, statusTable.RenderCompact())

	_, _ = fmt.Fprintln(out)
	green.Fprintln(out, "🧾 Last Request")
	if agent.Stats != nil {
		if table := buildLastRequestTable(agent.ProviderName, agent.CurrentModel, agent.Stats.LastUsage); table != nil {
			_, _ = fmt.Fprint(out, table.RenderCompact())
		} else {
			dim.Fprintln(out, "  No request usage data available")
		}
	} else {
		dim.Fprintln(out, "  No request usage data available")
	}

	printSessionSections(agent)
	_, _ = fmt.Fprintln(out)
	return true
}
