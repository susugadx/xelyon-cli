package agent

import (
	"testing"

	"github.com/susugadx/xelyon-cli/internal/agent/plan"
)

func TestExtractFilesFromStep(t *testing.T) {
	da := plan.NewDependencyAnalyzer(nil)

	tests := []struct {
		name           string
		step           plan.PlanStep
		wantReadFiles  []string // expected files that should be in reads
		wantWriteFiles []string // expected files that should be in writes
	}{
		{
			name: "read_file tool with quoted path",
			step: plan.PlanStep{
				ID:          1,
				Description: `Read the file "internal/agent/plan.go" to understand the structure`,
				Tools:       []string{"read_file"},
			},
			wantReadFiles:  []string{"internal/agent/plan.go"},
			wantWriteFiles: nil,
		},
		{
			name: "write_file tool",
			step: plan.PlanStep{
				ID:          2,
				Description: `Create new file internal/agent/plan_dependency.go`,
				Tools:       []string{"write_file"},
			},
			wantReadFiles:  nil,
			wantWriteFiles: []string{"internal/agent/plan_dependency.go"},
		},
		{
			name: "str_replace tool",
			step: plan.PlanStep{
				ID:          3,
				Description: `Update internal/agent/plan.go to add new function`,
				Tools:       []string{"str_replace"},
			},
			wantReadFiles:  nil,
			wantWriteFiles: []string{"internal/agent/plan.go"},
		},
		{
			name: "multiple tools",
			step: plan.PlanStep{
				ID:          4,
				Description: `Read main.go and update config.go`,
				Tools:       []string{"read_file", "str_replace"},
			},
			wantReadFiles:  []string{"config.go", "main.go"},
			wantWriteFiles: []string{"config.go", "main.go"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotReads, gotWrites := da.ExtractFilesFromStep(&tt.step)

			// Check that expected read files are present
			for _, wantFile := range tt.wantReadFiles {
				if !containsFile(gotReads, wantFile) {
					t.Errorf("ExtractFilesFromStep() gotReads = %v, missing expected file %q", gotReads, wantFile)
				}
			}

			// Check that expected write files are present
			for _, wantFile := range tt.wantWriteFiles {
				if !containsFile(gotWrites, wantFile) {
					t.Errorf("ExtractFilesFromStep() gotWrites = %v, missing expected file %q", gotWrites, wantFile)
				}
			}

			// Note: If no reads expected but gotReads has items, we allow false positives
			// from regex matching partial paths as long as they're not actual valid file paths
		})
	}
}

// containsFile checks if a file path is in the list
func containsFile(files []string, target string) bool {
	for _, f := range files {
		if f == target {
			return true
		}
	}
	return false
}

func TestDetectConflicts_WriteWrite(t *testing.T) {
	da := plan.NewDependencyAnalyzer(nil)

	steps := []plan.PlanStep{
		{
			ID:         1,
			WriteFiles: []string{"shared.go"},
		},
		{
			ID:         2,
			WriteFiles: []string{"shared.go"},
		},
	}

	// 並列実行で競合検出
	conflicts := da.DetectConflicts([]int{1, 2}, steps)

	if len(conflicts) == 0 {
		t.Error("Expected write-write conflict to be detected")
	}

	if conflicts[0].ConflictType != "write-write" {
		t.Errorf("Expected conflict type 'write-write', got '%s'", conflicts[0].ConflictType)
	}
}

func TestDetectConflicts_ReadWrite(t *testing.T) {
	da := plan.NewDependencyAnalyzer(nil)

	steps := []plan.PlanStep{
		{
			ID:        1,
			ReadFiles: []string{"data.go"},
		},
		{
			ID:         2,
			WriteFiles: []string{"data.go"},
		},
	}

	// 並列実行で競合検出
	conflicts := da.DetectConflicts([]int{1, 2}, steps)

	if len(conflicts) == 0 {
		t.Error("Expected read-write conflict to be detected")
	}

	if conflicts[0].ConflictType != "read-write" {
		t.Errorf("Expected conflict type 'read-write', got '%s'", conflicts[0].ConflictType)
	}
}

func TestDetectConflicts_NoConflict(t *testing.T) {
	da := plan.NewDependencyAnalyzer(nil)

	steps := []plan.PlanStep{
		{
			ID:         1,
			WriteFiles: []string{"file1.go"},
		},
		{
			ID:         2,
			WriteFiles: []string{"file2.go"},
		},
	}

	// 並列実行で競合なし
	conflicts := da.DetectConflicts([]int{1, 2}, steps)

	if len(conflicts) != 0 {
		t.Errorf("Expected no conflicts, got %d", len(conflicts))
	}
}

func TestAnalyze_IntegrationTest(t *testing.T) {
	da := plan.NewDependencyAnalyzer(nil)

	steps := []plan.PlanStep{
		{
			ID:          1,
			Description: `Read internal/agent/plan.go to understand structure`,
			Tools:       []string{"read_file"},
			DependsOn:   nil,
		},
		{
			ID:          2,
			Description: `Create internal/agent/plan_dependency.go based on plan.go patterns`,
			Tools:       []string{"write_file"},
			DependsOn:   nil, // AI forgot to set dependency
		},
		{
			ID:          3,
			Description: `Update internal/agent/plan.go to integrate dependency analyzer`,
			Tools:       []string{"str_replace"},
			DependsOn:   []int{2}, // Correctly set
		},
	}

	result := da.Analyze(steps)

	// Step 2 reads plan.go (implicitly for "based on plan.go patterns")
	// This test verifies the analyzer processes the steps correctly
	if len(result.Steps) != 3 {
		t.Errorf("Expected 3 steps, got %d", len(result.Steps))
	}
}

func TestEnhanceWithLSP_WithoutClient(t *testing.T) {
	da := plan.NewDependencyAnalyzer(nil)

	steps := []plan.PlanStep{
		{ID: 1, Description: "Test step"},
	}

	// LSPクライアントがない場合、そのまま返る
	result := da.EnhanceWithLSP(steps)

	if len(result) != 1 {
		t.Errorf("Expected 1 step, got %d", len(result))
	}
}

func TestFormatConflictWarning(t *testing.T) {
	tests := []struct {
		name     string
		conflict plan.Conflict
		want     string
	}{
		{
			name: "single file write-write conflict",
			conflict: plan.Conflict{
				StepIDs:      []int{1, 2},
				ConflictType: "write-write",
				Files:        []string{"main.go"},
				Message:      "Both steps modify the same file",
			},
			want: "Conflict detected: Both steps modify the same file (steps: 1, 2, files: main.go)",
		},
		{
			name: "multiple files read-write conflict",
			conflict: plan.Conflict{
				StepIDs:      []int{3, 5, 7},
				ConflictType: "read-write",
				Files:        []string{"config.yaml", "settings.json"},
				Message:      "Read after write dependency",
			},
			want: "Conflict detected: Read after write dependency (steps: 3, 5, 7, files: config.yaml, settings.json)",
		},
		{
			name: "empty files",
			conflict: plan.Conflict{
				StepIDs:      []int{1},
				ConflictType: "write-read",
				Files:        []string{},
				Message:      "Potential conflict",
			},
			want: "Conflict detected: Potential conflict (steps: 1, files: )",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := plan.FormatConflictWarning(tt.conflict)
			if got != tt.want {
				t.Errorf("FormatConflictWarning() = %q, want %q", got, tt.want)
			}
		})
	}
}
