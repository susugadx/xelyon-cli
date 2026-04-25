package viewport

import "testing"

func TestVisibleLines(t *testing.T) {
	got := VisibleLines([]string{"a", "b", "c"}, 1, 5)
	want := []string{"b", "c"}
	if len(got) != len(want) {
		t.Fatalf("VisibleLines() len = %d, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("VisibleLines()[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestRenderPadsHeightAndWidth(t *testing.T) {
	got := Render([]string{"a"}, 0, 3, 4)
	want := "a   \n    \n    "
	if got != want {
		t.Fatalf("Render() = %q, want %q", got, want)
	}
}

func TestScrollDownClamps(t *testing.T) {
	if got := ScrollDown(0, 10, 3, 2); got != 1 {
		t.Fatalf("ScrollDown() = %d, want 1", got)
	}
}
