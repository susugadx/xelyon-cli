package gathercontext

import (
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/tools"
)

func TestToolRunResult_DirectReadAttachesObservation(t *testing.T) {
	root := t.TempDir()
	targetPath := filepath.Join(root, "target.go")
	if err := os.WriteFile(targetPath, []byte("package sample\n\nconst selected = true\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	result, err := (&Tool{}).RunResult(tools.ExecutionContext{
		Stdout:             io.Discard,
		Stderr:             io.Discard,
		ProjectMapRootPath: root,
		InvocationCWD:      root,
	}, map[string]string{
		"query": "./target.go",
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Observation == nil {
		t.Fatal("Observation = nil, want direct read observation")
	}
	for _, file := range result.Observation.TouchedFiles {
		if file.ResolvedPath == targetPath {
			return
		}
	}
	t.Fatalf("TouchedFiles = %#v, want %s", result.Observation.TouchedFiles, targetPath)
}
