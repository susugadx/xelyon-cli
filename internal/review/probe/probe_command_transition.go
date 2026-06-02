package probe

import "fmt"

const probeExecMissingResolvedPathError = "resolved command path is required"

func buildProbeCommandTransition(cmdStatus ReviewProbeStatus, command string, args []string) (nextStatus ReviewProbeStatus, message string, stop bool) {
	formatted := formatProbeCommand(command, args)

	switch cmdStatus {
	case ReviewProbeBlocked:
		return ReviewProbeBlocked, fmt.Sprintf("probe command blocked: %s", formatted), true
	case ReviewProbeTimedOut:
		return ReviewProbeTimedOut, fmt.Sprintf("probe command timed out: %s", formatted), true
	case ReviewProbeFailed:
		return ReviewProbeFailed, fmt.Sprintf("probe command failed: %s", formatted), true
	default:
		return "", "", false
	}
}
