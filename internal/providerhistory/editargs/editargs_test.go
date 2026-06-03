package editargs_test

import (
	"encoding/json"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/providerhistory/editargs"
)

func TestProviderHistoryEditArgsPayloadDetectsEditArgumentKinds(t *testing.T) {
	writeContent := strings.Repeat("package main\n", 20)
	patch := strings.Repeat("*** Begin Patch\n*** Update File: a.go\n+line\n*** End Patch\n", 8)
	oldStr := strings.Repeat("old line\n", 12)
	newStr := strings.Repeat("new line\n", 12)
	edits := strings.Repeat(`[{"old_str":"before","new_str":"after"}]`, 8)
	deletePath := "tmp/generated/delete-target.txt"

	tests := []struct {
		name     string
		tool     string
		args     string
		want     editargs.PayloadSummary
		wantKeep string
	}{
		{
			name: "write_file content",
			tool: "write_file",
			args: jsonArgs(t, map[string]string{"path": "a.go", "content": writeContent}),
			want: editargs.PayloadSummary{Reason: "write_file_content", Bytes: len(writeContent)},
		},
		{
			name: "apply_patch patch",
			tool: "apply_patch",
			args: jsonArgs(t, map[string]string{"patch": patch}),
			want: editargs.PayloadSummary{Reason: "apply_patch_patch", Bytes: len(patch)},
		},
		{
			name: "str_replace strings",
			tool: "str_replace",
			args: jsonArgs(t, map[string]string{"path": "b.go", "old_str": oldStr, "new_str": newStr}),
			want: editargs.PayloadSummary{Reason: "str_replace_strings", Bytes: len(oldStr) + len(newStr)},
		},
		{
			name: "str_replace edits",
			tool: "str_replace",
			args: jsonArgs(t, map[string]string{"path": "c.go", "edits": edits}),
			want: editargs.PayloadSummary{Reason: "str_replace_edits", Bytes: len(edits)},
		},
		{
			name: "str_replace old_str wins over stale edits",
			tool: "str_replace",
			args: jsonArgs(t, map[string]string{"path": "b.go", "old_str": oldStr, "new_str": newStr, "edits": "[{"}),
			want: editargs.PayloadSummary{Reason: "str_replace_strings", Bytes: len(oldStr) + len(newStr)},
		},
		{
			name: "str_replace empty old_str uses edits",
			tool: "str_replace",
			args: jsonArgs(t, map[string]string{"path": "c.go", "old_str": "", "new_str": newStr, "edits": edits}),
			want: editargs.PayloadSummary{Reason: "str_replace_edits", Bytes: len(edits)},
		},
		{
			name: "str_replace line range counts new_str candidate",
			tool: "str_replace",
			args: jsonArgs(t, map[string]string{"path": "b.go", "old_str": "", "new_str": newStr, "start_line": "5", "end_line": "8"}),
			want: editargs.PayloadSummary{Reason: "str_replace_strings", Bytes: len(newStr)},
		},
		{
			name: "delete_file path",
			tool: "delete_file",
			args: jsonArgs(t, map[string]string{"path": deletePath}),
			want: editargs.PayloadSummary{Reason: "delete_file_path", Bytes: len(deletePath)},
		},
		{
			name:     "missing payload",
			tool:     "write_file",
			args:     jsonArgs(t, map[string]string{"path": "a.go"}),
			wantKeep: "missing_edit_argument_payload",
		},
		{
			name:     "invalid json",
			tool:     "write_file",
			args:     `{"path":`,
			wantKeep: "invalid_tool_call_arguments",
		},
		{
			name:     "non edit tool",
			tool:     "read_file",
			args:     jsonArgs(t, map[string]string{"path": "a.go"}),
			wantKeep: "tool_not_in_command_edit_allowlist",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, keep := editargs.Payload(tt.tool, tt.args)
			if keep != tt.wantKeep {
				t.Fatalf("Payload keepReason = %q, want %q", keep, tt.wantKeep)
			}
			if tt.wantKeep != "" {
				return
			}
			if got.Reason != tt.want.Reason || got.Bytes != tt.want.Bytes || got.Runes <= 0 || got.Tokens <= 0 {
				t.Fatalf("Payload() = %#v, want reason %q bytes %d with positive runes/tokens", got, tt.want.Reason, tt.want.Bytes)
			}
		})
	}

	for _, tool := range []string{"write_file", "apply_patch", "str_replace", "delete_file"} {
		if !editargs.IsTool(tool) {
			t.Fatalf("IsTool(%q) = false, want true", tool)
		}
	}
	if editargs.IsTool("read_file") {
		t.Fatal("IsTool(read_file) = true, want false")
	}
}

