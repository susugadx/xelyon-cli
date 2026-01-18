package agent

import (
	"strings"
	"testing"
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

	plan, err := ParsePlan(jsonStr)
	if err != nil {
		t.Fatalf("Failed to parse plan: %v", err)
	}

	if len(plan.Steps) != 2 {
		t.Errorf("Expected 2 steps, got %d", len(plan.Steps))
	}

	step1 := plan.Steps[0]
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

	step2 := plan.Steps[1]
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

	jsonStr, err := ExtractPlanJSON(response)
	if err != nil {
		t.Fatalf("Failed to extract JSON: %v", err)
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

	jsonStr, err := ExtractPlanJSON(response)
	if err != nil {
		t.Fatalf("Failed to extract JSON with newlines: %v", err)
	}

	if !strings.Contains(jsonStr, `"steps"`) {
		t.Errorf("Extracted JSON does not contain 'steps': %s", jsonStr)
	}
}

func TestExtractPlanJSON_NoJSON(t *testing.T) {
	response := "This response contains no JSON at all."

	_, err := ExtractPlanJSON(response)
	if err == nil {
		t.Error("Expected error for response without JSON, but got nil")
	}
}

func TestPlan_CanExecute(t *testing.T) {
	plan := &Plan{
		Steps: []PlanStep{
			{ID: 1, Status: "pending", DependsOn: []int{}},
			{ID: 2, Status: "pending", DependsOn: []int{1}},
			{ID: 3, Status: "pending", DependsOn: []int{1}},
		},
	}

	// Step 1 は依存なしで実行可能
	if !plan.CanExecute(1) {
		t.Error("Expected step 1 to be executable")
	}

	// Step 2 は Step 1 が pending なので実行不可
	if plan.CanExecute(2) {
		t.Error("Expected step 2 to be not executable (depends on step 1)")
	}

	// Step 1 を完了にする
	plan.UpdateStatus(1, "completed", "Success")

	// Step 2, 3 が実行可能になる
	if !plan.CanExecute(2) {
		t.Error("Expected step 2 to be executable after step 1 completed")
	}
	if !plan.CanExecute(3) {
		t.Error("Expected step 3 to be executable after step 1 completed")
	}
}

func TestPlan_GetParallelSteps(t *testing.T) {
	plan := &Plan{
		Steps: []PlanStep{
			{ID: 1, Status: "completed", DependsOn: []int{}},
			{ID: 2, Status: "pending", DependsOn: []int{1}},
			{ID: 3, Status: "pending", DependsOn: []int{1}},
			{ID: 4, Status: "pending", DependsOn: []int{2, 3}},
		},
	}

	parallelSteps := plan.GetParallelSteps()
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
	plan := &Plan{
		Steps: []PlanStep{
			{ID: 1, Status: "completed", DependsOn: []int{}},
			{ID: 2, Status: "completed", DependsOn: []int{}},
			{ID: 3, Status: "pending", DependsOn: []int{1}},
			{ID: 4, Status: "pending", DependsOn: []int{2}},
		},
	}

	parallelSteps := plan.GetParallelSteps()
	// Step 3 と 4 は異なる depends_on を持つので nil
	if parallelSteps != nil {
		t.Errorf("Expected nil for steps with different depends_on, got %v", parallelSteps)
	}
}

func TestSameDependencies(t *testing.T) {
	tests := []struct {
		name string
		a    []int
		b    []int
		want bool
	}{
		{"both empty", []int{}, []int{}, true},
		{"same single", []int{1}, []int{1}, true},
		{"same multiple", []int{1, 2}, []int{2, 1}, true},
		{"different length", []int{1}, []int{1, 2}, false},
		{"different values", []int{1}, []int{2}, false},
		{"nil and empty", nil, []int{}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := sameDependencies(tt.a, tt.b)
			if got != tt.want {
				t.Errorf("sameDependencies(%v, %v) = %v, want %v", tt.a, tt.b, got, tt.want)
			}
		})
	}
}

