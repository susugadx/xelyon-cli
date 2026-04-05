package file

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/testutil"
	"github.com/susugadx/xelyon-cli/internal/tools/common"
)

func TestValidateReadFilesPaths(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		paths []string
		want  string
	}{
		{name: "empty", paths: nil, want: "Error: paths is empty"},
		{name: "too many", paths: make([]string, MaxReadFilesPaths+1), want: "Error: too many paths"},
		{name: "valid", paths: []string{"a.go"}, want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := validateReadFilesPaths(tt.paths)
			if tt.want == "" {
				if got != "" {
					t.Fatalf("validateReadFilesPaths() = %q, want empty", got)
				}
				return
			}
			if !strings.Contains(got, tt.want) {
				t.Fatalf("validateReadFilesPaths() = %q, want substring %q", got, tt.want)
			}
		})
	}
}

func TestResolveReadFilesBudget(t *testing.T) {
	t.Parallel()

	if got := resolveReadFilesBudget(0); got != DefaultFullLines {
		t.Fatalf("resolveReadFilesBudget(0) = %d, want %d", got, DefaultFullLines)
	}
	if got := resolveReadFilesBudget(123); got != 123 {
		t.Fatalf("resolveReadFilesBudget(123) = %d, want 123", got)
	}
}

func TestExecuteReadBatchEntry_ParsesRange(t *testing.T) {
	setupTestMocks(t)
	defer withPermissiveValidatePath(t)()
	tmpDir := t.TempDir()

	testutil.CreateTempFile(t, tmpDir, "range.txt", "line1\nline2\nline3\nline4")
	entry := filepath.Join(tmpDir, "range.txt") + ":2-3"

	result := executeReadBatchEntry(common.DefaultOutput(), nil, nil, entry, DefaultFullLines)

	if result.entry != entry {
		t.Fatalf("unexpected entry: %q", result.entry)
	}
	if result.filePath != filepath.Join(tmpDir, "range.txt") {
		t.Fatalf("unexpected filePath: %q", result.filePath)
	}
	if result.startLine != 2 || result.endLine != 3 {
		t.Fatalf("unexpected range: %d-%d", result.startLine, result.endLine)
	}
	if !strings.Contains(result.result, "2: line2") || !strings.Contains(result.result, "3: line3") {
		t.Fatalf("expected ranged result, got: %s", result.result)
	}
}

func TestReadFilesInParallel_PreservesInputOrder(t *testing.T) {
	setupTestMocks(t)
	defer withPermissiveValidatePath(t)()
	tmpDir := t.TempDir()

	testutil.CreateTempFile(t, tmpDir, "first.txt", "first")
	testutil.CreateTempFile(t, tmpDir, "second.txt", "second")
	testutil.CreateTempFile(t, tmpDir, "third.txt", "third")

	paths := []string{
		filepath.Join(tmpDir, "third.txt"),
		filepath.Join(tmpDir, "first.txt"),
		filepath.Join(tmpDir, "second.txt"),
	}

	results := readFilesInParallel(common.DefaultOutput(), nil, nil, paths, DefaultFullLines)
	if len(results) != len(paths) {
		t.Fatalf("unexpected result len: %d", len(results))
	}
	for i, path := range paths {
		if results[i].entry != path {
			t.Fatalf("results[%d].entry = %q, want %q", i, results[i].entry, path)
		}
	}
}
