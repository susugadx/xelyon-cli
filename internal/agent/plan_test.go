package agent

import (
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/agent/plan"
)

func TestParsePlan(t *testing.T) {
	jsonStr := `{
		"steps": [
			{
				"id": 1,
				"description": "Read main.go",
				"tools": ["read_file"],
				"depends_on": []
			},
			{
				"id": 2,
				"description": "Run tests",
				"tools": ["bash"],
				"depends_on": [1]
			}
		]
	}`

	p, err := plan.ParsePlan(jsonStr)
	if err != nil {
		t.Fatalf("Failed to parse plan: %v", err)
	}

	if len(p.Steps) != 2 {
		t.Errorf("Expected 2 steps, got %d", len(p.Steps))
	}

	step1 := p.Steps[0]
	if step1.ID != 1 {
		t.Errorf("Expected step ID 1, got %d", step1.ID)
	}
	if step1.Description != "Read main.go" {
		t.Errorf("Expected description 'Read main.go', got '%s'", step1.Description)
	}
	if step1.Status != "pending" {
		t.Errorf("Expected status 'pending', got '%s'", step1.Status)
	}
	if len(step1.Tools) != 1 || step1.Tools[0] != "read_file" {
		t.Errorf("Expected tools ['read_file'], got %v", step1.Tools)
	}

	step2 := p.Steps[1]
	if len(step2.DependsOn) != 1 || step2.DependsOn[0] != 1 {
		t.Errorf("Expected depends_on [1], got %v", step2.DependsOn)
	}
}

func TestExtractPlanJSON_WithCodeBlock(t *testing.T) {
	response := `Here is the plan:

` + "```json" + `
{
  "steps": [
    {
      "id": 1,
      "description": "Test step",
      "tools": ["bash"],
      "depends_on": []
    }
  ]
}
` + "```" + `

This is the execution plan.`

	jsonStr := plan.ExtractPlanJSON(response)
	if jsonStr == "" {
		t.Fatal("Failed to extract JSON: got empty string")
	}

	if !strings.Contains(jsonStr, `"steps"`) {
		t.Errorf("Extracted JSON does not contain 'steps': %s", jsonStr)
	}
	if !strings.Contains(jsonStr, `"Test step"`) {
		t.Errorf("Extracted JSON does not contain 'Test step': %s", jsonStr)
	}
}

func TestExtractPlanJSON_WithNewlines(t *testing.T) {
	response := `I will create a plan:

{
  "steps": [
    {
      "id": 1,
      "description": "Step with newlines",
      "tools": ["read_file"],
      "depends_on": []
    }
  ]
}

Done!`

	jsonStr := plan.ExtractPlanJSON(response)
	if jsonStr == "" {
		t.Fatal("Failed to extract JSON with newlines: got empty string")
	}

	if !strings.Contains(jsonStr, `"steps"`) {
		t.Errorf("Extracted JSON does not contain 'steps': %s", jsonStr)
	}
}

func TestExtractPlanJSON_NoJSON(t *testing.T) {
	response := "This response contains no JSON at all."

	jsonStr := plan.ExtractPlanJSON(response)
	if jsonStr != "" {
		t.Errorf("Expected empty string for response without JSON, but got: %s", jsonStr)
	}
}

func TestPlan_CanExecute(t *testing.T) {
	p := &plan.Plan{
		Steps: []plan.PlanStep{
			{ID: 1, Status: "pending", DependsOn: []int{}},
			{ID: 2, Status: "pending", DependsOn: []int{1}},
			{ID: 3, Status: "pending", DependsOn: []int{1}},
		},
	}

	// Step 1 は依存なしで実行可能
	if !p.CanExecute(1) {
		t.Error("Expected step 1 to be executable")
	}

	// Step 2 は Step 1 が pending なので実行不可
	if p.CanExecute(2) {
		t.Error("Expected step 2 to be not executable (depends on step 1)")
	}

	// Step 1 を完了にする
	p.UpdateStatus(1, "completed", "Success")

	// Step 2, 3 が実行可能になる
	if !p.CanExecute(2) {
		t.Error("Expected step 2 to be executable after step 1 completed")
	}
	if !p.CanExecute(3) {
		t.Error("Expected step 3 to be executable after step 1 completed")
	}
}

func TestPlan_GetParallelSteps(t *testing.T) {
	p := &plan.Plan{
		Steps: []plan.PlanStep{
			{ID: 1, Status: "completed", DependsOn: []int{}},
			{ID: 2, Status: "pending", DependsOn: []int{1}},
			{ID: 3, Status: "pending", DependsOn: []int{1}},
			{ID: 4, Status: "pending", DependsOn: []int{2, 3}},
		},
	}

	parallelSteps := p.GetParallelSteps()
	if len(parallelSteps) != 2 {
		t.Errorf("Expected 2 parallel steps, got %d", len(parallelSteps))
	}

	// Step 2 と 3 は同じ depends_on を持つので並列実行可能
	if !containsInt(parallelSteps, 2) || !containsInt(parallelSteps, 3) {
		t.Errorf("Expected steps 2 and 3 to be parallel, got %v", parallelSteps)
	}
}

