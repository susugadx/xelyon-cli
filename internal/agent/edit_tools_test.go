package agent

import "testing"

func TestPlanModeExcludedTools_DefaultEditTool(t *testing.T) {
	excluded := planModeExcludedTools()

	if containsString(excluded, "ask_user_question") {
		t.Fatal("plan mode should not exclude planning tools")
	}
	for _, name := range []string{"str_replace", "write_file", "delete_file"} {
		if !containsString(excluded, name) {
			t.Fatalf("plan mode should exclude %s in default edit mode", name)
		}
	}
	if containsString(excluded, "apply_patch") {
		t.Fatal("plan mode should keep apply_patch visible in default edit mode")
	}
}

func TestPlanModeExcludedTools_LegacyEditTool(t *testing.T) {
	t.Setenv("XELYON_EDIT_TOOL", "str_replace")

	excluded := planModeExcludedTools()
	if containsString(excluded, "ask_user_question") {
		t.Fatal("plan mode should not exclude planning tools")
	}
	if !containsString(excluded, "apply_patch") {
		t.Fatal("plan mode should exclude apply_patch in legacy edit mode")
	}
	for _, name := range []string{"str_replace", "write_file", "delete_file"} {
		if containsString(excluded, name) {
			t.Fatalf("plan mode should keep %s visible in legacy edit mode", name)
		}
	}
}

func TestNormalModeExcludedTools_IncludesListDir(t *testing.T) {
	excluded := normalModeExcludedTools()
	if !containsString(excluded, "list_dir") {
		t.Fatal("normal mode should exclude list_dir")
	}
	// inspect_symbol は公開ツールとして廃止済み（search_code に統合）
	if containsString(excluded, "inspect_symbol") {
		t.Fatal("inspect_symbol is no longer a public tool, should not appear in excluded list")
	}
}

func TestPlanModeExcludedTools_IncludesListDir(t *testing.T) {
	excluded := planModeExcludedTools()
	if !containsString(excluded, "list_dir") {
		t.Fatal("plan mode should exclude list_dir")
	}
	if containsString(excluded, "inspect_symbol") {
		t.Fatal("inspect_symbol is no longer a public tool, should not appear in excluded list")
	}
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}
