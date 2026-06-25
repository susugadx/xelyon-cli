package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/susugadx/xelyon-cli/internal/tools"
)

func TestRunHeadlessWithConfig_SummaryChangedFilesFromEditTool(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XELYON_EDIT_TOOL", "str_replace")

	dir := testSubDir(t)
	writeTestFile(t, filepath.Join(dir, "target.txt"), "old content\n")

	provider := &sequenceMockProvider{
		name: "test-provider",
		responses: []string{
			`{"tool":"str_replace","args":{"path":"target.txt","old_str":"old content","new_str":"new content"}}`,
			"final text mentions fake changed_files: fake.go",
		},
	}

	result := RunHeadlessWithConfig(context.Background(), "edit target", "test-model", provider, newProjectMapDisabledConfig())

	if result.Status != HeadlessStatusSuccess {
		t.Fatalf("Status = %q, want success: %+v", result.Status, result.Error)
	}
	if result.Summary == nil {
		t.Fatal("Summary = nil, want changed_files")
	}
	if got, want := result.Summary.ChangedFiles, []string{"target.txt"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("Summary.ChangedFiles = %v, want %v", got, want)
	}
	if len(result.Summary.Commands) != 0 {
		t.Fatalf("Summary.Commands = %+v, want empty", result.Summary.Commands)
	}
}

func TestRunHeadlessWithConfig_SummaryBashTestCommand(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	dir := testSubDir(t)
	writeTestFile(t, filepath.Join(dir, "go.mod"), "module example.com/headless-summary\n\ngo 1.24\n")
	writeTestFile(t, filepath.Join(dir, "main_test.go"), "package main\n\nimport \"testing\"\n\nfunc TestProbe(t *testing.T) {}\n")

	provider := &sequenceMockProvider{
		name: "test-provider",
		responses: []string{
			`{"tool":"bash","args":{"command":"go test ./..."}}`,
			"done",
		},
	}

	result := RunHeadlessWithConfig(context.Background(), "run tests", "test-model", provider, newProjectMapDisabledConfig())

	if result.Status != HeadlessStatusSuccess {
		t.Fatalf("Status = %q, want success: %+v", result.Status, result.Error)
	}
	requireHeadlessCommandSummary(t, result, HeadlessCommandSummary{
		Command:  "go test ./...",
		ExitCode: 0,
		Status:   "passed",
		Source:   "tool",
	})
}

func TestRunHeadlessWithConfig_SummaryNonTestShellCommand(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	_ = testSubDir(t)

	provider := &sequenceMockProvider{
		name: "test-provider",
		responses: []string{
			`{"tool":"bash","args":{"command":"printf ok"}}`,
			"done",
		},
	}

	result := RunHeadlessWithConfig(context.Background(), "run command", "test-model", provider, newProjectMapDisabledConfig())

	if result.Status != HeadlessStatusSuccess {
		t.Fatalf("Status = %q, want success: %+v", result.Status, result.Error)
	}
	requireHeadlessCommandSummary(t, result, HeadlessCommandSummary{
		Command:  "printf ok",
		ExitCode: 0,
		Status:   "passed",
		Source:   "tool",
	})
	if len(result.Summary.FinalChecks) != 0 {
		t.Fatalf("Summary.FinalChecks = %+v, want empty", result.Summary.FinalChecks)
	}
}

func TestRunHeadlessWithConfig_SummaryFinalChecksPassed(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XELYON_EDIT_TOOL", "str_replace")

	dir := testSubDir(t)
	writeTestFile(t, filepath.Join(dir, "target.txt"), "old content\n")

	cfg := newProjectMapDisabledConfig()
	cfg.FinalChecks.Commands = []string{`test "$XELYON_CHANGED_FILES" = "target.txt"`}
	cfg.FinalChecks.Timeout = 10

	provider := &sequenceMockProvider{
		name: "test-provider",
		responses: []string{
			`{"tool":"str_replace","args":{"path":"target.txt","old_str":"old content","new_str":"new content"}}`,
			"done",
		},
	}

	result := RunHeadlessWithConfig(context.Background(), "edit target", "test-model", provider, cfg)

	if result.Status != HeadlessStatusSuccess {
		t.Fatalf("Status = %q, want success: %+v", result.Status, result.Error)
	}
	if result.Summary == nil {
		t.Fatal("Summary = nil, want final_checks")
	}
	if got, want := result.Summary.FinalChecks, []HeadlessFinalCheckSummary{{
		Command:  `test "$XELYON_CHANGED_FILES" = "target.txt"`,
		ExitCode: 0,
		Status:   "passed",
	}}; !reflect.DeepEqual(got, want) {
		t.Fatalf("Summary.FinalChecks = %+v, want %+v", got, want)
	}
}

