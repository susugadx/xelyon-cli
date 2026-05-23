package agent

import (
	"reflect"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/api"
)

func TestProviderHistoryCommandEditDryRunDetectsOldCommandOutput(t *testing.T) {
	output := strings.Repeat("--- FAIL: TestCommandDryRun\nFAIL\t./internal/agent\n", 8)
	report := providerHistoryCommandEditDryRunReportForTest(t,
		api.Message{Role: "user", Content: "run tests"},
		providerHistoryAssistantToolCalls(providerHistoryToolCallWithJSONArguments(t, "call_test", "bash", map[string]string{"command": "go test ./internal/agent"})),
		providerHistoryToolResult("call_test", "bash", output),
		api.Message{Role: "assistant", Content: "tests failed"},
		providerHistoryAssistantToolCall("call_latest", "read_file"),
		providerHistoryToolResult("call_latest", "read_file", "latest read"),
		api.Message{Role: "assistant", Content: "done"},
	)

	if report.ReplacementStatus != providerHistoryCommandEditReplacementStatusNotImplemented {
		t.Fatalf("ReplacementStatus = %q, want not_implemented", report.ReplacementStatus)
	}
	if report.CommandCandidates != 1 || report.CommandOriginalBytes != len(output) || report.ApproxCommandSavedTokens <= 0 {
		t.Fatalf("command dry-run metrics = candidates %d bytes %d tokens %d, want one command candidate with savings", report.CommandCandidates, report.CommandOriginalBytes, report.ApproxCommandSavedTokens)
	}
	if got := report.CandidateReasonCounts["test_failure_output"]; got != 1 {
		t.Fatalf("CandidateReasonCounts = %#v, want test_failure_output:1", report.CandidateReasonCounts)
	}
	if len(report.Candidates) != 1 || report.Candidates[0].ToolName != "bash" || report.Candidates[0].Reason != "test_failure_output" {
		t.Fatalf("Candidates = %#v, want bash test failure candidate", report.Candidates)
	}
}

func TestProviderHistoryCommandEditDryRunKeepsLatestAndTrailingCommandOutputs(t *testing.T) {
	t.Run("latest", func(t *testing.T) {
		report := providerHistoryCommandEditDryRunReportForTest(t,
			providerHistoryAssistantToolCalls(providerHistoryToolCallWithJSONArguments(t, "call_latest_cmd", "bash", map[string]string{"command": "ls"})),
			providerHistoryToolResult("call_latest_cmd", "bash", "latest output"),
			api.Message{Role: "assistant", Content: "done"},
		)

		if report.CommandCandidates != 0 {
			t.Fatalf("CommandCandidates = %d, want 0 for latest tool result", report.CommandCandidates)
		}
		if got := report.KeptReasonCounts["latest_tool_result"]; got != 1 {
			t.Fatalf("KeptReasonCounts = %#v, want latest_tool_result:1", report.KeptReasonCounts)
		}
	})

	t.Run("latest infers missing tool name from assistant call", func(t *testing.T) {
		report := providerHistoryCommandEditDryRunReportForTest(t,
			providerHistoryAssistantToolCalls(providerHistoryToolCallWithJSONArguments(t, "call_latest_missing_name", "bash", map[string]string{"command": "ls"})),
			providerHistoryToolResult("call_latest_missing_name", "", "latest output"),
			api.Message{Role: "assistant", Content: "done"},
		)

		if report.CommandCandidates != 0 {
			t.Fatalf("CommandCandidates = %d, want 0 for latest tool result", report.CommandCandidates)
		}
		if got := report.KeptReasonCounts["latest_tool_result"]; got != 1 {
			t.Fatalf("KeptReasonCounts = %#v, want latest_tool_result:1", report.KeptReasonCounts)
		}
		if len(report.Kept) != 1 || report.Kept[0].ToolName != "bash" {
			t.Fatalf("Kept = %#v, want inferred bash latest keep", report.Kept)
		}
	})

	t.Run("trailing", func(t *testing.T) {
		report := providerHistoryCommandEditDryRunReportForTest(t,
			api.Message{Role: "assistant", Content: "before tail"},
			providerHistoryAssistantToolCalls(providerHistoryToolCallWithJSONArguments(t, "call_tail_cmd", "bash", map[string]string{"command": "ls"})),
			providerHistoryToolResult("call_tail_cmd", "bash", "tail output"),
		)

		if report.CommandCandidates != 0 {
			t.Fatalf("CommandCandidates = %d, want 0 for trailing tool result", report.CommandCandidates)
		}
		if got := report.KeptReasonCounts["trailing_tool_suffix"]; got != 1 {
			t.Fatalf("KeptReasonCounts = %#v, want trailing_tool_suffix:1", report.KeptReasonCounts)
		}
	})

	t.Run("trailing infers missing tool name from assistant call", func(t *testing.T) {
		report := providerHistoryCommandEditDryRunReportForTest(t,
			api.Message{Role: "assistant", Content: "before tail"},
			providerHistoryAssistantToolCalls(providerHistoryToolCallWithJSONArguments(t, "call_tail_missing_name", "write_file", map[string]string{"path": "a.go", "content": "package main\n"})),
			providerHistoryToolResult("call_tail_missing_name", "", "tail output"),
		)

		if report.EditArgCandidates != 0 {
			t.Fatalf("EditArgCandidates = %d, want 0 for trailing tool result", report.EditArgCandidates)
		}
		if got := report.KeptReasonCounts["trailing_tool_suffix"]; got != 1 {
			t.Fatalf("KeptReasonCounts = %#v, want trailing_tool_suffix:1", report.KeptReasonCounts)
		}
		if len(report.Kept) != 1 || report.Kept[0].ToolName != "write_file" {
			t.Fatalf("Kept = %#v, want inferred write_file trailing keep", report.Kept)
		}
	})
}

