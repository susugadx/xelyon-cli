package tui

import (
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/susugadx/xelyon-cli/internal/tui/slash"
	"github.com/susugadx/xelyon-cli/internal/tui/theme"
)

func TestModel_RenderInputDock_VisualRowsKeepOrderAndWidth(t *testing.T) {
	m := newModelWithViewport(&stubAgent{statusLine: "ready"})
	m.width = 44
	m.height = 14
	m.textInput.Width = max(0, m.width-inputPromptWidth-1)
	m.padLineCache = fillANSITextWidth("", m.width, theme.Chrome.InputBg)
	m.textInput.SetValue("ask")
	m.appendAttachment(composerAttachment{
		Kind: composerAttachmentFile,
		Path: filepath.Join(string(filepath.Separator), "tmp", "notes.txt"),
		Size: 12,
	})
	m.composer.AppendText("draft summary")
	m.composer.AppendPasteBlock("line1\nline2")
	m.slashSuggestions = slashSuggestionState{
		suggestions: []slash.Suggestion{{
			Label:       "/review",
			Description: "Review current changes and find issues",
		}},
		selected: 0,
	}

	lines := strings.Split(m.renderInputDock(), "\n")
	if len(lines) != 7 {
		t.Fatalf("renderInputDock lines = %d, want 7; lines=%#v", len(lines), lines)
	}
	for i, line := range lines {
		if got := lipgloss.Width(line); got != m.width {
			t.Fatalf("line %d width = %d, want %d; line=%q", i, got, m.width, line)
		}
	}

	plain := make([]string, len(lines))
	for i, line := range lines {
		plain[i] = stripANSI(line)
	}
	checks := []struct {
		line int
		want string
	}{
		{0, "/review"},
		{1, "Review current changes and find issues"},
		{2, inputDockMetaPrefix + "[file notes.txt #1] [paste 11c/2l #1]"},
		{3, inputDockDraftPrefix + "draft summary"},
		{5, inputPrompt + "ask"},
	}
	for _, check := range checks {
		if !strings.Contains(plain[check.line], check.want) {
			t.Fatalf("line %d should contain %q, got %q", check.line, check.want, plain[check.line])
		}
	}
	if got := strings.TrimRight(plain[4], " "); got != "" {
		t.Fatalf("top input padding should stay blank, got %q", plain[4])
	}
	if got := strings.TrimRight(plain[6], " "); got != "" {
		t.Fatalf("bottom input padding should stay blank, got %q", plain[6])
	}
	if !strings.Contains(lines[2], theme.Chrome.InputPasteID+"#1") {
		t.Fatalf("chip row should highlight trailing number, got %q", lines[2])
	}
	if !strings.Contains(lines[3], theme.Chrome.InputRowMarkerFg+inputDockDraftPrefix) {
		t.Fatalf("draft row should use draft prefix styling, got %q", lines[3])
	}
}

func TestModel_InputDockRowGroupsKeepSourcesInOrder(t *testing.T) {
	m := newModelWithViewport(&stubAgent{statusLine: "ready"})
	m.width = 44
	m.textInput.Width = max(0, m.width-inputPromptWidth-1)
	m.padLineCache = fillANSITextWidth("", m.width, theme.Chrome.InputBg)
	m.textInput.SetValue("ask")
	m.appendAttachment(composerAttachment{
		Kind: composerAttachmentFile,
		Path: filepath.Join(string(filepath.Separator), "tmp", "notes.txt"),
		Size: 12,
	})
	m.composer.AppendText("draft summary")
	m.slashSuggestions = slashSuggestionState{
		suggestions: []slash.Suggestion{{
			Label:       "/review",
			Description: "Review current changes and find issues",
		}},
		selected: 0,
	}

	groups := m.inputDockRowGroups()
	wantKinds := []inputDockRowGroupKind{
		inputDockRowGroupSuggestions,
		inputDockRowGroupSelectedDetail,
		inputDockRowGroupCompactChips,
		inputDockRowGroupDrafts,
		inputDockRowGroupTopPadding,
		inputDockRowGroupInput,
		inputDockRowGroupBottomPadding,
	}
	if len(groups) != len(wantKinds) {
		t.Fatalf("row groups = %d, want %d", len(groups), len(wantKinds))
	}
	for i, want := range wantKinds {
		if groups[i].Kind != want {
			t.Fatalf("row group %d kind = %d, want %d", i, groups[i].Kind, want)
		}
	}

	flattened := flattenInputDockRowGroups(groups)
	rendered := m.renderInputDockLines()
	if len(flattened) != len(rendered) {
		t.Fatalf("flattened lines = %d, render lines = %d", len(flattened), len(rendered))
	}
	for i := range rendered {
		if flattened[i] != rendered[i] {
			t.Fatalf("line %d = %q, want %q", i, flattened[i], rendered[i])
		}
	}
	if !strings.Contains(stripANSI(groups[0].Lines[0]), "/review") {
		t.Fatalf("suggestion group should render slash row, got %#v", groups[0].Lines)
	}
	if !strings.Contains(stripANSI(groups[1].Lines[0]), "Review current changes and find issues") {
		t.Fatalf("detail group should render selected detail row, got %#v", groups[1].Lines)
	}
	if !strings.Contains(stripANSI(groups[2].Lines[0]), "[file notes.txt #1]") {
		t.Fatalf("chip group should render attachment chip, got %#v", groups[2].Lines)
	}
	if !strings.Contains(stripANSI(groups[3].Lines[0]), inputDockDraftPrefix+"draft summary") {
		t.Fatalf("draft group should render draft row, got %#v", groups[3].Lines)
	}
}

