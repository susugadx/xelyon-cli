package agent

import (
	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/config"
)

type normalModeNoToolHandler struct {
	runner *TurnRunner
	cfg    *config.Config
	state  *normalModeState
}

func newNormalModeNoToolHandler(r *TurnRunner, cfg *config.Config, state *normalModeState) *normalModeNoToolHandler {
	return &normalModeNoToolHandler{
		runner: r,
		cfg:    cfg,
		state:  state,
	}
}

func (h *normalModeNoToolHandler) Handle(response string) normalModeAction {
	if handled, action := h.handleTextPlanRecovery(response); handled {
		return action
	}
	recordedChanges := h.captureRecordedTaskChanges(response)
	if handled, action := h.handleCompletionFinalChecks(response, recordedChanges); handled {
		return action
	}

	h.runner.agent.handleNormalResponse(response)
	return normalModeDone
}

func (h *normalModeNoToolHandler) captureRecordedTaskChanges(response string) recordedTaskChangeSnapshot {
	if !isCompletionTriggerResponse(response) {
		h.state.recordedTaskChanges = recordedTaskChangeSnapshot{}
		return recordedTaskChangeSnapshot{}
	}

	snapshot := h.runner.agent.recordedTaskChangeSnapshot()
	h.state.recordedTaskChanges = snapshot
	return snapshot
}

func (h *normalModeNoToolHandler) handleCompletionFinalChecks(response string, changes recordedTaskChangeSnapshot) (bool, normalModeAction) {
	a := h.runner.agent
	if !isCompletionTriggerResponse(response) || len(h.cfg.FinalChecks.Commands) == 0 {
		return false, normalModeContinue
	}
	finalCheckTargets := h.finalCheckTargetSnapshot(changes)
	if len(finalCheckTargets.files) == 0 {
		return false, normalModeContinue
	}

	result := a.runFinalCheckCommands(finalCheckTargets.files)
	if !result.needsContinue {
		h.state.finalCheckRetry.reset()
		return false, normalModeContinue
	}

	h.runner.appendAssistantHistoryOnly(response)
	if h.state.finalCheckRetry.recordFailure(result, finalCheckTargets.progressFingerprint) {
		yellow.Fprintln(a.output(), "⚠️  Final checks failed again without any task progress. Returning control to user.")
		a.History = append(a.History, api.Message{
			Role:    "user",
			Content: result.feedback,
		})
		return true, normalModeBreak
	}

	yellow.Fprintln(a.output(), "⚠️  Final check command failed. Asking AI to fix...")
	a.History = append(a.History, api.Message{
		Role:    "user",
		Content: result.feedback,
	})
	return true, normalModeContinue
}

func (h *normalModeNoToolHandler) finalCheckTargetSnapshot(changes recordedTaskChangeSnapshot) recordedTaskChangeSnapshot {
	a := h.runner.agent
	mergedFiles := append([]string(nil), changes.files...)

	if a != nil && a.activeApprovedPlan != "" && a.pendingApprovedPlanHasChanges() {
		planFiles := a.pendingApprovedPlanChangedFiles()
		if len(planFiles) > 0 {
			seen := make(map[string]bool, len(mergedFiles)+len(planFiles))
			for _, file := range mergedFiles {
				seen[file] = true
			}
			for _, file := range planFiles {
				mergedFiles = appendRecordedChangedFile(mergedFiles, seen, file)
			}
		}
	}

	progressFingerprint := fingerprintFinalCheckTargetFiles(mergedFiles)
	if progressFingerprint == "" {
		progressFingerprint = changes.progressFingerprint
	}

	return recordedTaskChangeSnapshot{
		files:               mergedFiles,
		progressFingerprint: progressFingerprint,
	}
}