func TestProviderHistoryCommandEditDryRunClassifiesCommandReasons(t *testing.T) {
	tests := []struct {
		name    string
		command string
		output  string
		want    string
	}{
		{name: "git diff", command: "git diff -- internal/agent", output: "diff --git a/a.go b/a.go\n", want: "git_diff_output"},
		{name: "test failure", command: "go test ./...", output: "--- FAIL: TestX\nFAIL\t./pkg\n", want: "test_failure_output"},
		{name: "build failure", command: "go build ./...", output: "undefined: missingSymbol\n", want: "build_failure_output"},
		{name: "nonzero exit", command: "ls missing", output: "Error: exit status 2", want: "command_exit_nonzero"},
		{name: "nonzero exit code", command: "run-task", output: "Process exited with code 2", want: "command_exit_nonzero"},
		{name: "zero exit status", command: "run-task", output: "exit status 0", want: "command_output"},
		{name: "zero exit code", command: "run-task", output: "exit code 0", want: "command_output"},
		{name: "process exited with zero code", command: "run-task", output: "Process exited with code 0", want: "command_output"},
		{name: "fallback", command: "ls", output: "file.txt\n", want: "command_output"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			report := providerHistoryCommandEditDryRunReportForTest(t,
				providerHistoryAssistantToolCalls(providerHistoryToolCallWithJSONArguments(t, "call_cmd", "bash", map[string]string{"command": tt.command})),
				providerHistoryToolResult("call_cmd", "bash", tt.output),
				api.Message{Role: "assistant", Content: "after command"},
				providerHistoryAssistantToolCall("call_latest", "read_file"),
				providerHistoryToolResult("call_latest", "read_file", "latest"),
				api.Message{Role: "assistant", Content: "done"},
			)

			if report.CommandCandidates != 1 {
				t.Fatalf("CommandCandidates = %d, want 1", report.CommandCandidates)
			}
			if got := report.Candidates[0].Reason; got != tt.want {
				t.Fatalf("candidate reason = %q, want %q (report %#v)", got, tt.want, report)
			}
		})
	}
}

func TestProviderHistoryCommandEditDryRunKeepsInvalidLinkageFromAssistantToolName(t *testing.T) {
	report := providerHistoryCommandEditDryRunReportForTest(t,
		providerHistoryAssistantToolCalls(providerHistoryToolCallWithJSONArguments(t, "call_mismatch", "bash", map[string]string{"command": "ls"})),
		providerHistoryToolResult("call_mismatch", "read_file", "mismatched command output"),
		api.Message{Role: "assistant", Content: "after mismatch"},
		providerHistoryAssistantToolCalls(
			providerHistoryToolCallWithJSONArguments(t, "call_ambiguous", "read_file", map[string]string{"path": "README.md"}),
			providerHistoryToolCallWithJSONArguments(t, "call_ambiguous", "write_file", map[string]string{"path": "a.go", "content": "x"}),
		),
		providerHistoryToolResult("call_ambiguous", "", "ambiguous result"),
		api.Message{Role: "assistant", Content: "after ambiguous"},
		providerHistoryAssistantToolCalls(providerHistoryToolCallWithJSONArguments(t, "call_non_contiguous", "apply_patch", map[string]string{"patch": "*** Begin Patch\n*** End Patch"})),
		api.Message{Role: "assistant", Content: "intervening assistant"},
		providerHistoryToolResult("call_non_contiguous", "", "non-contiguous result"),
		api.Message{Role: "assistant", Content: "after non-contiguous"},
		providerHistoryAssistantToolCall("call_latest", "read_file"),
		providerHistoryToolResult("call_latest", "read_file", "latest"),
		api.Message{Role: "assistant", Content: "done"},
	)

	if report.CommandCandidates != 0 || report.EditArgCandidates != 0 {
		t.Fatalf("command/edit candidates = %d/%d, want no candidates for invalid linkage", report.CommandCandidates, report.EditArgCandidates)
	}
	for _, want := range []string{"mismatched_tool_name", "ambiguous_assistant_tool_call", "non_contiguous_tool_call_linkage"} {
		if got := report.KeptReasonCounts[want]; got != 1 {
			t.Fatalf("KeptReasonCounts[%q] = %d in %#v, want 1", want, got, report.KeptReasonCounts)
		}
	}
}

