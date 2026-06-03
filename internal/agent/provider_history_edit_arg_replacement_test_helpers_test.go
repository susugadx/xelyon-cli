package agent

import (
	"encoding/json"
	"strings"
	"testing"
)

func providerHistoryLargeApplyPatch(path string) string {
	return strings.Repeat("*** Begin Patch\n*** Update File: "+path+"\n@@\n-old line\n+new line\n*** End Patch\n", 80)
}

func providerHistoryApplyPatchSuccess(added, modified, deleted []string) string {
	return "✓ Patch applied successfully.\nAdded: " + providerHistoryApplyPatchPathList(added) +
		"\nModified: " + providerHistoryApplyPatchPathList(modified) +
		"\nDeleted: " + providerHistoryApplyPatchPathList(deleted)
}

func providerHistoryApplyPatchPathList(paths []string) string {
	if len(paths) == 0 {
		return "(none)"
	}
	return strings.Join(paths, ", ")
}

func providerHistoryLargeStrReplaceText(prefix string) string {
	return strings.Repeat(prefix+" with enough content to pass replacement threshold\n", 220)
}

func providerHistoryStrReplaceArguments(t *testing.T, path, oldStr, newStr string) string {
	t.Helper()
	return providerHistoryJSONAnyArguments(t, map[string]any{"path": path, "old_str": oldStr, "new_str": newStr})
}

func providerHistoryJSONAnyArguments(t *testing.T, arguments map[string]any) string {
	t.Helper()
	data, err := json.Marshal(arguments)
	if err != nil {
		t.Fatalf("json.Marshal(%#v) error = %v", arguments, err)
	}
	return string(data)
}

func providerHistoryArgumentFields(t *testing.T, args string) map[string]any {
	t.Helper()
	var fields map[string]any
	if err := json.Unmarshal([]byte(args), &fields); err != nil {
		t.Fatalf("arguments are not valid JSON: %v\nargs=%s", err, args)
	}
	return fields
}

func assertProviderHistoryApplyPatchArgReplacement(t *testing.T, args, path, originalPatch string) string {
	t.Helper()
	fields := providerHistoryArgumentFields(t, args)
	replacement, ok := fields["patch"].(string)
	if !ok {
		t.Fatalf("projected apply_patch.patch = %#v, want string", fields["patch"])
	}
	if replacement == "" || replacement == originalPatch || !strings.HasPrefix(replacement, "[omitted old apply_patch.patch; files="+path+"]") {
		t.Fatalf("projected apply_patch.patch = %q, want placeholder for %s", replacement, path)
	}
	return replacement
}

func assertProviderHistoryStrReplaceArgReplacement(t *testing.T, args, path, originalOldStr, originalNewStr string) {
	t.Helper()
	fields := providerHistoryArgumentFields(t, args)
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
