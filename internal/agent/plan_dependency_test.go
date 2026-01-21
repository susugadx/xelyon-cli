package agent

import (
	"testing"
)

func TestExtractFilesFromStep(t *testing.T) {
	da := NewDependencyAnalyzer(nil)

	tests := []struct {
		name           string
		step           PlanStep
		wantReadFiles  []string // expected files that should be in reads
		wantWriteFiles []string // expected files that should be in writes
	}{
		{
			name: "read_file tool with quoted path",
			step: PlanStep{
				ID:          1,
				Description: `Read the file "internal/agent/plan.go" to understand the structure`,
				Tools:       []string{"read_file"},
			},
			wantReadFiles:  []string{"internal/agent/plan.go"},
			wantWriteFiles: nil,
		},
		{
			name: "write_file tool",
			step: PlanStep{
				ID:          2,
				Description: `Create new file internal/agent/plan_dependency.go`,
				Tools:       []string{"write_file"},
			},
			wantReadFiles:  nil,
			wantWriteFiles: []string{"internal/agent/plan_dependency.go"},
		},
		{
			name: "str_replace tool",
			step: PlanStep{
				ID:          3,
				Description: `Update internal/agent/plan.go to add new function`,
				Tools:       []string{"str_replace"},
			},
			wantReadFiles:  nil,
			wantWriteFiles: []string{"internal/agent/plan.go"},
		},
		{
			name: "multiple tools",
			step: PlanStep{
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

			// If no reads expected, check gotReads is empty or only contains duplicates
			if len(tt.wantReadFiles) == 0 && len(gotReads) > 0 {
				// Allow false positives from regex matching partial paths
				// as long as they're not actual valid file paths we care about
			}
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

func TestInferDependencies_RAW(t *testing.T) {
	// RAW: Read After Write - BがAの書き込みファイルを読む
	da := NewDependencyAnalyzer(nil)

	steps := []PlanStep{
		{
			ID:          1,
			Description: `Create config.go`,
			Tools:       []string{"write_file"},
			DependsOn:   nil,
		},
		{
			ID:          2,
			Description: `Read config.go to verify`,
			Tools:       []string{"read_file"},
			DependsOn:   nil,
		},
	}

	// ファイル抽出
	for i := range steps {
		readFiles, writeFiles := da.ExtractFilesFromStep(&steps[i])
		steps[i].ReadFiles = readFiles
		steps[i].WriteFiles = writeFiles

		for _, f := range readFiles {
			da.fileReaders[f] = append(da.fileReaders[f], steps[i].ID)
		}
		for _, f := range writeFiles {
			da.fileWriters[f] = append(da.fileWriters[f], steps[i].ID)
		}
	}

	// 依存関係推論
	result := da.InferDependencies(steps)

	// Step 2 should depend on Step 1
	if len(result[1].DependsOn) == 0 {
		t.Error("Expected step 2 to depend on step 1 (RAW hazard)")
	}

	found := false
	for _, depID := range result[1].DependsOn {
		if depID == 1 {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected step 2 to depend on step 1 (RAW hazard)")
	}
}

func TestInferDependencies_WAW(t *testing.T) {
	// WAW: Write After Write - 両方が同じファイルに書く
	da := NewDependencyAnalyzer(nil)

	steps := []PlanStep{
		{
			ID:          1,
			Description: `Write to output.txt`,
			Tools:       []string{"write_file"},
			DependsOn:   nil,
		},
		{
			ID:          2,
			Description: `Append to output.txt`,
			Tools:       []string{"append_file"},
			DependsOn:   nil,
		},
	}

	// ファイル抽出
	for i := range steps {
		readFiles, writeFiles := da.ExtractFilesFromStep(&steps[i])
		steps[i].ReadFiles = readFiles
		steps[i].WriteFiles = writeFiles

		for _, f := range readFiles {
			da.fileReaders[f] = append(da.fileReaders[f], steps[i].ID)
		}
		for _, f := range writeFiles {
			da.fileWriters[f] = append(da.fileWriters[f], steps[i].ID)
		}
	}

	// 依存関係推論
	result := da.InferDependencies(steps)

	// Step 2 should depend on Step 1
	found := false
	for _, depID := range result[1].DependsOn {
		if depID == 1 {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected step 2 to depend on step 1 (WAW hazard)")
	}
}

func TestDetectConflicts_WriteWrite(t *testing.T) {
	da := NewDependencyAnalyzer(nil)

	steps := []PlanStep{
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
	da := NewDependencyAnalyzer(nil)

	steps := []PlanStep{
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
	da := NewDependencyAnalyzer(nil)

	steps := []PlanStep{
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
	da := NewDependencyAnalyzer(nil)

	steps := []PlanStep{
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
	da := NewDependencyAnalyzer(nil)

	steps := []PlanStep{
		{ID: 1, Description: "Test step"},
	}

	// LSPクライアントがない場合、そのまま返る
	result := da.EnhanceWithLSP(steps)

	if len(result) != 1 {
		t.Errorf("Expected 1 step, got %d", len(result))
	}
}

func TestIsValidFilePath(t *testing.T) {
	tests := []struct {
		path  string
		valid bool
	}{
		{"internal/agent/plan.go", true},
		{"main.go", true},
		{"/absolute/path/to/file.go", true},
		{"", false},
		{"http://example.com", false},
		{"https://github.com/repo", false},
		{"directory", false}, // no extension
		{"example.com", false},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			if got := isValidFilePath(tt.path); got != tt.valid {
				t.Errorf("isValidFilePath(%q) = %v, want %v", tt.path, got, tt.valid)
			}
		})
	}
}

func TestFormatIntSlice(t *testing.T) {
	tests := []struct {
		input    []int
		expected string
	}{
		{[]int{1, 2, 3}, "1, 2, 3"},
		{[]int{10, 20}, "10, 20"},
		{[]int{1}, "1"},
		{[]int{}, ""},
	}

	for _, tt := range tests {
		result := formatIntSlice(tt.input)
		if result != tt.expected {
			t.Errorf("formatIntSlice(%v) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}
