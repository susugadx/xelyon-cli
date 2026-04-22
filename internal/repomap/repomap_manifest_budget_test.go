package repomap

import (
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func TestGenerateManifest_StaysWithinBudgetWithManyChanges(t *testing.T) {
	pm := &ProjectMap{
		MaxTokens: 40,
		Files: []*FileEntry{
			{Path: "README.md"},
			{Path: "Makefile"},
			{Path: "internal/agent/compress.go"},
			{Path: "internal/config/project.go"},
		},
	}
	for i := 0; i < 30; i++ {
		pm.GitStatus = append(pm.GitStatus, GitChange{
			Status: "M",
			Path:   filepath.ToSlash(filepath.Join("internal", "agent", "file"+strconv.Itoa(i)+".go")),
		})
	}

	output := pm.GenerateManifest([]string{"internal/agent"})
	if !pm.fitsBudget(output) {
		t.Fatalf("manifest must stay within budget, got:\n%s", output)
	}
	if strings.Count(output, "\n- M ") >= len(pm.GitStatus) {
		t.Fatalf("expected uncommitted changes to be trimmed under budget:\n%s", output)
	}
}
