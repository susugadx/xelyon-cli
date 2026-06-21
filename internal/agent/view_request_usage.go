package agent

import (
	"fmt"

	"github.com/susugadx/xelyon-cli/internal/agent/viewfmt"
	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/cost"
	"github.com/susugadx/xelyon-cli/internal/termtext"
)

func renderLastRequestTable(cfg *config.Config, provider, model string, usage *api.Usage, costOverride *cost.CostEstimate) *termtext.Table {
	if usage == nil {
		return nil
	}

	table := termtext.NewTable().
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
	addWebSearchUsageRows(table, func(label string) string { return label }, usage.WebSearchCalls, usage.WebSearchResultTokens, usage.StorageCost)
	if detail := geminiServiceTierStatusDetail(cfg, provider, usage); detail != "" {
		table.AddRow("Service Tier", detail)
	}

	estimate := requestUsageCost(cfg, provider, model, *usage)
	if costOverride != nil {
		estimate = *costOverride
	}
	if estimate.PricingUnavailable {
		table.AddRow("Cost", "N/A (pricing unavailable)")
	} else if estimate.Cost > 0 {
		table.AddRow("Cost", viewfmt.USDWithSuffix(estimate.Cost))
	} else {
		table.AddRow("Cost", "Free (local)")
	}

	return table
}

func addWebSearchUsageRows(table *termtext.Table, label func(string) string, calls, resultTokens int, fee float64) {
	if table == nil || calls <= 0 {
		return
	}
	table.AddRow(label("Web Search Calls"), formatNumber(calls))
	if resultTokens > 0 {
		table.AddRow(label("Search Result Tokens"), formatNumber(resultTokens)+" tokens observed")
	}
	if fee > 0 {
		table.AddRow(label("Web Search Fee"), viewfmt.USDWithSuffix(fee))
	}
}
