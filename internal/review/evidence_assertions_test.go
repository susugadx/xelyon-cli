package review

import (
	"os"
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
