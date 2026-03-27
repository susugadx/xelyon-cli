package tui

import (
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

type programAction func(*tea.Program)
type programScript []programAction
type programSetup struct {
	script programScript
}

// resize builds a window resize message for program e2e scripts.
func resize(width, height int) tea.Msg {
	return tea.WindowSizeMsg{Width: width, Height: height}
}

// runeKey builds a single-rune key message.
func runeKey(r rune) tea.Msg {
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}}
}

// escKey builds an Escape key message.
func escKey() tea.Msg {
	return tea.KeyMsg{Type: tea.KeyEsc}
}

// enterKey builds an Enter key message.
func enterKey() tea.Msg {
	return tea.KeyMsg{Type: tea.KeyEnter}
}

// pageUpKey builds a PageUp key message.
func pageUpKey() tea.Msg {
	return tea.KeyMsg{Type: tea.KeyPgUp}
}

// pageDownKey builds a PageDown key message.
func pageDownKey() tea.Msg {
	return tea.KeyMsg{Type: tea.KeyPgDown}
}

// wheelUp builds a mouse wheel-up message.
func wheelUp() tea.Msg {
	return tea.MouseMsg{Button: tea.MouseButtonWheelUp, Action: tea.MouseActionPress}
}

// wheelDown builds a mouse wheel-down message.
func wheelDown() tea.Msg {
	return tea.MouseMsg{Button: tea.MouseButtonWheelDown, Action: tea.MouseActionPress}
}

// sendMsg converts a tea message into a program action.
func sendMsg(msg tea.Msg) programAction {
	return func(p *tea.Program) {
		p.Send(msg)
	}
}

// appendAssistant appends an assistant message during a running program script.
func appendAssistant(text string) programAction {
	return sendMsg(AppendMessageMsg{
		Message: ChatMessage{Role: "assistant", Content: text},
	})
}

// appendTool appends a tool result during a running program script.
func appendTool(tool ToolResult) programAction {
	return sendMsg(AppendToolResultMsg{Tool: tool})
}

// script builds a program script from program actions.
func script(actions ...programAction) programScript {
	return append(programScript(nil), actions...)
}

// msgScript builds a program script from tea messages.
func msgScript(msgs ...tea.Msg) programScript {
	s := make(programScript, 0, len(msgs))
	for _, msg := range msgs {
		s = append(s, sendMsg(msg))
	}
	return s
}

// keys expands a compact key sequence into a program script.
// It accepts plain rune sequences such as "wvj" and token forms such as
// "<Esc>", "<Enter>", "<PgUp>", "<PgDown>", "<Tab>", "<Ctrl+C>",
// "<WheelUp>", and "<WheelDown>".
func keys(seq string) programScript {
	var s programScript
	for len(seq) > 0 {
		if seq[0] == '<' {
			end := strings.IndexByte(seq, '>')
			if end > 0 {
				token := seq[1:end]
				if msg, ok := keyTokenMsg(token); ok {
					s = append(s, sendMsg(msg))
					seq = seq[end+1:]
					continue
				}
			}
		}

		r := rune(seq[0])
		seq = seq[1:]
		if r == ' ' || r == '\n' || r == '\t' {
			continue
		}
		s = append(s, sendMsg(runeKey(r)))
	}
	return s
}

// keyTokenMsg resolves a key token into a tea message.
func keyTokenMsg(token string) (tea.Msg, bool) {
	switch strings.ToLower(token) {
	case "esc":
		return escKey(), true
	case "enter", "cr", "return":
		return enterKey(), true
	case "pgup", "pageup":
		return pageUpKey(), true
	case "pgdown", "pagedown":
		return pageDownKey(), true
	case "tab":
		return tea.KeyMsg{Type: tea.KeyTab}, true
	case "ctrl+c", "c-c":
		return tea.KeyMsg{Type: tea.KeyCtrlC}, true
	case "wheelup", "wheel-up":
		return wheelUp(), true
	case "wheeldown", "wheel-down":
		return wheelDown(), true
	default:
		return nil, false
	}
}

// run executes the script against a Bubble Tea program and returns the final model.
func (s programScript) run(t *testing.T, model Model) Model {
	t.Helper()
	return runTUIProgramActions(t, model, s)
}

// append extends the script with additional actions.
func (s programScript) append(other ...programAction) programScript {
	return append(s, other...)
}

// appendScript extends the script with another script.
func (s programScript) appendScript(other programScript) programScript {
	return append(s, other...)
}

