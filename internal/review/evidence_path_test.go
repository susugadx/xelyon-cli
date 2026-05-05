package review

import (
	"path/filepath"
	"testing"
)

func TestReviewEvidencePathValidationRejectsUnsafePaths(t *testing.T) {
	repo := filepath.Clean(t.TempDir())
	outside := filepath.Join(t.TempDir(), "outside.txt")

	tests := []struct {
		name string
		path string
	}{
		{name: "absolute path", path: filepath.Join(repo, "file.txt")},
		{name: "dotdot escape", path: "../outside.txt"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, _, err := resolveReviewEvidenceRepoPathLexically(repo, tt.path); err == nil {
				t.Fatalf("resolveReviewEvidenceRepoPathLexically(%q) error = nil", tt.path)
			}
		})
	}

	if err := validateReviewEvidencePathWithinRepoRoot(repo, outside, "outside.txt"); err == nil {
		t.Fatalf("validateReviewEvidencePathWithinRepoRoot(%q) error = nil", outside)
	}
}
