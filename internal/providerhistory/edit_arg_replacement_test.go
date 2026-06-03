package providerhistory

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/api"
)

func TestProjectReplacesOldApplyPatchAndStrReplaceArgsAfterSuccessfulMatchingResults(t *testing.T) {
	patchPath := "internal/providerhistory/edit_arg.go"
	replacePath := "internal/providerhistory/replace.go"
	patch := providerHistoryTestLargeApplyPatch(patchPath)
	oldStr := providerHistoryTestLargeStrReplaceText("old line")
	newStr := providerHistoryTestLargeStrReplaceText("new line")
	patchArgs := providerHistoryTestJSONAnyArguments(t, map[string]any{"patch": patch})
	replaceArgs := providerHistoryTestJSONAnyArguments(t, map[string]any{"path": replacePath, "old_str": oldStr, "new_str": newStr})
	patchSuccess := providerHistoryTestApplyPatchSuccess(nil, []string{patchPath}, nil)
	replaceSuccess := "Successfully replaced text in " + replacePath + " (lines 12-40 → 12-41)"
	history := []api.Message{
		providerHistoryTestAssistantToolCalls(
			providerHistoryTestToolCallWithArguments("call_patch", "apply_patch", patchArgs),
			providerHistoryTestToolCallWithArguments("call_replace", "str_replace", replaceArgs),
		),
		providerHistoryTestToolResult("call_patch", "apply_patch", patchSuccess),
		providerHistoryTestToolResult("call_replace", "str_replace", replaceSuccess),
		{Role: "assistant", Content: "edits completed"},
		providerHistoryTestAssistantToolCall("call_latest", "read_file"),
		providerHistoryTestToolResult("call_latest", "read_file", "latest read"),
		{Role: "assistant", Content: "done"},
	}
	raw := api.CloneMessages(history)

	result := Project(ProjectionInput{
		Messages: history,
		Policy:   Policy{Mode: Apply},
	})

	assertProviderHistoryTestApplyPatchReplacement(t, result.History[0].ToolCalls[0].Function.Arguments, patchPath, patch)
	assertProviderHistoryTestStrReplaceReplacement(t, result.History[0].ToolCalls[1].Function.Arguments, replacePath, oldStr, newStr)
	if result.History[1].Content != patchSuccess || result.History[2].Content != replaceSuccess {
		t.Fatalf("tool results changed to %q / %q, want raw success outputs", result.History[1].Content, result.History[2].Content)
	}
	if !reflect.DeepEqual(history, raw) {
		t.Fatalf("raw history changed after edit arg projection:\n got %#v\nwant %#v", history, raw)
	}
	report := result.Report.CommandEditDryRun
	if report.EditArgReplacedCount != 2 ||
		report.EditArgReplacementSavedBytes <= 0 ||
		report.ApproxEditArgReplacementSavedTokens < providerHistoryEditArgReplacementMinSavedTokens*2 ||
		!result.Report.ResponsesChainDisabled {
		t.Fatalf("report = %#v / top-level %#v, want two edit arg replacements with response chain disabled", report, result.Report)
	}
}

func TestProjectEditArgReplacementSyncsAnthropicProviderState(t *testing.T) {
	patchPath := "internal/providerhistory/claude_patch.go"
	replacePath := "internal/providerhistory/claude_replace.go"
	patch := providerHistoryTestLargeApplyPatch(patchPath)
	oldStr := providerHistoryTestLargeStrReplaceText("claude old")
	newStr := providerHistoryTestLargeStrReplaceText("claude new")
	patchArgs := providerHistoryTestJSONAnyArguments(t, map[string]any{"patch": patch})
	replaceArgs := providerHistoryTestJSONAnyArguments(t, map[string]any{"path": replacePath, "old_str": oldStr, "new_str": newStr})
	assistant := providerHistoryTestAssistantToolCalls(
		providerHistoryTestToolCallWithArguments("call_patch", "apply_patch", patchArgs),
		providerHistoryTestToolCallWithArguments("call_replace", "str_replace", replaceArgs),
	)
	assistant.SetAnthropicContentBlocks([]api.AnthropicContentBlock{
		{Type: "thinking", Thinking: "private thought", Signature: "sig"},
		{Type: "tool_use", ID: "call_patch", Name: "apply_patch", Input: map[string]any{"patch": patch}},
		{Type: "tool_use", ID: "call_replace", Name: "str_replace", Input: map[string]any{"path": replacePath, "old_str": oldStr, "new_str": newStr}},
	})
	history := []api.Message{
		assistant,
		providerHistoryTestToolResult("call_patch", "apply_patch", providerHistoryTestApplyPatchSuccess(nil, []string{patchPath}, nil)),
		providerHistoryTestToolResult("call_replace", "str_replace", "Successfully replaced text in "+replacePath+" (lines 1-20 → 1-22)"),
		{Role: "assistant", Content: "edits done"},
		providerHistoryTestAssistantToolCall("call_latest", "read_file"),
		providerHistoryTestToolResult("call_latest", "read_file", "latest read"),
		{Role: "assistant", Content: "done"},
	}
	raw := api.CloneMessages(history)

	result := Project(ProjectionInput{
		Messages: history,
		Policy:   Policy{Mode: Apply},
	})

	patchReplacement := providerHistoryTestStringArgument(t, result.History[0].ToolCalls[0].Function.Arguments, "patch")
	replaceFields := providerHistoryTestArgumentFields(t, result.History[0].ToolCalls[1].Function.Arguments)
	blocks := result.History[0].AnthropicContentBlocks()
	if len(blocks) != 3 ||
		blocks[1].Input["patch"] != patchReplacement ||
		blocks[2].Input["old_str"] != replaceFields["old_str"] ||
		blocks[2].Input["new_str"] != replaceFields["new_str"] {
		t.Fatalf("projected AnthropicContentBlocks = %#v, want matching edit arg replacements", blocks)
	}
	if !reflect.DeepEqual(history, raw) {
		t.Fatalf("raw history changed after Anthropic edit arg projection:\n got %#v\nwant %#v", history, raw)
	}
}

