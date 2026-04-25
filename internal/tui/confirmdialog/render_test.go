package confirmdialog

import (
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/tui/theme"
)

func TestRender(t *testing.T) {
	view := Render(30, 12, "Unsaved changes", []string{"Save", "Discard", "Cancel"}, 1, theme.Config)
	if !strings.Contains(view, "Unsaved changes") {
		t.Fatalf("Render() should contain title, got %q", view)
	}
	if !strings.Contains(view, "(*) Discard") {
		t.Fatalf("Render() should mark selected option, got %q", view)
	}
	if got := strings.Count(view, "\n") + 1; got != 12 {
		t.Fatalf("line count = %d, want 12", got)
	}
}
