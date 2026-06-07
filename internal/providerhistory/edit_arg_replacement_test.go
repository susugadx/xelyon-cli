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
	if result.Report.EstimatedSavedBytes != report.EditArgReplacementSavedBytes ||
		result.Report.ApproxSavedTokens != report.ApproxEditArgReplacementSavedTokens {
		t.Fatalf("top-level savings = bytes %d tokens %d, want apply_patch/str_replace edit savings %d/%d", result.Report.EstimatedSavedBytes, result.Report.ApproxSavedTokens, report.EditArgReplacementSavedBytes, report.ApproxEditArgReplacementSavedTokens)
	}
}

func TestProjectReplacesOldStrReplaceEditsArgAfterSuccessfulMatchingResult(t *testing.T) {
	replacePath := "internal/providerhistory/batch_replace.go"
	edits := providerHistoryTestLargeStrReplaceEdits(160)
	editsBytes, err := json.Marshal(edits)
	if err != nil {
		t.Fatalf("json.Marshal(edits) error = %v", err)
	}
	editsPayload := string(editsBytes)
	args := providerHistoryTestJSONAnyArguments(t, map[string]any{"path": replacePath, "edits": editsPayload})
	replaceSuccess := "Successfully applied 160 edits to " + replacePath
	history := []api.Message{
		providerHistoryTestAssistantToolCalls(providerHistoryTestToolCallWithArguments("call_edits", "str_replace", args)),
		providerHistoryTestToolResult("call_edits", "str_replace", replaceSuccess),
		{Role: "assistant", Content: "batch edit completed"},
		providerHistoryTestAssistantToolCall("call_latest", "read_file"),
		providerHistoryTestToolResult("call_latest", "read_file", "latest read"),
		{Role: "assistant", Content: "done"},
	}
	raw := api.CloneMessages(history)

	result := Project(ProjectionInput{
		Messages: history,
		Policy:   Policy{Mode: Apply},
	})

	projectedEdits := providerHistoryTestStringArgument(t, result.History[0].ToolCalls[0].Function.Arguments, "edits")
	if projectedEdits == editsPayload || !strings.Contains(projectedEdits, "[omitted old str_replace.edits[0].old_str; path="+replacePath+"]") {
		t.Fatalf("projected str_replace edits = %q, want compacted edit placeholders", projectedEdits)
	}
	if !reflect.DeepEqual(history, raw) {
		t.Fatalf("raw history changed after str_replace edits projection:\n got %#v\nwant %#v", history, raw)
	}
	report := result.Report.CommandEditDryRun
	if report.EditArgReplacedCount != 1 || report.EditArgReplacementSavedBytes <= 0 || !result.Report.ResponsesChainDisabled {
		t.Fatalf("report = %#v / top-level %#v, want one str_replace edits replacement", report, result.Report)
	}
	if result.Report.EstimatedSavedBytes != report.EditArgReplacementSavedBytes ||
		result.Report.ApproxSavedTokens != report.ApproxEditArgReplacementSavedTokens {
		t.Fatalf("top-level savings = bytes %d tokens %d, want str_replace edits savings %d/%d", result.Report.EstimatedSavedBytes, result.Report.ApproxSavedTokens, report.EditArgReplacementSavedBytes, report.ApproxEditArgReplacementSavedTokens)
	}
}

func TestProjectReplacesOldStrReplaceEditsArrayArgAfterSuccessfulMatchingResult(t *testing.T) {
	replacePath := "internal/providerhistory/batch_array_replace.go"
	edits := providerHistoryTestLargeStrReplaceEdits(2)
	args := providerHistoryTestJSONAnyArguments(t, map[string]any{"path": replacePath, "edits": edits})
	replaceSuccess := "Successfully applied 2 edits to " + replacePath
	history := []api.Message{
		providerHistoryTestAssistantToolCalls(providerHistoryTestToolCallWithArguments("call_edits_array", "str_replace", args)),
		providerHistoryTestToolResult("call_edits_array", "str_replace", replaceSuccess),
		{Role: "assistant", Content: "batch edit completed"},
		providerHistoryTestAssistantToolCall("call_latest", "read_file"),
		providerHistoryTestToolResult("call_latest", "read_file", "latest read"),
		{Role: "assistant", Content: "done"},
	}
	raw := api.CloneMessages(history)

	result := Project(ProjectionInput{
		Messages: history,
		Policy:   Policy{Mode: Apply},
	})

	fields := providerHistoryTestArgumentFields(t, result.History[0].ToolCalls[0].Function.Arguments)
	projectedEdits, ok := fields["edits"].([]any)
	if !ok || len(projectedEdits) != 2 {
		t.Fatalf("projected str_replace edits = %#v, want compacted JSON array", fields["edits"])
	}
	firstEdit, ok := projectedEdits[0].(map[string]any)
	oldReplacement, oldOK := firstEdit["old_str"].(string)
	if !ok || !oldOK || !strings.Contains(oldReplacement, "[omitted old str_replace.edits[0].old_str; path="+replacePath+"]") {
		t.Fatalf("projected first edit = %#v, want compacted edit placeholders", projectedEdits[0])
	}
	if !reflect.DeepEqual(history, raw) {
		t.Fatalf("raw history changed after str_replace edits array projection:\n got %#v\nwant %#v", history, raw)
	}
	report := result.Report.CommandEditDryRun
	if report.EditArgReplacedCount != 1 || report.EditArgReplacementSavedBytes <= 0 || !result.Report.ResponsesChainDisabled {
		t.Fatalf("report = %#v / top-level %#v, want one str_replace edits array replacement", report, result.Report)
	}
}

