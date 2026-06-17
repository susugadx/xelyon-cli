package subagent

import (
	"fmt"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/config"
)

func applySubAgentReasoningEffort(cfg *config.Config, taskType, reasoningEffort string) error {
	effort := strings.TrimSpace(reasoningEffort)
	if effort == "" {
		effort = DefaultEffortForTaskType(taskType)
	}
	if effort == "" {
		effort = strings.TrimSpace(cfg.SubAgent.DefaultEffort)
	}
	return applyReasoningEffort(cfg, effort)
}

func applyReasoningEffort(cfg *config.Config, effort string) error {
	switch normalizeReasoningEffort(effort) {
	case "", "off":
		cfg.Thinking.Enabled = false
		return nil
	case "low", "medium", "high", "xhigh":
		cfg.Thinking.Enabled = true
		cfg.Thinking.Level = normalizeReasoningEffort(effort)
		return nil
	default:
		return fmt.Errorf("invalid reasoning_effort: %s", effort)
	}
}

func normalizeReasoningEffort(effort string) string {
	return strings.ToLower(strings.TrimSpace(effort))
}
