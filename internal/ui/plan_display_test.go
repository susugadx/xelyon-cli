package ui

import (
	"strings"
	"testing"
)

func TestNewPlanDisplay(t *testing.T) {
	p := NewPlanDisplay("Test Plan")
	if p.Title != "Test Plan" {
		t.Errorf("Expected title 'Test Plan', got %q", p.Title)
	}
	if p.Steps != nil {
		t.Error("Steps should be nil initially")
	}
}

func TestPlanDisplay_SetSummary(t *testing.T) {
	p := NewPlanDisplay("Test").SetSummary("This is a summary")
	if p.Summary != "This is a summary" {
		t.Errorf("Expected summary 'This is a summary', got %q", p.Summary)
	}
}

func TestPlanDisplay_AddStep(t *testing.T) {
	p := NewPlanDisplay("Test").
		AddStep(1, "First step", []string{"read_file"}, []string{"main.go"}).
		AddStep(2, "Second step", []string{"write_file"}, []string{"main.go"})

	if len(p.Steps) != 2 {
		t.Errorf("Expected 2 steps, got %d", len(p.Steps))
	}
	if p.Steps[0].Description != "First step" {
		t.Errorf("Expected 'First step', got %q", p.Steps[0].Description)
	}
	if p.Steps[1].ID != 2 {
		t.Errorf("Expected step ID 2, got %d", p.Steps[1].ID)
	}
}

func TestPlanDisplay_SetStepStatus(t *testing.T) {
	p := NewPlanDisplay("Test").
		AddStep(1, "Step 1", nil, nil).
		AddStep(2, "Step 2", nil, nil)

	p.SetStepStatus(1, "completed")
	p.SetStepStatus(2, "running")

	if p.Steps[0].Status != "completed" {
		t.Errorf("Expected status 'completed', got %q", p.Steps[0].Status)
	}
	if p.Steps[1].Status != "running" {
		t.Errorf("Expected status 'running', got %q", p.Steps[1].Status)
	}
}

func TestPlanDisplay_Render(t *testing.T) {
	p := NewPlanDisplay("Bracketed Paste Mode 対応").
		SetSummary("複数行ペーストを直接受け付けられるように実装").
		AddStep(1, "multiline.go を修正", []string{"str_replace"}, []string{"internal/ui/multiline.go"}).
		AddStep(2, "agent_repl.go を更新", []string{"str_replace"}, []string{"internal/agent/agent_repl.go"})

	result := p.Render()

	// タイトルが含まれているか
	if !strings.Contains(result, "Bracketed Paste Mode 対応") {
		t.Error("Missing title in output")
	}

	// 概要が含まれているか
	if !strings.Contains(result, "複数行ペースト") {
		t.Error("Missing summary in output")
	}

	// ステップが含まれているか
	if !strings.Contains(result, "multiline.go を修正") {
		t.Error("Missing step 1 in output")
	}

	// ツールが含まれているか
	if !strings.Contains(result, "str_replace") {
		t.Error("Missing tools in output")
	}

	// 罫線が含まれているか
	if !strings.Contains(result, "╌") {
		t.Error("Missing border in output")
	}
}

func TestPlanDisplay_RenderProgress(t *testing.T) {
	p := NewPlanDisplay("Test").
		AddStep(1, "Step 1", nil, nil).
		AddStep(2, "Step 2", nil, nil).
		AddStep(3, "Step 3", nil, nil)

	p.SetStepStatus(1, "completed")
	p.SetStepStatus(2, "running")

	result := p.RenderProgress()

	if !strings.Contains(result, "✅") {
		t.Error("Missing completed icon")
	}
	if !strings.Contains(result, "⏳") {
		t.Error("Missing running icon")
	}
	if !strings.Contains(result, "⬜") {
		t.Error("Missing pending icon")
	}
}

