package plan

import (
	"testing"
)

func TestNewDependencyAnalyzer(t *testing.T) {
	da := NewDependencyAnalyzer(nil)
	if da == nil {
		t.Fatal("NewDependencyAnalyzer returned nil")
	}
	if da.fileWriters == nil {
		t.Error("fileWriters map not initialized")
	}
	if da.fileReaders == nil {
		t.Error("fileReaders map not initialized")
	}
	if da.lspClient != nil {
		t.Error("lspClient should be nil")
	}
}

func TestExtractFilePaths(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		expected []string
	}{
		{
			name:     "quoted path",
			text:     `Read the file "internal/config/config.go"`,
			expected: []string{"internal/config/config.go"},
		},
		{
			name:     "unquoted path",
			text:     "Edit internal/api/client.go to fix the bug",
			expected: []string{"internal/api/client.go"},
		},
		{
			name:     "multiple paths",
			text:     `Read "main.go" and modify "cmd/root.go"`,
			expected: []string{"main.go", "cmd/root.go"},
		},
		{
			name:     "absolute path",
			text:     "File at /home/user/project/main.go",
			expected: []string{"/home/user/project/main.go"},
		},
		{
			name:     "no paths",
			text:     "Run the tests and check for errors",
			expected: []string{},
		},
		{
			name:     "URL should be excluded",
			text:     "Visit https://example.com/page.html",
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractFilePaths(tt.text)

			// Check if expected paths are present
			resultSet := make(map[string]bool)
			for _, r := range result {
				resultSet[r] = true
			}

			for _, exp := range tt.expected {
				if !resultSet[exp] {
					t.Errorf("Expected path %q not found in result: %v", exp, result)
				}
			}
		})
	}
}

func TestIsValidFilePath(t *testing.T) {
	tests := []struct {
		path     string
		expected bool
	}{
		{"main.go", true},
		{"internal/config/config.go", true},
		{"/absolute/path/file.txt", true},
		{"", false},                     // empty
		{"https://example.com", false},  // URL
		{"http://example.com", false},   // URL
		{"noextension", false},          // no extension
		{"example.com", false},          // domain-like
		{"github.io", false},            // domain-like
		{"test.org", false},             // domain-like
		{"internal/api/test.go", true},  // valid nested path
		{"README.md", true},             // valid file
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			result := isValidFilePath(tt.path)
			if result != tt.expected {
				t.Errorf("isValidFilePath(%q) = %v, want %v", tt.path, result, tt.expected)
			}
		})
	}
}

func TestExtractFilesFromStep(t *testing.T) {
	da := NewDependencyAnalyzer(nil)

	tests := []struct {
		name           string
		step           PlanStep
		expectedReads  []string
		expectedWrites []string
	}{
		{
			name: "read tool",
			step: PlanStep{
				ID:          1,
				Description: `Read "config.go" to understand the structure`,
				Tools:       []string{"read_file"},
			},
			expectedReads:  []string{"config.go"},
			expectedWrites: []string{},
		},
		{
			name: "write tool only",
			step: PlanStep{
				ID:          2,
				Description: `Write changes to "main.go"`,
				Tools:       []string{"write_file"},
			},
			expectedReads:  []string{}, // write-only tool doesn't imply read
			expectedWrites: []string{"main.go"},
		},
		{
			name: "mixed tools",
			step: PlanStep{
				ID:          3,
				Description: `Read "input.go" and write to "output.go"`,
				Tools:       []string{"read_file", "write_file"},
			},
			expectedReads:  []string{"input.go", "output.go"},
			expectedWrites: []string{"input.go", "output.go"},
		},
		{
			name: "no tools specified",
			step: PlanStep{
				ID:          4,
				Description: `Check "test.go" file`,
				Tools:       []string{},
			},
			expectedReads:  []string{"test.go"},
			expectedWrites: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reads, writes := da.ExtractFilesFromStep(&tt.step)

			readSet := make(map[string]bool)
			for _, r := range reads {
				readSet[r] = true
			}
			writeSet := make(map[string]bool)
			for _, w := range writes {
				writeSet[w] = true
			}

			for _, exp := range tt.expectedReads {
				if !readSet[exp] {
					t.Errorf("Expected read file %q not found in reads: %v", exp, reads)
				}
			}
			for _, exp := range tt.expectedWrites {
				if !writeSet[exp] {
					t.Errorf("Expected write file %q not found in writes: %v", exp, writes)
				}
			}
		})
	}
}