func TestProviderHistoryEditArgsBuildReplacementRewritesWriteFileContentAfterSuccessfulMatchingResult(t *testing.T) {
	path := "generated/write.go"
	content := largeWriteFileContent()
	args := jsonArgs(t, map[string]string{"path": path, "content": content, "mode": "0644"})

	replacement, ok := editargs.BuildReplacement(editargs.ReplacementRequest{
		ToolName:          "write_file",
		Arguments:         args,
		ToolResultContent: writeFileSuccess(content, path),
	})
	if !ok {
		t.Fatal("BuildReplacement(write_file) returned false")
	}
	fields := argumentFields(t, replacement.Arguments)
	if fields["path"] != path || fields["mode"] != "0644" {
		t.Fatalf("replacement fields = %#v, want path and extra fields preserved", fields)
	}
	contentReplacement := stringField(t, fields, "content")
	if contentReplacement == content || contentReplacement != "[omitted old write_file.content; path="+path+"]" {
		t.Fatalf("write_file content replacement = %q", contentReplacement)
	}
	input := map[string]any{"path": path, "content": content}
	if !replacement.ApplyAnthropicInput(input) || input["content"] != contentReplacement {
		t.Fatalf("ApplyAnthropicInput() did not sync write_file input: %#v", input)
	}
	if replacement.SavedBytes <= 0 || replacement.SavedTokens <= 0 {
		t.Fatalf("replacement savings = bytes %d tokens %d, want positive", replacement.SavedBytes, replacement.SavedTokens)
	}
}

func TestProviderHistoryEditArgsBuildReplacementKeepsNonReplaceableWriteFileCases(t *testing.T) {
	path := "generated/write.go"
	content := largeWriteFileContent()
	args := jsonArgs(t, map[string]string{"path": path, "content": content})

	tests := []struct {
		name   string
		args   string
		result string
	}{
		{name: "non success result", args: args, result: "Error writing file: permission denied"},
		{name: "small content below threshold", args: jsonArgs(t, map[string]string{"path": path, "content": "tiny"}), result: writeFileSuccess("tiny", path)},
		{name: "result path mismatch", args: args, result: writeFileSuccess(content, "generated/other.go")},
		{name: "unsafe path", args: jsonArgs(t, map[string]string{"path": "../outside.go", "content": content}), result: writeFileSuccess(content, "../outside.go")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if replacement, ok := editargs.BuildReplacement(editargs.ReplacementRequest{
				ToolName:          "write_file",
				Arguments:         tt.args,
				ToolResultContent: tt.result,
			}); ok {
				t.Fatalf("BuildReplacement() = %#v, want skipped", replacement)
			}
		})
	}
}

func TestProviderHistoryEditArgsBuildReplacementRewritesApplyPatchWithMultiFileSummary(t *testing.T) {
	paths := []string{
		"generated/a.go",
		"generated/b.go",
		"generated/c.go",
		"generated/d.go",
		"generated/e.go",
	}
	patch := largeMultiFileApplyPatch(paths)
	args := jsonArgs(t, map[string]string{"patch": patch, "note": "preserve"})

	replacement, ok := editargs.BuildReplacement(editargs.ReplacementRequest{
		ToolName:          "apply_patch",
		Arguments:         args,
		ToolResultContent: applyPatchSuccess(nil, paths, nil),
	})
	if !ok {
		t.Fatal("BuildReplacement(apply_patch) returned false")
	}
	fields := argumentFields(t, replacement.Arguments)
	if fields["note"] != "preserve" {
		t.Fatalf("replacement fields = %#v, want extra fields preserved", fields)
	}
	want := "[omitted old apply_patch.patch; files=generated/a.go, generated/b.go, generated/c.go, +2 more; result=success]"
	if got := stringField(t, fields, "patch"); got != want {
		t.Fatalf("apply_patch replacement = %q, want %q", got, want)
	}
	input := map[string]any{"patch": patch, "note": "preserve"}
	if !replacement.ApplyAnthropicInput(input) || input["patch"] != want {
		t.Fatalf("ApplyAnthropicInput() did not sync apply_patch input: %#v", input)
	}
}

