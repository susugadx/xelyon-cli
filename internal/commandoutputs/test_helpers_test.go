package commandoutputs

import (
	"fmt"
	"strings"
	"testing"
)

func numberedLines(prefix string, count int) string {
	var b strings.Builder
	for i := 1; i <= count; i++ {
		fmt.Fprintf(&b, "%s-%03d\n", prefix, i)
	}
	return b.String()
}

func assertSectionContains(t *testing.T, text, heading, want string) {
	t.Helper()
	section := commandOutputTestSection(text, heading)
	if !strings.Contains(section, want) {
		t.Fatalf("section %q missing %q:\n%s", heading, want, text)
	}
}

func assertSectionNotContains(t *testing.T, text, heading, reject string) {
	t.Helper()
	section := commandOutputTestSection(text, heading)
	if strings.Contains(section, reject) {
		t.Fatalf("section %q unexpectedly contains %q:\n%s", heading, reject, text)
	}
}

func commandOutputTestSection(text, heading string) string {
	start := strings.Index(text, heading)
	if start < 0 {
		return ""
	}
	rest := text[start+len(heading):]
	for _, next := range []string{"\nstaged:", "\nunstaged:", "\nuntracked:", "\nconflicted:", "\nother:"} {
		if index := strings.Index(rest, next); index >= 0 {
			return rest[:index]
		}
	}
	return rest
}
