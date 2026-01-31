package agent

import (
	"fmt"
	"strings"
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

// globalAgentStatus は Agent からアクセスされる共有ステータス
// Agent 構造体を変更せずに利用するための簡易実装
var globalAgentStatus = statusHolder{
	status: defaultStatus(),
}

// SetStatus updates the current agent status.
func (a *Agent) SetStatus(state AgentState, reasonEN, reasonJP, nextEN, nextJP string) {
	globalAgentStatus.setStatus(AgentStatus{
		State:    state,
		ReasonEN: reasonEN,
		ReasonJP: reasonJP,
		NextEN:   nextEN,
		NextJP:   nextJP,
	})
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

	// 区切り線（dim色）
	fmt.Println()
	statusDim.Println(dividerLine)

	// ステータス行: ● model │ Mode │ tokens │ ~$cost
	// Ollama の場合はコスト非表示
	providerLower := strings.ToLower(a.ProviderName)
	if providerLower == "ollama" {
		fmt.Printf("%s %s %s %s %s %s\n",
			indicator,
			statusCyan.Sprint(a.CurrentModel),
			sep,
			modeText,
			sep,
			tokenStr)
	} else {
		fmt.Printf("%s %s %s %s %s %s %s ~$%.3f\n",
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
	statusDim.Println(dividerLine)
}
