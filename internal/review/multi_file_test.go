package review

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMultiFileApplier_ValidateMultiFileChange(t *testing.T) {
	tmpDir := t.TempDir()
	applier := &MultiFileApplier{BackupDir: filepath.Join(tmpDir, "backups")}

	// Create test file
	testFile := filepath.Join(tmpDir, "test.go")
	content := `package main

func main() {
	client := &http.Client{}
}
`
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	tests := []struct {
		name    string
		change  *MultiFileChange
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid code-based change",
			change: &MultiFileChange{
				ID:          "test1",
				Description: "Test change",
				Changes: []FileChange{
					{
						FilePath: testFile,
						Patches: []Patch{
							{OldCode: "http.Client{}", NewCode: "http.Client{Timeout: 30 * time.Second}"},
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "empty changes",
			change: &MultiFileChange{
				ID:      "test2",
				Changes: []FileChange{},
			},
			wantErr: true,
			errMsg:  "no changes",
		},
		{
			name: "duplicate file paths",
			change: &MultiFileChange{
				ID: "test3",
				Changes: []FileChange{
					{FilePath: testFile, Patches: []Patch{{OldCode: "http.Client{}", NewCode: "new"}}},
					{FilePath: testFile, Patches: []Patch{{OldCode: "main", NewCode: "new"}}},
				},
			},
			wantErr: true,
			errMsg:  "duplicate",
		},
		{
			name: "file not found",
			change: &MultiFileChange{
				ID: "test4",
				Changes: []FileChange{
					{FilePath: filepath.Join(tmpDir, "nonexistent.go"), Patches: []Patch{{OldCode: "old", NewCode: "new"}}},
				},
			},
			wantErr: true,
			errMsg:  "not found",
		},
		{
			name: "OldCode not found",
			change: &MultiFileChange{
				ID: "test5",
				Changes: []FileChange{
					{FilePath: testFile, Patches: []Patch{{OldCode: "nonexistent code", NewCode: "new"}}},
				},
			},
			wantErr: true,
			errMsg:  "not found",
		},
		{
			name: "empty OldCode without line range",
			change: &MultiFileChange{
				ID: "test6",
				Changes: []FileChange{
					{FilePath: testFile, Patches: []Patch{{OldCode: "", NewCode: "new"}}},
				},
			},
			wantErr: true,
			errMsg:  "OldCode is empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := applier.ValidateMultiFileChange(tt.change)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error but got nil")
				} else if tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("error = %q, want to contain %q", err.Error(), tt.errMsg)
				}
			} else if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestMultiFileApplier_ApplyMultiFileChange(t *testing.T) {
	tmpDir := t.TempDir()
	applier := &MultiFileApplier{BackupDir: filepath.Join(tmpDir, "backups")}

	t.Run("successful apply", func(t *testing.T) {
		// Create test file
		testFile := filepath.Join(tmpDir, "apply_test.go")
		content := `package main

func main() {
	client := &http.Client{}
}
`
		if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
			t.Fatalf("failed to create test file: %v", err)
		}

		change := &MultiFileChange{
			ID:          "apply1",
			Description: "Add timeout",
			Changes: []FileChange{
				{
					FilePath: testFile,
					Patches: []Patch{
						{OldCode: "http.Client{}", NewCode: "http.Client{Timeout: 30 * time.Second}"},
					},
				},
			},
		}

		if err := applier.ApplyMultiFileChange(change); err != nil {
			t.Fatalf("ApplyMultiFileChange() error = %v", err)
		}

		// Verify change was applied
		newContent, err := os.ReadFile(testFile)
		if err != nil {
			t.Fatalf("failed to read file: %v", err)
		}

		if !strings.Contains(string(newContent), "Timeout: 30 * time.Second") {
			t.Error("change was not applied")
		}

		// Verify backup was created
		if change.Changes[0].BackupPath == "" {
			t.Error("backup path not set")
		}

		if _, err := os.Stat(change.Changes[0].BackupPath); os.IsNotExist(err) {
			t.Error("backup file does not exist")
		}
	})

	t.Run("apply multiple files", func(t *testing.T) {
		file1 := filepath.Join(tmpDir, "multi1.go")
		file2 := filepath.Join(tmpDir, "multi2.go")

		if err := os.WriteFile(file1, []byte("package a\nfunc F1() {}"), 0644); err != nil {
			t.Fatalf("failed to create file1: %v", err)
		}
		if err := os.WriteFile(file2, []byte("package b\nfunc F2() {}"), 0644); err != nil {
			t.Fatalf("failed to create file2: %v", err)
		}

		change := &MultiFileChange{
			ID: "multi",
			Changes: []FileChange{
				{FilePath: file1, Patches: []Patch{{OldCode: "F1", NewCode: "Func1"}}},
				{FilePath: file2, Patches: []Patch{{OldCode: "F2", NewCode: "Func2"}}},
			},
		}

		if err := applier.ApplyMultiFileChange(change); err != nil {
			t.Fatalf("error = %v", err)
		}

		c1, _ := os.ReadFile(file1)
		c2, _ := os.ReadFile(file2)

		if !strings.Contains(string(c1), "Func1") {
			t.Error("file1 not updated")
		}
		if !strings.Contains(string(c2), "Func2") {
			t.Error("file2 not updated")
		}
	})
}

func TestMultiFileApplier_Rollback(t *testing.T) {
	tmpDir := t.TempDir()
	applier := &MultiFileApplier{BackupDir: filepath.Join(tmpDir, "backups")}

	// Create test file
	testFile := filepath.Join(tmpDir, "rollback_test.go")
	originalContent := `package main

func original() {}
`
	if err := os.WriteFile(testFile, []byte(originalContent), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	change := &MultiFileChange{
		ID: "rollback1",
		Changes: []FileChange{
			{
				FilePath: testFile,
				Patches:  []Patch{{OldCode: "original", NewCode: "modified"}},
			},
		},
	}

	// Apply change
	if err := applier.ApplyMultiFileChange(change); err != nil {
		t.Fatalf("apply error: %v", err)
	}

	// Verify change was applied
	afterApply, _ := os.ReadFile(testFile)
	if !strings.Contains(string(afterApply), "modified") {
		t.Fatal("change was not applied")
	}

	// Rollback
	if err := applier.RollbackMultiFileChange(change); err != nil {
		t.Fatalf("rollback error: %v", err)
	}

	// Verify rollback
	afterRollback, _ := os.ReadFile(testFile)
	if !strings.Contains(string(afterRollback), "original") {
		t.Error("rollback failed")
	}
}

func TestMultiFileApplier_LinePatch(t *testing.T) {
	tmpDir := t.TempDir()
	applier := &MultiFileApplier{BackupDir: filepath.Join(tmpDir, "backups")}

	testFile := filepath.Join(tmpDir, "line_test.go")
	content := `line1
line2
line3
line4
line5
`
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	change := &MultiFileChange{
		ID: "line1",
		Changes: []FileChange{
			{
				FilePath: testFile,
				Patches: []Patch{
					{StartLine: 2, EndLine: 3, NewCode: "replaced2\nreplaced3"},
				},
			},
		},
	}

	if err := applier.ApplyMultiFileChange(change); err != nil {
		t.Fatalf("error = %v", err)
	}

	result, _ := os.ReadFile(testFile)
	lines := strings.Split(string(result), "\n")

	if lines[0] != "line1" {
		t.Errorf("line 1 = %q, want %q", lines[0], "line1")
	}
	if lines[1] != "replaced2" {
		t.Errorf("line 2 = %q, want %q", lines[1], "replaced2")
	}
	if lines[2] != "replaced3" {
		t.Errorf("line 3 = %q, want %q", lines[2], "replaced3")
	}
	if lines[3] != "line4" {
		t.Errorf("line 4 = %q, want %q", lines[3], "line4")
	}
}

func TestApplyLinePatch(t *testing.T) {
	tests := []struct {
		name  string
		lines []string
		patch Patch
		want  []string
	}{
		{
			name:  "replace middle lines",
			lines: []string{"a", "b", "c", "d"},
			patch: Patch{StartLine: 2, EndLine: 3, NewCode: "X\nY"},
			want:  []string{"a", "X", "Y", "d"},
		},
		{
			name:  "replace first line",
			lines: []string{"a", "b", "c"},
			patch: Patch{StartLine: 1, EndLine: 1, NewCode: "X"},
			want:  []string{"X", "b", "c"},
		},
		{
			name:  "replace last line",
			lines: []string{"a", "b", "c"},
			patch: Patch{StartLine: 3, EndLine: 3, NewCode: "X"},
			want:  []string{"a", "b", "X"},
		},
		{
			name:  "replace with more lines",
			lines: []string{"a", "b"},
			patch: Patch{StartLine: 1, EndLine: 1, NewCode: "X\nY\nZ"},
			want:  []string{"X", "Y", "Z", "b"},
		},
		{
			name:  "replace with fewer lines",
			lines: []string{"a", "b", "c", "d"},
			patch: Patch{StartLine: 2, EndLine: 3, NewCode: "X"},
			want:  []string{"a", "X", "d"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := applyLinePatch(tt.lines, tt.patch)
			if len(got) != len(tt.want) {
				t.Errorf("len = %d, want %d", len(got), len(tt.want))
			}
			for i := range tt.want {
				if i < len(got) && got[i] != tt.want[i] {
					t.Errorf("line[%d] = %q, want %q", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestConvertFixProposalsToMultiFileChange(t *testing.T) {
	proposals := []FixProposal{
		{Actionable: true, FilePath: "/tmp/a.go", OldCode: "old1", NewCode: "new1"},
		{Actionable: true, FilePath: "/tmp/a.go", OldCode: "old2", NewCode: "new2"},
		{Actionable: true, FilePath: "/tmp/b.go", OldCode: "old3", NewCode: "new3"},
		{Actionable: false, FilePath: "/tmp/c.go"}, // non-actionable, should be skipped
	}

	change := ConvertFixProposalsToMultiFileChange(proposals, "Test conversion")

	if change.Description != "Test conversion" {
		t.Errorf("Description = %q, want %q", change.Description, "Test conversion")
	}

	if len(change.Changes) != 2 {
		t.Errorf("len(Changes) = %d, want 2", len(change.Changes))
	}

	// Find /tmp/a.go change
	var aChange *FileChange
	for i := range change.Changes {
		if change.Changes[i].FilePath == "/tmp/a.go" {
			aChange = &change.Changes[i]
			break
		}
	}

	if aChange == nil {
		t.Fatal("change for /tmp/a.go not found")
	}

	if len(aChange.Patches) != 2 {
		t.Errorf("len(Patches) for a.go = %d, want 2", len(aChange.Patches))
	}
}

func TestMultiFileApplier_CleanupBackups(t *testing.T) {
	tmpDir := t.TempDir()
	applier := &MultiFileApplier{BackupDir: filepath.Join(tmpDir, "backups")}

	// Create test file
	testFile := filepath.Join(tmpDir, "cleanup_test.go")
	if err := os.WriteFile(testFile, []byte("content"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	change := &MultiFileChange{
		ID: "cleanup1",
		Changes: []FileChange{
			{FilePath: testFile, Patches: []Patch{{OldCode: "content", NewCode: "new"}}},
		},
	}

	if err := applier.ApplyMultiFileChange(change); err != nil {
		t.Fatalf("failed to apply change: %v", err)
	}

	backupPath := change.Changes[0].BackupPath
	if _, err := os.Stat(backupPath); os.IsNotExist(err) {
		t.Fatal("backup not created")
	}

	applier.CleanupBackups(change)

	if _, err := os.Stat(backupPath); !os.IsNotExist(err) {
		t.Error("backup not cleaned up")
	}
}
