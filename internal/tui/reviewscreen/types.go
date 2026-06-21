package reviewscreen

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/susugadx/xelyon-cli/internal/tui/termtext"
)

// Mode は review screen の表示モードを表す。
type Mode int

const (
	// ModePreset は preset 選択画面を表す。
	ModePreset Mode = iota
	// ModeCustom は custom focus 入力画面を表す。
	ModeCustom
)

// Command は review screen が root Model に要求する操作を表す。
type Command int

const (
	// CommandNone は root 側の操作が不要な入力処理を表す。
	CommandNone Command = iota
	// CommandClose は review screen を閉じる要求を表す。
	CommandClose
	// CommandSubmit は選択済み review request の実行要求を表す。
	CommandSubmit
	// CommandDelegateCtrlC は Ctrl+C を chat 側 cancellation に委譲する要求を表す。
	CommandDelegateCtrlC
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

// Screen は /review preset/custom 画面の state/input/render を保持する。
type Screen struct {
	mode         Mode
	presetIndex  int
	bodyViewport reviewBodyViewport
	customInput  textinput.Model
	notice       string
}

// Snapshot は root tests が review screen の公開状態を確認するための読み取り専用状態。
type Snapshot struct {
	Mode               Mode
	CustomInputFocused bool
}

// New は review screen を構築する。
func New(width int) *Screen {
	input := textinput.New()
	input.Prompt = ""
	input.Placeholder = "Add custom focus..."
	input.CharLimit = 0
	input.Width = inputWidth(width)

	return &Screen{
		mode:        ModePreset,
		customInput: input,
	}
}

// Resize は review screen の入力幅を現在の terminal 幅へ同期する。
func (rs *Screen) Resize(width int) {
	rs.customInput.Width = inputWidth(width)
}

// Snapshot は review screen の公開状態を返す。
func (rs *Screen) Snapshot() Snapshot {
	if rs == nil {
		return Snapshot{}
	}
	return Snapshot{
		Mode:               rs.mode,
		CustomInputFocused: rs.customInput.Focused(),
	}
}

// SetNotice は画面上部に表示する通知を設定する。
func (rs *Screen) SetNotice(text string) {
	rs.notice = termtext.SanitizeSingleLineANSI(strings.TrimSpace(text))
	rs.bodyViewport.reset()
	if rs.mode == ModeCustom {
		rs.customInput.Focus()
	}
}

// ClearNotice は通知表示を消す。
func (rs *Screen) ClearNotice() {
	rs.clearNotice()
}

func (rs *Screen) openCustomInput() {
	rs.mode = ModeCustom
	rs.clearNotice()
	rs.bodyViewport.reset()
	rs.customInput.Focus()
}

func (rs *Screen) backToPreset() {
	rs.mode = ModePreset
	rs.clearNotice()
	rs.bodyViewport.reset()
	rs.customInput.Blur()
}

func (rs *Screen) clearNotice() {
	rs.notice = ""
}

func (rs *Screen) selectedPreset() (reviewPreset, bool) {
	if rs.presetIndex < 0 || rs.presetIndex >= len(reviewPresets) {
		return reviewPreset{}, false
	}
	return reviewPresets[rs.presetIndex], true
}

func inputWidth(width int) int {
	return max(0, width-4)
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