func TestInferDependencies_RAW(t *testing.T) {
	// RAW: Read After Write - Step 2 reads what Step 1 writes
	da := NewDependencyAnalyzer(nil)

	// Setup: Step 1 writes to config.go
	da.fileWriters["config.go"] = []int{1}

	steps := []PlanStep{
		{ID: 1, WriteFiles: []string{"config.go"}},
		{ID: 2, ReadFiles: []string{"config.go"}},
	}

	result := da.InferDependencies(steps)

	// Step 2 should depend on Step 1
	found := false
	for _, dep := range result[1].DependsOn {
		if dep == 1 {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("RAW: Step 2 should depend on Step 1, got DependsOn: %v", result[1].DependsOn)
	}
}

func TestInferDependencies_WAW(t *testing.T) {
	// WAW: Write After Write - Both steps write to the same file
	da := NewDependencyAnalyzer(nil)

	// Setup: Step 1 writes to config.go
	da.fileWriters["config.go"] = []int{1}

	steps := []PlanStep{
		{ID: 1, WriteFiles: []string{"config.go"}},
		{ID: 2, WriteFiles: []string{"config.go"}},
	}

	result := da.InferDependencies(steps)

	// Step 2 should depend on Step 1
	found := false
	for _, dep := range result[1].DependsOn {
		if dep == 1 {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("WAW: Step 2 should depend on Step 1, got DependsOn: %v", result[1].DependsOn)
	}
}

func TestInferDependencies_WAR(t *testing.T) {
	// WAR: Write After Read - Step 2 writes what Step 1 reads
	da := NewDependencyAnalyzer(nil)

	// Setup: Step 1 reads config.go
	da.fileReaders["config.go"] = []int{1}

	steps := []PlanStep{
		{ID: 1, ReadFiles: []string{"config.go"}},
		{ID: 2, WriteFiles: []string{"config.go"}},
	}

	result := da.InferDependencies(steps)

	// Step 2 should depend on Step 1
	found := false
	for _, dep := range result[1].DependsOn {
		if dep == 1 {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("WAR: Step 2 should depend on Step 1, got DependsOn: %v", result[1].DependsOn)
	}
}

func TestDetectConflicts_WriteWrite(t *testing.T) {
	da := NewDependencyAnalyzer(nil)

	steps := []PlanStep{
		{ID: 1, WriteFiles: []string{"config.go"}},
		{ID: 2, WriteFiles: []string{"config.go"}},
	}

	conflicts := da.DetectConflicts([]int{1, 2}, steps)

	if len(conflicts) == 0 {
		t.Fatal("Expected write-write conflict, got none")
	}

	found := false
	for _, c := range conflicts {
		if c.ConflictType == "write-write" {
			found = true
			if len(c.Files) != 1 || c.Files[0] != "config.go" {
				t.Errorf("Expected conflict on config.go, got: %v", c.Files)
			}
			break
		}
	}
	if !found {
		t.Error("Expected write-write conflict type")
	}
}

func TestDetectConflicts_ReadWrite(t *testing.T) {
	da := NewDependencyAnalyzer(nil)

	steps := []PlanStep{
		{ID: 1, ReadFiles: []string{"config.go"}},
		{ID: 2, WriteFiles: []string{"config.go"}},
	}

	conflicts := da.DetectConflicts([]int{1, 2}, steps)

	if len(conflicts) == 0 {
		t.Fatal("Expected read-write conflict, got none")
	}

	found := false
	for _, c := range conflicts {
		if c.ConflictType == "read-write" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected read-write conflict type")
	}
}

func TestDetectConflicts_NoConflict(t *testing.T) {
	da := NewDependencyAnalyzer(nil)

	// Different files - no conflict
	steps := []PlanStep{
		{ID: 1, WriteFiles: []string{"file1.go"}},
		{ID: 2, WriteFiles: []string{"file2.go"}},
	}

	conflicts := da.DetectConflicts([]int{1, 2}, steps)

	if len(conflicts) != 0 {
		t.Errorf("Expected no conflicts for different files, got: %v", conflicts)
	}
}

func TestEnhanceWithLSP_NilClient(t *testing.T) {
	da := NewDependencyAnalyzer(nil) // nil LSP client

	steps := []PlanStep{
		{ID: 1, WriteFiles: []string{"config.go"}},
		{ID: 2, ReadFiles: []string{"main.go"}},
	}

	// Should return steps unchanged when LSP client is nil
	result := da.EnhanceWithLSP(steps)

	if len(result) != len(steps) {
		t.Errorf("EnhanceWithLSP should return same number of steps")
	}
}

func TestAnalyze(t *testing.T) {
	da := NewDependencyAnalyzer(nil)

	steps := []PlanStep{
		{ID: 1, Description: `Write to "config.go"`, Tools: []string{"write_file"}},
		{ID: 2, Description: `Read "config.go" and modify`, Tools: []string{"read_file"}},
	}

	result := da.Analyze(steps)

	if result == nil {
		t.Fatal("Analyze returned nil")
	}
	if len(result.Steps) != 2 {
		t.Errorf("Expected 2 steps, got %d", len(result.Steps))
	}
}

func TestFormatConflictWarning(t *testing.T) {
	conflict := Conflict{
		StepIDs:      []int{1, 2},
		ConflictType: "write-write",
		Files:        []string{"config.go"},
		Message:      "Multiple steps write to the same file",
	}

	warning := FormatConflictWarning(conflict)

	if warning == "" {
		t.Error("FormatConflictWarning returned empty string")
	}
	if !contains(warning, "config.go") {
		t.Errorf("Warning should contain file name: %s", warning)
	}
}

func TestContainsInt(t *testing.T) {
	tests := []struct {
		slice    []int
		val      int
		expected bool
	}{
		{[]int{1, 2, 3}, 2, true},
		{[]int{1, 2, 3}, 4, false},
		{[]int{}, 1, false},
		{nil, 1, false},
	}

	for _, tt := range tests {
		result := containsInt(tt.slice, tt.val)
		if result != tt.expected {
			t.Errorf("containsInt(%v, %d) = %v, want %v", tt.slice, tt.val, result, tt.expected)
		}
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsStr(s, substr))
}

func containsStr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
