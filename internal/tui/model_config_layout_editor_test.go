package tui

import (
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/tui/configscreen"
)

func TestConfigScreen_NarrowWidth_StringEdit_RemainsVisible(t *testing.T) {
	m := newConfigTestModel()
	m.width = 72
	m.height = 20

	selectConfigField(t, &m, "provider", "default_model")

	_, _, rightW := configscreen.PaneWidths(m.width)
	if rightW != 0 {
		t.Fatalf("rightW = %d, want 0 for narrow width", rightW)
	}

	m = sendConfigKey(m, "enter")
	snapshot := configSnapshot(t, m)
	if snapshot.editMode != editInput {
		t.Fatalf("editMode = %d, want editInput", snapshot.editMode)
	}
	if snapshot.activePane != paneField {
		t.Fatalf("activePane = %d, want paneField when detail pane is hidden", snapshot.activePane)
	}

	view := stripANSI(m.View())
	if !strings.Contains(view, "Edit:") {
		t.Fatal("narrow width view should render the input editor")
	}
	if !strings.Contains(view, "deepseek-v4-flash") {
		t.Fatal("narrow width view should include the current input value")
	}
}

func TestConfigScreen_NarrowWidth_SelectEdit_RemainsVisible(t *testing.T) {
	m := newConfigTestModel()
	m.width = 72
	m.height = 20

	selectConfigField(t, &m, "execution", "execution.mode")

	_, _, rightW := configscreen.PaneWidths(m.width)
	if rightW != 0 {
		t.Fatalf("rightW = %d, want 0 for narrow width", rightW)
	}

	m = sendConfigKey(m, "enter")
	snapshot := configSnapshot(t, m)
	if snapshot.editMode != editSelect {
		t.Fatalf("editMode = %d, want editSelect", snapshot.editMode)
	}
	if snapshot.activePane != paneField {
		t.Fatalf("activePane = %d, want paneField when detail pane is hidden", snapshot.activePane)
	}

	view := stripANSI(m.View())
	if !strings.Contains(view, "Select:") {
		t.Fatal("narrow width view should render the select editor")
	}
	if !strings.Contains(view, "balanced") {
		t.Fatal("narrow width view should render visible select options")
	}
}

func TestConfigScreen_VeryNarrowWidth_ConfigDoesNotEnterInvisiblePane(t *testing.T) {
	m := newConfigTestModel()
	m.width = 30
	m.height = 20

	selectConfigField(t, &m, "lsp", "lsp.servers")

	leftW, midW, rightW := configscreen.PaneWidths(m.width)
	if leftW != 30 || midW != 0 || rightW != 0 {
		t.Fatalf("configscreen.PaneWidths(%d) = (%d, %d, %d), want (30, 0, 0)", m.width, leftW, midW, rightW)
	}

	m = sendConfigKey(m, "enter")
	snapshot := configSnapshot(t, m)
	if snapshot.editMode != editStructMap {
		t.Fatalf("editMode = %d, want editStructMap", snapshot.editMode)
	}
	if snapshot.activePane == paneDetail {
		t.Fatalf("activePane = %d, should not enter invisible detail pane", snapshot.activePane)
	}

	view := stripANSI(m.View())
	if !strings.Contains(view, "Keys:") {
		t.Fatal("very narrow view should render a visible struct map editor")
	}
	if !strings.Contains(view, "go") {
		t.Fatal("very narrow view should render struct map entries")
	}
}

func TestConfigScreen_NormalWidth_BehaviorUnchanged(t *testing.T) {
	m := newConfigTestModel()
	m.width = 120
	m.height = 30

	selectConfigField(t, &m, "provider", "default_model")

	_, _, rightW := configscreen.PaneWidths(m.width)
	if rightW <= 0 {
		t.Fatalf("rightW = %d, want visible detail pane on normal width", rightW)
	}

	m = sendConfigKey(m, "enter")
	snapshot := configSnapshot(t, m)
	if snapshot.editMode != editInput {
		t.Fatalf("editMode = %d, want editInput", snapshot.editMode)
	}
	if snapshot.activePane != paneDetail {
		t.Fatalf("activePane = %d, want paneDetail on normal width", snapshot.activePane)
	}

	view := stripANSI(m.View())
	if !strings.Contains(view, "Edit:") {
		t.Fatal("normal width view should render the input editor")
	}
}
