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

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}
