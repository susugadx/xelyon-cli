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
		providerHistoryAssistantToolCalls(providerHistoryToolCallWithJSONArguments(t, "call_test", "bash", map[string]string{"command": providerHistoryUnsafeFormattedTestCommand()})),
		providerHistoryToolResult("call_test", "bash", output),
		api.Message{Role: "assistant", Content: "tests failed"},
		providerHistoryAssistantToolCall("call_latest", "read_file"),
		providerHistoryToolResult("call_latest", "read_file", "latest read"),
		api.Message{Role: "assistant", Content: "done"},
	)

	if report.ReplacementStatus != providerHistoryCommandEditReplacementStatusNotImplemented {
		t.Fatalf("ReplacementStatus = %q, want not_implemented", report.ReplacementStatus)
	}
	if report.CommandCandidates != 1 || report.CommandOriginalBytes != len(output) {
		t.Fatalf("command dry-run metrics = candidates %d bytes %d, want one command candidate with original bytes %d", report.CommandCandidates, report.CommandOriginalBytes, len(output))
	}
	if report.CommandEstimatedSavedBytes != 0 || report.ApproxCommandSavedTokens != 0 {
		t.Fatalf("command dry-run safe estimate = bytes %d tokens %d, want zero for failed test output", report.CommandEstimatedSavedBytes, report.ApproxCommandSavedTokens)
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
		{name: "test success", command: "go test ./...", output: "ok\t./internal/agent\t0.1s\n", want: "test_success_output"},
		{name: "build success", command: "go build ./...", output: "build completed successfully\n", want: "build_success_output"},
		{name: "lint success", command: "npm run lint", output: "lint clean\n", want: "lint_success_output"},
		{name: "test command without success evidence", command: "go test ./...", output: "running tests\n", want: "command_success_output"},
		{name: "interrupted test command with partial success output", command: "go test ./...", output: "Command interrupted.\nPartial output:\nok\t./internal/agent\t0.1s\n", want: "command_success_output"},
		{name: "grep containing test command", command: "grep 'go test' README.md", output: "ok\t./internal/agent\t0.1s\n", want: "command_success_output"},
		{name: "compound test command", command: "go test ./... && cat coverage.out", output: "ok\t./internal/agent\t0.1s\n", want: "command_success_output"},
		{name: "piped lint command", command: "npm run lint | tee lint.log", output: "lint clean\n", want: "command_success_output"},
		{name: "redirected test command", command: "go test ./... > coverage.out", output: "ok\t./internal/agent\t0.1s\n", want: "command_success_output"},
		{name: "newline separated test command", command: "go test ./...\ncat coverage.out", output: "ok\t./internal/agent\t0.1s\n", want: "command_success_output"},
		{name: "double quoted command substitution test command", command: `go test "$(go list ./...)"`, output: "ok\t./internal/agent\t0.1s\n", want: "command_success_output"},
		{name: "double quoted backtick substitution test command", command: "go test \"`go list ./...`\"", output: "ok\t./internal/agent\t0.1s\n", want: "command_success_output"},
		{name: "test command with quoted regex pipe", command: "go test ./... -run 'TestA|TestB'", output: "ok\t./internal/agent\t0.1s\n", want: "test_success_output"},
		{name: "partial passing mocha test output", command: "npm test", output: "1 passing\n1 failing\n", want: "test_failure_output"},
		{name: "mixed cargo suite test output", command: "cargo test", output: "test result: ok. 1 passed; 0 failed\nfailures:\n    suite_x\ntest result: FAILED. 0 passed; 1 failed\n", want: "test_failure_output"},
		{name: "mixed pytest summary test output", command: "pytest", output: "1 failed, 1 passed in 0.12s\n", want: "test_failure_output"},
		{name: "build completed with errors", command: "npm run build", output: "Build completed with errors\n", want: "build_failure_output"},
		{name: "ambiguous build complete", command: "npm run build", output: "build complete\n", want: "command_success_output"},
		{name: "lint warning summary", command: "npm run lint", output: "0 errors, 3 warnings\n", want: "command_success_output"},
		{name: "lint nonzero errors summary", command: "npm run lint", output: "10 errors\n", want: "command_success_output"},
		{name: "lint nonzero problems summary", command: "npm run lint", output: "10 problems\n", want: "command_success_output"},
		{name: "lint warning summary with zero exit", command: "npm run lint", output: "0 errors, 3 warnings\nProcess exited with code 0\n", want: "command_success_output"},
		{name: "lint uncounted error with zero exit", command: "npm run lint", output: "error: unexpected console statement\nProcess exited with code 0\n", want: "command_success_output"},
		{name: "lint warnings found with zero exit", command: "npm run lint", output: "Warnings found\nProcess exited with code 0\n", want: "command_success_output"},
		{name: "lint issue count with zero exit", command: "npm run lint", output: "1 issue found\nProcess exited with code 0\n", want: "command_success_output"},
		{name: "lint issues summary with zero exit", command: "npm run lint", output: "2 issues\nProcess exited with code 0\n", want: "command_success_output"},
		{name: "lint clean summary", command: "npm run lint", output: "0 errors, 0 warnings\n", want: "lint_success_output"},
		{name: "lint zero exit only", command: "npm run lint", output: "Process exited with code 0\n", want: "lint_success_output"},
		{name: "zero exit status", command: "run-task", output: "exit status 0", want: "command_success_output"},
		{name: "zero exit code", command: "run-task", output: "exit code 0", want: "command_success_output"},
		{name: "process exited with zero code", command: "run-task", output: "Process exited with code 0", want: "command_success_output"},
		{name: "fallback", command: "ls", output: "file.txt\n", want: "command_success_output"},
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
	if report.EditArgCandidates != 5 || report.EditArgOriginalBytes != wantBytes {
		t.Fatalf("edit dry-run metrics = candidates %d bytes %d, want 5/%d", report.EditArgCandidates, report.EditArgOriginalBytes, wantBytes)
	}
	if report.EditArgEstimatedSavedBytes != 0 || report.ApproxEditArgSavedTokens != 0 {
		t.Fatalf("edit dry-run replacement estimate = bytes %d tokens %d, want zero without successful matching edit results", report.EditArgEstimatedSavedBytes, report.ApproxEditArgSavedTokens)
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

	result := agent.buildProviderHistoryProjection(ProviderHistoryReductionPolicy{Mode: ProviderHistoryReductionDryRun})

	if !reflect.DeepEqual(result.History, agent.History) {
		t.Fatalf("dry-run projection changed command/edit-only history:\n got %#v\nwant %#v", result.History, agent.History)
	}
	if result.Report.ResponsesChainDisabled {
		t.Fatalf("ResponsesChainDisabled = true, want false for command/edit dry-run candidates")
	}
	if result.Report.CommandEditDryRun.CommandCandidates != 1 || result.Report.CommandEditDryRun.CommandReplacedCount != 0 || result.Report.CommandEditDryRun.ReplacementStatus != providerHistoryCommandEditReplacementStatusNotImplemented {
		t.Fatalf("CommandEditDryRun = %#v, want one unreplaced command candidate", result.Report.CommandEditDryRun)
	}
}

func TestProviderHistoryCommandEditApplyReplacesOldSuccessfulCommandOutputs(t *testing.T) {
	testOutput := providerHistoryLargeSuccessfulTestOutput()
	buildOutput := providerHistoryLargeSuccessfulBuildOutput()
	lintOutput := providerHistoryLargeSuccessfulLintOutput()
	agent := &Agent{History: []api.Message{
		providerHistoryAssistantToolCalls(providerHistoryToolCallWithJSONArguments(t, "call_test", "bash", map[string]string{"command": providerHistoryUnsafeFormattedTestCommand()})),
		providerHistoryToolResult("call_test", "bash", testOutput),
		api.Message{Role: "assistant", Content: "tests passed"},
		providerHistoryAssistantToolCalls(providerHistoryToolCallWithJSONArguments(t, "call_build", "command", map[string]string{"command": providerHistorySuccessfulBuildCommand})),
		providerHistoryToolResult("call_build", "command", buildOutput),
		api.Message{Role: "assistant", Content: "build passed"},
		providerHistoryAssistantToolCalls(providerHistoryToolCallWithJSONArguments(t, "call_lint", "bash", map[string]string{"command": providerHistorySuccessfulLintCommand})),
		providerHistoryToolResult("call_lint", "bash", lintOutput),
		api.Message{Role: "assistant", Content: "lint passed"},
		providerHistoryAssistantToolCall("call_latest", "read_file"),
		providerHistoryToolResult("call_latest", "read_file", "latest read"),
		api.Message{Role: "assistant", Content: "done"},
	}}
	raw := api.CloneMessages(agent.History)

	result := agent.buildProviderHistoryProjection(ProviderHistoryReductionPolicy{Mode: ProviderHistoryReductionApply})

	assertProviderHistoryCommandReplacement(t, result, 1, testOutput, providerHistorySuccessfulTestReplacementLabel)
	assertProviderHistoryCommandReplacement(t, result, 4, buildOutput, providerHistorySuccessfulBuildReplacementLabel)
	assertProviderHistoryCommandReplacement(t, result, 7, lintOutput, providerHistorySuccessfulLintReplacementLabel)
	if !reflect.DeepEqual(agent.History, raw) {
		t.Fatalf("Agent.History changed after command replacement:\n got %#v\nwant %#v", agent.History, raw)
	}
	report := result.Report
	if report.ReplacedCount != 0 || !report.ResponsesChainDisabled {
		t.Fatalf("projection report = %#v, want command-only replacement to disable response chain without read/search replacement", report)
	}
	commandReport := report.CommandEditDryRun
	if commandReport.ReplacementStatus != providerHistoryCommandEditReplacementStatusPartialApply ||
		commandReport.CommandCandidates != 3 ||
		commandReport.CommandReplacedCount != 3 ||
		commandReport.CommandReplacementSavedBytes <= 0 ||
		commandReport.ApproxCommandReplacementSavedTokens < providerHistoryCommandReplacementMinSavedTokens*3 {
		t.Fatalf("CommandEditDryRun = %#v, want three partial_apply command replacements with savings", commandReport)
	}
	wantReasons := map[string]int{"test_success_output": 1, "build_success_output": 1, "lint_success_output": 1}
	if !reflect.DeepEqual(commandReport.CandidateReasonCounts, wantReasons) {
		t.Fatalf("CandidateReasonCounts = %#v, want %#v", commandReport.CandidateReasonCounts, wantReasons)
	}
}

func TestProviderHistoryCommandEditApplyKeepsUnsafeAndGenericCommandOutputs(t *testing.T) {
	agent := &Agent{History: []api.Message{
		providerHistoryAssistantToolCalls(providerHistoryToolCallWithJSONArguments(t, "call_diff", "bash", map[string]string{"command": "git diff"})),
		providerHistoryToolResult("call_diff", "bash", providerHistoryLargeCommandOutput("diff --git a/a.go b/a.go\n")),
		api.Message{Role: "assistant", Content: "diff checked"},
		providerHistoryAssistantToolCalls(providerHistoryToolCallWithJSONArguments(t, "call_test_fail", "bash", map[string]string{"command": "go test ./..."})),
		providerHistoryToolResult("call_test_fail", "bash", providerHistoryLargeCommandOutput("--- FAIL: TestX\nFAIL\t./internal/agent\n")),
		api.Message{Role: "assistant", Content: "tests failed"},
		providerHistoryAssistantToolCalls(providerHistoryToolCallWithJSONArguments(t, "call_build_fail", "bash", map[string]string{"command": "go build ./..."})),
		providerHistoryToolResult("call_build_fail", "bash", providerHistoryLargeCommandOutput("undefined: missingSymbol\n")),
		api.Message{Role: "assistant", Content: "build failed"},
		providerHistoryAssistantToolCalls(providerHistoryToolCallWithJSONArguments(t, "call_nonzero", "bash", map[string]string{"command": "ls missing"})),
		providerHistoryToolResult("call_nonzero", "bash", providerHistoryLargeCommandOutput("Error: exit status 2\n")),
		api.Message{Role: "assistant", Content: "command failed"},
		providerHistoryAssistantToolCalls(providerHistoryToolCallWithJSONArguments(t, "call_generic", "bash", map[string]string{"command": "ls -la"})),
		providerHistoryToolResult("call_generic", "bash", providerHistoryLargeCommandOutput("file.txt\n")),
		api.Message{Role: "assistant", Content: "command succeeded"},
		providerHistoryAssistantToolCall("call_latest", "read_file"),
		providerHistoryToolResult("call_latest", "read_file", "latest read"),
		api.Message{Role: "assistant", Content: "done"},
	}}
	raw := api.CloneMessages(agent.History)

	result := agent.buildProviderHistoryProjection(ProviderHistoryReductionPolicy{Mode: ProviderHistoryReductionApply})

	assertProviderHistoryCommandProjectionUnchanged(t, result, raw)
	if result.Report.ResponsesChainDisabled {
		t.Fatalf("ResponsesChainDisabled = true, want false without command replacements")
	}
	commandReport := result.Report.CommandEditDryRun
	if commandReport.CommandCandidates != 5 {
		t.Fatalf("CommandCandidates = %d, want five unreplaced command diagnostics", commandReport.CommandCandidates)
	}
	assertProviderHistoryCommandReportNoReplacement(t, commandReport)
	wantReasons := map[string]int{
		"git_diff_output":        1,
		"test_failure_output":    1,
		"build_failure_output":   1,
		"command_exit_nonzero":   1,
		"command_success_output": 1,
	}
	if !reflect.DeepEqual(commandReport.CandidateReasonCounts, wantReasons) {
		t.Fatalf("CandidateReasonCounts = %#v, want %#v", commandReport.CandidateReasonCounts, wantReasons)
	}
}

func TestProviderHistoryCommandEditApplyKeepsAmbiguousBuildAndLintOutputs(t *testing.T) {
	buildOutput := providerHistoryLargeCommandOutput("Build completed with errors\n")
	lintOutput := providerHistoryLargeCommandOutput("0 errors, 3 warnings\n")
	lintErrorCountOutput := providerHistoryLargeCommandOutput("10 errors\n")
	lintWarningZeroExitOutput := providerHistoryLargeCommandOutput("0 errors, 3 warnings\nProcess exited with code 0\n")
	lintUncountedErrorOutput := providerHistoryLargeCommandOutput("error: unexpected console statement\nProcess exited with code 0\n")
	lintWarningsFoundOutput := providerHistoryLargeCommandOutput("Warnings found\nProcess exited with code 0\n")
	lintIssueFoundOutput := providerHistoryLargeCommandOutput("1 issue found\nProcess exited with code 0\n")
	lintIssuesOutput := providerHistoryLargeCommandOutput("2 issues\nProcess exited with code 0\n")
	agent := &Agent{History: []api.Message{
		providerHistoryAssistantToolCalls(providerHistoryToolCallWithJSONArguments(t, "call_build_errors", "bash", map[string]string{"command": "npm run build"})),
		providerHistoryToolResult("call_build_errors", "bash", buildOutput),
		api.Message{Role: "assistant", Content: "build had errors"},
		providerHistoryAssistantToolCalls(providerHistoryToolCallWithJSONArguments(t, "call_lint_warnings", "bash", map[string]string{"command": providerHistorySuccessfulLintCommand})),
		providerHistoryToolResult("call_lint_warnings", "bash", lintOutput),
		api.Message{Role: "assistant", Content: "lint had warnings"},
		providerHistoryAssistantToolCalls(providerHistoryToolCallWithJSONArguments(t, "call_lint_errors", "bash", map[string]string{"command": providerHistorySuccessfulLintCommand})),
		providerHistoryToolResult("call_lint_errors", "bash", lintErrorCountOutput),
		api.Message{Role: "assistant", Content: "lint had errors"},
		providerHistoryAssistantToolCalls(providerHistoryToolCallWithJSONArguments(t, "call_lint_warning_zero_exit", "bash", map[string]string{"command": providerHistorySuccessfulLintCommand})),
		providerHistoryToolResult("call_lint_warning_zero_exit", "bash", lintWarningZeroExitOutput),
		api.Message{Role: "assistant", Content: "lint had warnings with zero exit"},
		providerHistoryAssistantToolCalls(providerHistoryToolCallWithJSONArguments(t, "call_lint_uncounted_error", "bash", map[string]string{"command": providerHistorySuccessfulLintCommand})),
		providerHistoryToolResult("call_lint_uncounted_error", "bash", lintUncountedErrorOutput),
		api.Message{Role: "assistant", Content: "lint had uncounted error"},
		providerHistoryAssistantToolCalls(providerHistoryToolCallWithJSONArguments(t, "call_lint_warnings_found", "bash", map[string]string{"command": providerHistorySuccessfulLintCommand})),
		providerHistoryToolResult("call_lint_warnings_found", "bash", lintWarningsFoundOutput),
		api.Message{Role: "assistant", Content: "lint had warnings found"},
		providerHistoryAssistantToolCalls(providerHistoryToolCallWithJSONArguments(t, "call_lint_issue_found", "bash", map[string]string{"command": providerHistorySuccessfulLintCommand})),
		providerHistoryToolResult("call_lint_issue_found", "bash", lintIssueFoundOutput),
		api.Message{Role: "assistant", Content: "lint had issue found"},
		providerHistoryAssistantToolCalls(providerHistoryToolCallWithJSONArguments(t, "call_lint_issues", "bash", map[string]string{"command": providerHistorySuccessfulLintCommand})),
		providerHistoryToolResult("call_lint_issues", "bash", lintIssuesOutput),
		api.Message{Role: "assistant", Content: "lint had issues"},
		providerHistoryAssistantToolCall("call_latest", "read_file"),
		providerHistoryToolResult("call_latest", "read_file", "latest read"),
		api.Message{Role: "assistant", Content: "done"},
	}}
	raw := api.CloneMessages(agent.History)

	result := agent.buildProviderHistoryProjection(ProviderHistoryReductionPolicy{Mode: ProviderHistoryReductionApply})

	assertProviderHistoryCommandProjectionUnchanged(t, result, raw)
	if result.Report.ResponsesChainDisabled {
		t.Fatal("ResponsesChainDisabled = true, want false without command replacements")
	}
	commandReport := result.Report.CommandEditDryRun
	if commandReport.CommandCandidates != 8 {
		t.Fatalf("CommandCandidates = %d, want eight unreplaced command diagnostics", commandReport.CommandCandidates)
	}
	assertProviderHistoryCommandReportNoReplacement(t, commandReport)
	wantReasons := map[string]int{
		"build_failure_output":   1,
		"command_success_output": 7,
	}
	if !reflect.DeepEqual(commandReport.CandidateReasonCounts, wantReasons) {
		t.Fatalf("CandidateReasonCounts = %#v, want %#v", commandReport.CandidateReasonCounts, wantReasons)
	}
}

func TestProviderHistoryCommandEditApplyKeepsUnprovenSuccessfulCommandOutputs(t *testing.T) {
	agent := &Agent{History: []api.Message{
		providerHistoryAssistantToolCalls(providerHistoryToolCallWithJSONArguments(t, "call_interrupted", "bash", map[string]string{"command": providerHistorySuccessfulTestCommand})),
		providerHistoryToolResult("call_interrupted", "bash", providerHistoryLargeInterruptedCommandOutput()),
		api.Message{Role: "assistant", Content: "tests interrupted"},
		providerHistoryAssistantToolCalls(providerHistoryToolCallWithJSONArguments(t, "call_grep", "bash", map[string]string{"command": "grep 'go test' README.md"})),
		providerHistoryToolResult("call_grep", "bash", providerHistoryLargeSuccessfulTestOutput()),
		api.Message{Role: "assistant", Content: "grep done"},
		providerHistoryAssistantToolCalls(providerHistoryToolCallWithJSONArguments(t, "call_no_evidence", "bash", map[string]string{"command": providerHistorySuccessfulTestCommand})),
		providerHistoryToolResult("call_no_evidence", "bash", providerHistoryLargeCommandOutput("running tests\n")),
		api.Message{Role: "assistant", Content: "tests done"},
		providerHistoryAssistantToolCalls(providerHistoryToolCallWithJSONArguments(t, "call_compound", "bash", map[string]string{"command": providerHistorySuccessfulTestCommand + " && cat coverage.out"})),
		providerHistoryToolResult("call_compound", "bash", providerHistoryLargeSuccessfulTestOutput()),
		api.Message{Role: "assistant", Content: "compound command done"},
		providerHistoryAssistantToolCalls(providerHistoryToolCallWithJSONArguments(t, "call_piped", "bash", map[string]string{"command": providerHistorySuccessfulLintCommand + " | tee lint.log"})),
		providerHistoryToolResult("call_piped", "bash", providerHistoryLargeSuccessfulLintOutput()),
		api.Message{Role: "assistant", Content: "piped command done"},
		providerHistoryAssistantToolCalls(providerHistoryToolCallWithJSONArguments(t, "call_redirected", "bash", map[string]string{"command": providerHistorySuccessfulTestCommand + " > coverage.out"})),
		providerHistoryToolResult("call_redirected", "bash", providerHistoryLargeSuccessfulTestOutput()),
		api.Message{Role: "assistant", Content: "redirected command done"},
		providerHistoryAssistantToolCalls(providerHistoryToolCallWithJSONArguments(t, "call_substitution", "bash", map[string]string{"command": `go test "$(go list ./...)"`})),
		providerHistoryToolResult("call_substitution", "bash", providerHistoryLargeSuccessfulTestOutput()),
		api.Message{Role: "assistant", Content: "substitution command done"},
		providerHistoryAssistantToolCalls(providerHistoryToolCallWithJSONArguments(t, "call_backtick_substitution", "bash", map[string]string{"command": "go test \"`go list ./...`\""})),
		providerHistoryToolResult("call_backtick_substitution", "bash", providerHistoryLargeSuccessfulTestOutput()),
		api.Message{Role: "assistant", Content: "backtick substitution command done"},
		providerHistoryAssistantToolCall("call_latest", "read_file"),
		providerHistoryToolResult("call_latest", "read_file", "latest read"),
		api.Message{Role: "assistant", Content: "done"},
	}}
	raw := api.CloneMessages(agent.History)

	result := agent.buildProviderHistoryProjection(ProviderHistoryReductionPolicy{Mode: ProviderHistoryReductionApply})

	assertProviderHistoryCommandProjectionUnchanged(t, result, raw)
	if result.Report.ResponsesChainDisabled {
		t.Fatal("ResponsesChainDisabled = true, want false without proven successful command replacement")
	}
	commandReport := result.Report.CommandEditDryRun
	if commandReport.CommandCandidates != 8 {
		t.Fatalf("CommandCandidates = %d, want eight command diagnostics", commandReport.CommandCandidates)
	}
	assertProviderHistoryCommandReportNoReplacement(t, commandReport)
	if got := commandReport.CandidateReasonCounts["command_success_output"]; got != 8 {
		t.Fatalf("CandidateReasonCounts = %#v, want eight generic command diagnostics without success replacement", commandReport.CandidateReasonCounts)
	}
}

func TestProviderHistoryCommandEditApplyKeepsMixedFailedTestOutputs(t *testing.T) {
	mochaOutput := providerHistoryLargeCommandOutput("1 passing\n1 failing\n")
	cargoOutput := providerHistoryLargeCommandOutput("test result: ok. 1 passed; 0 failed\nfailures:\n    suite_x\ntest result: FAILED. 0 passed; 1 failed\n")
	pytestOutput := providerHistoryLargeCommandOutput("1 failed, 1 passed in 0.12s\n")
	agent := &Agent{History: []api.Message{
		providerHistoryAssistantToolCalls(providerHistoryToolCallWithJSONArguments(t, "call_mocha", "bash", map[string]string{"command": "npm test"})),
		providerHistoryToolResult("call_mocha", "bash", mochaOutput),
		api.Message{Role: "assistant", Content: "mocha tests failed"},
		providerHistoryAssistantToolCalls(providerHistoryToolCallWithJSONArguments(t, "call_cargo", "bash", map[string]string{"command": "cargo test"})),
		providerHistoryToolResult("call_cargo", "bash", cargoOutput),
		api.Message{Role: "assistant", Content: "cargo tests failed"},
		providerHistoryAssistantToolCalls(providerHistoryToolCallWithJSONArguments(t, "call_pytest", "bash", map[string]string{"command": "pytest"})),
		providerHistoryToolResult("call_pytest", "bash", pytestOutput),
		api.Message{Role: "assistant", Content: "pytest tests failed"},
		providerHistoryAssistantToolCall("call_latest", "read_file"),
		providerHistoryToolResult("call_latest", "read_file", "latest read"),
		api.Message{Role: "assistant", Content: "done"},
	}}
	raw := api.CloneMessages(agent.History)

	result := agent.buildProviderHistoryProjection(ProviderHistoryReductionPolicy{Mode: ProviderHistoryReductionApply})

	assertProviderHistoryCommandProjectionUnchanged(t, result, raw)
	if result.Report.ResponsesChainDisabled {
		t.Fatal("ResponsesChainDisabled = true, want false without command replacements")
	}
	commandReport := result.Report.CommandEditDryRun
	if commandReport.CommandCandidates != 3 {
		t.Fatalf("CommandCandidates = %d, want three failed test diagnostics", commandReport.CommandCandidates)
	}
	assertProviderHistoryCommandReportNoReplacement(t, commandReport)
	if got := commandReport.CandidateReasonCounts["test_failure_output"]; got != 3 {
		t.Fatalf("CandidateReasonCounts = %#v, want three failed test diagnostics", commandReport.CandidateReasonCounts)
	}
}

func TestProviderHistoryCommandEditApplyKeepsCommandBoundaryOutputs(t *testing.T) {
	largeSuccessOutput := providerHistoryLargeSuccessfulTestOutput()
	tests := []struct {
		name           string
		history        []api.Message
		wantKeepReason string
	}{
		{
			name: "latest tool result",
			history: []api.Message{
				providerHistoryAssistantToolCalls(providerHistoryToolCallWithJSONArguments(t, "call_latest_cmd", "bash", map[string]string{"command": providerHistorySuccessfulTestCommand})),
				providerHistoryToolResult("call_latest_cmd", "bash", largeSuccessOutput),
				api.Message{Role: "assistant", Content: "done"},
			},
			wantKeepReason: "latest_tool_result",
		},
		{
			name: "trailing tool suffix",
			history: []api.Message{
				api.Message{Role: "assistant", Content: "before tail"},
				providerHistoryAssistantToolCalls(providerHistoryToolCallWithJSONArguments(t, "call_tail_cmd", "bash", map[string]string{"command": providerHistorySuccessfulTestCommand})),
				providerHistoryToolResult("call_tail_cmd", "bash", largeSuccessOutput),
			},
			wantKeepReason: "trailing_tool_suffix",
		},
		{
			name: "invalid linkage",
			history: []api.Message{
				providerHistoryAssistantToolCalls(providerHistoryToolCallWithJSONArguments(t, "call_mismatch", "bash", map[string]string{"command": providerHistorySuccessfulTestCommand})),
				providerHistoryToolResult("call_mismatch", "read_file", largeSuccessOutput),
				api.Message{Role: "assistant", Content: "after mismatch"},
				providerHistoryAssistantToolCall("call_latest", "read_file"),
				providerHistoryToolResult("call_latest", "read_file", "latest"),
				api.Message{Role: "assistant", Content: "done"},
			},
			wantKeepReason: "mismatched_tool_name",
		},
		{
			name: "no later assistant",
			history: []api.Message{
				providerHistoryAssistantToolCalls(providerHistoryToolCallWithJSONArguments(t, "call_no_later", "bash", map[string]string{"command": providerHistorySuccessfulTestCommand})),
				providerHistoryToolResult("call_no_later", "bash", largeSuccessOutput),
				api.Message{Role: "user", Content: "next"},
				providerHistoryToolResult("call_other", "read_file", "later non-assistant tool result"),
			},
			wantKeepReason: "no_later_assistant_message",
		},
		{
			name: "empty command output",
			history: []api.Message{
				providerHistoryAssistantToolCalls(providerHistoryToolCallWithJSONArguments(t, "call_empty", "bash", map[string]string{"command": providerHistorySuccessfulTestCommand})),
				providerHistoryToolResult("call_empty", "bash", ""),
				api.Message{Role: "assistant", Content: "after empty"},
				providerHistoryAssistantToolCall("call_latest", "read_file"),
				providerHistoryToolResult("call_latest", "read_file", "latest"),
				api.Message{Role: "assistant", Content: "done"},
			},
			wantKeepReason: "empty_command_output",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			agent := &Agent{History: tt.history}
			raw := api.CloneMessages(agent.History)

			result := agent.buildProviderHistoryProjection(ProviderHistoryReductionPolicy{Mode: ProviderHistoryReductionApply})

			assertProviderHistoryCommandProjectionUnchanged(t, result, raw)
			commandReport := result.Report.CommandEditDryRun
			assertProviderHistoryCommandReportNoReplacement(t, commandReport)
			if got := commandReport.KeptReasonCounts[tt.wantKeepReason]; got != 1 {
				t.Fatalf("KeptReasonCounts[%q] = %d in %#v, want 1", tt.wantKeepReason, got, commandReport.KeptReasonCounts)
			}
			if result.Report.ResponsesChainDisabled {
				t.Fatalf("ResponsesChainDisabled = true for %s, want false without replacement", tt.name)
			}
		})
	}
}