func TestRunHeadlessWithConfig_SummaryFinalCheckFailurePromotesResult(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XELYON_EDIT_TOOL", "str_replace")

	dir := testSubDir(t)
	writeTestFile(t, filepath.Join(dir, "target.txt"), "old content\n")

	cfg := newProjectMapDisabledConfig()
	cfg.FinalChecks.Commands = []string{"exit 7"}
	cfg.FinalChecks.Timeout = 10

	provider := &sequenceMockProvider{
		name: "test-provider",
		responses: []string{
			`{"tool":"str_replace","args":{"path":"target.txt","old_str":"old content","new_str":"new content"}}`,
			"done after edit",
		},
	}

	result := RunHeadlessWithConfig(context.Background(), "edit target", "test-model", provider, cfg)

	if result.Status != HeadlessStatusError {
		t.Fatalf("Status = %q, want error", result.Status)
	}
	if result.Error == nil || result.Error.Type != HeadlessErrorTypeFinalCheckFailed {
		t.Fatalf("Error = %+v, want %s", result.Error, HeadlessErrorTypeFinalCheckFailed)
	}
	if result.FailureReason != HeadlessFailureReasonFinalCheckFailed {
		t.Fatalf("FailureReason = %q, want %q", result.FailureReason, HeadlessFailureReasonFinalCheckFailed)
	}
	if result.Response != "done after edit" {
		t.Fatalf("Response = %q, want preserved final response", result.Response)
	}
	if result.Summary == nil {
		t.Fatal("Summary = nil, want final_checks")
	}
	if got, want := result.Summary.FinalChecks, []HeadlessFinalCheckSummary{{
		Command:  "exit 7",
		ExitCode: 7,
		Status:   "failed",
	}}; !reflect.DeepEqual(got, want) {
		t.Fatalf("Summary.FinalChecks = %+v, want %+v", got, want)
	}
}

