package agent

import (
	"fmt"
	"io"
	"strconv"

	"github.com/susugadx/xelyon-cli/internal/agent/viewfmt"
	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/tools/subagent"
	"github.com/susugadx/xelyon-cli/internal/ui"
)

func renderLastRequestTable(provider, model string, usage *api.Usage, costOverride *float64) *ui.Table {
	if usage == nil {
		return nil
	}

	table := ui.NewTable().
		AddRow("Input", formatNumber(usage.InputTokens)+" tokens").
		AddRow("Cache Mode", requestCacheMode(*usage))

	if usage.CachedInputTokens > 0 || usage.CacheCreationTokens > 0 {
		table.AddRow("Cached", formatNumber(usage.CachedInputTokens)+" tokens").
			AddRow("Cache Creation", formatNumber(usage.CacheCreationTokens)+" tokens").
			AddRow("Hit Rate", fmt.Sprintf("%.1f%%", requestCacheHitRate(*usage)))
	}

	table.AddRow("Output", formatNumber(usage.OutputTokens)+" tokens")
	if usage.ThinkingTokens > 0 {
		table.AddRow("Thinking", formatNumber(usage.ThinkingTokens)+" tokens")
	}

	cost := requestUsageCost(provider, model, *usage)
	if costOverride != nil {
		cost = *costOverride
	}
	if cost > 0 {
		table.AddRow("Cost", viewfmt.USDWithSuffix(cost))
	} else {
		table.AddRow("Cost", "Free (local)")
	}

	return table
}

func renderSessionOverviewTable(agent *Agent, stats *SessionStats) *ui.Table {
	sessionPath, sessionSize := getSessionFileInfo(agent)
	table := ui.NewTable().
		AddRow("Elapsed", stats.FormatElapsedTime()).
		AddRow("User Messages", strconv.Itoa(stats.UserMessages)).
		AddRow("Assistant Messages", strconv.Itoa(stats.AssistantMessages)).
		AddRow("Total Messages", strconv.Itoa(stats.TotalMessages())).
		AddRow("Tool Executions", strconv.Itoa(stats.TotalToolExecutions()))

	if sessionPath != "" {
		table.AddRow("Session File", sessionPath)
		if sessionSize > 0 {
			table.AddRow("Session Size", FormatFileSize(sessionSize))
		}
	}
	return table
}

func renderSessionTokenTable(agent *Agent, stats *SessionStats, subSummary *subagent.SubAgentSummary) *ui.Table {
	hasSubAgents := subSummary != nil && subSummary.TotalSpawned > 0
	if stats.TotalTokens() <= 0 && !hasSubAgents {
		return nil
	}

	tokenTable := ui.NewTable()
	currentTokens := agent.EstimateTokens()
	limit := agent.currentModelTokenLimit(agent.cfg())
	if limit > 0 {
		contextPct := float64(currentTokens) / float64(limit) * 100
		tokenTable.AddRow(tokenRowLabel(hasSubAgents, "Parent", "Context"), fmt.Sprintf("%s / %s (%.1f%%)", formatNumber(currentTokens), formatNumber(limit), contextPct))
	}

	cost := stats.EstimatedCostForConfig(agent.cfg())
	if stats.TotalTokens() > 0 {
		tokenTable.AddRow(tokenRowLabel(hasSubAgents, "Parent", "Input"), formatNumber(stats.InputTokens)+" tokens")

		if stats.CachedInputTokens > 0 || stats.CacheCreationTokens > 0 {
			tokenTable.AddRow(tokenRowLabel(hasSubAgents, "Parent", "Cached"), formatNumber(stats.CachedInputTokens)+" tokens").
				AddRow(tokenRowLabel(hasSubAgents, "Parent", "Cache Creation"), formatNumber(stats.CacheCreationTokens)+" tokens").
				AddRow(tokenRowLabel(hasSubAgents, "Parent", "Hit Rate"), fmt.Sprintf("%.1f%%", sessionCacheHitRate(stats)))
		}

		tokenTable.AddRow(tokenRowLabel(hasSubAgents, "Parent", "Output"), formatNumber(stats.OutputTokens)+" tokens")
		if stats.ThinkingTokens > 0 {
			tokenTable.AddRow(tokenRowLabel(hasSubAgents, "Parent", "Thinking"), formatNumber(stats.ThinkingTokens)+" tokens")
		}

		tokenTable.AddRow(tokenRowLabel(hasSubAgents, "Parent", "Total"), formatNumber(stats.TotalTokens())+" tokens")
	}

	if hasSubAgents {
		tokenTable.AddRow("Sub-agent Input", formatNumber(subSummary.TotalInput)+" tokens")
		if subSummary.TotalCached > 0 {
			tokenTable.AddRow("Sub-agent Cached", formatNumber(subSummary.TotalCached)+" tokens")
			if subSummary.TotalInput > 0 {
				tokenTable.AddRow("Sub-agent Hit Rate", fmt.Sprintf("%.1f%%", subAgentCacheHitRate(*subSummary)))
			}
		}
		tokenTable.AddRow("Sub-agent Output", formatNumber(subSummary.TotalOutput)+" tokens")
		if subSummary.TotalThinking > 0 {
			tokenTable.AddRow("Sub-agent Thinking", formatNumber(subSummary.TotalThinking)+" tokens")
		}
		tokenTable.AddRow("Sub-agent Total", formatNumber(subAgentTotalTokens(*subSummary))+" tokens")
	}

	if hasSubAgents {
		tokenTable.AddRow("Parent Cost", formatParentCost(agent.ProviderName, cost)).
			AddRow("Sub-agent Cost", formatUSDWithSuffix(subSummary.TotalCost)).
			AddRow("Total Cost", formatUSDWithSuffix(cost+subSummary.TotalCost))
	} else if cost > 0 {
		tokenTable.AddRow("Cost", formatUSDWithSuffix(cost))
	} else {
		tokenTable.AddRow("Cost", "Free (local)")
	}
	return tokenTable
}

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
			formatSubAgentCost(agentStats.Cost, pending),
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
		formatUSD(summary.TotalCost),
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

