package termtext

import "testing"

func TestLayoutCursorMapping_RoundTripsWrappedLine(t *testing.T) {
	layout := BuildLayout([]string{"abcdef"}, 4)

	if len(layout.Rows) != 2 {
		t.Fatalf("rows = %d, want 2", len(layout.Rows))
	}
	if got := layout.GetRawColumnForVisualRow(1); got != 4 {
		t.Fatalf("GetRawColumnForVisualRow(1) = %d, want 4", got)
	}
	row, col := layout.GetVisualCursor(0, 5)
	if row != 1 || col != 1 {
		t.Fatalf("GetVisualCursor(0, 5) = (%d, %d), want (1, 1)", row, col)
	}
}

func TestLayoutCursorMapping_AccountsForContinuationPrefix(t *testing.T) {
	layout := BuildLayout([]string{" abcdef"}, 4)

	if len(layout.Rows) != 2 {
		t.Fatalf("rows = %d, want 2", len(layout.Rows))
	}
	if got := layout.Rows[1].PrefixWidth; got != 1 {
		t.Fatalf("rows[1].PrefixWidth = %d, want 1", got)
	}
	if got := layout.GetRawColumnForVisualRow(1); got != 4 {
		t.Fatalf("GetRawColumnForVisualRow(1) = %d, want 4", got)
	}
	row, col := layout.GetVisualCursor(0, 4)
	if row != 1 || col != 1 {
		t.Fatalf("GetVisualCursor(0, 4) = (%d, %d), want (1, 1)", row, col)
	}
}
