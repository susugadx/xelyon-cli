package uisummary

import (
	"strings"
	"testing"
)

func TestNewTaskSummary(t *testing.T) {
	ts := NewTaskSummary()
	if ts == nil {
		t.Fatal("NewTaskSummary() should not return nil")
	}
	if len(ts.Changes) != 0 {
		t.Error("Changes should be empty initially")
	}
}

func TestTaskSummary_AddChange(t *testing.T) {
	ts := NewTaskSummary().
		AddChange("file1.go", "created", 100, 0).
		AddChange("file2.go", "modified", 20, 10)

	if len(ts.Changes) != 2 {
		t.Errorf("Expected 2 changes, got %d", len(ts.Changes))
	}
	if ts.Changes[0].FilePath != "file1.go" {
		t.Errorf("Expected 'file1.go', got %q", ts.Changes[0].FilePath)
	}
	if ts.Changes[1].Action != "modified" {
		t.Errorf("Expected 'modified', got %q", ts.Changes[1].Action)
	}
}

func TestTaskSummary_SetTestResult(t *testing.T) {
	ts := NewTaskSummary().SetTestResult(true)
	if ts.TestsPassed == nil {
		t.Error("TestsPassed should not be nil after SetTestResult")
	}
	if !*ts.TestsPassed {
		t.Error("TestsPassed should be true")
	}

	ts2 := NewTaskSummary().SetTestResult(false)
	if *ts2.TestsPassed {
		t.Error("TestsPassed should be false")
	}
}

func TestTaskSummary_SetPlannedVerification(t *testing.T) {
	ts := NewTaskSummary().SetPlannedVerification([]string{
		" go test ./internal/agent ",
		"",
		"go test ./internal/agent",
		"make ci-check",
	})

	want := []string{"go test ./internal/agent", "make ci-check"}
	if len(ts.PlannedVerification) != len(want) {
		t.Fatalf("PlannedVerification = %#v, want %#v", ts.PlannedVerification, want)
	}
	for i := range want {
		if ts.PlannedVerification[i] != want[i] {
			t.Fatalf("PlannedVerification = %#v, want %#v", ts.PlannedVerification, want)
		}
	}
}

func TestTaskSummary_Calculate(t *testing.T) {
	ts := NewTaskSummary().
		AddChange("file1.go", "created", 100, 0).
		AddChange("file2.go", "modified", 20, 10).
		AddChange("file3.go", "deleted", 0, 50).
		Calculate()

	if ts.Stats.FilesChanged != 3 {
		t.Errorf("Expected 3 files, got %d", ts.Stats.FilesChanged)
	}
	if ts.Stats.TotalLinesAdded != 120 {
		t.Errorf("Expected 120 lines added, got %d", ts.Stats.TotalLinesAdded)
	}
	if ts.Stats.TotalLinesRemoved != 60 {
		t.Errorf("Expected 60 lines removed, got %d", ts.Stats.TotalLinesRemoved)
	}
}

func TestTaskSummary_Calculate_Dedup(t *testing.T) {
	ts := NewTaskSummary().
		AddChange("file1.go", "modified", 1, 0).
		AddChange("file1.go", "modified", 2, 3).
		AddChange("file2.go", "created", 10, 0).
		AddChange("file2.go", "deleted", 0, 10).
		Calculate()

	if ts.Stats.FilesChanged != 2 {
		t.Errorf("Expected 2 unique files, got %d", ts.Stats.FilesChanged)
	}
	if ts.Stats.TotalLinesAdded != 13 {
		t.Errorf("Expected 13 lines added, got %d", ts.Stats.TotalLinesAdded)
	}
	if ts.Stats.TotalLinesRemoved != 13 {
		t.Errorf("Expected 13 lines removed, got %d", ts.Stats.TotalLinesRemoved)
	}

	// file2 は created+deleted の場合、action は deleted が優先される
	for _, c := range ts.Changes {
		if c.FilePath == "file2.go" && c.Action != "deleted" {
			t.Errorf("Expected file2.go action 'deleted', got %q", c.Action)
		}
	}
}

func TestTaskSummary_Render_Empty(t *testing.T) {
	ts := NewTaskSummary()
	result := ts.Render()
	if result != "" {
		t.Error("Empty summary should render to empty string")
	}
}

func TestTaskSummary_Render(t *testing.T) {
	ts := NewTaskSummary().
		AddChange("internal/agent/agent_chat.go", "modified", 5, 3).
		AddChange("internal/uisummary/task_summary.go", "created", 120, 0).
		SetTestResult(true).
		SetTestCommand("make ci-check").
		SetPlannedVerification([]string{"go test ./internal/uisummary"})

	result := ts.Render()

	// ヘッダーチェック
	if !strings.Contains(result, "Task Completed") {
		t.Error("Missing 'Task Completed' header")
	}

	// ファイル名チェック
	if !strings.Contains(result, "agent_chat.go") {
		t.Error("Missing file name 'agent_chat.go'")
	}
	if !strings.Contains(result, "task_summary.go") {
		t.Error("Missing file name 'task_summary.go'")
	}

	// アクションチェック
	if !strings.Contains(result, "modified") {
		t.Error("Missing action 'modified'")
	}
	if !strings.Contains(result, "created") {
		t.Error("Missing action 'created'")
	}

	// テスト結果チェック
	if !strings.Contains(result, "Test result (make ci-check): passed") {
		t.Error("Missing test result")
	}
	if !strings.Contains(result, "Planned verification") || !strings.Contains(result, "go test ./internal/uisummary") {
		t.Error("Missing planned verification")
	}

	// サマリーチェック
	if !strings.Contains(result, "2 file(s)") {
		t.Error("Missing file count in summary")
	}
	if !strings.Contains(result, "+125") {
		t.Error("Missing lines added in summary")
	}
}

func TestTaskSummary_Render_NoTests(t *testing.T) {
	ts := NewTaskSummary().
		AddChange("file.go", "modified", 10, 5)

	result := ts.Render()

	// テスト結果が含まれていないこと
	if strings.Contains(result, "tests passed") || strings.Contains(result, "Tests failed") {
		t.Error("Should not show test result when TestsPassed is nil")
	}
}

func TestTaskSummary_Render_TestsFailed(t *testing.T) {
	ts := NewTaskSummary().
		AddChange("file.go", "modified", 10, 5).
		SetTestResult(false)

	result := ts.Render()

	if !strings.Contains(result, "Test result: failed") {
		t.Error("Missing failed test result message")
	}
}

func TestFormatLines(t *testing.T) {
	tests := []struct {
		added    int
		removed  int
		expected string
	}{
		{0, 0, "-"},
		{10, 0, "+10"},
		{0, 5, "-5"},
		{10, 5, "+10 -5"},
		{100, 50, "+100 -50"},
	}

	for _, tt := range tests {
		result := formatLines(tt.added, tt.removed)
		if result != tt.expected {
			t.Errorf("formatLines(%d, %d) = %q, want %q", tt.added, tt.removed, result, tt.expected)
		}
	}
}

func TestInferAction(t *testing.T) {
	tests := []struct {
		tool     string
		expected string
	}{
		{"write_file", "created"},
		{"str_replace", "modified"},
		{"delete_file", "deleted"},
		{"unknown_tool", "modified"},
	}

	for _, tt := range tests {
		result := InferAction(tt.tool)
		if result != tt.expected {
			t.Errorf("InferAction(%q) = %q, want %q", tt.tool, result, tt.expected)
		}
	}
}
