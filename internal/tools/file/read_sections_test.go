package file

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/locator"
	"github.com/susugadx/xelyon-cli/internal/tools"
)

func TestExecuteReadTargetsWithDetailSections_AttachesObservationToSuccessfulSections(t *testing.T) {
	root := t.TempDir()
	targetPath := filepath.Join(root, "target.go")
	if err := os.WriteFile(targetPath, []byte("package sample\n\nconst selected = true\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	reg := locator.NewRegistry()
	id := reg.Register(locator.Location{
		FilePath:     "target.go",
		ResolvedPath: targetPath,
		Line:         3,
		EndLine:      3,
		Name:         "selected",
	})
	execCtx := tools.ExecutionContext{
		Stdout:             io.Discard,
		Stderr:             io.Discard,
		LocatorRegistry:    reg,
		ProjectMapRootPath: root,
		InvocationCWD:      root,
	}

	sections := ExecuteReadTargetsWithDetailSections(execCtx, id, "compact")
	if len(sections) != 1 {
		t.Fatalf("sections len = %d, want 1", len(sections))
	}
	section := sections[0]
	if section.Failed {
		t.Fatalf("section failed: %s", section.Output)
	}
	if section.Observation == nil {
		t.Fatal("Observation = nil, want successful read observation")
	}
	if len(section.Observation.TouchedFiles) != 1 || section.Observation.TouchedFiles[0].ResolvedPath != targetPath {
		t.Fatalf("TouchedFiles = %#v, want target path", section.Observation.TouchedFiles)
	}
	if len(section.Observation.Evidence) != 1 {
		t.Fatalf("Evidence = %#v, want one line evidence", section.Observation.Evidence)
	}
	evidence := section.Observation.Evidence[0]
	if evidence.ResolvedPath != targetPath || evidence.StartLine != 3 || evidence.EndLine != 3 {
		t.Fatalf("Evidence = %#v, want target.go:3", section.Observation.Evidence)
	}
	if !strings.Contains(evidence.Excerpt, "selected") {
		t.Fatalf("Evidence excerpt = %q, want selected line", evidence.Excerpt)
	}
}

func TestExecuteReadTargetsWithDetailSections_FailedSectionsHaveNoObservation(t *testing.T) {
	sections := ExecuteReadTargetsWithDetailSections(tools.ExecutionContext{
		Stdout:          io.Discard,
		Stderr:          io.Discard,
		LocatorRegistry: locator.NewRegistry(),
	}, "[L999]", "compact")

	if len(sections) != 1 {
		t.Fatalf("sections len = %d, want 1", len(sections))
	}
	if !sections[0].Failed {
		t.Fatalf("Failed = false, want failed section for invalid target: %s", sections[0].Output)
	}
	if sections[0].Observation != nil {
		t.Fatalf("Observation = %#v, want nil for failed section", sections[0].Observation)
	}
}

func TestReadFileToolRunResult_AttachesObservation(t *testing.T) {
	root := t.TempDir()
	targetPath := filepath.Join(root, "target.go")
	if err := os.WriteFile(targetPath, []byte("package sample\n\nconst selected = true\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Chdir(root)

	result, err := (&ReadFileTool{}).RunResult(tools.ExecutionContext{
		Stdout:             io.Discard,
		Stderr:             io.Discard,
		LocatorRegistry:    locator.NewRegistry(),
		ProjectMapRootPath: root,
		InvocationCWD:      root,
	}, map[string]string{
		"paths": `["target.go:3-3"]`,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(result.Output, "selected") {
		t.Fatalf("Output = %q, want selected line", result.Output)
	}
	if result.Observation == nil {
		t.Fatal("Observation = nil, want read_file structured observation")
	}
	if len(result.Observation.TouchedFiles) != 1 || result.Observation.TouchedFiles[0].ResolvedPath != targetPath {
		t.Fatalf("TouchedFiles = %#v, want target path", result.Observation.TouchedFiles)
	}
	if len(result.Observation.Evidence) != 1 {
		t.Fatalf("Evidence = %#v, want one line evidence", result.Observation.Evidence)
	}
	evidence := result.Observation.Evidence[0]
	if evidence.ResolvedPath != targetPath || evidence.StartLine != 3 || evidence.EndLine != 3 {
		t.Fatalf("Evidence = %#v, want target.go:3", result.Observation.Evidence)
	}
}
