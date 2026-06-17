package agent

import (
	"fmt"
	"strconv"

	"github.com/susugadx/xelyon-cli/internal/tools/subagent"
	"github.com/susugadx/xelyon-cli/internal/ui"
)

func renderSessionSections(agent *Agent) {
	out := agent.output()
	if agent.Stats == nil {
		dim.Fprintln(out, "  Statistics not available")
		return
	}

	stats := agent.Stats
	var subSummary *subagent.SubAgentSummary
	if manager := agent.subAgentManager(); manager != nil {
		summary := manager.GetSummary()
		if summary.TotalSpawned > 0 {
			subSummary = &summary
		}
	}

	_, _ = fmt.Fprintln(out)
	green.Fprintln(out, "📚 Session")
	_, _ = fmt.Fprint(out, renderSessionOverviewTable(agent, stats).RenderCompact())

	_, _ = fmt.Fprintln(out)
	green.Fprintln(out, "🔧 Tool Executions")
	if stats.TotalToolExecutions() > 0 {
		toolTable := ui.NewTable()
		for tool, count := range stats.ToolExecutions {
			toolTable.AddRow(tool, strconv.Itoa(count))
		}
		toolTable.AddRow("Total", strconv.Itoa(stats.TotalToolExecutions()))
		_, _ = fmt.Fprint(out, toolTable.RenderCompact())
	} else {
		dim.Fprintln(out, "  No tools executed yet")
	}

	_, _ = fmt.Fprintln(out)
	green.Fprintln(out, "💰 Session Tokens & Cost")
	if tokenTable := renderSessionTokenTable(agent, stats, subSummary); tokenTable != nil {
		_, _ = fmt.Fprint(out, tokenTable.RenderCompact())
	} else {
		dim.Fprintln(out, "  No token usage data available")
	}

	if subSummary != nil {
		renderSubAgentStats(out, *subSummary)
	}

	renderToolObservabilitySection(out, stats)
	renderSavingsSection(out, stats)

	_, _ = fmt.Fprintln(out)
	green.Fprintln(out, "⚡ Optimizations")
	opt := stats.Optimizations
	if opt.hasAny() {
		optTable := ui.NewTable()
		if opt.NegativeCacheHits > 0 {
			optTable.AddRow("Negative cache", fmt.Sprintf("%d hits", opt.NegativeCacheHits))
		}
		if opt.ErrorCompressions > 0 {
			optTable.AddRow("Error compression", fmt.Sprintf("%d times", opt.ErrorCompressions))
		}
		if opt.FailedPairCompressions > 0 {
			optTable.AddRow("Failed-pair compression", fmt.Sprintf("%d times", opt.FailedPairCompressions))
		}
		if opt.OutlineFirstCount > 0 {
			optTable.AddRow("Outline-first mode", fmt.Sprintf("%d times", opt.OutlineFirstCount))
		}
		if opt.CompactionCount > 0 {
			optTable.AddRow("Auto-compress", fmt.Sprintf("%d times", opt.CompactionCount))
		}
		if opt.CostAwareCompressions > 0 {
			optTable.AddRow("Cost-aware auto-compress", fmt.Sprintf("%d times", opt.CostAwareCompressions))
		}
		_, _ = fmt.Fprint(out, optTable.RenderCompact())
	} else {
		dim.Fprintln(out, "  No optimizations triggered yet")
	}
}