func TestPlan_GetNextStep(t *testing.T) {
	plan := &Plan{
		Steps: []PlanStep{
			{ID: 1, Status: "completed", DependsOn: []int{}},
			{ID: 2, Status: "pending", DependsOn: []int{1}},
			{ID: 3, Status: "pending", DependsOn: []int{2}},
		},
	}

	nextStep := plan.GetNextStep()
	if nextStep != 2 {
		t.Errorf("Expected next step to be 2, got %d", nextStep)
	}

	plan.UpdateStatus(2, "completed", "Success")
	nextStep = plan.GetNextStep()
	if nextStep != 3 {
		t.Errorf("Expected next step to be 3, got %d", nextStep)
	}

	plan.UpdateStatus(3, "completed", "Success")
	nextStep = plan.GetNextStep()
	if nextStep != -1 {
		t.Errorf("Expected next step to be -1 (all completed), got %d", nextStep)
	}
}

func TestPlan_IsCompleted(t *testing.T) {
	plan := &Plan{
		Steps: []PlanStep{
			{ID: 1, Status: "completed"},
			{ID: 2, Status: "pending"},
		},
	}

	if plan.IsCompleted() {
		t.Error("Expected plan to be not completed")
	}

	plan.UpdateStatus(2, "completed", "Success")
	if !plan.IsCompleted() {
		t.Error("Expected plan to be completed")
	}
}

func TestPlan_HasFailed(t *testing.T) {
	plan := &Plan{
		Steps: []PlanStep{
			{ID: 1, Status: "completed"},
			{ID: 2, Status: "pending"},
		},
	}

	if plan.HasFailed() {
		t.Error("Expected plan to have no failures")
	}

	plan.UpdateStatus(2, "failed", "Error occurred")
	if !plan.HasFailed() {
		t.Error("Expected plan to have failed")
	}
}

