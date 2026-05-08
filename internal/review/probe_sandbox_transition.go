package review

import "fmt"

func newIsolatedProbeResult(req ReviewProbeRequest) ReviewProbeResult {
	return ReviewProbeResult{
		ID:     req.ID,
		Mode:   req.Mode,
		Status: ReviewProbePassed,
	}
}

func blockIsolatedProbeResult(result *ReviewProbeResult, message string) {
	result.Status = ReviewProbeBlocked
	result.Error = message
}

func appendIsolatedCleanupError(result *ReviewProbeResult, cleanupTargetName, cleanupTargetPath string, err error) {
	result.Error = appendError(result.Error, fmt.Sprintf("failed to remove %s %q: %v", cleanupTargetName, cleanupTargetPath, err))
}

func applyIsolatedProbeSnapshotError(result *ReviewProbeResult, phase string, err error) {
	result.Status = ReviewProbeBlocked
	result.Error = appendError(result.Error, fmt.Sprintf("failed to capture worktree snapshot %s probe: %v", phase, err))
}

func applyIsolatedProbeCommandTransition(result *ReviewProbeResult, command string, args []string, cmdResult ReviewProbeCommandResult) (stop bool) {
	result.CommandResults = append(result.CommandResults, cmdResult)
	result.OutputTruncated = result.OutputTruncated || cmdResult.OutputTruncated

	nextStatus, message, stop := buildProbeCommandTransition(cmdResult.Status, command, args)
	if !stop {
		return false
	}
	result.Status = nextStatus
	result.Error = appendError(result.Error, message)
	return true
}

func applyIsolatedProbeMutationTransition(result *ReviewProbeResult, mutatedFiles []string) {
	if len(mutatedFiles) == 0 {
		return
	}

	result.MutatedWorktree = true
	result.MutatedFiles = mutatedFiles
	result.Status = ReviewProbeMutatedWorktree
	result.Error = appendError(result.Error, "probe command changed the working tree")
}