func TestPlanDisplay_RenderCompact(t *testing.T) {
	p := NewPlanDisplay("Quick Task").
		AddStep(1, "Do something", nil, nil)

	result := p.RenderCompact()

	if !strings.Contains(result, "Quick Task") {
		t.Error("Missing title")
	}
	if !strings.Contains(result, "Do something") {
		t.Error("Missing step")
	}
	// コンパクト形式は短い罫線
	if !strings.Contains(result, "─") {
		t.Error("Missing compact border")
	}
}

func TestGetStepStatusIcon(t *testing.T) {
	tests := []struct {
		status   string
		expected string
	}{
		{"completed", "✅"},
		{"running", "⏳"},
		{"failed", "❌"},
		{"skipped", "⏭️"},
		{"pending", "⬜"},
		{"unknown", "⬜"},
		{"", "⬜"},
	}

	for _, tt := range tests {
		result := getStepStatusIcon(tt.status)
		if result != tt.expected {
			t.Errorf("getStepStatusIcon(%q) = %q, expected %q", tt.status, result, tt.expected)
		}
	}
}

func TestInferActionFromDescription(t *testing.T) {
	tests := []struct {
		desc     string
		expected string
	}{
		{"Create new file", "新規作成"},
		{"Add new feature", "新規作成"},
		{"Delete old code", "削除"},
		{"Remove unused imports", "削除"},
		{"Update configuration", "更新"},
		{"Fix bug in handler", "更新"},
		{"Refactor the module", "リファクタリング"},
		{"Add test coverage", "テスト追加"},   // testが優先される
		{"Run tests", "テスト追加"},           // testが優先される
		{"新規ファイル作成", "新規作成"},
		{"設定を更新", "更新"},
		{"Some generic task", "変更"},
	}

	for _, tt := range tests {
		result := inferActionFromDescription(tt.desc)
		if result != tt.expected {
			t.Errorf("inferActionFromDescription(%q) = %q, expected %q", tt.desc, result, tt.expected)
		}
	}
}

func TestFormatPlanHeader(t *testing.T) {
	result := FormatPlanHeader("My Plan")

	if !strings.Contains(result, "📋 Plan: My Plan") {
		t.Error("Missing plan header")
	}
	if !strings.Contains(result, "╌") {
		t.Error("Missing border")
	}
}

func TestFormatStepProgress(t *testing.T) {
	tests := []struct {
		stepID int
		total  int
		desc   string
		status string
		want   string
	}{
		{1, 3, "First step", "completed", "✅ [1/3] First step"},
		{2, 3, "Second step", "running", "⏳ [2/3] Second step"},
		{3, 3, "Third step", "pending", "⬜ [3/3] Third step"},
	}

	for _, tt := range tests {
		result := FormatStepProgress(tt.stepID, tt.total, tt.desc, tt.status)
		if result != tt.want {
			t.Errorf("FormatStepProgress(%d, %d, %q, %q) = %q, want %q",
				tt.stepID, tt.total, tt.desc, tt.status, result, tt.want)
		}
	}
}

func TestPlanDisplay_EmptyPlan(t *testing.T) {
	p := NewPlanDisplay("")
	result := p.Render()

	// 空でも罫線は含まれる
	if !strings.Contains(result, "╌") {
		t.Error("Missing border even for empty plan")
	}
}

func TestPlanDisplay_FilesTable(t *testing.T) {
	p := NewPlanDisplay("Test").
		AddStep(1, "Create new handler", nil, []string{"handler.go"}).
		AddStep(2, "Update tests", nil, []string{"handler_test.go"})

	result := p.Render()

	// ファイルテーブルヘッダー
	if !strings.Contains(result, "修正ファイル") {
		t.Error("Missing files section header")
	}

	// ファイル名
	if !strings.Contains(result, "handler.go") {
		t.Error("Missing file handler.go")
	}
	if !strings.Contains(result, "handler_test.go") {
		t.Error("Missing file handler_test.go")
	}
}