func TestFormatPlan(t *testing.T) {
	plan := &Plan{
		Steps: []PlanStep{
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

	formatted := FormatPlan(plan)
	if !strings.Contains(formatted, "📋 Plan:") {
		t.Error("Expected formatted plan to contain header")
	}
	if !strings.Contains(formatted, "Read file") {
		t.Error("Expected formatted plan to contain step description")
	}
	if !strings.Contains(formatted, "Depends on:") {
		t.Error("Expected formatted plan to contain depends_on info")
	}
}

func TestFindClosingBrace(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		start    int
		expected int
	}{
		{
			name:     "simple object",
			input:    `{"key": "value"}`,
			start:    0,
			expected: 16,
		},
		{
			name:     "nested object",
			input:    `{"outer": {"inner": "value"}}`,
			start:    0,
			expected: 29, // closing brace at index 28, +1 = 29
		},
		{
			name:     "with string containing brace",
			input:    `{"key": "value with } brace"}`,
			start:    0,
			expected: 29, // closing brace at index 28, +1 = 29
		},
		{
			name:     "with escaped quote",
			input:    `{"key": "value with \" quote"}`,
			start:    0,
			expected: 30, // closing brace at index 29, +1 = 30
		},
		{
			name:     "incomplete object",
			input:    `{"key": "value"`,
			start:    0,
			expected: -1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := findClosingBrace(tt.input, tt.start)
			if result != tt.expected {
				t.Errorf("Expected %d, got %d for input: %s", tt.expected, result, tt.input)
			}
		})
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
			failed, _ := containsFailure(tt.result)
			if failed != tt.want {
				t.Errorf("containsFailure() = %v, want %v", failed, tt.want)
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

	failed, reason := containsFailure(result)
	if !failed {
		t.Error("containsFailure() should detect panic")
	}
	if reason != "Panic detected" {
		t.Errorf("containsFailure() reason = %q, want 'Panic detected'", reason)
	}
}

func TestContainsFailure_NpmError(t *testing.T) {
	result := `npm ERR! code ENOENT
npm ERR! syscall open
npm ERR! path /home/user/project/package.json
npm ERR! errno -2
npm ERR! enoent ENOENT: no such file or directory`

	failed, reason := containsFailure(result)
	if !failed {
		t.Error("containsFailure() should detect npm error")
	}
	if reason != "npm error" {
		t.Errorf("containsFailure() reason = %q, want 'npm error'", reason)
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
			name:   "exit status 2",
			result: "Process exited with exit status 2",
			want:   true,
		},
		{
			name:   "exit status 0 (success)",
			result: "Command completed with exit status 0",
			want:   true, // The pattern matches "exit status" regardless of code
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			failed, _ := containsFailure(tt.result)
			if failed != tt.want {
				t.Errorf("containsFailure() = %v, want %v", failed, tt.want)
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
			failed, _ := containsFailure(tt.result)
			if failed {
				t.Errorf("containsFailure() should not detect failure for %q", tt.name)
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
			name:   "SyntaxError",
			result: "SyntaxError: Unexpected token",
			want:   true,
		},
		{
			name:   "TypeError",
			result: "TypeError: Cannot read property 'x' of undefined",
			want:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			failed, _ := containsFailure(tt.result)
			if failed != tt.want {
				t.Errorf("containsFailure() = %v, want %v", failed, tt.want)
			}
		})
	}
}

// isAIQuestion tests

func TestIsAIQuestion_JapanesePatterns(t *testing.T) {
	tests := []struct {
		name     string
		response string
		want     bool
	}{
		{
			name:     "続行しますか",
			response: "この変更を適用してもよろしいでしょうか？続行しますか？",
			want:     true,
		},
		{
			name:     "よろしいですか",
			response: "ファイルを削除してもよろしいですか？",
			want:     true,
		},
		{
			name:     "確認してください",
			response: "設定を確認してください。",
			want:     true,
		},
		{
			name:     "どうしますか",
			response: "エラーが発生しました。どうしますか？",
			want:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isAIQuestion(tt.response)
			if got != tt.want {
				t.Errorf("isAIQuestion() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsAIQuestion_EnglishPatterns(t *testing.T) {
	tests := []struct {
		name     string
		response string
		want     bool
	}{
		{
			name:     "Should I",
			response: "Should I proceed with the changes?",
			want:     true,
		},
		{
			name:     "Do you want",
			response: "Do you want me to create the file?",
			want:     true,
		},
		{
			name:     "Would you like",
			response: "Would you like me to run the tests?",
			want:     true,
		},
		{
			name:     "Shall I",
			response: "Shall I continue?",
			want:     true,
		},
		{
			name:     "Can you confirm",
			response: "Can you confirm this is correct?",
			want:     true,
		},
		{
			name:     "proceed?",
			response: "The changes are ready. Proceed?",
			want:     true,
		},
		{
			name:     "continue?",
			response: "File saved successfully. Continue?",
			want:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isAIQuestion(tt.response)
			if got != tt.want {
				t.Errorf("isAIQuestion() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsAIQuestion_NotQuestion(t *testing.T) {
	tests := []struct {
		name     string
		response string
	}{
		{
			name:     "simple statement",
			response: "I have read the file successfully.",
		},
		{
			name:     "action report",
			response: "The function has been added to the file.",
		},
		{
			name:     "empty response",
			response: "",
		},
		{
			name:     "code only",
			response: "```go\nfunc main() {}\n```",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isAIQuestion(tt.response)
			if got {
				t.Errorf("isAIQuestion() = true, want false for %q", tt.name)
			}
		})
	}
}

func TestIsAIQuestion_WithToolCall(t *testing.T) {
	// Even if the response contains question patterns, tool calls take precedence
	response := `Should I proceed with reading the file?

{"tool": "read_file", "args": {"path": "/test.txt"}}`

	got := isAIQuestion(response)
	if got {
		t.Error("isAIQuestion() should return false when tool call is present")
	}
}
