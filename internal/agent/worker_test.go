package agent

import (
	"testing"

	"github.com/susugadx/xelyon-cli/internal/agent/plan"
)

func TestNewWorker(t *testing.T) {
	sharedCtx := NewSharedContext()
	commands := make(chan WorkerCommand, 10)
	results := make(chan WorkerResult, 10)

	w := NewWorker(
		1,            // id
		nil,          // provider (nil for test)
		"test-model", // model
		sharedCtx,
		nil, // agent (nil for test)
		600, // stepTimeout
		commands,
		results,
	)

	if w == nil {
		t.Fatal("NewWorker returned nil")
	}
	if w.id != 1 {
		t.Errorf("worker id = %d, want 1", w.id)
	}
	if w.model != "test-model" {
		t.Errorf("worker model = %s, want test-model", w.model)
	}
	if w.stepTimeout != 600 {
		t.Errorf("worker stepTimeout = %d, want 600", w.stepTimeout)
	}
	if w.status != WorkerIdle {
		t.Errorf("worker initial status = %v, want WorkerIdle", w.status)
	}
}

func TestWorker_GetStatus(t *testing.T) {
	sharedCtx := NewSharedContext()
	commands := make(chan WorkerCommand, 10)
	results := make(chan WorkerResult, 10)

	w := NewWorker(0, nil, "test", sharedCtx, nil, 600, commands, results)

	// 初期状態は Idle
	if w.GetStatus() != WorkerIdle {
		t.Errorf("initial status = %v, want WorkerIdle", w.GetStatus())
	}

	// 状態を手動で変更してテスト
	w.mu.Lock()
	w.status = WorkerRunning
	w.mu.Unlock()

	if w.GetStatus() != WorkerRunning {
		t.Errorf("status = %v, want WorkerRunning", w.GetStatus())
	}
}

func TestWorker_GetCurrentStep(t *testing.T) {
	sharedCtx := NewSharedContext()
	commands := make(chan WorkerCommand, 10)
	results := make(chan WorkerResult, 10)

	w := NewWorker(0, nil, "test", sharedCtx, nil, 600, commands, results)

	// 初期状態は nil
	if w.GetCurrentStep() != nil {
		t.Error("initial currentStep should be nil")
	}

	// ステップを設定
	step := &plan.PlanStep{ID: 1, Description: "Test step"}
	w.mu.Lock()
	w.currentStep = step
	w.mu.Unlock()

	current := w.GetCurrentStep()
	if current == nil {
		t.Fatal("GetCurrentStep returned nil")
	}
	if current.ID != 1 {
		t.Errorf("currentStep.ID = %d, want 1", current.ID)
	}
}

func TestWorker_buildWorkerHistory(t *testing.T) {
	sharedCtx := NewSharedContext()
	commands := make(chan WorkerCommand, 10)
	results := make(chan WorkerResult, 10)

	// Agent をモックするのは複雑なので、nil で初期化してテスト
	w := NewWorker(0, nil, "test", sharedCtx, nil, 600, commands, results)

	step := &plan.PlanStep{
		ID:          1,
		Description: "Test step description",
		Tools:       []string{"read_file", "write_file"},
	}

	// コンテキストなしの場合
	history := w.buildWorkerHistory(step, "")
	if len(history) != 1 {
		t.Errorf("history length = %d, want 1", len(history))
	}
	if history[0].Role != "user" {
		t.Errorf("history[0].Role = %s, want user", history[0].Role)
	}

	// コンテキストありの場合
	history = w.buildWorkerHistory(step, "Previous step output")
	if len(history) != 1 {
		t.Errorf("history length = %d, want 1", len(history))
	}
	if !containsSubstring(history[0].Content, "Previous step") {
		t.Error("history should contain context info")
	}
}

func TestStepTimeout_Error(t *testing.T) {
	err := &StepTimeout{StepID: 5, Timeout: 600}
	expected := "step 5 timed out after 600 seconds"
	if err.Error() != expected {
		t.Errorf("StepTimeout.Error() = %s, want %s", err.Error(), expected)
	}
}

func TestStepFailure_Error(t *testing.T) {
	err := &StepFailure{StepID: 3, Reason: "file not found"}
	expected := "step 3 failed: file not found"
	if err.Error() != expected {
		t.Errorf("StepFailure.Error() = %s, want %s", err.Error(), expected)
	}
}

func TestEscalationNeeded_Error(t *testing.T) {
	err := &EscalationNeeded{
		StepID:   2,
		ToolName: "delete_file",
		Reason:   "confirmation required",
	}
	expected := "step 2 requires escalation: confirmation required (tool: delete_file)"
	if err.Error() != expected {
		t.Errorf("EscalationNeeded.Error() = %s, want %s", err.Error(), expected)
	}
}

func TestWorkerStatus_Values(t *testing.T) {
	tests := []struct {
		status   WorkerStatus
		expected string
	}{
		{WorkerIdle, "idle"},
		{WorkerRunning, "running"},
		{WorkerDone, "done"},
		{WorkerFailed, "failed"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if string(tt.status) != tt.expected {
				t.Errorf("WorkerStatus = %s, want %s", string(tt.status), tt.expected)
			}
		})
	}
}

func TestCommandType_Constants(t *testing.T) {
	// コマンドタイプの定数値を確認
	if CmdExecuteStep != 0 {
		t.Errorf("CmdExecuteStep = %d, want 0", CmdExecuteStep)
	}
	if CmdInvestigate != 1 {
		t.Errorf("CmdInvestigate = %d, want 1", CmdInvestigate)
	}
	if CmdStop != 2 {
		t.Errorf("CmdStop = %d, want 2", CmdStop)
	}
}

func TestWorkerResult_Fields(t *testing.T) {
	result := WorkerResult{
		WorkerID:         1,
		StepID:           5,
		Success:          true,
		NeedsEscalation:  false,
		EscalationReason: "",
		Output:           "Step completed successfully",
		ToolsExecuted:    []string{"read_file", "write_file"},
		Query:            "test query",
	}

	if result.WorkerID != 1 {
		t.Errorf("WorkerID = %d, want 1", result.WorkerID)
	}
	if result.StepID != 5 {
		t.Errorf("StepID = %d, want 5", result.StepID)
	}
	if !result.Success {
		t.Error("Success should be true")
	}
	if result.NeedsEscalation {
		t.Error("NeedsEscalation should be false")
	}
	if len(result.ToolsExecuted) != 2 {
		t.Errorf("ToolsExecuted length = %d, want 2", len(result.ToolsExecuted))
	}
	if result.Query != "test query" {
		t.Errorf("Query = %s, want 'test query'", result.Query)
	}
}

func TestWorkerCommand_Fields(t *testing.T) {
	step := &plan.PlanStep{ID: 1}
	cmd := WorkerCommand{
		Type:         CmdExecuteStep,
		StepID:       1,
		Step:         step,
		Query:        "",
		ConfirmLevel: "dangerous",
	}

	if cmd.Type != CmdExecuteStep {
		t.Errorf("Type = %d, want CmdExecuteStep", cmd.Type)
	}
	if cmd.StepID != 1 {
		t.Errorf("StepID = %d, want 1", cmd.StepID)
	}
	if cmd.Step.ID != 1 {
		t.Errorf("Step.ID = %d, want 1", cmd.Step.ID)
	}
	if cmd.ConfirmLevel != "dangerous" {
		t.Errorf("ConfirmLevel = %s, want 'dangerous'", cmd.ConfirmLevel)
	}
}
