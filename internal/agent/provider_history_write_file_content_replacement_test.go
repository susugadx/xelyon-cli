package agent

import (
	"reflect"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/api"
)

func TestProviderHistoryCommandEditApplyReplacesOldSuccessfulWriteFileContent(t *testing.T) {
	path := "generated/write.go"
	content := providerHistoryLargeWriteFileContent()
	args := providerHistoryWriteFileArguments(t, path, content)
	success := providerHistoryWriteFileSuccess(content, path)
	agent := &Agent{History: []api.Message{
		providerHistoryAssistantToolCalls(providerHistoryToolCallWithArguments("call_write", "write_file", args)),
		providerHistoryToolResult("call_write", "write_file", success),
		api.Message{Role: "assistant", Content: "write done"},
		providerHistoryAssistantToolCall("call_latest", "read_file"),
		providerHistoryToolResult("call_latest", "read_file", "latest read"),
		api.Message{Role: "assistant", Content: "done"},
	}}
	raw := api.CloneMessages(agent.History)

	result := agent.buildProviderHistoryProjection(ProviderHistoryReductionPolicy{Mode: ProviderHistoryReductionApply})

	assertProviderHistoryWriteFileContentReplacement(t, result.History[0].ToolCalls[0].Function.Arguments, path, content)
	if result.History[1].Content != success {
		t.Fatalf("write_file tool result content = %q, want raw success output", result.History[1].Content)
	}
	if !reflect.DeepEqual(agent.History, raw) {
		t.Fatalf("Agent.History changed after write_file.content replacement:\n got %#v\nwant %#v", agent.History, raw)
	}

	report := result.Report
	if report.ReplacedCount != 0 || !report.ResponsesChainDisabled {
		t.Fatalf("projection report = %#v, want edit-arg-only replacement to disable response chain", report)
	}
	if report.EstimatedSavedBytes != 0 || report.ApproxSavedTokens != 0 {
		t.Fatalf("top-level content savings = bytes %d tokens %d, want unchanged for arg-only replacement", report.EstimatedSavedBytes, report.ApproxSavedTokens)
	}
	commandReport := report.CommandEditDryRun
	if commandReport.ReplacementStatus != providerHistoryCommandEditReplacementStatusPartialApply ||
		commandReport.EditArgCandidates != 1 ||
		commandReport.EditArgReplacedCount != 1 ||
		commandReport.EditArgReplacementSavedBytes <= 0 ||
		commandReport.ApproxEditArgReplacementSavedTokens < providerHistoryEditArgReplacementMinSavedTokens {
		t.Fatalf("CommandEditDryRun = %#v, want one write_file.content replacement with savings", commandReport)
	}
	if got := commandReport.CandidateReasonCounts["write_file_content"]; got != 1 {
		t.Fatalf("CandidateReasonCounts = %#v, want write_file_content:1", commandReport.CandidateReasonCounts)
	}
}

func TestProviderHistoryWriteFileContentApplyKeepsNonReplaceableCases(t *testing.T) {
	path := "generated/write.go"
	content := providerHistoryLargeWriteFileContent()
	validArgs := providerHistoryWriteFileArguments(t, path, content)
	success := providerHistoryWriteFileSuccess(content, path)

	tests := []struct {
		name    string
		history []api.Message
	}{
		{
			name: "latest tool result",
			history: []api.Message{
				providerHistoryAssistantToolCalls(providerHistoryToolCallWithArguments("call_write", "write_file", validArgs)),
				providerHistoryToolResult("call_write", "write_file", success),
				api.Message{Role: "assistant", Content: "write done"},
			},
		},
		{
			name: "trailing tool suffix",
			history: []api.Message{
				api.Message{Role: "assistant", Content: "before tail"},
				providerHistoryAssistantToolCalls(providerHistoryToolCallWithArguments("call_write", "write_file", validArgs)),
				providerHistoryToolResult("call_write", "write_file", success),
			},
		},
		{
			name: "no later assistant",
			history: []api.Message{
				providerHistoryAssistantToolCalls(providerHistoryToolCallWithArguments("call_write", "write_file", validArgs)),
				providerHistoryToolResult("call_write", "write_file", success),
				api.Message{Role: "user", Content: "next"},
				providerHistoryToolResult("call_other", "read_file", "later tool result"),
			},
		},
		{
			name: "invalid linkage",
			history: []api.Message{
				providerHistoryAssistantToolCalls(providerHistoryToolCallWithArguments("call_write", "write_file", validArgs)),
				providerHistoryToolResult("call_write", "read_file", success),
				api.Message{Role: "assistant", Content: "after mismatch"},
				providerHistoryAssistantToolCall("call_latest", "read_file"),
				providerHistoryToolResult("call_latest", "read_file", "latest read"),
				api.Message{Role: "assistant", Content: "done"},
			},
		},
		{
			name:    "missing path",
			history: providerHistoryWriteFileReplacementHistory(t, "call_write", providerHistoryJSONArguments(t, map[string]string{"content": content}), success),
		},
		{
			name:    "missing content",
			history: providerHistoryWriteFileReplacementHistory(t, "call_write", providerHistoryJSONArguments(t, map[string]string{"path": path}), success),
		},
		{
			name:    "invalid JSON",
			history: providerHistoryWriteFileReplacementHistory(t, "call_write", `{"path":`, success),
		},
		{
			name:    "unsafe path",
			history: providerHistoryWriteFileReplacementHistory(t, "call_write", providerHistoryWriteFileArguments(t, "../outside.go", content), providerHistoryWriteFileSuccess(content, "../outside.go")),
		},
		{
			name:    "non-success result",
			history: providerHistoryWriteFileReplacementHistory(t, "call_write", validArgs, "Error writing file: permission denied"),
		},
		{
			name:    "small content below threshold",
			history: providerHistoryWriteFileReplacementHistory(t, "call_write", providerHistoryWriteFileArguments(t, path, "tiny"), providerHistoryWriteFileSuccess("tiny", path)),
		},
		{
			name:    "result path mismatch",
			history: providerHistoryWriteFileReplacementHistory(t, "call_write", validArgs, providerHistoryWriteFileSuccess(content, "generated/other.go")),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			agent := &Agent{History: tt.history}
			raw := api.CloneMessages(agent.History)

			result := agent.buildProviderHistoryProjection(ProviderHistoryReductionPolicy{Mode: ProviderHistoryReductionApply})

			if !reflect.DeepEqual(result.History, raw) {
				t.Fatalf("apply projection changed non-replaceable write_file history:\n got %#v\nwant %#v", result.History, raw)
			}
			if result.Report.CommandEditDryRun.EditArgReplacedCount != 0 || result.Report.ResponsesChainDisabled {
				t.Fatalf("report = %#v, want no write_file.content replacement", result.Report)
			}
		})
	}
}

