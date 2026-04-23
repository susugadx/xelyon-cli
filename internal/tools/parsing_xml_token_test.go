package tools

import "testing"

func TestFindNextXMLOpenTag(t *testing.T) {
	text := "before <read_file>body</read_file>"
	token, ok := findNextXMLOpenTag(text, 0)
	if !ok {
		t.Fatal("findNextXMLOpenTag() ok = false, want true")
	}
	if token.openStart != 7 {
		t.Fatalf("openStart = %d, want 7", token.openStart)
	}
	if token.contentStart != 18 {
		t.Fatalf("contentStart = %d, want 18", token.contentStart)
	}
	if token.tagName != "read_file" {
		t.Fatalf("tagName = %q, want read_file", token.tagName)
	}
}

func TestFindXMLCloseTagIndex(t *testing.T) {
	text := "<bash>echo ok</bash>"
	contentStart := len("<bash>")
	got := findXMLCloseTagIndex(text, contentStart, "bash")
	if got != len("echo ok") {
		t.Fatalf("findXMLCloseTagIndex() = %d, want %d", got, len("echo ok"))
	}
}