func TestProviderHistoryEditArgsBuildReplacementRewritesApplyPatchMoveTarget(t *testing.T) {
	sourcePath := "generated/source.go"
	movePath := "generated/moved.go"
	patch := strings.Repeat("*** Begin Patch\n*** Update File: "+sourcePath+"\n*** Move to: "+movePath+"\n@@\n-old line\n+new line\n*** End Patch\n", 80)
	args := jsonArgs(t, map[string]string{"patch": patch})

	replacement, ok := editargs.BuildReplacement(editargs.ReplacementRequest{
		ToolName:          "apply_patch",
		Arguments:         args,
		ToolResultContent: applyPatchSuccess(nil, []string{movePath}, nil),
	})
	if !ok {
		t.Fatal("BuildReplacement(apply_patch move) returned false")
	}
	fields := argumentFields(t, replacement.Arguments)
	want := "[omitted old apply_patch.patch; files=" + movePath + "; result=success]"
	if got := stringField(t, fields, "patch"); got != want {
		t.Fatalf("apply_patch move replacement = %q, want %q", got, want)
	}

	if replacement, ok := editargs.BuildReplacement(editargs.ReplacementRequest{
		ToolName:          "apply_patch",
		Arguments:         args,
		ToolResultContent: applyPatchSuccess(nil, []string{sourcePath}, nil),
	}); ok {
		t.Fatalf("BuildReplacement(apply_patch move source result) = %#v, want skipped", replacement)
	}
}

func TestProviderHistoryEditArgsBuildReplacementKeepsNonReplaceableApplyPatchCases(t *testing.T) {
	patch := largeApplyPatch("internal/providerhistory/edit_arg.go")
	args := jsonArgs(t, map[string]string{"patch": patch})
	success := applyPatchSuccess(nil, []string{"internal/providerhistory/edit_arg.go"}, nil)

	tests := []struct {
		name   string
		args   string
		result string
	}{
		{name: "failed result", args: args, result: "Error: patch failed"},
		{name: "result path mismatch", args: args, result: applyPatchSuccess(nil, []string{"internal/providerhistory/other.go"}, nil)},
		{name: "incomplete result summary", args: args, result: "✓ Patch applied successfully.\nAdded: (none)\nModified: internal/providerhistory/edit_arg.go"},
		{name: "unsafe result path", args: args, result: "✓ Patch applied successfully.\nAdded: (none)\nModified: ../outside.go\nDeleted: (none)"},
		{name: "ambiguous result paths", args: args, result: "✓ Patch applied successfully.\nAdded: (none)\nModified: internal/providerhistory/edit_arg.go\nModified: internal/providerhistory/edit_arg.go\nDeleted: (none)"},
		{name: "unsafe patch path", args: jsonArgs(t, map[string]string{"patch": largeApplyPatch("../outside.go")}), result: success},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if replacement, ok := editargs.BuildReplacement(editargs.ReplacementRequest{
				ToolName:          "apply_patch",
				Arguments:         tt.args,
				ToolResultContent: tt.result,
			}); ok {
				t.Fatalf("BuildReplacement() = %#v, want skipped", replacement)
			}
		})
	}
}

