package agent

import "github.com/susugadx/xelyon-cli/internal/config"

func newCompletionTestAgent(cfg *config.Config) *Agent {
	return &Agent{Runtime: NewAgentRuntimeWithConfig(cfg)}
}
