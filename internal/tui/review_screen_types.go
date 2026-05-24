package tui

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/susugadx/xelyon-cli/internal/tui/termtext"
)

const reviewRunnerNotImplementedMessage = "review runner is not implemented yet"
const reviewRunnerCancelledMessage = "review canceled"

type reviewScreenMode int

const (
	reviewScreenPreset reviewScreenMode = iota
	reviewScreenCustom
)

type reviewCommand int

const (
	reviewCommandNone reviewCommand = iota
	reviewCommandClose
	reviewCommandSubmit
	reviewCommandDelegateCtrlC
)

type reviewPresetAction int

const (
	reviewPresetActionCurrentChanges reviewPresetAction = iota
	reviewPresetActionCustomInstructions
)

type reviewPreset struct {
	label  string
	action reviewPresetAction
}

var reviewPresets = []reviewPreset{
	{label: "Review current changes", action: reviewPresetActionCurrentChanges},
	{label: "Review current changes with custom focus", action: reviewPresetActionCustomInstructions},
}

type reviewScreen struct {
	mode         reviewScreenMode
	presetIndex  int
	bodyViewport reviewBodyViewport
	customInput  textinput.Model
	notice       string
}

func newReviewScreen() *reviewScreen {
	input := textinput.New()
	input.Prompt = ""
	input.Placeholder = "Add custom focus..."
	input.CharLimit = 0
	input.Width = 80

	return &reviewScreen{
		mode:        reviewScreenPreset,
		customInput: input,
	}
}

func (rs *reviewScreen) openCustomInput() {
	rs.mode = reviewScreenCustom
	rs.clearNotice()
	rs.bodyViewport.reset()
	rs.customInput.Focus()
}

func (rs *reviewScreen) backToPreset() {
	rs.mode = reviewScreenPreset
	rs.clearNotice()
	rs.bodyViewport.reset()
	rs.customInput.Blur()
}

func (rs *reviewScreen) setNotice(text string) {
	rs.notice = termtext.SanitizeSingleLineANSI(strings.TrimSpace(text))
	rs.bodyViewport.reset()
}

func (rs *reviewScreen) clearNotice() {
	rs.notice = ""
}

func (rs *reviewScreen) selectedPreset() (reviewPreset, bool) {
	if rs.presetIndex < 0 || rs.presetIndex >= len(reviewPresets) {
		return reviewPreset{}, false
	}
	return reviewPresets[rs.presetIndex], true
}