func TestProviderHistoryEditArgsBuildReplacementRewritesStrReplaceSingleAndBatch(t *testing.T) {
	singlePath := "internal/providerhistory/replace.go"
	oldStr := largeStrReplaceText("old line")
	newStr := largeStrReplaceText("new line")
	singleArgs := jsonAnyArgs(t, map[string]any{
		"path":       singlePath,
		"old_str":    oldStr,
		"new_str":    newStr,
		"start_line": 12,
		"end_line":   40,
	})

	single, ok := editargs.BuildReplacement(editargs.ReplacementRequest{
		ToolName:          "str_replace",
		Arguments:         singleArgs,
		ToolResultContent: "Successfully replaced text in " + singlePath + " (lines 12-40 → 12-41)",
	})
	if !ok {
		t.Fatal("BuildReplacement(str_replace single) returned false")
	}
	singleFields := argumentFields(t, single.Arguments)
	if singleFields["path"] != singlePath || singleFields["start_line"] != float64(12) || singleFields["end_line"] != float64(40) {
		t.Fatalf("single replacement fields = %#v, want metadata preserved", singleFields)
	}
	assertStrReplacePlaceholder(t, singleFields["old_str"], "old_str", singlePath)
	assertStrReplacePlaceholder(t, singleFields["new_str"], "new_str", singlePath)

	lineRangePath := "internal/providerhistory/line_range_replace.go"
	lineRangeArgs := jsonAnyArgs(t, map[string]any{
		"path":    lineRangePath,
		"old_str": oldStr,
		"new_str": newStr,
	})
	lineRange, ok := editargs.BuildReplacement(editargs.ReplacementRequest{
		ToolName:          "str_replace",
		Arguments:         lineRangeArgs,
		ToolResultContent: "Successfully replaced lines 12-40 in " + lineRangePath + " (new range: 12-41)",
	})
	if !ok {
		t.Fatal("BuildReplacement(str_replace line range single) returned false")
	}
	lineRangeFields := argumentFields(t, lineRange.Arguments)
	assertStrReplacePlaceholder(t, lineRangeFields["old_str"], "old_str", lineRangePath)
	assertStrReplacePlaceholder(t, lineRangeFields["new_str"], "new_str", lineRangePath)

	staleEditsArgs := jsonAnyArgs(t, map[string]any{
		"path":    singlePath,
		"old_str": oldStr,
		"new_str": newStr,
		"edits":   "[{",
	})
	staleEditsReplacement, ok := editargs.BuildReplacement(editargs.ReplacementRequest{
		ToolName:          "str_replace",
		Arguments:         staleEditsArgs,
		ToolResultContent: "Successfully replaced text in " + singlePath + " (lines 12-40 → 12-41)",
	})
	if !ok {
		t.Fatal("BuildReplacement(str_replace single with stale edits) returned false")
	}
	staleEditsFields := argumentFields(t, staleEditsReplacement.Arguments)
	if staleEditsFields["edits"] != "[{" {
		t.Fatalf("stale edits field = %#v, want preserved unused edits payload", staleEditsFields["edits"])
	}
	assertStrReplacePlaceholder(t, staleEditsFields["old_str"], "old_str", singlePath)
	assertStrReplacePlaceholder(t, staleEditsFields["new_str"], "new_str", singlePath)

	arrayPath := "internal/providerhistory/batch_array.go"
	stringPath := "internal/providerhistory/batch_string.go"
	arrayArgs := jsonAnyArgs(t, map[string]any{
		"path": arrayPath,
		"edits": []map[string]any{
			{
				"old_str": largeStrReplaceText("array old"),
				"new_str": largeStrReplaceText("array new"),
				"note":    "preserve me",
			},
			{
				"old_str": largeStrReplaceText("array old second"),
				"new_str": largeStrReplaceText("array new second"),
			},
		},
	})
	stringEdits := jsonAnyString(t, []map[string]any{
		{
			"old_str": largeStrReplaceText("string old"),
			"new_str": largeStrReplaceText("string new"),
		},
		{
			"old_str": largeStrReplaceText("string old second"),
			"new_str": largeStrReplaceText("string new second"),
		},
	})
	stringArgs := jsonAnyArgs(t, map[string]any{
		"path":  stringPath,
		"edits": stringEdits,
	})

	arrayReplacement, ok := editargs.BuildReplacement(editargs.ReplacementRequest{
		ToolName:          "str_replace",
		Arguments:         arrayArgs,
		ToolResultContent: "Successfully applied 2 edits to " + arrayPath,
	})
	if !ok {
		t.Fatal("BuildReplacement(str_replace array edits) returned false")
	}
	arrayFields := argumentFields(t, arrayReplacement.Arguments)
	arrayEdits, ok := arrayFields["edits"].([]any)
	if !ok || len(arrayEdits) != 2 {
		t.Fatalf("array edits = %#v, want JSON array shape", arrayFields["edits"])
	}
	arrayEdit := arrayEdits[0].(map[string]any)
	assertStrReplacePlaceholder(t, arrayEdit["old_str"], "old_str", arrayPath)
	assertStrReplacePlaceholder(t, arrayEdit["new_str"], "new_str", arrayPath)
	if arrayEdit["note"] != "preserve me" {
		t.Fatalf("array edit = %#v, want extra fields preserved", arrayEdit)
	}

	stringReplacement, ok := editargs.BuildReplacement(editargs.ReplacementRequest{
		ToolName:          "str_replace",
		Arguments:         stringArgs,
		ToolResultContent: "Successfully applied 2 edits to " + stringPath,
	})
	if !ok {
		t.Fatal("BuildReplacement(str_replace string edits) returned false")
	}
	stringFields := argumentFields(t, stringReplacement.Arguments)
	stringReplacementPayload, ok := stringFields["edits"].(string)
	if !ok {
		t.Fatalf("string edits = %#v, want JSON string shape", stringFields["edits"])
	}
	var parsedStringEdits []map[string]any
	if err := json.Unmarshal([]byte(stringReplacementPayload), &parsedStringEdits); err != nil {
		t.Fatalf("projected edits string is not JSON array: %v\nedits=%s", err, stringReplacementPayload)
	}
	if len(parsedStringEdits) != 2 {
		t.Fatalf("projected edits string length = %d, want 2", len(parsedStringEdits))
	}
	assertStrReplacePlaceholder(t, parsedStringEdits[0]["old_str"], "old_str", stringPath)
	assertStrReplacePlaceholder(t, parsedStringEdits[0]["new_str"], "new_str", stringPath)
}

