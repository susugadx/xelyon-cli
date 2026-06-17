package agent

import "github.com/susugadx/xelyon-cli/internal/config"

func providerHistoryRawOutputActiveContextBudget(runtime *AgentRuntime) int {
	defaults := config.DefaultProviderHistoryRawOutputArtifactsConfig()
	if runtime == nil {
		return defaults.ActiveContextBudgetTokens
	}
	cfg := runtime.Options.ProviderHistoryRawOutputArtifacts
	budget := cfg.ActiveContextBudgetTokens
	if budget <= 0 {
		budget = defaults.ActiveContextBudgetTokens
	}
	maxBudget := cfg.ActiveContextBudgetMaxTokens
	if maxBudget <= 0 {
		maxBudget = defaults.ActiveContextBudgetMaxTokens
	}
	if budget > maxBudget {
		return maxBudget
	}
	return budget
}