func TestHighlightSummaryNumber_UsesTrailingNumber(t *testing.T) {
	got := highlightSummaryNumber("[Attached file has#hash.txt] #3", "<num>", "<text>")
	if strings.Contains(got, "has<num>#hash") {
		t.Fatalf("highlightSummaryNumber should not style # inside filename, got %q", got)
	}
	if !strings.Contains(got, "has#hash.txt] <num>#3<text>") {
		t.Fatalf("highlightSummaryNumber should style trailing number, got %q", got)
	}
}

func TestModel_RenderSlashSuggestionRow_SelectionContrastAndNarrowWidth(t *testing.T) {
	m := Model{width: 24}
	suggestion := slash.Suggestion{
		Label:       "/very-long-command",
		Description: "Long description\nwith controls\tand extra words",
	}

	normal := m.renderSlashSuggestionRow(suggestion, false)
	selected := m.renderSlashSuggestionRow(suggestion, true)

	if normal == selected {
		t.Fatal("selected slash suggestion row should differ from normal row")
	}
	for name, line := range map[string]string{"normal": normal, "selected": selected} {
		if got := lipgloss.Width(line); got != m.width {
			t.Fatalf("%s row width = %d, want %d; line=%q", name, got, m.width, line)
		}
		if strings.ContainsAny(stripANSI(line), "\r\n\t") {
			t.Fatalf("%s row should sanitize control chars, got %q", name, stripANSI(line))
		}
	}
	if !strings.Contains(selected, theme.Chrome.SuggestionSelectedBg) {
		t.Fatalf("selected row should use selected background, got %q", selected)
	}
	if !strings.Contains(selected, theme.Chrome.SuggestionSelectedFg) {
		t.Fatalf("selected row should use selected foreground, got %q", selected)
	}
	if strings.Contains(normal, theme.Chrome.SuggestionSelectedBg) {
		t.Fatalf("normal row should not use selected background, got %q", normal)
	}
	if !strings.Contains(stripANSI(selected), "› ") {
		t.Fatalf("selected row should keep selection prefix, got %q", stripANSI(selected))
	}
}

func TestModel_RenderStatusBar_PreservesPrioritySegmentsWithBadges(t *testing.T) {
	t.Setenv("HOME", filepath.Join(string(filepath.Separator), "tmp", "xelyon-test-home"))

	m := newModelWithViewport(&stubAgent{statusLine: "nav status"})
	m.width = 220
	m.navigationMode = true
	m.newOutput = true
	m.transientStatus = "Copied 2 lines"
	m.transientStatusUntil = time.Now().Add(time.Minute)
	m.workingDir = filepath.Join(string(filepath.Separator), "opt", "xelyon", "dev", "xelyon-cli")
	m.vp = lightViewport{
		lines:   []string{"one", "two", "three", "four"},
		yOffset: 0,
		width:   m.width,
		height:  2,
	}

	bar := m.renderStatusBar()
	if got := lipgloss.Width(bar); got != m.width {
		t.Fatalf("status bar width = %d, want %d; bar=%q", got, m.width, bar)
	}
	plain := stripANSI(bar)
	for _, fragment := range []string{"NAV", "nav status", "New output", "Copied 2 lines", "cwd: /opt/xelyon/dev/xelyon-cli", "count+j/k"} {
		if !strings.Contains(plain, fragment) {
			t.Fatalf("status bar missing %q, got %q", fragment, plain)
		}
	}
	for _, style := range []string{theme.Chrome.NavBadge, theme.Chrome.NewOutput, theme.Chrome.SuccessFg, theme.Chrome.StatusPathFg, theme.Chrome.HintFg} {
		if !strings.Contains(bar, style) {
			t.Fatalf("status bar missing style %q, got %q", style, bar)
		}
	}
}
