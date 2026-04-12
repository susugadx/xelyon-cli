package tui

import (
	"testing"

	"github.com/charmbracelet/lipgloss"
)

func TestLayout_TabWrapStaysWithinNarrowViewportWidth(t *testing.T) {
	testCases := []struct {
		width    int
		wantRows int
	}{
		{width: 1, wantRows: 4},
		{width: 2, wantRows: 2},
		{width: 3, wantRows: 2},
	}

	for _, tc := range testCases {
		layout := BuildLayout([]string{"\t"}, tc.width)
		if len(layout.Rows) != tc.wantRows {
			t.Fatalf("width=%d rows=%d, want %d", tc.width, len(layout.Rows), tc.wantRows)
		}
		total := 0
		for i, row := range layout.Rows {
			if row.Width > tc.width {
				t.Fatalf("width=%d row[%d].Width=%d exceeds viewport width", tc.width, i, row.Width)
			}
			plainWidth := lipgloss.Width(stripANSI(row.Content))
			if plainWidth > tc.width {
				t.Fatalf("width=%d row[%d] content width=%d exceeds viewport width", tc.width, i, plainWidth)
			}
			total += row.Width
		}
		if total != visualTabWidth {
			t.Fatalf("width=%d total rendered tab width=%d, want %d", tc.width, total, visualTabWidth)
		}
	}
}

func TestLayout_CombiningCharacterWidth(t *testing.T) {
	line := "e\u0301e\u0301e\u0301"
	layout := BuildLayout([]string{line}, 3)
	if len(layout.Rows) != 1 {
		t.Fatalf("expected 1 row, got %d", len(layout.Rows))
	}
	if layout.Rows[0].Width != 3 {
		t.Fatalf("expected width 3, got %d", layout.Rows[0].Width)
	}

	layout = BuildLayout([]string{line}, 2)
	if len(layout.Rows) != 2 {
		t.Fatalf("expected 2 rows for width=2, got %d", len(layout.Rows))
	}
}
