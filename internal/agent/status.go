package agent

import (
	"fmt"
	"strings"
	"sync"

	"github.com/fatih/color"
	"github.com/susugadx/xelyon-cli/internal/agent/token"
	"github.com/susugadx/xelyon-cli/internal/config"
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
	statusDim    = color.New(color.Faint)
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
// Format: ● model │ Mode │ tokens/limit │ ~$cost
// This should be called right before showing the input prompt.
func (a *Agent) PrintStatusFooter() {
	const dividerLine = "────────────────────────────────────────────"

	// Mode 表示
	modeText := "Normal"
	if a.PlanModeEnabled {
		modeText = "Plan"
	}

	// トークン使用量（API実測値）
	tokens := a.Stats.TotalTokens()
	limit := token.GetModelTokenLimit(a.CurrentModel)
	tokenStr := FormatTokens(tokens)
	limitStr := FormatTokens(limit)
	percentage := float64(tokens) / float64(limit) * 100

	// コスト
	cost := a.Stats.EstimatedCost()

	// セパレータ（dim色）
	sep := statusDim.Sprint("│")

	// 警告閾値を config から取得
	cfg := config.GetGlobalConfig()
	threshold := float64(cfg.Compression.ThresholdPercent)
	if threshold == 0 {
		threshold = 80 // デフォルト値
	}

	// 色分け（緑 < threshold < 黄 < 100% < 赤）
	var indicator string
	var tokenDisplay string
	if percentage >= 100 {
		// 上限超過: 赤で警告
		indicator = statusRed.Sprint("●")
		tokenDisplay = statusRed.Sprintf("%s/%s ⚠️", tokenStr, limitStr)
	} else if percentage > threshold {
		// 警告閾値超過: 黄色
		indicator = statusYellow.Sprint("●")
		tokenDisplay = statusYellow.Sprintf("%s/%s", tokenStr, limitStr)
	} else {
		// 正常: 緑
		indicator = statusGreen.Sprint("●")
		tokenDisplay = fmt.Sprintf("%s/%s", tokenStr, limitStr)
	}

	// 区切り線（dim色）
	fmt.Println()
	statusDim.Println(dividerLine)

	// ステータス行: ● model │ Mode │ tokens/limit │ ~$cost
	// Ollama の場合はコスト非表示（strings.ToLower で判定）
	providerLower := strings.ToLower(a.ProviderName)
	if providerLower == "ollama" {
		fmt.Printf("%s %s %s %s %s %s\n",
			indicator,
			statusCyan.Sprint(a.CurrentModel),
			sep,
			modeText,
			sep,
			tokenDisplay)
	} else {
		fmt.Printf("%s %s %s %s %s %s %s ~$%.3f\n",
			indicator,
			statusCyan.Sprint(a.CurrentModel),
			sep,
			modeText,
			sep,
			tokenDisplay,
			sep,
			cost)
	}

	// 下の区切り線
	statusDim.Println(dividerLine)
}
