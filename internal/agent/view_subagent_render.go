package agent

import (
	"fmt"
	"io"

	"github.com/susugadx/xelyon-cli/internal/cost"
	"github.com/susugadx/xelyon-cli/internal/tools/subagent"
	"github.com/susugadx/xelyon-cli/internal/ui"
)

func renderSubAgentStats(out io.Writer, summary subagent.SubAgentSummary) {
	if summary.TotalSpawned == 0 {
		return
	}

	_, _ = fmt.Fprintln(out)
	green.Fprintln(out, "🤖 Sub-agents")

	table := ui.NewTable().SetHeaders("ID", "Model", "Status", "Input", "Cached", "Output", "Thinking", "Cost", "Tools", "Error")
	for _, agentStats := range summary.Agents {
		pending := agentStats.Status == "running"
		model := agentStats.Model
		if model == "" {
			model = "-"
		}
		table.AddRow(
			agentStats.ID,
			model,
			agentStats.Status,
			formatSubAgentNumber(agentStats.InputTokens, pending),
			formatSubAgentNumber(agentStats.CachedTokens, pending),
			formatSubAgentNumber(agentStats.OutputTokens, pending),
			formatSubAgentNumber(agentStats.ThinkingTokens, pending),
			formatSubAgentCost(cost.CostEstimate{
				Cost:               agentStats.Cost,
				PricingUnavailable: agentStats.PricingUnavailable,
			}, pending),
			formatSubAgentNumber(agentStats.ToolExecutions, pending),
			formatSubAgentError(agentStats.Status, agentStats.ErrorMessage),
		)
	}

	table.AddRow(
		"Total",
		formatNumber(summary.TotalSpawned),
		"",
		formatNumber(summary.TotalInput),
		formatNumber(summary.TotalCached),
		formatNumber(summary.TotalOutput),
		formatNumber(summary.TotalThinking),
		formatSubAgentCost(cost.CostEstimate{
			Cost:               summary.TotalCost,
			PricingUnavailable: summary.PricingUnavailable,
		}, false),
		formatNumber(summary.TotalTools),
		"",
	)
	_, _ = fmt.Fprint(out, table.RenderCompact())

	hasBreakdown := false
	for _, agentStats := range summary.Agents {
		if len(agentStats.ToolBreakdown) > 0 {
			hasBreakdown = true
			break
		}
	}
	if hasBreakdown {
		_, _ = fmt.Fprintln(out)
		green.Fprintln(out, "🔧 Sub-agent Tool Breakdown")
		for _, agentStats := range summary.Agents {
			if len(agentStats.ToolBreakdown) == 0 {
				continue
			}
			_, _ = fmt.Fprintf(out, "  %s (%s):\n", agentStats.ID, agentStats.Model)
			bdTable := ui.NewTable().SetHeaders("Tool", "✓", "✗", "Total")
			for _, entry := range agentStats.ToolBreakdown {
				total := entry.Success + entry.Failures
				failStr := fmt.Sprintf("%d", entry.Failures)
				if entry.Failures > 0 {
					failStr = red.Sprintf("%d", entry.Failures)
				}
				bdTable.AddRow(
					entry.Tool,
					fmt.Sprintf("%d", entry.Success),
					failStr,
					fmt.Sprintf("%d", total),
				)
			}
			_, _ = fmt.Fprint(out, bdTable.RenderCompact())
		}
	}
}
