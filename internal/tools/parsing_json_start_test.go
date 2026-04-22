package tools

import "testing"

func TestPatternJSONToolCallStartFinder_Find(t *testing.T) {
	finder := newPatternJSONToolCallStartFinder([]string{"{\"id\"", "{\"tool\""})

	input := `prefix {"tool": "read_file"} middle {"id": "call_1", "tool": "bash"}`
	if got := finder.Find(input, 0); got != 7 {
		t.Fatalf("Find(input, 0) = %d, want 7", got)
	}
	if got := finder.Find(input, 20); got <= 20 {
		t.Fatalf("Find(input, 20) = %d, want index after 20", got)
	}
}

func TestPatternJSONToolCallStartFinder_NoMatch(t *testing.T) {
	finder := newPatternJSONToolCallStartFinder([]string{"{\"tool\""})
	if got := finder.Find("plain text", 0); got != -1 {
		t.Fatalf("Find(no match) = %d, want -1", got)
	}
}