func TestPlan_GetParallelSteps_DifferentDeps(t *testing.T) {
	// 異なる depends_on を持つステップは並列実行されない
	p := &plan.Plan{
		Steps: []plan.PlanStep{
			{ID: 1, Status: "completed", DependsOn: []int{}},
			{ID: 2, Status: "completed", DependsOn: []int{}},
			{ID: 3, Status: "pending", DependsOn: []int{1}},
			{ID: 4, Status: "pending", DependsOn: []int{2}},
		},
	}

	parallelSteps := p.GetParallelSteps()
	// Step 3 と 4 は異なる depends_on を持つので nil
	if parallelSteps != nil {
		t.Errorf("Expected nil for steps with different depends_on, got %v", parallelSteps)
	}
}

func TestPlan_GetNextStep(t *testing.T) {
	p := &plan.Plan{
		Steps: []plan.PlanStep{
			{ID: 1, Status: "completed", DependsOn: []int{}},
			{ID: 2, Status: "pending", DependsOn: []int{1}},
			{ID: 3, Status: "pending", DependsOn: []int{2}},
		},
	}

	nextStep := p.GetNextStep()
	if nextStep != 2 {
		t.Errorf("Expected next step to be 2, got %d", nextStep)
	}

	p.UpdateStatus(2, "completed", "Success")
	nextStep = p.GetNextStep()
	if nextStep != 3 {
		t.Errorf("Expected next step to be 3, got %d", nextStep)
	}

	p.UpdateStatus(3, "completed", "Success")
	nextStep = p.GetNextStep()
	if nextStep != -1 {
		t.Errorf("Expected next step to be -1 (all completed), got %d", nextStep)
	}
}

func TestPlan_IsCompleted(t *testing.T) {
	p := &plan.Plan{
		Steps: []plan.PlanStep{
			{ID: 1, Status: "completed"},
			{ID: 2, Status: "pending"},
		},
	}

	if p.IsCompleted() {
		t.Error("Expected plan to be not completed")
	}

	p.UpdateStatus(2, "completed", "Success")
	if !p.IsCompleted() {
		t.Error("Expected plan to be completed")
	}
}

func TestPlan_HasFailed(t *testing.T) {
	p := &plan.Plan{
		Steps: []plan.PlanStep{
			{ID: 1, Status: "completed"},
			{ID: 2, Status: "pending"},
		},
	}

	if p.HasFailed() {
		t.Error("Expected plan to have no failures")
	}

	p.UpdateStatus(2, "failed", "Error occurred")
	if !p.HasFailed() {
		t.Error("Expected plan to have failed")
	}
}

func TestFormatPlan(t *testing.T) {
	p := &plan.Plan{
		Steps: []plan.PlanStep{
			{
				ID:          1,
				Description: "Read file",
				Tools:       []string{"read_file"},
				DependsOn:   []int{},
			},
			{
				ID:          2,
				Description: "Write file",
				Tools:       []string{"write_file"},
				DependsOn:   []int{1},
			},
		},
	}

	formatted := plan.FormatPlan(p)
	if !strings.Contains(formatted, "Plan:") {
		t.Error("Expected formatted plan to contain header")
	}
	if !strings.Contains(formatted, "Read file") {
		t.Error("Expected formatted plan to contain step description")
	}
	if !strings.Contains(formatted, "Depends on:") {
		t.Error("Expected formatted plan to contain depends_on info")
	}
}

// Helper function
func containsInt(slice []int, val int) bool {
	for _, v := range slice {
		if v == val {
			return true
		}
	}
	return false
}

// containsFailure tests

func TestContainsFailure_GoTestFail(t *testing.T) {
	tests := []struct {
		name   string
		result string
		want   bool
	}{
		{
			name:   "Go test FAIL pattern",
			result: "--- FAIL: TestSomething (0.00s)\n    main_test.go:10: expected true, got false",
			want:   true,
		},
		{
			name:   "Go test FAIL tab pattern",
			result: "FAIL\tgithub.com/example/pkg\t0.010s",
			want:   true,
		},
		{
			name:   "Go test pass",
			result: "ok  \tgithub.com/example/pkg\t0.010s",
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			failed, _ := plan.ContainsFailure(tt.result)
			if failed != tt.want {
				t.Errorf("ContainsFailure() = %v, want %v", failed, tt.want)
			}
		})
	}
}

