package configscreen

import "testing"

func TestPaneWidths(t *testing.T) {
	tests := []struct {
		width int
		left  int
		mid   int
		right int
	}{
		{width: 30, left: 30, mid: 0, right: 0},
		{width: 60, left: 18, mid: 42, right: 0},
		{width: 120, left: 24, mid: 36, right: 60},
	}

	for _, tt := range tests {
		left, mid, right := PaneWidths(tt.width)
		if left != tt.left || mid != tt.mid || right != tt.right {
			t.Fatalf("PaneWidths(%d) = (%d, %d, %d), want (%d, %d, %d)", tt.width, left, mid, right, tt.left, tt.mid, tt.right)
		}
	}
}

func TestLayout_FieldPaneVisibleRows(t *testing.T) {
	layout := NewLayout(120, 10)
	if got := layout.FieldPaneVisibleRows(false); got != 8 {
		t.Fatalf("FieldPaneVisibleRows(false) = %d, want 8", got)
	}
	if got := layout.FieldPaneVisibleRows(true); got != 7 {
		t.Fatalf("FieldPaneVisibleRows(true) = %d, want 7", got)
	}
}