func TestProviderHistoryEditArgApplyKeepsUnsuccessfulPatchReplaceAndDeleteFileCandidateOnlyRaw(t *testing.T) {
	patch := strings.Repeat("*** Begin Patch\n*** Update File: a.go\n+line\n*** End Patch\n", 40)
	oldStr := strings.Repeat("old line\n", 180)
	newStr := strings.Repeat("new line\n", 180)
	deletePath := "generated/delete.txt"
	agent := &Agent{History: []api.Message{
		providerHistoryAssistantToolCalls(
			providerHistoryToolCallWithJSONArguments(t, "call_patch", "apply_patch", map[string]string{"patch": patch}),
			providerHistoryToolCallWithJSONArguments(t, "call_replace", "str_replace", map[string]string{"path": "b.go", "old_str": oldStr, "new_str": newStr}),
			providerHistoryToolCallWithJSONArguments(t, "call_delete", "delete_file", map[string]string{"path": deletePath}),
		),
		providerHistoryToolResult("call_patch", "apply_patch", "patched"),
		providerHistoryToolResult("call_replace", "str_replace", "replaced"),
		providerHistoryToolResult("call_delete", "delete_file", "deleted"),
		api.Message{Role: "assistant", Content: "edits done"},
		providerHistoryAssistantToolCall("call_latest", "read_file"),
		providerHistoryToolResult("call_latest", "read_file", "latest read"),
		api.Message{Role: "assistant", Content: "done"},
	}}
	raw := api.CloneMessages(agent.History)

	result := agent.buildProviderHistoryProjection(ProviderHistoryReductionPolicy{Mode: ProviderHistoryReductionApply})

	if !reflect.DeepEqual(result.History, raw) {
		t.Fatalf("apply projection changed unsuccessful/candidate-only edit tool args:\n got %#v\nwant %#v", result.History, raw)
	}
	if result.Report.CommandEditDryRun.EditArgCandidates != 3 ||
		result.Report.CommandEditDryRun.EditArgReplacedCount != 0 ||
		result.Report.ResponsesChainDisabled {
		t.Fatalf("report = %#v, want edit diagnostics without replacement or response chain disable", result.Report)
	}
	wantReasons := map[string]int{
		"apply_patch_patch":   1,
		"str_replace_strings": 1,
		"delete_file_path":    1,
	}
	if !reflect.DeepEqual(result.Report.CommandEditDryRun.CandidateReasonCounts, wantReasons) {
		t.Fatalf("CandidateReasonCounts = %#v, want %#v", result.Report.CommandEditDryRun.CandidateReasonCounts, wantReasons)
	}
}

