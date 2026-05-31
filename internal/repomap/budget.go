package repomap

import agenttoken "github.com/susugadx/xelyon-cli/internal/token"

const defaultMaxTokens = 4000

func (pm *ProjectMap) fitsBudget(text string) bool {
	maxTokens := pm.MaxTokens
	if maxTokens <= 0 {
		maxTokens = defaultMaxTokens
	}
	return agenttoken.EstimateTokenCount(text) <= maxTokens
}
