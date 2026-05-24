package review

import (
	"fmt"
	"strings"
	"time"
)

const maxReviewProgressDetailRunes = 140

type reviewProgressItem struct {
	id           string
	phase        ReviewProgressPhase
	runningLabel string
	okLabel      string
}

var (
	reviewProgressEvidenceItem = reviewProgressItem{
		id:           "evidence",
		phase:        ReviewProgressPhaseEvidence,
		runningLabel: "collecting current changes",
		okLabel:      "evidence collected",
	}
	reviewProgressProbePlanItem = reviewProgressItem{
		id:           "probe_plan",
		phase:        ReviewProgressPhaseProbePlan,
		runningLabel: "planning probes",
		okLabel:      "planned probes",
	}
	reviewProgressReportItem = reviewProgressItem{
		id:           "report",
		phase:        ReviewProgressPhaseReport,
		runningLabel: "writing report",
		okLabel:      "report drafted",
	}
	reviewProgressSaturationCheckItem = reviewProgressItem{
		id:           "saturation_check",
		phase:        ReviewProgressPhaseSaturationCheck,
		runningLabel: "checking review coverage",
		okLabel:      "coverage checked",
	}
	reviewProgressSaturationRepairItem = reviewProgressItem{
		id:           "saturation_check",
		phase:        ReviewProgressPhaseSaturationCheck,
		runningLabel: "repairing coverage check",
		okLabel:      "coverage checked",
	}
	reviewProgressReportRevisionItem = reviewProgressItem{
		id:           "report_revision",
		phase:        ReviewProgressPhaseReportRevision,
		runningLabel: "revising report",
		okLabel:      "report revised",
	}
	reviewProgressReportRevisionRepairItem = reviewProgressItem{
		id:           "report_revision",
		phase:        ReviewProgressPhaseReportRevision,
		runningLabel: "repairing report revision",
		okLabel:      "report revised",
	}
)

func (item reviewProgressItem) event(status ReviewProgressStatus, label string, detail string) ReviewProgressEvent {
	return ReviewProgressEvent{
		ID:     item.id,
		Phase:  item.phase,
		Status: status,
		Label:  label,
		Detail: detail,
	}
}

func (r *ReviewRunner) emitProgressRunning(item reviewProgressItem) {
	r.emitProgress(item.event(ReviewProgressRunning, item.runningLabel, ""))
}

func (r *ReviewRunner) emitProgressOK(item reviewProgressItem, detail string) {
	r.emitProgress(item.event(ReviewProgressOK, item.okLabel, detail))
}

func (r *ReviewRunner) emitProgressError(item reviewProgressItem, err error) {
	if err == nil {
		return
	}
	r.emitProgress(item.event(ReviewProgressError, item.runningLabel, truncateReviewProgressDetail(err.Error())))
}

func (r *ReviewRunner) emitProgress(event ReviewProgressEvent) {
	if r == nil {
		return
	}
	if event.ID == "" {
		event.ID = string(event.Phase)
	}
	r.progressSink.emit(event)
}

func reviewEvidenceProgressDetail(bundle ReviewEvidenceBundle) string {
	staged, unstaged := reviewChangedFileStageCounts(bundle.ChangedFiles)
	untracked := len(bundle.Inventory.Untracked)
	if untracked == 0 {
		untracked = len(bundle.UntrackedFiles)
	}
	contextFiles := countReviewProgressContextFiles(bundle.ChangedFileContext) + countReviewProgressContextFiles(bundle.RelatedContextFiles)
	parts := []string{
		fmt.Sprintf("staged %d", staged),
		fmt.Sprintf("unstaged %d", unstaged),
		fmt.Sprintf("untracked %d", untracked),
		fmt.Sprintf("context %d", contextFiles),
	}
	if len(bundle.Diffs) > 0 {
		parts = append(parts, fmt.Sprintf("diffs %d", len(bundle.Diffs)))
	}
	return strings.Join(parts, " · ")
}

func reviewChangedFileStageCounts(files []ReviewChangedFile) (int, int) {
	staged := 0
	unstaged := 0
	for _, file := range files {
		if file.Staged {
			staged++
		}
		if file.Unstaged {
			unstaged++
		}
	}
	return staged, unstaged
}

func countReviewProgressContextFiles(files []ReviewContextFileEvidence) int {
	count := 0
	for _, file := range files {
		if !file.Skipped {
			count++
		}
	}
	return count
}

func reviewProgressProbeCountDetail(count int) string {
	return fmt.Sprintf("%d checks", count)
}

func reviewProgressProbeID(probeID string, commandIndex int) string {
	if probeID == "" {
		probeID = "unknown"
	}
	if commandIndex < 0 {
		return "probe:" + probeID
	}
	return fmt.Sprintf("probe:%s:%d", probeID, commandIndex)
}

type reviewProgressProbeScope struct {
	probeID   string
	startedID string
}

func reviewProgressProbeScopeForRequest(req ReviewProbeRequest) reviewProgressProbeScope {
	commandIndex := 0
	if len(req.Commands) == 0 {
		commandIndex = -1
	}
	return reviewProgressProbeScope{
		probeID:   req.ID,
		startedID: reviewProgressProbeID(req.ID, commandIndex),
	}
}

func (scope reviewProgressProbeScope) eventID(commandIndex int) string {
	if commandIndex < 0 && scope.startedID != "" {
		return scope.startedID
	}
	return reviewProgressProbeID(scope.probeID, commandIndex)
}

func reviewProgressProbeLabel(mode ReviewProbeMode) string {
	modeText := strings.TrimSpace(string(mode))
	if modeText == "" {
		return "probe"
	}
	return "probe " + modeText
}

func reviewProgressProbeDetail(req ReviewProbeRequest) string {
	if len(req.Commands) == 0 {
		if strings.TrimSpace(req.Purpose) != "" {
			return truncateReviewProgressDetail(req.Purpose)
		}
		return truncateReviewProgressDetail(req.ID)
	}
	return truncateReviewProgressDetail(formatProbeCommand(req.Commands[0].Command, req.Commands[0].Args))
}

func reviewProgressProbeCommandDetail(result ReviewProbeCommandResult) string {
	detail := formatProbeCommand(result.Command, result.Args)
	if strings.TrimSpace(detail) == "" && strings.TrimSpace(result.Error) != "" {
		detail = result.Error
	}
	return truncateReviewProgressDetail(detail)
}

func reviewProgressProbeResultDetail(result ReviewProbeResult) string {
	if strings.TrimSpace(result.Error) != "" {
		return truncateReviewProgressDetail(result.Error)
	}
	if result.ID != "" {
		return truncateReviewProgressDetail(result.ID)
	}
	return ""
}

func reviewProgressStatusForProbeStatus(status ReviewProbeStatus) ReviewProgressStatus {
	switch status {
	case ReviewProbePassed:
		return ReviewProgressOK
	default:
		return ReviewProgressError
	}
}

func reviewProgressDuration(start time.Time) time.Duration {
	if start.IsZero() {
		return 0
	}
	return time.Since(start)
}

func truncateReviewProgressDetail(text string) string {
	text = strings.TrimSpace(strings.ReplaceAll(text, "\n", " "))
	if text == "" {
		return ""
	}
	runes := []rune(text)
	if len(runes) <= maxReviewProgressDetailRunes {
		return text
	}
	return string(runes[:maxReviewProgressDetailRunes-1]) + "…"
}