func TestProviderHistoryCommandEditApplyPreservesInferredCommandToolName(t *testing.T) {
	commandOutput := providerHistoryLargeSuccessfulTestOutput()
	agent := &Agent{History: []api.Message{
		providerHistoryAssistantToolCalls(providerHistoryToolCallWithJSONArguments(t, "call_missing_tool_name", "bash", map[string]string{"command": providerHistorySuccessfulTestCommand})),
		providerHistoryToolResult("call_missing_tool_name", "", commandOutput),
		api.Message{Role: "assistant", Content: "tests passed"},
		providerHistoryAssistantToolCall("call_latest", "read_file"),
		providerHistoryToolResult("call_latest", "read_file", "latest read"),
		api.Message{Role: "assistant", Content: "done"},
	}}
	raw := api.CloneMessages(agent.History)

	result := agent.buildProviderHistoryProjection(ProviderHistoryReductionPolicy{Mode: ProviderHistoryReductionApply})

	assertProviderHistoryCommandReplacement(t, result, 1, commandOutput, providerHistorySuccessfulTestReplacementLabel)
	if got := result.History[1].ToolName; got != "bash" {
		t.Fatalf("projected ToolName = %q, want inferred bash", got)
	}
	if !reflect.DeepEqual(agent.History, raw) {
		t.Fatalf("Agent.History changed after inferred command tool name replacement:\n got %#v\nwant %#v", agent.History, raw)
	}
	if agent.History[1].ToolName != "" {
		t.Fatalf("raw Agent.History[1].ToolName = %q, want unchanged empty", agent.History[1].ToolName)
	}
}

func providerHistoryCommandEditDryRunReportForTest(t *testing.T, history ...api.Message) ProviderHistoryCommandEditDryRunReport {
	t.Helper()
	agent := &Agent{History: history}
	return agent.buildProviderHistoryProjection(ProviderHistoryReductionPolicy{Mode: ProviderHistoryReductionDryRun}).Report.CommandEditDryRun
}
