package tui

import (
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/susugadx/xelyon-cli/internal/review"
)

const reviewRunnerNotImplementedMessage = "review runner is not implemented yet"

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

type reviewScreen struct {
	mode        reviewScreenMode
	presetIndex int
	customInput textinput.Model
	request     *review.ReviewRequest
	message     string
}

func newReviewScreen() *reviewScreen {
	input := textinput.New()
	input.Prompt = ""
	input.Placeholder = "Add review instructions..."
	input.CharLimit = 0
	input.Width = 80

	return &reviewScreen{
		mode:        reviewScreenPreset,
		customInput: input,
	}
}

func (rs *reviewScreen) submitUncommitted(customInstructions string) {
	req := review.NewUncommittedRequest(customInstructions)
	rs.request = &req
	rs.mode = reviewScreenSubmitted
	rs.customInput.Blur()
}

func (rs *reviewScreen) openCustomInput() {
	rs.mode = reviewScreenCustom
	rs.customInput.Focus()
}

func (rs *reviewScreen) backToPreset() {
	rs.mode = reviewScreenPreset
	rs.customInput.Blur()
}

func reviewPresetLabels() []string {
	return []string{
		"Review uncommitted changes",
		"Custom review instructions",
	}
}