func TestProviderHistoryCommandEditDryRunDetectsEditArguments(t *testing.T) {
	writeContent := strings.Repeat("package main\n", 20)
	patch := strings.Repeat("*** Begin Patch\n*** Update File: a.go\n+line\n*** End Patch\n", 8)
	oldStr := strings.Repeat("old line\n", 12)
	newStr := strings.Repeat("new line\n", 12)
	edits := strings.Repeat(`[{"old_str":"before","new_str":"after"}]`, 8)
	deletePath := "tmp/generated/delete-target.txt"
	report := providerHistoryCommandEditDryRunReportForTest(t,
		providerHistoryAssistantToolCalls(
			providerHistoryToolCallWithJSONArguments(t, "call_write", "write_file", map[string]string{"path": "a.go", "content": writeContent}),
			providerHistoryToolCallWithJSONArguments(t, "call_patch", "apply_patch", map[string]string{"patch": patch}),
			providerHistoryToolCallWithJSONArguments(t, "call_replace", "str_replace", map[string]string{"path": "b.go", "old_str": oldStr, "new_str": newStr}),
			providerHistoryToolCallWithJSONArguments(t, "call_edits", "str_replace", map[string]string{"path": "c.go", "edits": edits}),
			providerHistoryToolCallWithJSONArguments(t, "call_delete", "delete_file", map[string]string{"path": deletePath}),
		),
		providerHistoryToolResult("call_write", "write_file", "wrote"),
		providerHistoryToolResult("call_patch", "apply_patch", "patched"),
		providerHistoryToolResult("call_replace", "str_replace", "replaced"),
		providerHistoryToolResult("call_edits", "str_replace", "batch replaced"),
		providerHistoryToolResult("call_delete", "delete_file", "deleted"),
		api.Message{Role: "assistant", Content: "edits done"},
		providerHistoryAssistantToolCall("call_latest", "read_file"),
		providerHistoryToolResult("call_latest", "read_file", "latest"),
		api.Message{Role: "assistant", Content: "done"},
	)

	wantBytes := len(writeContent) + len(patch) + len(oldStr) + len(newStr) + len(edits) + len(deletePath)
	if report.EditArgCandidates != 5 || report.EditArgOriginalBytes != wantBytes || report.ApproxEditArgSavedTokens <= 0 {
		t.Fatalf("edit dry-run metrics = candidates %d bytes %d tokens %d, want 5/%d/positive", report.EditArgCandidates, report.EditArgOriginalBytes, report.ApproxEditArgSavedTokens, wantBytes)
	}
	wantReasons := map[string]int{
		"write_file_content":  1,
		"apply_patch_patch":   1,
		"str_replace_strings": 1,
		"str_replace_edits":   1,
		"delete_file_path":    1,
	}
	if !reflect.DeepEqual(report.CandidateReasonCounts, wantReasons) {
		t.Fatalf("CandidateReasonCounts = %#v, want %#v", report.CandidateReasonCounts, wantReasons)
	}
}

func TestProviderHistoryCommandEditDryRunDoesNotApplyReplacementOrDisableResponseChain(t *testing.T) {
	commandOutput := strings.Repeat("command output\n", 20)
	agent := &Agent{History: []api.Message{
		providerHistoryAssistantToolCalls(providerHistoryToolCallWithJSONArguments(t, "call_cmd", "bash", map[string]string{"command": "ls"})),
		providerHistoryToolResult("call_cmd", "bash", commandOutput),
		{Role: "assistant", Content: "after command"},
		providerHistoryAssistantToolCall("call_latest", "read_file"),
		providerHistoryToolResult("call_latest", "read_file", "latest"),
		{Role: "assistant", Content: "done"},
	}}

	result := agent.buildProviderHistoryProjection(ProviderHistoryReductionPolicy{Mode: ProviderHistoryReductionApply})

	if !reflect.DeepEqual(result.History, agent.History) {
		t.Fatalf("apply projection changed command/edit-only history:\n got %#v\nwant %#v", result.History, agent.History)
	}
	if result.Report.ResponsesChainDisabled {
		t.Fatalf("ResponsesChainDisabled = true, want false for command/edit dry-run candidates")
	}
	if result.Report.CommandEditDryRun.CommandCandidates != 1 || result.Report.CommandEditDryRun.ReplacementStatus != providerHistoryCommandEditReplacementStatusNotImplemented {
		t.Fatalf("CommandEditDryRun = %#v, want one not_implemented command candidate", result.Report.CommandEditDryRun)
	}
}

func providerHistoryCommandEditDryRunReportForTest(t *testing.T, history ...api.Message) ProviderHistoryCommandEditDryRunReport {
	t.Helper()
	agent := &Agent{History: history}
	return agent.buildProviderHistoryProjection(ProviderHistoryReductionPolicy{Mode: ProviderHistoryReductionDryRun}).Report.CommandEditDryRun
}
