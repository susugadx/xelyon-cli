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
	return h.state != nil && h.state.turnMutations.hasMutations()
}

func (h *normalModeNoToolHandler) handlePostMutationFinalChecks(response string) (bool, normalModeAction) {
	a := h.runner.agent
	if len(h.cfg.FinalChecks.Commands) == 0 {
		return false, normalModeContinue
	}

	turnMutations := h.state.turnMutations.snapshot()
	finalCheckTargets := h.finalCheckTargetSnapshot(turnMutations)

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

func (h *normalModeNoToolHandler) finalCheckTargetSnapshot(changes turnMutationSnapshot) turnMutationSnapshot {
	files := append([]string(nil), changes.files...)
	progressFingerprint := fingerprintFinalCheckTargetFiles(files)
	if progressFingerprint == "" {
		// 対象ファイルを特定できない mutation でも retry 進捗は追跡したいので、
		// turn-local FileChange イベント由来の fingerprint にフォールバックする。
		progressFingerprint = changes.progressFingerprint
	}

	return turnMutationSnapshot{
		mutationCount:       changes.mutationCount,
		files:               files,
		progressFingerprint: progressFingerprint,
	}
}
