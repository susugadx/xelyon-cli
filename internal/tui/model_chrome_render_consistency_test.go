package tui

import (
	"strings"
	"testing"
)

func TestModel_RebuildChrome_ConsistentWithHelpers(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := newModelWithViewport(agent)
	m.textInput.SetValue("test")
	m.rebuildChrome()

	want := m.renderInputDock() + "\n" + m.renderStatusBar()
	if m.chromeCache != want {
		t.Fatalf("chromeCache should equal renderInputDock + \\n + renderStatusBar")
	}
}

func TestModel_RebuildChrome_TotalLineCount(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := newModelWithViewport(agent)
	m.rebuildChrome()

	lines := strings.Split(m.chromeCache, "\n")
	if len(lines) != m.footerHeight() {
		t.Fatalf("chromeCache line count = %d, want footerHeight() = %d", len(lines), m.footerHeight())
	}
}
