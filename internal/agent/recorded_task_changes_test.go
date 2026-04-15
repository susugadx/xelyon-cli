package agent

import (
	"testing"

	"github.com/susugadx/xelyon-cli/internal/tools"
)

func TestRecordedTaskChangedFiles(t *testing.T) {
	tests := []struct {
		name      string
		stack     []tools.FileChange
		offset    int
		wantFiles []string
	}{
		{name: "no_changes"},
		{name: "single_file", stack: []tools.FileChange{{FilePath: "/src/main.go"}}, wantFiles: []string{"/src/main.go"}},
		{name: "deduplicates_same_file", stack: []tools.FileChange{{FilePath: "/src/main.go"}, {FilePath: "/src/main.go"}}, wantFiles: []string{"/src/main.go"}},
		{name: "uses_detail_paths", stack: []tools.FileChange{{Details: []tools.FileChangeDetail{{FilePath: "/src/a.go"}, {FilePath: "/src/b.go"}, {FilePath: "/src/a.go"}}}}, wantFiles: []string{"/src/a.go", "/src/b.go"}},
		{name: "respects_task_offset", stack: []tools.FileChange{{FilePath: "/old/a.go"}, {FilePath: "/old/b.go"}, {FilePath: "/src/current.go"}}, offset: 2, wantFiles: []string{"/src/current.go"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := &Agent{agentWorkspaceState: agentWorkspaceState{changeStack: tt.stack, taskChangeOffset: tt.offset}}
			got := a.recordedTaskChangedFiles()
			if len(got) != len(tt.wantFiles) {
				t.Fatalf("recordedTaskChangedFiles() = %v, want %v", got, tt.wantFiles)
			}
			for i, want := range tt.wantFiles {
				if got[i] != want {
					t.Fatalf("recordedTaskChangedFiles()[%d] = %q, want %q", i, got[i], want)
				}
			}
		})
	}
}

func TestRecordedTaskChangeFingerprint_ChangesWhenNewRecordedWriteIsAdded(t *testing.T) {
	a := &Agent{
		agentWorkspaceState: agentWorkspaceState{
			changeStack: []tools.FileChange{{
				FilePath: "/src/main.go",
				Tool:     "write_file",
				Details: []tools.FileChangeDetail{{
					FilePath: "/src/main.go",
					Action:   "modified",
				}},
			}},
		},
	}

	first := a.recordedTaskChangeFingerprint()
	if first == "" {
		t.Fatal("expected non-empty fingerprint after first recorded change")
	}

	a.changeStack = append(a.changeStack, tools.FileChange{
		FilePath: "/src/main.go",
		Tool:     "write_file",
		Details: []tools.FileChangeDetail{{
			FilePath: "/src/main.go",
			Action:   "modified",
		}},
	})

	second := a.recordedTaskChangeFingerprint()
	if second == "" {
		t.Fatal("expected non-empty fingerprint after second recorded change")
	}
	if second == first {
		t.Fatal("expected fingerprint to change when a new recorded write is added")
	}
}
