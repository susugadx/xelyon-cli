package probe

import (
	"fmt"

	"github.com/susugadx/xelyon-cli/internal/review/domain"
)

const probeExecMissingResolvedPathError = "resolved command path is required"

func buildProbeCommandTransition(cmdStatus domain.ReviewProbeStatus, command string, args []string) (nextStatus domain.ReviewProbeStatus, message string, stop bool) {
	formatted := formatProbeCommand(command, args)

	switch cmdStatus {
	case domain.ReviewProbeBlocked:
		return domain.ReviewProbeBlocked, fmt.Sprintf("probe command blocked: %s", formatted), true
	case domain.ReviewProbeTimedOut:
		return domain.ReviewProbeTimedOut, fmt.Sprintf("probe command timed out: %s", formatted), true
	case domain.ReviewProbeFailed:
		return domain.ReviewProbeFailed, fmt.Sprintf("probe command failed: %s", formatted), true
	default:
		return "", "", false
	}
}
