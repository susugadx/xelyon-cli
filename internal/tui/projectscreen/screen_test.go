package projectscreen

import (
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/config"
)

func TestProjectListWindowStartKeepsSelectionVisible(t *testing.T) {
	if got := projectListWindowStart(10, 12, 4); got != 7 {
		t.Fatalf("projectListWindowStart = %d, want 7", got)
	}
}

func TestScreenHandleSaveResultQueuesNewerSnapshot(t *testing.T) {
	screen := New(&config.ProjectConfig{Context: "old"}, 7)

	pending, ok := screen.BeginSave(false)
	if !ok {
		t.Fatal("BeginSave returned ok=false")
	}
	screen.pc.Context = "newer"
	screen.saveQueued = true

	action := screen.HandleSaveResult(SaveResult{
		Snapshot: pending.Snapshot,
		ScreenID: pending.ScreenID,
		SaveSeq:  pending.SaveSeq,
	})
	if !action.StartQueued {
		t.Fatal("HandleSaveResult should request queued save")
	}
	if !screen.dirty {
		t.Fatal("screen should stay dirty after stale save snapshot")
	}
}

func TestScreenViewSanitizesListItemText(t *testing.T) {
	screen := New(&config.ProjectConfig{
		Context: "ctx",
		Rules:   []string{"first line\nsecond line\tthird line"},
	}, 1)
	screen.sectionIndex = int(projectSectionRules)
	screen.activePane = projectPaneItem

	view := stripANSIForTest(screen.View(80, 10))
	if !strings.Contains(view, "first line second line third line") {
		t.Fatalf("View did not sanitize multiline list item:\n%s", view)
	}
}

func stripANSIForTest(s string) string {
	var b strings.Builder
	inEsc := false
	for i := 0; i < len(s); i++ {
		c := s[i]
		if inEsc {
			if (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') {
				inEsc = false
			}
			continue
		}
		if c == 0x1b {
			inEsc = true
			continue
		}
		b.WriteByte(c)
	}
	return b.String()
}