func TestProjectEditArgReplacementSkipsLineRangeStrReplaceToPreserveMode(t *testing.T) {
	replacePath := "internal/providerhistory/line_range.go"
	newStr := providerHistoryTestLargeStrReplaceText("line range new")
	args := providerHistoryTestJSONAnyArguments(t, map[string]any{
		"path":       replacePath,
		"old_str":    "",
		"new_str":    newStr,
		"start_line": "5",
		"end_line":   "8",
	})
	assistant := providerHistoryTestAssistantToolCalls(providerHistoryTestToolCallWithArguments("call_range", "str_replace", args))
	assistant.SetAnthropicContentBlocks([]api.AnthropicContentBlock{
		{
			Type: "tool_use",
			ID:   "call_range",
			Name: "str_replace",
			Input: map[string]any{
				"path":       replacePath,
				"old_str":    "",
				"new_str":    newStr,
				"start_line": "5",
				"end_line":   "8",
			},
		},
	})
	replaceSuccess := "Successfully replaced lines 5-8 in " + replacePath + " (new range: 5-10)"
	history := []api.Message{
		assistant,
		providerHistoryTestToolResult("call_range", "str_replace", replaceSuccess),
		{Role: "assistant", Content: "line range edit done"},
		providerHistoryTestAssistantToolCall("call_latest", "read_file"),
		providerHistoryTestToolResult("call_latest", "read_file", "latest read"),
		{Role: "assistant", Content: "done"},
	}
	raw := api.CloneMessages(history)

	result := Project(ProjectionInput{
		Messages: history,
		Policy:   Policy{Mode: Apply},
	})

	if !reflect.DeepEqual(result.History, raw) {
		t.Fatalf("line-range str_replace projection changed history:\n got %#v\nwant %#v", result.History, raw)
	}
	report := result.Report.CommandEditDryRun
	if report.EditArgCandidates != 1 ||
		report.CandidateReasonCounts["str_replace_strings"] != 1 ||
		report.EditArgReplacedCount != 0 ||
		result.Report.ResponsesChainDisabled {
		t.Fatalf("report = %#v / top-level %#v, want line-range candidate without replacement or chain disable", report, result.Report)
	}
}

func TestProjectEditArgReplacementSkipsWhenAnthropicProviderStateCannotSync(t *testing.T) {
	patchPath := "internal/providerhistory/claude_patch.go"
	patch := providerHistoryTestLargeApplyPatch(patchPath)
	args := providerHistoryTestJSONAnyArguments(t, map[string]any{"patch": patch})
	assistant := providerHistoryTestAssistantToolCalls(providerHistoryTestToolCallWithArguments("call_patch", "apply_patch", args))
	assistant.SetAnthropicContentBlocks([]api.AnthropicContentBlock{
		{Type: "tool_use", ID: "call_patch", Name: "apply_patch", Input: map[string]any{"patch": "different patch"}},
	})
	history := []api.Message{
		assistant,
		providerHistoryTestToolResult("call_patch", "apply_patch", providerHistoryTestApplyPatchSuccess(nil, []string{patchPath}, nil)),
		{Role: "assistant", Content: "patch done"},
		providerHistoryTestAssistantToolCall("call_latest", "read_file"),
		providerHistoryTestToolResult("call_latest", "read_file", "latest read"),
		{Role: "assistant", Content: "done"},
	}
	raw := api.CloneMessages(history)

	result := Project(ProjectionInput{
		Messages: history,
		Policy:   Policy{Mode: Apply},
	})

	if !reflect.DeepEqual(result.History, raw) {
		t.Fatalf("projection changed history despite stale Anthropic state:\n got %#v\nwant %#v", result.History, raw)
	}
	if result.Report.CommandEditDryRun.EditArgReplacedCount != 0 || result.Report.ResponsesChainDisabled {
		t.Fatalf("report = %#v, want replacement skipped when Anthropic state cannot sync", result.Report)
	}
}

