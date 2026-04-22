package repomap

import "testing"

func TestCollectPriorityFiles_ResolvesFileAndDirectoryCandidates(t *testing.T) {
	pm := &ProjectMap{
		Files: []*FileEntry{
			{Path: "README.md"},
			{Path: "internal/agent/compress.go"},
			{Path: "internal/agent/prompt.go"},
			{Path: "internal/config/project.go"},
		},
	}

	got := pm.collectPriorityFiles([]string{
		" internal/agent/ ",
		"README.md",
		"internal/agent/compress.go",
		"missing/path",
		"",
	})

	want := map[string]struct{}{
		"README.md":                  {},
		"internal/agent/compress.go": {},
		"internal/agent/prompt.go":   {},
	}
	if len(got) != len(want) {
		t.Fatalf("collectPriorityFiles() len = %d, want %d (got=%v)", len(got), len(want), got)
	}
	for _, path := range got {
		if _, ok := want[path]; !ok {
			t.Fatalf("collectPriorityFiles() contains unexpected path %q (all=%v)", path, got)
		}
		delete(want, path)
	}
	if len(want) != 0 {
		t.Fatalf("collectPriorityFiles() missing expected paths: %v (got=%v)", want, got)
	}
}

func TestCollectPriorityFiles_NormalizesSlashWrappedCandidate(t *testing.T) {
	pm := &ProjectMap{
		Files: []*FileEntry{
			{Path: "internal/agent/compress.go"},
			{Path: "internal/agent/prompt.go"},
			{Path: "internal/config/project.go"},
		},
	}

	got := pm.collectPriorityFiles([]string{"/internal/agent/"})
	if len(got) != 2 {
		t.Fatalf("collectPriorityFiles() len = %d, want 2 (got=%v)", len(got), got)
	}
	found := map[string]bool{}
	for _, path := range got {
		found[path] = true
	}
	if !found["internal/agent/compress.go"] || !found["internal/agent/prompt.go"] {
		t.Fatalf("collectPriorityFiles() = %v, want internal/agent files", got)
	}
}

func TestCollectPriorityFiles_EmptyCandidatesReturnsNil(t *testing.T) {
	pm := &ProjectMap{
		Files: []*FileEntry{
			{Path: "README.md"},
		},
	}
	if got := pm.collectPriorityFiles(nil); got != nil {
		t.Fatalf("collectPriorityFiles(nil) = %v, want nil", got)
	}
}
