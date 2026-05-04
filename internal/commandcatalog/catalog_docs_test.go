package commandcatalog

import (
	"os"
	"strings"
	"testing"
)

func TestAttachLimitDocumentationConsistency(t *testing.T) {
	paths := []string{
		"../../README.md",
		"../../docs/commands.md",
		"../../docs/usage.md",
	}
	for _, path := range paths {
		t.Run(path, func(t *testing.T) {
			body, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("ReadFile(%q) error = %v", path, err)
			}
			text := string(body)
			if !strings.Contains(text, "/attach") {
				t.Fatalf("%s should mention /attach", path)
			}
			if !strings.Contains(text, "最大12件") && !strings.Contains(text, "最大 12 件") {
				t.Fatalf("%s should mention attachment limit (12)", path)
			}
		})
	}
}