func TestProjectDeleteFilePathCandidateOnlyDoesNotReplaceOrDisableResponseChain(t *testing.T) {
	deletePath := "generated/delete.txt"
	args := providerHistoryTestJSONAnyArguments(t, map[string]any{"path": deletePath})
	history := []api.Message{
		providerHistoryTestAssistantToolCalls(providerHistoryTestToolCallWithArguments("call_delete", "delete_file", args)),
		providerHistoryTestToolResult("call_delete", "delete_file", "Deleted "+deletePath),
		{Role: "assistant", Content: "delete done"},
		providerHistoryTestAssistantToolCall("call_latest", "read_file"),
		providerHistoryTestToolResult("call_latest", "read_file", "latest read"),
		{Role: "assistant", Content: "done"},
	}
	raw := api.CloneMessages(history)

	result := Project(ProjectionInput{
		Messages: history,
		Policy:   Policy{Mode: Apply},
	})

	if !reflect.DeepEqual(result.History, raw) {
		t.Fatalf("delete_file projection changed history:\n got %#v\nwant %#v", result.History, raw)
	}
	report := result.Report.CommandEditDryRun
	if report.EditArgCandidates != 1 ||
		report.CandidateReasonCounts["delete_file_path"] != 1 ||
		report.EditArgReplacedCount != 0 ||
		result.Report.ResponsesChainDisabled {
		t.Fatalf("report = %#v / top-level %#v, want candidate-only delete_file_path", report, result.Report)
	}
}

func providerHistoryTestLargeApplyPatch(path string) string {
	return strings.Repeat("*** Begin Patch\n*** Update File: "+path+"\n@@\n-old line\n+new line\n*** End Patch\n", 80)
}

func providerHistoryTestLargeStrReplaceText(prefix string) string {
	return strings.Repeat(prefix+" with enough content to pass replacement threshold\n", 220)
}

func providerHistoryTestApplyPatchSuccess(added, modified, deleted []string) string {
	return "✓ Patch applied successfully.\nAdded: " + providerHistoryTestApplyPatchPathList(added) +
		"\nModified: " + providerHistoryTestApplyPatchPathList(modified) +
		"\nDeleted: " + providerHistoryTestApplyPatchPathList(deleted)
}

func providerHistoryTestApplyPatchPathList(paths []string) string {
	if len(paths) == 0 {
		return "(none)"
	}
	return strings.Join(paths, ", ")
}

func providerHistoryTestJSONAnyArguments(t *testing.T, arguments map[string]any) string {
	t.Helper()
	data, err := json.Marshal(arguments)
	if err != nil {
		t.Fatalf("json.Marshal(%#v) error = %v", arguments, err)
	}
	return string(data)
}

func providerHistoryTestArgumentFields(t *testing.T, args string) map[string]any {
	t.Helper()
	var fields map[string]any
	if err := json.Unmarshal([]byte(args), &fields); err != nil {
		t.Fatalf("arguments are not valid JSON: %v\nargs=%s", err, args)
	}
	return fields
}

func providerHistoryTestStringArgument(t *testing.T, args, key string) string {
	t.Helper()
	fields := providerHistoryTestArgumentFields(t, args)
	value, ok := fields[key].(string)
	if !ok {
		t.Fatalf("argument %q = %#v, want string", key, fields[key])
	}
	return value
}

func assertProviderHistoryTestApplyPatchReplacement(t *testing.T, args, path, originalPatch string) {
	t.Helper()
	replacement := providerHistoryTestStringArgument(t, args, "patch")
	if replacement == "" || replacement == originalPatch || !strings.HasPrefix(replacement, "[omitted old apply_patch.patch; files="+path+"]") {
		t.Fatalf("projected apply_patch.patch = %q, want placeholder for %s", replacement, path)
	}
}

func assertProviderHistoryTestStrReplaceReplacement(t *testing.T, args, path, originalOldStr, originalNewStr string) {
	t.Helper()
	fields := providerHistoryTestArgumentFields(t, args)
	if fields["path"] != path {
		t.Fatalf("projected str_replace path = %#v, want %q", fields["path"], path)
	}
	oldReplacement, ok := fields["old_str"].(string)
	if !ok || oldReplacement == originalOldStr || !strings.Contains(oldReplacement, "str_replace.old_str") || !strings.Contains(oldReplacement, "path="+path) {
		t.Fatalf("projected old_str = %#v, want placeholder for %s", fields["old_str"], path)
	}
	newReplacement, ok := fields["new_str"].(string)
	if !ok || newReplacement == originalNewStr || !strings.Contains(newReplacement, "str_replace.new_str") || !strings.Contains(newReplacement, "path="+path) {
		t.Fatalf("projected new_str = %#v, want placeholder for %s", fields["new_str"], path)
	}
}
