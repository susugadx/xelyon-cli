package agent

import (
	"fmt"
	"io"
	"sync"

	"github.com/fatih/color"
	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/commandcatalog"
	"github.com/susugadx/xelyon-cli/internal/cost"
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
		NextEN:   "Type your request or / for commands",
		NextJP:   "リクエスト、または / でコマンド候補を入力",
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

// FormatStatusLine はステータスバーに表示する文字列を返す。
// Format: ● model │ provider │ Plan: ON/OFF │ tokens │ ~$cost
func (a *Agent) FormatStatusLine() string {
	modeText := planModeStatusText(a.PlanModeEnabled)

	// TUI の spinner tick goroutine と agent.chat goroutine から同時にアクセスされるためロック
	a.statsMu.Lock()
	tokens := a.Stats.TotalTokens()
	tokenStr := FormatTokens(tokens)
	estimate := a.Stats.EstimatedCostEstimateForConfig(a.cfg())
	a.statsMu.Unlock()

	if manager := a.subAgentManager(); manager != nil {
		summary := manager.GetSummary()
		estimate.Cost += summary.TotalCost
		if summary.PricingUnavailable {
			estimate.PricingUnavailable = true
		}
	}

	sep := statusDim.Sprint("│")
	indicator := statusGreen.Sprint("●")

	providerDisplay := a.ProviderName
	if shouldSuppressLocalCostDisplay(providerDisplay, estimate) {
		return fmt.Sprintf("%s %s %s %s %s %s %s %s",
			indicator,
			statusCyan.Sprint(a.CurrentModel),
			sep,
			providerDisplay,
			sep,
			modeText,
			sep,
			tokenStr)
	}
	return fmt.Sprintf("%s %s %s %s %s %s %s %s %s %s",
		indicator,
		statusCyan.Sprint(a.CurrentModel),
		sep,
		providerDisplay,
		sep,
		modeText,
		sep,
		tokenStr,
		sep,
		formatCompactCostEstimate(estimate))
}

// PrintStatusFooter prints a status bar with divider lines.
// Format: ● model │ provider │ Plan: ON/OFF │ tokens │ ~$cost
// This should be called right before showing the input prompt.
func (a *Agent) PrintStatusFooter() {
	const dividerLine = "────────────────────────────────────────────"

	out := a.output()

	// 区切り線（dim色）
	_, _ = fmt.Fprintln(out)
	statusDim.Fprintln(out, dividerLine)
	_, _ = fmt.Fprintln(out, a.FormatStatusLine())
	statusDim.Fprintln(out, dividerLine)
}

func handleStatusCommandForSurface(agent *Agent, commandSurface commandcatalog.CommandSurface) bool {
	out := agent.output()
	printCommandHeaderToWriter(out, "Status / 状態")
	_, _ = fmt.Fprintln(out)

	modeText := planModeStatusText(agent.PlanModeEnabled)

	status := agent.statusRef().getStatus()
	currentTokens := agent.EstimateTokens()
	limit := agent.currentModelTokenLimit(agent.cfg())
	contextText := formatNumber(currentTokens)
	if limit > 0 {
		contextText = fmt.Sprintf("%s / %s (%.1f%%)", formatNumber(currentTokens), formatNumber(limit), float64(currentTokens)/float64(limit)*100)
	}

	statusTable := ui.NewTable().
		AddRow("State", string(status.State)).
		AddRow("Surface", statusSurfaceSummary(commandSurface)).
		AddRow("Reason", status.ReasonEN).
		AddRow("Next", statusNextForSurface(status, commandSurface)).
		AddRow("Mode", modeText).
		AddRow("Provider", agent.ProviderName).
		AddRow("Model", agent.CurrentModel).
		AddRow("Context", contextText)
	_, _ = fmt.Fprint(out, statusTable.RenderCompact())

	printStatusUsageSection(out, agent, "🧾 Last Chat Turn", "No chat turn usage data available", lastChatTurnUsageForStatus)
	printStatusUsageSection(out, agent, "🧾 Last Review", "No review usage data available", lastReviewUsageForStatus)

	if summary, ok := providerHistoryReductionStatusSummary(agent.Runtime); ok {
		_, _ = fmt.Fprintln(out)
		green.Fprintln(out, "🧾 Provider history reduction")
		_, _ = fmt.Fprintf(out, "  %s\n", summary)
		if commandEditSummary, ok := providerHistoryCommandEditDryRunStatusSummary(agent.Runtime); ok {
			_, _ = fmt.Fprintf(out, "  %s\n", commandEditSummary)
		}
	}

	printSessionSections(agent)
	_, _ = fmt.Fprintln(out)
	return true
}

type statusUsageSelector func(*SessionStats) (*api.Usage, *cost.CostEstimate)

func printStatusUsageSection(out io.Writer, agent *Agent, title, emptyText string, selectUsage statusUsageSelector) {
	_, _ = fmt.Fprintln(out)
	green.Fprintln(out, title)
	if agent == nil || agent.Stats == nil {
		dim.Fprintf(out, "  %s\n", emptyText)
		return
	}

	cfg := agent.cfg()
	usage, costOverride := selectUsage(agent.Stats)
	if table := buildLastRequestTable(cfg, agent.activeModelProviderConfigKey(cfg), agent.CurrentModel, usage, costOverride); table != nil {
		_, _ = fmt.Fprint(out, table.RenderCompact())
		return
	}
	dim.Fprintf(out, "  %s\n", emptyText)
}

func planModeStatusText(enabled bool) string {
	if enabled {
		return "Plan: ON"
	}
	return "Plan: OFF"
}

func statusSurfaceSummary(commandSurface commandcatalog.CommandSurface) string {
	switch commandSurface {
	case commandcatalog.CommandSurfaceTUI:
		return "TUI primary"
	default:
		return "classic legacy fallback (--no-tui)"
	}
}

func statusNextForSurface(status AgentStatus, commandSurface commandcatalog.CommandSurface) string {
	if status.State != StateWaitingInput {
		return status.NextEN
	}
	switch commandSurface {
	case commandcatalog.CommandSurfaceTUI:
		return "Type your request or / for command candidates"
	default:
		return "Type your request or /help (classic fallback; new UI commands are TUI-only)"
	}
}
