package evidence

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func assertChangedFiles(t *testing.T, got, want []ReviewChangedFile) {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("ChangedFiles = %#v, want %#v", got, want)
	}
}

func diffEvidenceBySource(t *testing.T, bundle ReviewEvidenceBundle, source string) ReviewDiffEvidence {
	t.Helper()
	for _, diff := range bundle.Diffs {
		if diff.Source == source {
			return diff
		}
	}
	t.Fatalf("diff source %q not found in %#v", source, bundle.Diffs)
	return ReviewDiffEvidence{}
}

func ruleFileByPath(t *testing.T, bundle ReviewEvidenceBundle, path string) ReviewRuleFileEvidence {
	t.Helper()
	for _, file := range bundle.RuleFiles {
		if file.Path == path {
			return file
		}
	}
	t.Fatalf("rule file %q not found in %#v", path, bundle.RuleFiles)
	return ReviewRuleFileEvidence{}
}

func assertStringSlice(t *testing.T, got, want []string) {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("slice = %#v, want %#v", got, want)
	}
}

func assertReviewEvidenceNoANSI(t *testing.T, label, value string) {
	t.Helper()
	if strings.Contains(value, "\x1b[") {
		t.Fatalf("%s contains ANSI escape: %q", label, value)
	}
}

func assertFileAbsent(t *testing.T, path string) {
	t.Helper()

	if _, err := os.Stat(path); err == nil {
		content, readErr := os.ReadFile(path)
		if readErr != nil {
			t.Fatalf("file %q exists and ReadFile failed: %v", path, readErr)
		}
		t.Fatalf("file %q was created with %q", path, string(content))
	} else if !os.IsNotExist(err) {
		t.Fatalf("Stat(%q) error = %v", path, err)
	}
}

func requireReviewUntrackedFile(t *testing.T, bundle ReviewEvidenceBundle, path string) ReviewUntrackedFile {
	t.Helper()
	for _, file := range bundle.UntrackedFiles {
		if file.Path == path {
			return file
		}
	}
	t.Fatalf("UntrackedFiles = %#v, want %q", bundle.UntrackedFiles, path)
	return ReviewUntrackedFile{}
}

func requireReviewContextFile(t *testing.T, files []ReviewContextFileEvidence, path string) ReviewContextFileEvidence {
	t.Helper()
	for _, file := range files {
		if filepath.ToSlash(file.Path) == filepath.ToSlash(path) {
			return file
		}
	}
	t.Fatalf("context files = %#v, want %q", files, path)
	return ReviewContextFileEvidence{}
}

func hasReviewContextFile(files []ReviewContextFileEvidence, path string) bool {
	for _, file := range files {
		if filepath.ToSlash(file.Path) == filepath.ToSlash(path) {
			return true
		}
	}
	return false
}

func hasReviewSearchHit(hits []ReviewRelatedSearchHit, path string) bool {
	for _, hit := range hits {
		if filepath.ToSlash(hit.Path) == filepath.ToSlash(path) {
			return true
		}
	}
	return false
}

func hasReviewSearchHitWithReason(hits []ReviewRelatedSearchHit, path, reason string) bool {
	for _, hit := range hits {
		if filepath.ToSlash(hit.Path) == filepath.ToSlash(path) && hit.Reason == reason {
			return true
		}
	}
	return false
}

func reviewContextEvidenceContainsText(files []ReviewContextFileEvidence, text string) bool {
	for _, file := range files {
		if strings.Contains(file.Path, text) || strings.Contains(file.Content, text) || strings.Contains(file.SkipReason, text) {
			return true
		}
	}
	return false
}

func reviewSearchEvidenceContainsText(hits []ReviewRelatedSearchHit, text string) bool {
	for _, hit := range hits {
		if strings.Contains(hit.Path, text) || strings.Contains(hit.Snippet, text) || strings.Contains(hit.Reason, text) {
			return true
		}
	}
	return false
}

func requireReviewSearchHit(t *testing.T, hits []ReviewRelatedSearchHit, path string) ReviewRelatedSearchHit {
	t.Helper()
	for _, hit := range hits {
		if filepath.ToSlash(hit.Path) == filepath.ToSlash(path) {
			return hit
		}
	}
	t.Fatalf("search hits = %#v, want path %q", hits, path)
	return ReviewRelatedSearchHit{}
}
