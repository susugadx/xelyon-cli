package subagent

import (
	"fmt"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/config"
)

var supportedReasoningEffortValues = []string{"off", "low", "medium", "high", "xhigh"}

func reasoningEffortSchemaEnum() []string {
	return append([]string(nil), supportedReasoningEffortValues...)
}

func isSupportedReasoningEffort(effort string) bool {
	effort = normalizeReasoningEffort(effort)
	for _, supported := range supportedReasoningEffortValues {
		if effort == supported {
			return true
		}
	}
	return false
}

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
	normalized := normalizeReasoningEffort(effort)
	if normalized == "" || normalized == "off" {
		cfg.Thinking.Enabled = false
		return nil
	}
	if !isSupportedReasoningEffort(normalized) {
		return fmt.Errorf("invalid reasoning_effort: %s", effort)
	}
	cfg.Thinking.Enabled = true
	cfg.Thinking.Level = normalized
	return nil
}

func normalizeReasoningEffort(effort string) string {
	return strings.ToLower(strings.TrimSpace(effort))
}