func TestRunHeadlessWithConfig_FinalCheckCancelPromotesCancelledAndPreservesSummary(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XELYON_EDIT_TOOL", "str_replace")

	dir := testSubDir(t)
	writeTestFile(t, filepath.Join(dir, "target.txt"), "old content\n")
	startedFile := filepath.Join(t.TempDir(), "final-check-started")

	cfg := newProjectMapDisabledConfig()
	cfg.FinalChecks.Commands = []string{fmt.Sprintf("touch %q; sleep 30", startedFile)}
	cfg.FinalChecks.Timeout = 60

	provider := &headlessToolErrorUsageProvider{
		responses: []string{
			`{"tool":"str_replace","args":{"path":"target.txt","old_str":"old content","new_str":"new content"}}`,
			"done after edit",
		},
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	resultCh := make(chan *HeadlessResult, 1)
	go func() {
		resultCh <- RunHeadlessWithConfig(ctx, "edit target", "gpt-5.4-nano", provider, cfg)
	}()

	waitForFile(t, startedFile)
	cancel()

	select {
	case result := <-resultCh:
		if result.Status != HeadlessStatusError {
			t.Fatalf("Status = %q, want error", result.Status)
		}
		if result.Error == nil || result.Error.Type != HeadlessErrorTypeCancelled {
			t.Fatalf("Error = %+v, want %s", result.Error, HeadlessErrorTypeCancelled)
		}
		if result.FailureReason != HeadlessFailureReasonCancelled {
			t.Fatalf("FailureReason = %q, want %q", result.FailureReason, HeadlessFailureReasonCancelled)
		}
		if result.Response != "done after edit" {
			t.Fatalf("Response = %q, want preserved final response", result.Response)
		}
		if len(result.ToolCalls) != 1 {
			t.Fatalf("ToolCalls = %+v, want one preserved tool call", result.ToolCalls)
		}
		if result.Tokens == nil || result.Tokens.Total == 0 {
			t.Fatalf("Tokens = %+v, want preserved usage", result.Tokens)
		}
		if result.Cost <= 0 {
			t.Fatalf("Cost = %f, want preserved positive cost", result.Cost)
		}
		if result.Summary == nil {
			t.Fatal("Summary = nil, want preserved summary")
		}
		if got, want := result.Summary.ChangedFiles, []string{"target.txt"}; !reflect.DeepEqual(got, want) {
			t.Fatalf("Summary.ChangedFiles = %v, want %v", got, want)
		}
		if got, want := result.Summary.FinalChecks, []HeadlessFinalCheckSummary{{
			Command:  cfg.FinalChecks.Commands[0],
			ExitCode: -1,
			Status:   "failed",
		}}; !reflect.DeepEqual(got, want) {
			t.Fatalf("Summary.FinalChecks = %+v, want %+v", got, want)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("RunHeadlessWithConfig did not return promptly after final-check cancellation")
	}
}

func TestRunHeadlessWithConfig_FinalCheckTimeoutStaysFinalCheckFailed(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XELYON_EDIT_TOOL", "str_replace")

	dir := testSubDir(t)
	writeTestFile(t, filepath.Join(dir, "target.txt"), "old content\n")

	cfg := newProjectMapDisabledConfig()
	cfg.FinalChecks.Commands = []string{"sleep 30"}
	cfg.FinalChecks.Timeout = 1

	provider := &sequenceMockProvider{
		name: "test-provider",
		responses: []string{
			`{"tool":"str_replace","args":{"path":"target.txt","old_str":"old content","new_str":"new content"}}`,
			"done after edit",
		},
	}

	result := RunHeadlessWithConfig(context.Background(), "edit target", "test-model", provider, cfg)

	if result.Status != HeadlessStatusError {
		t.Fatalf("Status = %q, want error", result.Status)
	}
	if result.Error == nil || result.Error.Type != HeadlessErrorTypeFinalCheckFailed {
		t.Fatalf("Error = %+v, want %s", result.Error, HeadlessErrorTypeFinalCheckFailed)
	}
	if result.FailureReason != HeadlessFailureReasonFinalCheckFailed {
		t.Fatalf("FailureReason = %q, want %q", result.FailureReason, HeadlessFailureReasonFinalCheckFailed)
	}
	if result.Summary == nil || len(result.Summary.FinalChecks) != 1 {
		t.Fatalf("Summary.FinalChecks = %+v, want one timed-out final check", result.Summary)
	}
	if result.Summary.FinalChecks[0].Command != "sleep 30" || result.Summary.FinalChecks[0].Status != "failed" {
		t.Fatalf("Summary.FinalChecks = %+v, want failed sleep command", result.Summary.FinalChecks)
	}
}

func TestHeadlessResult_ToJSON_SummaryOmitsCommandOutput(t *testing.T) {
	result := NewSuccessResult("openai", "gpt-5.4", "ok", nil, 10)
	result.Summary = &HeadlessSummary{
		Commands: []HeadlessCommandSummary{{
			Command:  "printf secret",
			ExitCode: 0,
			Status:   "passed",
			Source:   "tool",
		}},
		FinalChecks: []HeadlessFinalCheckSummary{{
			Command:  "make verify-fast",
			ExitCode: 0,
			Status:   "passed",
		}},
	}

	jsonStr, err := result.ToJSON()
	if err != nil {
		t.Fatalf("ToJSON() error = %v", err)
	}
	var parsed map[string]any
	if err := json.Unmarshal([]byte(jsonStr), &parsed); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	summary, ok := parsed["summary"].(map[string]any)
	if !ok {
		t.Fatalf("summary = %#v, want object", parsed["summary"])
	}
	for _, key := range []string{"commands", "final_checks"} {
		items, ok := summary[key].([]any)
		if !ok || len(items) != 1 {
			t.Fatalf("summary[%s] = %#v, want one item", key, summary[key])
		}
		item, ok := items[0].(map[string]any)
		if !ok {
			t.Fatalf("summary[%s][0] = %#v, want object", key, items[0])
		}
		if _, exists := item["output"]; exists {
			t.Fatalf("summary[%s][0] contains output: %#v", key, item)
		}
	}
}

func requireHeadlessCommandSummary(t *testing.T, result *HeadlessResult, want HeadlessCommandSummary) {
	t.Helper()

	if result.Summary == nil {
		t.Fatal("Summary = nil, want command summary")
	}
	if got := result.Summary.Commands; len(got) != 1 || got[0] != want {
		t.Fatalf("Summary.Commands = %+v, want [%+v]", got, want)
	}
	if len(result.Summary.FinalChecks) != 0 {
		t.Fatalf("Summary.FinalChecks = %+v, want empty", result.Summary.FinalChecks)
	}
}

func TestClassifyHeadlessCommandResult(t *testing.T) {
	tests := []struct {
		name      string
		output    string
		toolError bool
		want      headlessCommandClassification
	}{
		{name: "success", output: "ok", want: headlessCommandClassification{exitCode: 0, status: "passed"}},
		{name: "explicit exit status", output: "Error: exit status 2\nOutput: failed", want: headlessCommandClassification{exitCode: 2, status: "failed"}},
		{name: "error prefix", output: "Error: blocked", want: headlessCommandClassification{exitCode: -1, status: "failed"}},
		{name: "tool error flag", output: "blocked by policy", toolError: true, want: headlessCommandClassification{exitCode: -1, status: "failed"}},
		{name: "cancelled by user", output: "Cancelled by user", want: headlessCommandClassification{exitCode: -1, status: "failed"}},
		{name: "command interrupted", output: "Command interrupted.\nPartial output:\nwork", want: headlessCommandClassification{exitCode: -1, status: "failed"}},
		{name: "cancelled marker", output: "[CANCELLED] apply_patch was not approved", want: headlessCommandClassification{exitCode: -1, status: "failed"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := classifyHeadlessCommandResult(tt.output, tt.toolError); got != tt.want {
				t.Fatalf("classifyHeadlessCommandResult(%q, %v) = %+v, want %+v", tt.output, tt.toolError, got, tt.want)
			}
		})
	}
}

func TestNewHeadlessCommandSummary_DoesNotChangeToolCallSuccessCompatibility(t *testing.T) {
	execResult := tools.ExecutionResult{
		Result: "Command interrupted.\nPartial output:\n",
		Error:  false,
	}
	if !isHeadlessToolCallSuccess(execResult) {
		t.Fatal("isHeadlessToolCallSuccess() = false, want existing compatibility to keep non-Error interrupted output successful")
	}
	summary, ok := newHeadlessCommandSummary(&tools.ToolCall{
		Tool: "bash",
		Args: map[string]string{"command": "sleep 30"},
	}, execResult)
	if !ok {
		t.Fatal("newHeadlessCommandSummary() ok = false, want true")
	}
	if summary.Status != "failed" || summary.ExitCode != -1 {
		t.Fatalf("summary = %+v, want failed/-1", summary)
	}
}

func TestHeadlessSummaryChangedFilesFromLedger(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	dir := testSubDir(t)
	if err := os.MkdirAll(filepath.Join(dir, "nested"), 0o755); err != nil {
		t.Fatal(err)
	}

	agent := NewAgentWithRuntime("test-model", &sequenceMockProvider{name: "test-provider"}, true, NewAgentRuntimeWithConfig(newProjectMapDisabledConfig()))
	agent.Runtime.TaskLedger.Recorder().RecordChangedFile(filepath.Join(dir, "nested", "changed.go"))

	got := headlessSummaryChangedFiles(agent)
	if want := []string{filepath.Join("nested", "changed.go")}; !reflect.DeepEqual(got, want) {
		t.Fatalf("headlessSummaryChangedFiles() = %v, want %v", got, want)
	}
}

func ExampleHeadlessSummary() {
	result := NewSuccessResult("openai", "gpt-5.4", "ok", nil, 10)
	result.Summary = &HeadlessSummary{
		ChangedFiles: []string{"internal/example.go"},
	}
	fmt.Println(result.Summary.ChangedFiles[0])
	// Output: internal/example.go
}