func TestProviderHistoryWriteFileContentReplacementUpdatesAnthropicProviderState(t *testing.T) {
	path := "generated/claude.go"
	content := providerHistoryLargeWriteFileContent()
	args := providerHistoryWriteFileArguments(t, path, content)
	assistant := providerHistoryAssistantToolCalls(providerHistoryToolCallWithArguments("call_write", "write_file", args))
	assistant.SetAnthropicContentBlocks([]api.AnthropicContentBlock{
		{Type: "thinking", Thinking: "private thought", Signature: "sig"},
		{Type: "tool_use", ID: "call_write", Name: "write_file", Input: map[string]any{"path": path, "content": content}},
	})
	agent := &Agent{History: []api.Message{
		assistant,
		providerHistoryToolResult("call_write", "write_file", providerHistoryWriteFileSuccess(content, path)),
		api.Message{Role: "assistant", Content: "write done"},
		providerHistoryAssistantToolCall("call_latest", "read_file"),
		providerHistoryToolResult("call_latest", "read_file", "latest read"),
		api.Message{Role: "assistant", Content: "done"},
	}}
	raw := api.CloneMessages(agent.History)

	result := agent.buildProviderHistoryProjection(ProviderHistoryReductionPolicy{Mode: ProviderHistoryReductionApply})

	replacement := assertProviderHistoryWriteFileContentReplacement(t, result.History[0].ToolCalls[0].Function.Arguments, path, content)
	blocks := result.History[0].AnthropicContentBlocks()
	if len(blocks) != 2 || blocks[1].Input["content"] != replacement || blocks[1].Input["path"] != path {
		t.Fatalf("projected AnthropicContentBlocks = %#v, want matching write_file.content replacement", blocks)
	}
	if !reflect.DeepEqual(agent.History, raw) {
		t.Fatalf("Agent.History changed after Anthropic write_file projection:\n got %#v\nwant %#v", agent.History, raw)
	}
}

func TestProviderHistoryWriteFileContentReplacementSkipsWhenAnthropicProviderStateCannotUpdate(t *testing.T) {
	path := "generated/claude.go"
	content := providerHistoryLargeWriteFileContent()
	args := providerHistoryWriteFileArguments(t, path, content)
	assistant := providerHistoryAssistantToolCalls(providerHistoryToolCallWithArguments("call_write", "write_file", args))
	assistant.SetAnthropicContentBlocks([]api.AnthropicContentBlock{
		{Type: "tool_use", ID: "call_write", Name: "write_file", Input: map[string]any{"path": path, "content": "different raw content"}},
	})
	agent := &Agent{History: []api.Message{
		assistant,
		providerHistoryToolResult("call_write", "write_file", providerHistoryWriteFileSuccess(content, path)),
		api.Message{Role: "assistant", Content: "write done"},
		providerHistoryAssistantToolCall("call_latest", "read_file"),
		providerHistoryToolResult("call_latest", "read_file", "latest read"),
		api.Message{Role: "assistant", Content: "done"},
	}}
	raw := api.CloneMessages(agent.History)

	result := agent.buildProviderHistoryProjection(ProviderHistoryReductionPolicy{Mode: ProviderHistoryReductionApply})

	if !reflect.DeepEqual(result.History, raw) {
		t.Fatalf("apply projection changed history despite stale Anthropic provider state:\n got %#v\nwant %#v", result.History, raw)
	}
	if result.Report.CommandEditDryRun.EditArgReplacedCount != 0 || result.Report.ResponsesChainDisabled {
		t.Fatalf("report = %#v, want replacement skipped when provider state cannot be updated", result.Report)
	}
}

func TestProviderHistoryWriteFileContentReplacementDoesNotRequireActiveContextTransport(t *testing.T) {
	writePath := "generated/unsupported.go"
	writeContent := providerHistoryLargeWriteFileContent()
	writeArgs := providerHistoryWriteFileArguments(t, writePath, writeContent)
	readContent := phase5DOutput("old read that needs active context")
	agent := &Agent{
		Runtime: &AgentRuntime{
			TaskLedger: providerHistoryTaskLedgerWithEvidence(t,
				providerHistoryEvidenceItem{ToolName: "read_file", ToolCallID: "call_read", Path: "README.md", StartLine: 1},
			),
			Options: RuntimeOptions{EnableProviderHistoryRehydrateContext: true},
		},
		History: []api.Message{
			providerHistoryAssistantToolCall("call_read", "read_file"),
			providerHistoryToolResult("call_read", "read_file", readContent),
			api.Message{Role: "assistant", Content: "read done"},
			providerHistoryAssistantToolCalls(providerHistoryToolCallWithArguments("call_write", "write_file", writeArgs)),
			providerHistoryToolResult("call_write", "write_file", providerHistoryWriteFileSuccess(writeContent, writePath)),
			api.Message{Role: "assistant", Content: "write done"},
			providerHistoryAssistantToolCall("call_latest", "read_file"),
			providerHistoryToolResult("call_latest", "read_file", "latest read"),
			api.Message{Role: "assistant", Content: "done"},
		},
	}

	result := agent.buildProviderHistoryProjection(ProviderHistoryReductionPolicy{Mode: ProviderHistoryReductionApply})

	if result.History[1].Content != readContent {
		t.Fatalf("read_file content = %q, want raw when active context transport is unsupported", result.History[1].Content)
	}
	assertProviderHistoryWriteFileContentReplacement(t, result.History[3].ToolCalls[0].Function.Arguments, writePath, writeContent)
	if got := countKeptByToolCallIDAndReason(result.Report, "call_read", "active_context_transport_unsupported"); got != 1 {
		t.Fatalf("active_context_transport_unsupported keep count = %d, want 1", got)
	}
	if result.Report.CommandEditDryRun.EditArgReplacedCount != 1 || !result.Report.ResponsesChainDisabled {
		t.Fatalf("report = %#v, want write_file.content replacement despite unsupported active context transport", result.Report)
	}
}
