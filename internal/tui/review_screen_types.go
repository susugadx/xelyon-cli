package tui

import (
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/susugadx/xelyon-cli/internal/review"
)

const reviewRunnerNotImplementedMessage = "review runner is not implemented yet"
const reviewRunnerRunningMessage = "review is running..."
const reviewRunnerCancellingMessage = "review cancellation requested..."
const reviewRunnerBusyMessage = "another request is still running; wait for it to finish before starting review"

type reviewScreenMode int

const (
	reviewScreenPreset reviewScreenMode = iota
	reviewScreenCustom
	reviewScreenSubmitted
)

type reviewCommand int

const (
	reviewCommandNone reviewCommand = iota
	reviewCommandClose
	reviewCommandSubmit
	reviewCommandDelegateCtrlC
)

type reviewRunState int

const (
	reviewRunIdle reviewRunState = iota
	reviewRunRunning
	reviewRunSucceeded
	reviewRunFailed
)

type reviewScreen struct {
	mode         reviewScreenMode
	screenID     int
	runSeq       int
	runState     reviewRunState
	presetIndex  int
	bodyViewport reviewBodyViewport
	activeRun    *reviewRunContext
	customInput  textinput.Model
	request      *review.ReviewRequest
	report       *review.ReviewReport
	message      string
	errMessage   string
}

func newReviewScreen(screenID int) *reviewScreen {
	input := textinput.New()
	input.Prompt = ""
	input.Placeholder = "Add review instructions..."
	input.CharLimit = 0
	input.Width = 80

	return &reviewScreen{
		screenID:    screenID,
		mode:        reviewScreenPreset,
		runState:    reviewRunIdle,
		customInput: input,
	}
}

func (rs *reviewScreen) submitCurrentChanges(customInstructions string) {
	req := review.NewCurrentChangesRequest(customInstructions)
	rs.request = &req
	rs.mode = reviewScreenSubmitted
	rs.bodyViewport.reset()
	rs.customInput.Blur()
}

func (rs *reviewScreen) openCustomInput() {
	rs.mode = reviewScreenCustom
	rs.bodyViewport.reset()
	rs.customInput.Focus()
}

func (rs *reviewScreen) backToPreset() {
	rs.mode = reviewScreenPreset
	rs.bodyViewport.reset()
	rs.customInput.Blur()
}

func (rs *reviewScreen) startReview(req review.ReviewRequest) *reviewRunContext {
	rs.runSeq++
	runCtx := rs.startActiveReviewRun()
	reqCopy := req
	rs.request = &reqCopy
	rs.report = nil
	rs.errMessage = ""
	rs.message = reviewRunnerRunningMessage
	rs.runState = reviewRunRunning
	rs.mode = reviewScreenSubmitted
	rs.bodyViewport.reset()
	rs.customInput.Blur()
	return runCtx
}

func (rs *reviewScreen) markReviewNotImplemented(req review.ReviewRequest) {
	rs.markReviewBlocked(req, reviewRunnerNotImplementedMessage)
}

func (rs *reviewScreen) markReviewBlocked(req review.ReviewRequest, message string) {
	rs.cancelActiveReviewRun()
	reqCopy := req
	rs.request = &reqCopy
	rs.report = nil
	rs.errMessage = message
	rs.message = message
	rs.runState = reviewRunFailed
	rs.mode = reviewScreenSubmitted
	rs.bodyViewport.reset()
	rs.customInput.Blur()
}

func (rs *reviewScreen) completeReview(report review.ReviewReport) {
	rs.clearActiveReviewRun()
	reportCopy := report
	rs.report = &reportCopy
	rs.errMessage = ""
	rs.message = "review complete"
	rs.runState = reviewRunSucceeded
	rs.bodyViewport.reset()
}

func (rs *reviewScreen) failReview(err error) {
	rs.clearActiveReviewRun()
	rs.report = nil
	if err == nil {
		rs.errMessage = "review failed"
	} else {
		rs.errMessage = err.Error()
	}
	rs.message = "review failed"
	rs.runState = reviewRunFailed
	rs.bodyViewport.reset()
}

func (rs *reviewScreen) cancelRunningReview() {
	if rs.runState != reviewRunRunning {
		return
	}
	rs.cancelActiveReviewRun()
	rs.message = reviewRunnerCancellingMessage
	rs.bodyViewport.reset()
}

func reviewPresetLabels() []string {
	return []string{
		"Review current changes",
		"Custom review instructions",
	}
}
