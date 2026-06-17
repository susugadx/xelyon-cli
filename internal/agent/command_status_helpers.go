package agent

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/agent/viewfmt"
	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/commandcatalog"
	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/cost"
	"github.com/susugadx/xelyon-cli/internal/providerdiag"
	"github.com/susugadx/xelyon-cli/internal/tools/subagent"
	"github.com/susugadx/xelyon-cli/internal/ui"
)

func requestCacheMode(usage api.Usage) string {
	switch {
	case usage.CachedInputTokens > 0 && usage.CacheCreationTokens > 0:
		return "read + create"
	case usage.CachedInputTokens > 0:
		return "read"
	case usage.CacheCreationTokens > 0:
		return "create"
	default:
		return "none"
	}
}

func requestCacheHitRate(usage api.Usage) float64 {
	if usage.InputTokens <= 0 {
		return 0
	}
	return float64(usage.CachedInputTokens) / float64(usage.InputTokens) * 100.0
}

func requestUsageCost(cfg *config.Config, provider, model string, usage api.Usage) cost.CostEstimate {
	estimate := cost.EstimateRequestCostWithCacheForConfig(cfg, provider, model, usage)
	estimate.Cost += usage.StorageCost
	return estimate
}

func buildLastRequestTable(cfg *config.Config, provider, model string, usage *api.Usage, costOverride *cost.CostEstimate) *ui.Table {
	return renderLastRequestTable(cfg, provider, model, usage, costOverride)
}

func geminiServiceTierStatusDetail(cfg *config.Config, provider string, usage *api.Usage) string {
	if config.ActiveProviderConfigKey(provider) != "gemini" {
		return ""
	}
	return providerdiag.NewGeminiServiceTierSnapshot(cfg, usage).Detail()
}

func lastChatTurnUsageForStatus(stats *SessionStats) (*api.Usage, *cost.CostEstimate) {
	if stats == nil || stats.LastTurnUsage == nil {
		return nil, nil
	}
	return stats.LastTurnUsage, &cost.CostEstimate{
		Cost:               stats.LastTurnCost,
		PricingUnavailable: stats.LastTurnCostUnknown,
	}
}

func lastReviewUsageForStatus(stats *SessionStats) (*api.Usage, *cost.CostEstimate) {
	if stats == nil || stats.LastReviewUsage == nil {
		return nil, nil
	}
	return stats.LastReviewUsage, &cost.CostEstimate{
		Cost:               stats.LastReviewCost,
		PricingUnavailable: stats.LastReviewCostUnknown,
	}
}

func getSessionFileInfo(agent *Agent) (string, int64) {
	sessionPath := ""
	sessionSize := int64(0)
	if agent.session != nil {
		sessionPath = fmt.Sprintf("~/.xelyon/history/%s.jsonl", agent.session.ID)
		if agent.storage != nil {
			homeDir, err := os.UserHomeDir()
			if err == nil {
				fullPath := fmt.Sprintf("%s/.xelyon/history/%s.jsonl", homeDir, agent.session.ID)
				if size, err := GetSessionFileSize(fullPath); err == nil {
					sessionSize = size
				}
			}
		}
	}
	return sessionPath, sessionSize
}

func sessionCacheHitRate(stats *SessionStats) float64 {
	if stats.InputTokens <= 0 {
		return 0
	}
	return float64(stats.CachedInputTokens) / float64(stats.InputTokens) * 100.0
}

func buildSessionTokenTable(agent *Agent, stats *SessionStats, subSummary *subagent.SubAgentSummary) *ui.Table {
	return renderSessionTokenTable(agent, stats, subSummary)
}

func tokenRowLabel(hasSubAgents bool, scope, label string) string {
	if !hasSubAgents {
		return label
	}
	return scope + " " + label
}

func subAgentCacheHitRate(summary subagent.SubAgentSummary) float64 {
	if summary.TotalInput <= 0 {
		return 0
	}
	return float64(summary.TotalCached) / float64(summary.TotalInput) * 100.0
}

func subAgentTotalTokens(summary subagent.SubAgentSummary) int {
	return summary.TotalInput + summary.TotalOutput + summary.TotalThinking
}

func printSessionSections(agent *Agent) {
	renderSessionSections(agent)
}

func formatCostEstimate(estimate cost.CostEstimate) string {
	if estimate.PricingUnavailable {
		return "N/A (pricing unavailable)"
	}
	return viewfmt.USDWithSuffix(estimate.Cost)
}

func formatCompactCostEstimate(estimate cost.CostEstimate) string {
	if estimate.PricingUnavailable {
		return "cost N/A"
	}
	return fmt.Sprintf("~$%.3f", estimate.Cost)
}

func shouldSuppressLocalCostDisplay(providerName string, estimate cost.CostEstimate) bool {
	return strings.EqualFold(providerName, "ollama") && estimate.Cost == 0 && !estimate.PricingUnavailable
}

func formatParentCost(providerName string, estimate cost.CostEstimate) string {
	if estimate.PricingUnavailable {
		return "N/A (pricing unavailable)"
	}
	if shouldSuppressLocalCostDisplay(providerName, estimate) {
		return "Free (local)"
	}
	return viewfmt.USDWithSuffix(estimate.Cost)
}

func formatSubAgentNumber(value int, pending bool) string {
	if pending {
		return "-"
	}
	return formatNumber(value)
}

func formatSubAgentCost(estimate cost.CostEstimate, pending bool) string {
	if pending {
		return "-"
	}
	if estimate.PricingUnavailable {
		return "N/A"
	}
	return viewfmt.USD(estimate.Cost)
}

func formatSubAgentError(status, message string) string {
	if status == "running" {
		return ""
	}
	if status != "error" {
		return ""
	}
	message = viewfmt.FirstLine(strings.TrimSpace(message))
	if message == "" {
		return "unknown error"
	}
	return viewfmt.Truncate(message, 120)
}

func printSubAgentStats(out io.Writer, summary subagent.SubAgentSummary) {
	renderSubAgentStats(out, summary)
}

// printToolObservabilitySection はツール選択のobservabilityセクションを表示する。
func printToolObservabilitySection(out io.Writer, stats *SessionStats) {
	renderToolObservabilitySection(out, stats)
}

func handleStatsCommandForSurface(agent *Agent, commandSurface commandcatalog.CommandSurface) bool {
	return handleStatusCommandForSurface(agent, commandSurface)
}