// setup starts a setup builder with an initial window size.
func setup(width, height int) *programSetup {
	return &programSetup{script: msgScript(resize(width, height))}
}

// nav enters navigation mode in the setup sequence.
func (b *programSetup) nav() *programSetup {
	b.script = b.script.append(sendMsg(escKey()))
	return b
}

// assistant appends an assistant message in the setup sequence.
func (b *programSetup) assistant(text string) *programSetup {
	b.script = b.script.append(appendAssistant(text))
	return b
}

// assistantLines appends a multi-line assistant message in the setup sequence.
func (b *programSetup) assistantLines(lines ...string) *programSetup {
	return b.assistant(strings.Join(lines, "\n"))
}

// pads appends numbered padding lines in the setup sequence.
func (b *programSetup) pads(n int) *programSetup {
	lines := make([]string, n)
	for i := 0; i < n; i++ {
		lines[i] = "pad-" + strconv.Itoa(i)
	}
	return b.assistantLines(lines...)
}

// tool appends a tool result in the setup sequence.
func (b *programSetup) tool(tool ToolResult) *programSetup {
	b.script = b.script.append(appendTool(tool))
	return b
}

// status appends a status update in the setup sequence.
func (b *programSetup) status(line string) *programSetup {
	b.script = b.script.append(sendMsg(UpdateStatusMsg{Line: line}))
	return b
}

// build finalizes the setup builder into a runnable script.
func (b *programSetup) build() programScript {
	return append(programScript(nil), b.script...)
}

// runTUIProgramActions executes actions through a real Bubble Tea program loop.
func runTUIProgramActions(t *testing.T, model Model, actions []programAction) Model {
	t.Helper()

	p := tea.NewProgram(
		model,
		tea.WithInput(nil),
		tea.WithOutput(io.Discard),
		tea.WithoutRenderer(),
		tea.WithoutSignals(),
	)

	go func() {
		for _, action := range actions {
			action(p)
		}
		p.Quit()
	}()

	finalModel, err := p.Run()
	if err != nil {
		t.Fatalf("program run failed: %v", err)
	}

	got, ok := finalModel.(Model)
	if !ok {
		t.Fatalf("final model type = %T, want tui.Model", finalModel)
	}
	return got
}

// normalizedViewLines strips ANSI sequences and right-side padding for snapshot checks.
func normalizedViewLines(view string) []string {
	lines := strings.Split(stripANSI(view), "\n")
	for i, line := range lines {
		lines[i] = strings.TrimRight(line, " ")
	}
	return lines
}

// assertGoldenText compares text against a golden file and optionally rewrites it.
func assertGoldenText(t *testing.T, name, got string) {
	t.Helper()

	path := filepath.Join("testdata", "program_e2e", name+".golden")
	got = strings.TrimRight(got, "\n")

	if os.Getenv("XELYON_UPDATE_GOLDEN") == "1" {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("failed to create golden dir %s: %v", filepath.Dir(path), err)
		}
		if err := os.WriteFile(path, []byte(got+"\n"), 0o644); err != nil {
			t.Fatalf("failed to update golden file %s: %v", path, err)
		}
		return
	}

	wantBytes, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read golden file %s: %v", path, err)
	}
	want := strings.TrimRight(string(wantBytes), "\n")
	if got != want {
		t.Fatalf("golden mismatch for %s\n%s", name, formatGoldenDiff(want, got))
	}
}

// formatGoldenDiff returns a compact line-oriented diff for golden mismatches.
func formatGoldenDiff(want, got string) string {
	wantLines := strings.Split(want, "\n")
	gotLines := strings.Split(got, "\n")
	maxLines := len(wantLines)
	if len(gotLines) > maxLines {
		maxLines = len(gotLines)
	}

	var b strings.Builder
	b.WriteString("--- want\n+++ got\n")
	for i := 0; i < maxLines; i++ {
		var wantLine, gotLine string
		if i < len(wantLines) {
			wantLine = wantLines[i]
		}
		if i < len(gotLines) {
			gotLine = gotLines[i]
		}
		if wantLine == gotLine {
			continue
		}
		b.WriteString("@@ line ")
		b.WriteString(strconv.Itoa(i + 1))
		b.WriteString(" @@\n")
		b.WriteString("-")
		b.WriteString(wantLine)
		b.WriteByte('\n')
		b.WriteString("+")
		b.WriteString(gotLine)
		b.WriteByte('\n')
	}
	return strings.TrimRight(b.String(), "\n")
}
