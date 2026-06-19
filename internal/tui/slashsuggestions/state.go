package slashsuggestions

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/susugadx/xelyon-cli/internal/tui/slash"
	"github.com/susugadx/xelyon-cli/internal/tui/termtext"
	"github.com/susugadx/xelyon-cli/internal/tui/theme"
)

const maxRows = 8

// State は slash suggestion の選択状態と表示 window を保持する。
type State struct {
	prefix          string
	suggestions     []slash.Suggestion
	selected        int
	selectionActive bool
}

// Snapshot は slash suggestion state の読み取り専用状態。
type Snapshot struct {
	Prefix          string
	Suggestions     []slash.Suggestion
	Selected        int
	SelectionActive bool
}

// KeyCommand は slash suggestion key handling が root に要求する操作。
type KeyCommand int

const (
	// KeyCommandNone は root 側の操作が不要な入力を表す。
	KeyCommandNone KeyCommand = iota
	// KeyCommandClear は suggestion を閉じる要求を表す。
	KeyCommandClear
	// KeyCommandCompleteWithSpace は選択 suggestion を引数 space 付きで補完する要求。
	KeyCommandCompleteWithSpace
	// KeyCommandExpand は選択 suggestion を展開する要求。
	KeyCommandExpand
	// KeyCommandComplete は選択 suggestion を補完する要求。
	KeyCommandComplete
	// KeyCommandSubmit は選択 suggestion を submit 用 text にして実行する要求。
	KeyCommandSubmit
)

// KeyResult は slash suggestion key handling の結果を表す。
type KeyResult struct {
	Command    KeyCommand
	Suggestion slash.Suggestion
	Handled    bool
}

// RenderRow は slash suggestion の表示行データ。
type RenderRow struct {
	Category     string
	CommandLabel string
	Description  string
	Selected     bool
}

// Snapshot は state の公開状態を返す。
func (s State) Snapshot() Snapshot {
	return Snapshot{
		Prefix:          s.prefix,
		Suggestions:     append([]slash.Suggestion(nil), s.suggestions...),
		Selected:        s.selected,
		SelectionActive: s.selectionActive,
	}
}

// Visible は suggestion が表示可能かを返す。
func (s State) Visible() bool {
	return len(s.suggestions) > 0
}

// SelectedSuggestion は現在選択中の suggestion を返す。
func (s State) SelectedSuggestion() (slash.Suggestion, bool) {
	if !s.Visible() || s.selected < 0 || s.selected >= len(s.suggestions) {
		return slash.Suggestion{}, false
	}
	return s.suggestions[s.selected], true
}

// Refresh は prefix と候補から state を更新する。
func (s State) Refresh(prefix string, suggestions []slash.Suggestion) State {
	if len(suggestions) == 0 {
		return State{}
	}
	selected := s.selected
	selectionActive := s.selectionActive
	if prefix != s.prefix {
		selected = 0
		selectionActive = false
	}
	if selected < 0 {
		selected = 0
	}
	if selected >= len(suggestions) {
		selected = len(suggestions) - 1
	}
	return State{
		prefix:          prefix,
		suggestions:     append([]slash.Suggestion(nil), suggestions...),
		selected:        selected,
		selectionActive: selectionActive,
	}
}

// Clear は state を空にする。
func (s State) Clear() State {
	return State{}
}

// ChromeRowCount は現在表示される suggestion chrome 行数を返す。
func (s State) ChromeRowCount(availableRows int) int {
	return len(s.VisibleRows(availableRows)) + s.detailRowCount(availableRows)
}

// VisibleRows は footer budget 内で表示する suggestion を返す。
func (s State) VisibleRows(availableRows int) []slash.Suggestion {
	if !s.Visible() {
		return nil
	}
	limit := s.maxVisibleRows(availableRows)
	if limit <= 0 {
		return nil
	}
	start := s.windowStart(availableRows)
	end := start + limit
	if end > len(s.suggestions) {
		end = len(s.suggestions)
	}
	return append([]slash.Suggestion(nil), s.suggestions[start:end]...)
}

// VisibleRenderRows は footer budget 内で表示する render row を返す。
func (s State) VisibleRenderRows(availableRows int) []RenderRow {
	rows := s.VisibleRows(availableRows)
	if len(rows) == 0 {
		return nil
	}
	start := s.windowStart(availableRows)
	out := make([]RenderRow, 0, len(rows))
	for i, suggestion := range rows {
		out = append(out, NewRenderRow(suggestion, start+i == s.selected))
	}
	return out
}

// SelectedDetail は選択中 suggestion の detail text を返す。
func (s State) SelectedDetail() string {
	suggestion, ok := s.SelectedSuggestion()
	if !ok {
		return ""
	}
	return detailText(suggestion)
}

func (s State) detailRowCount(availableRows int) int {
	if s.SelectedDetail() == "" || availableRows <= 1 || len(s.VisibleRows(availableRows)) == 0 {
		return 0
	}
	return 1
}

func (s State) maxVisibleRows(availableRows int) int {
	if s.SelectedDetail() != "" && availableRows > 1 {
		availableRows--
	}
	if availableRows <= 0 {
		return 0
	}
	if availableRows > maxRows {
		return maxRows
	}
	return availableRows
}

func (s State) windowStart(availableRows int) int {
	if !s.Visible() {
		return 0
	}
	limit := s.maxVisibleRows(availableRows)
	if limit <= 0 || s.selected < limit {
		return 0
	}
	return s.selected - limit + 1
}

