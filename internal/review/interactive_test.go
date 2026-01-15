package review

import (
	"os"
	"path/filepath"
	"testing"
)

func TestTruncateCode(t *testing.T) {
	tests := []struct {
		name   string
		code   string
		maxLen int
		want   string
	}{
		{
			name:   "short code",
			code:   "http.Client{}",
			maxLen: 50,
			want:   "http.Client{}",
		},
		{
			name:   "exact length",
			code:   "12345",
			maxLen: 5,
			want:   "12345",
		},
		{
			name:   "truncated",
			code:   "this is a very long code snippet that needs truncation",
			maxLen: 20,
			want:   "this is a very lo...",
		},
		{
			name:   "with newlines",
			code:   "line1\nline2\nline3",
			maxLen: 50,
			want:   "line1 line2 line3",
		},
		{
			name:   "with tabs",
			code:   "func() {\n\treturn\n}",
			maxLen: 50,
			want:   "func() {  return }",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := truncateCode(tt.code, tt.maxLen)
			if got != tt.want {
				t.Errorf("truncateCode() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestApplyFix(t *testing.T) {
	// Create temp directory
	tmpDir := t.TempDir()

	t.Run("successful fix", func(t *testing.T) {
		// Create test file
		testFile := filepath.Join(tmpDir, "test1.go")
		content := `package main

import "net/http"

func main() {
	client := &http.Client{}
	_ = client
}
`
		if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
			t.Fatalf("failed to create test file: %v", err)
		}

		proposal := FixProposal{
			Actionable: true,
			FilePath:   testFile,
			OldCode:    "http.Client{}",
			NewCode:    "http.Client{Timeout: 30 * time.Second}",
		}

		err := ApplyFix(proposal)
		if err != nil {
			t.Fatalf("ApplyFix() error = %v", err)
		}

		// Verify the fix was applied
		newContent, err := os.ReadFile(testFile)
		if err != nil {
			t.Fatalf("failed to read file: %v", err)
		}

		if string(newContent) != `package main

import "net/http"

func main() {
	client := &http.Client{Timeout: 30 * time.Second}
	_ = client
}
` {
			t.Errorf("file content after fix:\n%s", string(newContent))
		}
	})

	t.Run("non-actionable fix", func(t *testing.T) {
		proposal := FixProposal{
			Actionable: false,
			FilePath:   "test.go",
		}

		err := ApplyFix(proposal)
		if err == nil {
			t.Error("expected error for non-actionable fix")
		}
	})

	t.Run("missing file path", func(t *testing.T) {
		proposal := FixProposal{
			Actionable: true,
			FilePath:   "",
		}

		err := ApplyFix(proposal)
		if err == nil {
			t.Error("expected error for missing file path")
		}
	})

	t.Run("missing replacement code", func(t *testing.T) {
		proposal := FixProposal{
			Actionable: true,
			FilePath:   "test.go",
			OldCode:    "",
			NewCode:    "",
		}

		err := ApplyFix(proposal)
		if err == nil {
			t.Error("expected error for missing replacement code")
		}
	})

	t.Run("pattern not found", func(t *testing.T) {
		testFile := filepath.Join(tmpDir, "test2.go")
		content := `package main

func main() {}
`
		if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
			t.Fatalf("failed to create test file: %v", err)
		}

		proposal := FixProposal{
			Actionable: true,
			FilePath:   testFile,
			OldCode:    "http.Client{}",
			NewCode:    "http.Client{Timeout: 30 * time.Second}",
		}

		err := ApplyFix(proposal)
		if err == nil {
			t.Error("expected error when pattern not found")
		}
	})

	t.Run("file not found", func(t *testing.T) {
		proposal := FixProposal{
			Actionable: true,
			FilePath:   filepath.Join(tmpDir, "nonexistent.go"),
			OldCode:    "old",
			NewCode:    "new",
		}

		err := ApplyFix(proposal)
		if err == nil {
			t.Error("expected error for nonexistent file")
		}
	})
}

func TestNewInteractiveFixer(t *testing.T) {
	fixer := NewInteractiveFixer()

	if fixer == nil {
		t.Fatal("NewInteractiveFixer() returned nil")
	}

	if fixer.Reader == nil {
		t.Error("Reader should not be nil")
	}

	if fixer.AutoAll {
		t.Error("AutoAll should be false by default")
	}
}

func TestFixResult(t *testing.T) {
	result := FixResult{
		Applied: 5,
		Skipped: 2,
		Failed:  1,
		Quit:    false,
	}

	if result.Applied != 5 {
		t.Errorf("Applied = %d, want 5", result.Applied)
	}
	if result.Skipped != 2 {
		t.Errorf("Skipped = %d, want 2", result.Skipped)
	}
	if result.Failed != 1 {
		t.Errorf("Failed = %d, want 1", result.Failed)
	}
	if result.Quit {
		t.Error("Quit should be false")
	}
}

func TestRunInteractiveFixes_NoActionable(t *testing.T) {
	issues := []Issue{
		{ID: "issue1", Title: "Issue 1"},
	}

	// All proposals are non-actionable
	proposals := []FixProposal{
		{IssueID: "issue1", Actionable: false},
	}

	result := RunInteractiveFixes(issues, proposals, true)

	// No actionable fixes should result in no applied/skipped/failed
	if result.Applied != 0 {
		t.Errorf("Applied = %d, want 0", result.Applied)
	}
}
