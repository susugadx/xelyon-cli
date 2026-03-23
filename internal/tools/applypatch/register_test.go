package applypatch

import (
	"bytes"
	"io"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/tools"
	"github.com/susugadx/xelyon-cli/internal/tools/common"
)

func TestBuildApplyPatchFileChange_MultipleFiles(t *testing.T) {
	result := &ApplyResult{
		details: []applyResultDetail{
			{Path: "a.go", Action: "modified"},
			{Path: "b.go", Action: "deleted"},
			{Path: "c.go", Action: "created"},
		},
	}

	change := buildApplyPatchFileChange(result)
	if change == nil {
		t.Fatal("buildApplyPatchFileChange() should return a change for multi-file patches")
	}
	if change.FilePath != "a.go" {
		t.Fatalf("FilePath = %q, want %q", change.FilePath, "a.go")
	}
	if len(change.Details) != 3 {
		t.Fatalf("len(Details) = %d, want 3", len(change.Details))
	}
	if change.Description != "Patched 3 files" {
		t.Fatalf("Description = %q, want %q", change.Description, "Patched 3 files")
	}
}

func TestApplyPatchTool_AutoApproveMediumForNonDestructivePatch(t *testing.T) {
	t.Setenv("XELYON_INTERACTIVE_CONFIRM", "0")

	withTempWorkdir(t, func() {
		tool := &ApplyPatchTool{}
		cfg := config.DefaultConfig()
		cfg.ToolConfirm.AutoApproveMedium = true

		output, _, err := tool.Run(newApplyPatchExecContext(cfg, "n\n"), map[string]string{
			"patch": "*** Begin Patch\n*** Add File: hello.txt\n+Hello\n*** End Patch",
		})
		if err != nil {
			t.Fatalf("Run() error = %v", err)
		}
		if strings.HasPrefix(output, "[CANCELLED]") {
			t.Fatalf("Run() should auto-approve non-destructive patch, got %q", output)
		}
		assertFileContent(t, "hello.txt", "Hello\n")
	})
}

func TestApplyPatchTool_DestructivePatchIgnoresAutoApproveMedium(t *testing.T) {
	t.Setenv("XELYON_INTERACTIVE_CONFIRM", "0")

	withTempWorkdir(t, func() {
		writeTestFile(t, "delete.txt", "bye\n")

		tool := &ApplyPatchTool{}
		cfg := config.DefaultConfig()
		cfg.ToolConfirm.AutoApproveMedium = true

		output, _, err := tool.Run(newApplyPatchExecContext(cfg, "n\n"), map[string]string{
			"patch": "*** Begin Patch\n*** Delete File: delete.txt\n*** End Patch",
		})
		if err != nil {
			t.Fatalf("Run() error = %v", err)
		}
		if output != "[CANCELLED] apply_patch was not approved" {
			t.Fatalf("Run() = %q, want destructive patch to require confirmation", output)
		}
		assertFileContent(t, "delete.txt", "bye\n")
	})
}

func TestGetApplyPatchSafety_SamePathMoveIsMedium(t *testing.T) {
	withTempWorkdir(t, func() {
		parsed, err := ParsePatch("*** Begin Patch\n*** Update File: file.txt\n*** Move to: file.txt\n@@\n-old\n+new\n*** End Patch")
		if err != nil {
			t.Fatalf("ParsePatch() error = %v", err)
		}
		if got := getApplyPatchSafety(parsed); got != common.SafetyMedium {
			t.Fatalf("getApplyPatchSafety() = %v, want %v", got, common.SafetyMedium)
		}
	})
}

func newApplyPatchExecContext(cfg *config.Config, stdin string) tools.ExecutionContext {
	var out bytes.Buffer
	return tools.ExecutionContext{
		Stdin:  strings.NewReader(stdin),
		Stdout: &out,
		Stderr: io.Discard,
		Config: cfg,
	}
}

func TestCountLinesPerPath(t *testing.T) {
	tests := []struct {
		name  string
		patch string
		want  map[string][2]int
	}{
		{
			name:  "single file modification",
			patch: "*** Begin Patch\n*** Update File: test.go\n@@ func main()\n-line1\n+new line1\n-line2\n+new line2\n*** End Patch",
			want: map[string][2]int{
				"test.go": {2, 2},
			},
		},
		{
			name:  "multiple files",
			patch: "*** Begin Patch\n*** Update File: a.go\n@@ func a()\n-old a\n+new a\n*** Add File: b.go\n+line1\n+line2\n*** Delete File: c.go\n*** End Patch",
			want: map[string][2]int{
				"a.go": {1, 1},
				"b.go": {2, 0},
			},
		},
		{
			name:  "with move operation",
			patch: "*** Begin Patch\n*** Update File: old.txt\n*** Move to: new.txt\n@@\n-old content\n+new content\n*** End Patch",
			want: map[string][2]int{
				"old.txt": {1, 1},
			},
		},
		{
			name: "empty patch",
			patch: `*** Begin Patch
*** End Patch`,
			want: map[string][2]int{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := countLinesPerPath(tt.patch)
			if len(got) != len(tt.want) {
				t.Errorf("countLinesPerPath() map length = %d, want %d", len(got), len(tt.want))
			}
			for path, wantCounts := range tt.want {
				gotCounts, ok := got[path]
				if !ok {
					t.Errorf("countLinesPerPath() missing path %q", path)
					continue
				}
				if gotCounts[0] != wantCounts[0] || gotCounts[1] != wantCounts[1] {
					t.Errorf("countLinesPerPath()[%q] = %v, want %v", path, gotCounts, wantCounts)
				}
			}
		})
	}
}

func TestFormatApplyResult_ContainsSuccessMarker(t *testing.T) {
	result := &ApplyResult{
		Modified: []string{"test.go"},
	}

	output := formatApplyResult(result)
	if !strings.Contains(output, "✓ Patch applied successfully.") {
		t.Fatalf("formatApplyResult() = %q, want success marker", output)
	}
}
