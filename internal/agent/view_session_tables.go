package agent

import (
	"fmt"
	"strconv"

	"github.com/susugadx/xelyon-cli/internal/agent/viewfmt"
	"github.com/susugadx/xelyon-cli/internal/cost"
	"github.com/susugadx/xelyon-cli/internal/termtext"
	"github.com/susugadx/xelyon-cli/internal/tools/subagent"
)

func renderSessionOverviewTable(agent *Agent, stats *SessionStats) *termtext.Table {
	sessionPath, sessionSize := getSessionFileInfo(agent)
	table := termtext.NewTable().
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

func renderSessionTokenTable(agent *Agent, stats *SessionStats, subSummary *subagent.SubAgentSummary) *termtext.Table {
	hasSubAgents := subSummary != nil && subSummary.TotalSpawned > 0
	if stats.TotalTokens() <= 0 && stats.WebSearchCalls <= 0 && !stats.HasReviewUsage() && !hasSubAgents {
		return nil
	}

	tokenTable := termtext.NewTable()
	currentTokens := agent.EstimateTokens()
	limit := agent.currentModelTokenLimit(agent.cfg())
	if limit > 0 {
		contextPct := float64(currentTokens) / float64(limit) * 100
		tokenTable.AddRow(tokenRowLabel(hasSubAgents, "Parent", "Context"), fmt.Sprintf("%s / %s (%.1f%%)", formatNumber(currentTokens), formatNumber(limit), contextPct))
	}

	parentEstimate := stats.EstimatedCostEstimateForConfig(agent.cfg())
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

	addWebSearchUsageRows(tokenTable, func(label string) string {
		return tokenRowLabel(hasSubAgents, "Parent", label)
	}, stats.WebSearchCalls, stats.WebSearchResultTokens, stats.WebSearchCost)

	addReviewUsageRows(tokenTable, stats)

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
		subEstimate := cost.CostEstimate{
			Cost:               subSummary.TotalCost,
			PricingUnavailable: subSummary.PricingUnavailable,
		}
		totalEstimate := cost.CostEstimate{
			Cost:               parentEstimate.Cost + subSummary.TotalCost,
			PricingUnavailable: parentEstimate.PricingUnavailable || subSummary.PricingUnavailable,
		}
		tokenTable.AddRow("Parent Cost", formatParentCost(agent.ProviderName, parentEstimate)).
			AddRow("Sub-agent Cost", formatCostEstimate(subEstimate)).
			AddRow("Total Cost", formatCostEstimate(totalEstimate))
	} else if parentEstimate.PricingUnavailable {
		tokenTable.AddRow("Cost", "N/A (pricing unavailable)")
	} else if parentEstimate.Cost > 0 {
		tokenTable.AddRow("Cost", viewfmt.USDWithSuffix(parentEstimate.Cost))
	} else {
		tokenTable.AddRow("Cost", "Free (local)")
	}
	addReviewCostRow(tokenTable, stats)
	return tokenTable
}

func addReviewUsageRows(table *termtext.Table, stats *SessionStats) {
	if table == nil || stats == nil || !stats.HasReviewUsage() {
		return
	}

	reviewUsage := stats.ReviewUsage
	if reviewUsage.HasTokenObservation() {
		table.AddRow("Review Input", formatNumber(reviewUsage.InputTokens)+" tokens").
			AddRow("Review Output", formatNumber(reviewUsage.OutputTokens)+" tokens")
		if reviewUsage.ThinkingTokens > 0 {
			table.AddRow("Review Thinking", formatNumber(reviewUsage.ThinkingTokens)+" tokens")
		}
		table.AddRow("Review Total", formatNumber(stats.ReviewTotalTokens())+" tokens")
	}
	addWebSearchUsageRows(table, func(label string) string {
		return "Review " + label
	}, reviewUsage.WebSearchCalls, reviewUsage.WebSearchResultTokens, reviewUsage.StorageCost)
}

func addReviewCostRow(table *termtext.Table, stats *SessionStats) {
	if table == nil || stats == nil || !stats.HasReviewUsage() {
		return
	}
	table.AddRow("Review Cost", formatCostEstimate(stats.ReviewCostEstimate()))
}