func TestContainsFailure_Panic(t *testing.T) {
	result := `goroutine 1 [running]:
panic: runtime error: index out of range [1] with length 1

goroutine 1 [running]:
main.main()
	/home/user/project/main.go:10 +0x45`

	failed, reason := plan.ContainsFailure(result)
	if !failed {
		t.Error("ContainsFailure() should detect panic")
	}
	if reason != "Panic detected" {
		t.Errorf("ContainsFailure() reason = %q, want 'Panic detected'", reason)
	}
}

func TestContainsFailure_NpmError(t *testing.T) {
	result := `npm ERR! code ENOENT
npm ERR! syscall open
npm ERR! path /home/user/project/package.json
npm ERR! errno -2
npm ERR! enoent ENOENT: no such file or directory`

	failed, reason := plan.ContainsFailure(result)
	if !failed {
		t.Error("ContainsFailure() should detect npm error")
	}
	if reason != "npm error" {
		t.Errorf("ContainsFailure() reason = %q, want 'npm error'", reason)
	}
}

func TestContainsFailure_ExitStatus(t *testing.T) {
	tests := []struct {
		name   string
		result string
		want   bool
	}{
		{
			name:   "exit status 1",
			result: "Command failed with exit status 1",
			want:   true,
		},
		{
			name:   "exit status 0 (success)",
			result: "Command completed with exit status 0",
			want:   false, // exit status 0 is success, should not be detected as failure
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			failed, _ := plan.ContainsFailure(tt.result)
			if failed != tt.want {
				t.Errorf("ContainsFailure() = %v, want %v", failed, tt.want)
			}
		})
	}
}

func TestContainsFailure_NoFailure(t *testing.T) {
	tests := []struct {
		name   string
		result string
	}{
		{
			name:   "success output",
			result: "Build successful\nAll tests passed",
		},
		{
			name:   "empty output",
			result: "",
		},
		{
			name:   "normal command output",
			result: "total 16\ndrwxr-xr-x 5 user user 4096 Jan 15 10:00 .",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			failed, _ := plan.ContainsFailure(tt.result)
			if failed {
				t.Errorf("ContainsFailure() should not detect failure for %q", tt.name)
			}
		})
	}
}

func TestContainsFailure_BuildAndCompileErrors(t *testing.T) {
	tests := []struct {
		name   string
		result string
		want   bool
	}{
		{
			name:   "build failed",
			result: "go build: build failed: exit status 1",
			want:   true,
		},
		{
			name:   "compile error",
			result: "compile error: undefined: someFunction",
			want:   true,
		},
		{
			name:   "SyntaxError with colon",
			result: "SyntaxError: Unexpected token",
			want:   true,
		},
		{
			name:   "TypeError with colon",
			result: "TypeError: Cannot read property 'x' of undefined",
			want:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			failed, _ := plan.ContainsFailure(tt.result)
			if failed != tt.want {
				t.Errorf("ContainsFailure() = %v, want %v", failed, tt.want)
			}
		})
	}
}

// TestContainsFailure_FalsePositives は誤検知を起こさないことをテスト
// コード検索結果やログ出力に "Error" 文字列が含まれていても失敗と判定しない
func TestContainsFailure_FalsePositives(t *testing.T) {
	tests := []struct {
		name   string
		result string
		want   bool
	}{
		{
			name: "t.Errorf in test code",
			result: `internal/agent/plan_test.go:397:	t.Errorf("ContainsFailure() = %v, want %v", failed, tt.want)
internal/agent/plan_test.go:485:	t.Errorf("ContainsFailure() should not detect failure for %q", tt.name)`,
			want: false, // コード検索結果は失敗ではない
		},
		{
			name:   "ErrorHandler function name",
			result: "func ErrorHandler(err error) {\n    log.Printf(\"Error: %v\", err)\n}",
			want:   false, // 関数定義は失敗ではない
		},
		{
			name:   "fmt.Errorf in code",
			result: `return fmt.Errorf("failed to parse: %w", err)`,
			want:   false, // コード内のfmt.Errorfは失敗ではない
		},
		{
			name:   "log with error message",
			result: "2024-01-15 10:00:00 INFO: Processing completed\n2024-01-15 10:00:01 DEBUG: Error count: 0",
			want:   false, // ログ出力（エラーカウント0）は失敗ではない
		},
		{
			name:   "grep result with Error string",
			result: "grep result:\ninternal/api/client.go:50: type ErrorResponse struct {\ninternal/api/client.go:51:     Error string `json:\"error\"`",
			want:   false, // コード検索結果は失敗ではない
		},
		{
			name:   "markdown documentation",
			result: "## Error Handling\nThis section describes how errors are handled.\n\n### ErrorTypes\n- ValidationError\n- NetworkError",
			want:   false, // ドキュメントは失敗ではない
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			failed, reason := plan.ContainsFailure(tt.result)
			if failed != tt.want {
				t.Errorf("ContainsFailure() = %v (reason: %s), want %v", failed, reason, tt.want)
			}
		})
	}
}