func TestProviderHistoryEditArgsBuildReplacementKeepsNonReplaceableStrReplaceCases(t *testing.T) {
	path := "internal/providerhistory/replace.go"
	args := strReplaceSingleArgs(t, path)
	success := "Successfully replaced lines 5-6 in " + path + " (new range: 5-7)"
	batchEdits := []map[string]any{
		{"old_str": largeStrReplaceText("batch old"), "new_str": largeStrReplaceText("batch new")},
		{"old_str": largeStrReplaceText("batch old second"), "new_str": largeStrReplaceText("batch new second")},
	}
	batchArrayArgs := jsonAnyArgs(t, map[string]any{"path": path, "edits": batchEdits})
	batchStringArgs := jsonAnyArgs(t, map[string]any{"path": path, "edits": jsonAnyString(t, batchEdits)})

	tests := []struct {
		name   string
		args   string
		result string
	}{
		{name: "failed result", args: args, result: "Error: old_str not found in " + path},
		{name: "cancelled result", args: args, result: "[CANCELLED] str_replace (single) not applied for " + path + "."},
		{name: "no-op result", args: args, result: "Error: old_str and new_str are identical in " + path + " (no change needed)"},
		{name: "result path mismatch", args: args, result: "Successfully replaced lines 5-6 in internal/providerhistory/other.go (new range: 5-7)"},
		{name: "ambiguous multiple success lines", args: args, result: success + "\nSuccessfully replaced text in " + path + " (lines 8-9 → 8-10)"},
		{name: "missing new_str", args: jsonAnyArgs(t, map[string]any{"path": path, "old_str": largeStrReplaceText("old")}), result: success},
		{name: "line range keeps empty old_str mode", args: jsonAnyArgs(t, map[string]any{"path": path, "old_str": "", "new_str": largeStrReplaceText("new"), "start_line": "5", "end_line": "6"}), result: success},
		{name: "unsafe path", args: strReplaceSingleArgs(t, "../outside.go"), result: success},
		{name: "malformed edits string", args: jsonAnyArgs(t, map[string]any{"path": path, "edits": `[{`}), result: "Successfully applied 1 edits to " + path},
		{name: "missing old_str in edits", args: jsonAnyArgs(t, map[string]any{"path": path, "edits": []map[string]any{{"new_str": largeStrReplaceText("new")}}}), result: "Successfully applied 1 edits to " + path},
		{name: "batch array count mismatch", args: batchArrayArgs, result: "Successfully applied 1 edits to " + path},
		{name: "batch string count mismatch", args: batchStringArgs, result: "Successfully applied 1 edits to " + path},
		{name: "batch args with single success result", args: batchArrayArgs, result: success},
		{name: "batch result path mismatch", args: batchArrayArgs, result: "Successfully applied 2 edits to internal/providerhistory/other.go"},
		{name: "single args with batch success result", args: args, result: "Successfully applied 1 edits to " + path},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if replacement, ok := editargs.BuildReplacement(editargs.ReplacementRequest{
				ToolName:          "str_replace",
				Arguments:         tt.args,
				ToolResultContent: tt.result,
			}); ok {
				t.Fatalf("BuildReplacement() = %#v, want skipped", replacement)
			}
		})
	}
}