func TestProjectKeepsStrReplaceEditsArgsWhenSuccessfulEditCountMismatches(t *testing.T) {
	replacePath := "internal/providerhistory/batch_mismatch.go"
	edits := providerHistoryTestLargeStrReplaceEdits(2)
	editsBytes, err := json.Marshal(edits)
	if err != nil {
		t.Fatalf("json.Marshal(edits) error = %v", err)
	}
	editsPayload := string(editsBytes)
	args := providerHistoryTestJSONAnyArguments(t, map[string]any{"path": replacePath, "edits": editsPayload})
	replaceSuccess := "Successfully applied 1 edits to " + replacePath
	history := []api.Message{
		providerHistoryTestAssistantToolCalls(providerHistoryTestToolCallWithArguments("call_edits_mismatch", "str_replace", args)),
		providerHistoryTestToolResult("call_edits_mismatch", "str_replace", replaceSuccess),
		{Role: "assistant", Content: "batch edit completed"},
		providerHistoryTestAssistantToolCall("call_latest", "read_file"),
		providerHistoryTestToolResult("call_latest", "read_file", "latest read"),
		{Role: "assistant", Content: "done"},
	}
	raw := api.CloneMessages(history)

	result := Project(ProjectionInput{
		Messages: history,
		Policy:   Policy{Mode: Apply},
	})

	if result.History[0].ToolCalls[0].Function.Arguments != args {
		t.Fatalf("projected str_replace args = %q, want original args", result.History[0].ToolCalls[0].Function.Arguments)
	}
	if !reflect.DeepEqual(result.History, raw) {
		t.Fatalf("projection changed history for count mismatch:\n got %#v\nwant %#v", result.History, raw)
	}
	if !reflect.DeepEqual(history, raw) {
		t.Fatalf("raw history changed after count mismatch projection:\n got %#v\nwant %#v", history, raw)
	}
	report := result.Report.CommandEditDryRun
	if report.EditArgCandidates != 1 ||
		report.CandidateReasonCounts["str_replace_edits"] != 1 ||
		report.EditArgReplacedCount != 0 ||
		result.Report.ResponsesChainDisabled {
		t.Fatalf("report = %#v / top-level %#v, want str_replace edits candidate without replacement", report, result.Report)
	}
}

func TestProjectKeepsStrReplaceEditsArgsAndAnthropicInputWhenSuccessfulEditCountMismatches(t *testing.T) {
	replacePath := "internal/providerhistory/claude_batch_mismatch.go"
	edits := providerHistoryTestLargeStrReplaceEdits(2)
	args := providerHistoryTestJSONAnyArguments(t, map[string]any{"path": replacePath, "edits": edits})
	assistant := providerHistoryTestAssistantToolCalls(providerHistoryTestToolCallWithArguments("call_claude_edits", "str_replace", args))
	assistant.SetAnthropicContentBlocks([]api.AnthropicContentBlock{
		{Type: "thinking", Thinking: "private thought", Signature: "sig"},
		{Type: "tool_use", ID: "call_claude_edits", Name: "str_replace", Input: map[string]any{"path": replacePath, "edits": edits}},
	})
	replaceSuccess := "Successfully applied 1 edits to " + replacePath
	history := []api.Message{
		assistant,
		providerHistoryTestToolResult("call_claude_edits", "str_replace", replaceSuccess),
		{Role: "assistant", Content: "batch edit completed"},
		providerHistoryTestAssistantToolCall("call_latest", "read_file"),
		providerHistoryTestToolResult("call_latest", "read_file", "latest read"),
		{Role: "assistant", Content: "done"},
	}
	raw := api.CloneMessages(history)

	result := Project(ProjectionInput{
		Messages: history,
		Policy:   Policy{Mode: Apply},
	})

	if result.History[0].ToolCalls[0].Function.Arguments != args {
		t.Fatalf("projected ToolCalls arguments = %q, want original args", result.History[0].ToolCalls[0].Function.Arguments)
	}
	blocks := result.History[0].AnthropicContentBlocks()
	if !reflect.DeepEqual(blocks, raw[0].AnthropicContentBlocks()) {
		t.Fatalf("projected AnthropicContentBlocks = %#v, want original blocks %#v", blocks, raw[0].AnthropicContentBlocks())
	}
	if !reflect.DeepEqual(result.History, raw) {
		t.Fatalf("projection changed history for Anthropic count mismatch:\n got %#v\nwant %#v", result.History, raw)
	}
	if result.Report.CommandEditDryRun.EditArgReplacedCount != 0 || result.Report.ResponsesChainDisabled {
		t.Fatalf("report = %#v, want replacement skipped for Anthropic count mismatch", result.Report)
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
		report.KeptReasonCounts["delete_file_path_kept_context"] != 1 ||
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

func providerHistoryTestLargeStrReplaceEdits(count int) []map[string]string {
	edits := make([]map[string]string, 0, count)
	for i := 0; i < count; i++ {
		edits = append(edits, map[string]string{
			"old_str": strings.Repeat("old batch line with enough content to compact\n", 8),
			"new_str": strings.Repeat("new batch line with enough content to compact\n", 8),
		})
	}
	return edits
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
	if replacement == "" || replacement == originalPatch || !strings.HasPrefix(replacement, "[omitted old apply_patch.patch; files="+path+"; result=success]") {
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
