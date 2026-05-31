package agent

import (
	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/finalcheck"
	"github.com/susugadx/xelyon-cli/internal/turnsupport"
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
	// completion wording は使わず、変更が発生したターンの最初の no-tool 応答で
	// final checks を機械的に実行する。
	// 早めに走るケースはあるが、open-world な wording 判定を避けて単純化を優先する。
	if h.hasTaskMutationsThisTurn() {
		if handled, action := h.handlePostMutationFinalChecks(response); handled {
			return action
		}
		h.runner.agent.handleNormalResponse(response)
		return normalModeDone
	}

	if handled, action := h.handleTextPlanRecovery(response); handled {
		return action
	}

	h.runner.agent.handleNormalResponse(response)
	return normalModeDone
}

func (h *normalModeNoToolHandler) hasTaskMutationsThisTurn() bool {
	return h.state != nil && h.state.turnMutations.HasMutations()
}

func (h *normalModeNoToolHandler) handlePostMutationFinalChecks(response string) (bool, normalModeAction) {
	a := h.runner.agent
	if len(h.cfg.FinalChecks.Commands) == 0 {
		return false, normalModeContinue
	}

	turnMutations := h.state.turnMutations.Snapshot()
	finalCheckTargets := h.finalCheckTargetSnapshot(turnMutations)

	result := a.runFinalCheckCommands(finalCheckTargets.Files)
	if !result.NeedsContinue {
		h.state.finalCheckRetry.Reset()
		return false, normalModeContinue
	}

	h.runner.appendAssistantHistoryOnly(response)
	if h.state.finalCheckRetry.RecordFailure(result, finalCheckTargets.ProgressFingerprint) {
		yellow.Fprintln(a.output(), "⚠️  Final checks failed again without any task progress. Returning control to user.")
		a.History = append(a.History, api.Message{
			Role:    "user",
			Content: result.Feedback,
		})
		return true, normalModeBreak
	}

	yellow.Fprintln(a.output(), "⚠️  Final check command failed. Asking AI to fix...")
	a.History = append(a.History, api.Message{
		Role:    "user",
		Content: result.Feedback,
	})
	return true, normalModeContinue
}

func (h *normalModeNoToolHandler) finalCheckTargetSnapshot(changes turnsupport.MutationSnapshot) finalcheck.TargetSnapshot {
	return finalcheck.BuildTargetSnapshot(finalcheck.TargetInput{
		Files:               changes.Files,
		ProgressFingerprint: changes.ProgressFingerprint,
	})
}