func TestProviderHistoryEditArgsApplyAnthropicInputRequiresMatchingOriginalShape(t *testing.T) {
	patch := largeApplyPatch("internal/providerhistory/claude.go")
	args := jsonArgs(t, map[string]string{"patch": patch})
	replacement, ok := editargs.BuildReplacement(editargs.ReplacementRequest{
		ToolName:          "apply_patch",
		Arguments:         args,
		ToolResultContent: applyPatchSuccess(nil, []string{"internal/providerhistory/claude.go"}, nil),
	})
	if !ok {
		t.Fatal("BuildReplacement(apply_patch) returned false")
	}
	if replacement.ApplyAnthropicInput(map[string]any{"patch": "different patch"}) {
		t.Fatal("ApplyAnthropicInput() accepted stale apply_patch input")
	}

	path := "internal/providerhistory/claude_batch.go"
	edits := []map[string]any{{
		"old_str": largeStrReplaceText("old"),
		"new_str": largeStrReplaceText("new"),
	}}
	batchArgs := jsonAnyArgs(t, map[string]any{"path": path, "edits": edits})
	batchReplacement, ok := editargs.BuildReplacement(editargs.ReplacementRequest{
		ToolName:          "str_replace",
		Arguments:         batchArgs,
		ToolResultContent: "Successfully applied 1 edits to " + path,
	})
	if !ok {
		t.Fatal("BuildReplacement(str_replace batch) returned false")
	}
	if batchReplacement.ApplyAnthropicInput(map[string]any{"path": path, "edits": jsonAnyString(t, edits)}) {
		t.Fatal("ApplyAnthropicInput() accepted str_replace edits with different input shape")
	}
}

func TestProviderHistoryEditArgsBuildReplacementDoesNotReplaceDeleteFilePathCandidate(t *testing.T) {
	path := "generated/delete.txt"
	payload, keep := editargs.Payload("delete_file", jsonArgs(t, map[string]string{"path": path}))
	if keep != "" || payload.Reason != "delete_file_path" || payload.Bytes != len(path) {
		t.Fatalf("Payload(delete_file) = %#v keep %q, want delete_file_path candidate", payload, keep)
	}
	if replacement, ok := editargs.BuildReplacement(editargs.ReplacementRequest{
		ToolName:          "delete_file",
		Arguments:         jsonArgs(t, map[string]string{"path": path}),
		ToolResultContent: "Deleted " + path,
	}); ok {
		t.Fatalf("BuildReplacement(delete_file) = %#v, want candidate-only skip", replacement)
	}
}

func jsonArgs(t *testing.T, value map[string]string) string {
	t.Helper()
	return jsonAnyString(t, value)
}