// HandleKey は suggestion 表示中の key decision を返す。
func (s *State) HandleKey(msg tea.KeyMsg, availableRows int) KeyResult {
	if len(s.VisibleRows(availableRows)) == 0 {
		return KeyResult{}
	}

	switch {
	case msg.Type == tea.KeyEsc:
		*s = s.Clear()
		return KeyResult{Command: KeyCommandClear, Handled: true}
	case msg.Type == tea.KeyUp || msg.Type == tea.KeyShiftTab || msg.String() == "ctrl+p":
		s.move(-1)
		return KeyResult{Handled: true}
	case msg.Type == tea.KeyDown || msg.String() == "ctrl+n":
		s.move(1)
		return KeyResult{Handled: true}
	case msg.Type == tea.KeyTab:
		if suggestion, ok := s.SelectedSuggestion(); ok {
			return KeyResult{Command: KeyCommandCompleteWithSpace, Suggestion: suggestion, Handled: true}
		}
		return KeyResult{Handled: true}
	case isEnterKey(msg):
		if suggestion, ok := s.SelectedSuggestion(); ok {
			if suggestion.ExpandOnEnter && (s.selectionActive || !suggestion.SubmitOnEnter) {
				return KeyResult{Command: KeyCommandExpand, Suggestion: suggestion, Handled: true}
			}
			if suggestion.CompleteOnEnter {
				if !suggestion.SubmitOnEnter && !s.selectionActive {
					s.activateSelection()
					return KeyResult{Handled: true}
				}
				return KeyResult{Command: KeyCommandComplete, Suggestion: suggestion, Handled: true}
			}
			if !suggestion.SubmitOnEnter && !s.selectionActive {
				s.activateSelection()
				return KeyResult{Handled: true}
			}
			return KeyResult{Command: KeyCommandSubmit, Suggestion: suggestion, Handled: true}
		}
	}
	return KeyResult{}
}

func (s *State) move(delta int) {
	if !s.Visible() {
		return
	}
	count := len(s.suggestions)
	s.selected = (s.selected + delta + count) % count
	s.selectionActive = true
}

func (s *State) activateSelection() {
	if !s.Visible() {
		return
	}
	s.selectionActive = true
}

// ActivateSelection は選択を明示的に active にする。
func (s *State) ActivateSelection() {
	s.activateSelection()
}

func detailText(suggestion slash.Suggestion) string {
	detail := strings.TrimSpace(suggestion.Detail)
	if detail == "" {
		detail = strings.TrimSpace(suggestion.Description)
	}
	argHint := strings.TrimSpace(suggestion.ArgHint)
	if argHint != "" && detail != "" {
		return argHint + " · " + detail
	}
	if argHint != "" {
		return argHint
	}
	return detail
}

// NewRenderRow は suggestion から表示行 DTO を構築する。
func NewRenderRow(suggestion slash.Suggestion, selected bool) RenderRow {
	return RenderRow{
		Category:     suggestion.CategoryDisplayLabel(),
		CommandLabel: suggestion.Label,
		Description:  suggestion.Description,
		Selected:     selected,
	}
}

// RenderRowString は slash suggestion の1行を描画する。
func RenderRowString(row RenderRow, width int) string {
	chrome := theme.Chrome
	bg := chrome.SuggestionBg
	prefix := "  "
	prefixFg := chrome.SuggestionPrefixFg
	commandFg := chrome.SuggestionCommandFg
	descriptionFg := chrome.SuggestionDescFg
	if row.Selected {
		bg = chrome.SuggestionSelectedBg
		prefix = "› "
		prefixFg = chrome.SuggestionSelectedFg
		commandFg = chrome.SuggestionSelectedFg
		descriptionFg = chrome.SuggestionSelectedDimFg
	}

	layout := rowLayoutForWidth(width)
	label := paddedPlainText(row.CommandLabel, layout.commandWidth)
	description := termtext.TruncateWithANSI(termtext.SanitizeSingleLineANSI(row.Description), layout.descriptionWidth)
	line := bg + prefixFg + prefix
	line += commandFg + label + chrome.Reset + bg
	if layout.descriptionWidth > 0 {
		line += prefixFg + "  " + chrome.Reset + bg + descriptionFg + description + chrome.Reset + bg
	}
	return termtext.FillANSITextWidth(line+chrome.Reset, width, bg)
}

type rowLayout struct {
	commandWidth     int
	descriptionWidth int
}

func rowLayoutForWidth(width int) rowLayout {
	commandWidth := commandWidth(width)
	separatorWidth := 4
	return rowLayout{
		commandWidth:     commandWidth,
		descriptionWidth: max(0, width-commandWidth-separatorWidth),
	}
}

func commandWidth(width int) int {
	if width <= 24 {
		return max(8, width-4)
	}
	if width <= 44 {
		return min(24, max(14, width/2))
	}
	return min(28, max(14, width/3))
}

func paddedPlainText(text string, width int) string {
	if width <= 0 {
		return ""
	}
	text = termtext.TruncateWithANSI(termtext.SanitizeSingleLineANSI(text), width)
	padding := width - lipgloss.Width(text)
	if padding > 0 {
		text += strings.Repeat(" ", padding)
	}
	return text
}

func isEnterKey(msg tea.KeyMsg) bool {
	return msg.Type == tea.KeyEnter || msg.String() == "enter"
}