func renderToolObservabilitySection(out io.Writer, stats *SessionStats) {
	obs := stats.ToolObs

	_, _ = fmt.Fprintln(out)
	green.Fprintln(out, "📈 Tool Selection")
	selTable := ui.NewTable()
	selTable.AddRow("read_file(batch)", strconv.Itoa(obs.ReadFileBatchCalls))
	selTable.AddRow("search_code(multi)", strconv.Itoa(obs.SearchCodeMultiPatternCalls))
	selTable.AddRow("search_code(missed multi)", strconv.Itoa(obs.SearchCodeMissedMultiPattern))
	_, _ = fmt.Fprint(out, selTable.RenderCompact())

	_, _ = fmt.Fprintln(out)
	green.Fprintln(out, "📍 Exploration")
	explorationTable := ui.NewTable()
	explorationTable.AddRow("search_code(impact)", strconv.Itoa(obs.SearchCodeImpactCalls))
	explorationTable.AddRow("search_code(explicit multi)", strconv.Itoa(obs.SearchCodeExplicitMultiCalls))
	explorationTable.AddRow("read_file(targets)", strconv.Itoa(obs.ReadFileTargetCalls))
	explorationTable.AddRow("search_code(batch merges)", strconv.Itoa(obs.SearchCodeBatchMerges))
	explorationTable.AddRow("read_file(batch merges)", strconv.Itoa(obs.ReadFileBatchMerges))
	_, _ = fmt.Fprint(out, explorationTable.RenderCompact())

	if obs.ApplyPatchAttempts > 0 || obs.ApplyPatchRepairAttempts > 0 {
		_, _ = fmt.Fprintln(out)
		green.Fprintln(out, "🩹 Patch Reliability")
		patchTable := ui.NewTable()
		patchTable.AddRow("apply_patch(success)", fmt.Sprintf("%d/%d", obs.ApplyPatchSuccesses, obs.ApplyPatchAttempts))
		patchTable.AddRow("apply_patch(repair)", fmt.Sprintf("%d/%d", obs.ApplyPatchRepairSuccesses, obs.ApplyPatchRepairAttempts))
		_, _ = fmt.Fprint(out, patchTable.RenderCompact())
	}
}

func renderSavingsSection(out io.Writer, stats *SessionStats) {
	sav := stats.Savings
	if !sav.hasAny() {
		return
	}

	_, _ = fmt.Fprintln(out)
	green.Fprintln(out, "💰 Estimated Savings (API input)")
	savTable := ui.NewTable()
	if sav.SavedCalls > 0 {
		savTable.AddRow("Executions skipped", strconv.Itoa(sav.SavedCalls))
	}
	if sav.EstimatedInputTokensSaved > 0 {
		savTable.AddRow("~Input tokens saved", fmt.Sprintf("~%s", formatNumber(sav.EstimatedInputTokensSaved)))
	}
	if sav.EstimatedCostSaved > 0 {
		savTable.AddRow("~Cost saved", fmt.Sprintf("~%s", viewfmt.USDWithSuffix(sav.EstimatedCostSaved)))
	}
	_, _ = fmt.Fprint(out, savTable.RenderCompact())
	dim.Fprintln(out, "  (~ = estimated, dedup result diff + compaction)")
}