func jsonAnyArgs(t *testing.T, value map[string]any) string {
	t.Helper()
	return jsonAnyString(t, value)
}

func jsonAnyString(t *testing.T, value any) string {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("json.Marshal(%#v) error = %v", value, err)
	}
	return string(data)
}

func argumentFields(t *testing.T, args string) map[string]any {
	t.Helper()
	var fields map[string]any
	if err := json.Unmarshal([]byte(args), &fields); err != nil {
		t.Fatalf("arguments are not valid JSON: %v\nargs=%s", err, args)
	}
	return fields
}

func stringField(t *testing.T, fields map[string]any, key string) string {
	t.Helper()
	value, ok := fields[key].(string)
	if !ok {
		t.Fatalf("field %q = %#v, want string", key, fields[key])
	}
	return value
}

func largeWriteFileContent() string {
	return strings.Repeat("package generated\n", 260)
}

func largeApplyPatch(path string) string {
	return strings.Repeat("*** Begin Patch\n*** Update File: "+path+"\n@@\n-old line\n+new line\n*** End Patch\n", 80)
}

func largeMultiFileApplyPatch(paths []string) string {
	var builder strings.Builder
	builder.WriteString("*** Begin Patch\n")
	for _, path := range paths {
		builder.WriteString("*** Update File: " + path + "\n@@\n")
		for i := 0; i < 40; i++ {
			builder.WriteString("-old line\n+new line\n")
		}
	}
	builder.WriteString("*** End Patch\n")
	return builder.String()
}

func applyPatchSuccess(added, modified, deleted []string) string {
	return "✓ Patch applied successfully.\nAdded: " + pathList(added) +
		"\nModified: " + pathList(modified) +
		"\nDeleted: " + pathList(deleted)
}

func pathList(paths []string) string {
	if len(paths) == 0 {
		return "(none)"
	}
	return strings.Join(paths, ", ")
}

func writeFileSuccess(content, path string) string {
	lines := strings.Count(content, "\n") + 1
	return "Successfully wrote " + strconv.Itoa(len(content)) + " bytes (" + strconv.Itoa(lines) + " lines) to " + path
}

func largeStrReplaceText(prefix string) string {
	return strings.Repeat(prefix+" with enough content to pass replacement threshold\n", 220)
}

func strReplaceSingleArgs(t *testing.T, path string) string {
	t.Helper()
	return jsonAnyArgs(t, map[string]any{
		"path":    path,
		"old_str": largeStrReplaceText("old"),
		"new_str": largeStrReplaceText("new"),
	})
}

func assertStrReplacePlaceholder(t *testing.T, value any, field, path string) {
	t.Helper()
	text, ok := value.(string)
	if !ok {
		t.Fatalf("%s replacement = %#v, want string placeholder", field, value)
	}
	if !strings.Contains(text, "str_replace") || !strings.Contains(text, field) || !strings.Contains(text, "path="+path) {
		t.Fatalf("%s replacement = %q, want str_replace placeholder for %s", field, text, path)
	}
}

func TestProviderHistoryEditArgsReplacementDoesNotMutateOriginalArguments(t *testing.T) {
	args := jsonAnyArgs(t, map[string]any{
		"path":    "generated/replace.go",
		"old_str": largeStrReplaceText("old"),
		"new_str": largeStrReplaceText("new"),
	})
	var before map[string]any
	if err := json.Unmarshal([]byte(args), &before); err != nil {
		t.Fatal(err)
	}

	_, ok := editargs.BuildReplacement(editargs.ReplacementRequest{
		ToolName:          "str_replace",
		Arguments:         args,
		ToolResultContent: "Successfully replaced text in generated/replace.go (lines 1-2 → 1-3)",
	})
	if !ok {
		t.Fatal("BuildReplacement(str_replace) returned false")
	}
	var after map[string]any
	if err := json.Unmarshal([]byte(args), &after); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(after, before) {
		t.Fatalf("BuildReplacement mutated original arguments string parse result:\n got %#v\nwant %#v", after, before)
	}
}
