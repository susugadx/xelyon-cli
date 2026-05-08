package toolblock

import "testing"

func TestSummaryLine(t *testing.T) {
	if got := SummaryLine("search_code: test", true, false); got != "  search_code: test" {
		t.Fatalf("SummaryLine(collapsed) = %q", got)
	}
	if got := SummaryLine("search_code: test", true, true); got != "▶ search_code: test" {
		t.Fatalf("SummaryLine(focused) = %q", got)
	}
	if got := SummaryLine("search_code: test", false, true); got != "▶ search_code: test" {
		t.Fatalf("SummaryLine(expanded) = %q", got)
	}
}

func TestLines(t *testing.T) {
	got := Lines("search_code: test", "a\nb", false, false)
	want := []string{"  search_code: test", "  a", "  b"}
	if len(got) != len(want) {
		t.Fatalf("Lines() len = %d, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("Lines()[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestLines_CollapsedKeepsOnlySummary(t *testing.T) {
	got := Lines("bash: echo ok", "ok\nextra", true, true)
	want := []string{"▶ bash: echo ok"}
	if len(got) != len(want) {
		t.Fatalf("Lines() len = %d, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("Lines()[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}
