package tools

import "testing"

func TestToolDescriptions_AllKeysNonEmpty(t *testing.T) {
	for name, desc := range ToolDescriptions {
		if desc == "" {
			t.Errorf("ToolDescriptions[%q] is empty", name)
		}
	}
}

func TestToolDescriptions_ExpectedToolCount(t *testing.T) {
	expected := 17
	if len(ToolDescriptions) != expected {
		t.Errorf("ToolDescriptions has %d entries, want %d", len(ToolDescriptions), expected)
	}
}

func TestToolDescriptions_KnownToolsExist(t *testing.T) {
	knownTools := []string{
		"read_file", "write_file", "str_replace", "delete_file", "list_dir",
		"search_code", "search_file", "grep_replace", "web_search",
		"bash",
		"lsp_find",
		"ask_user_question", "create_plan", "get_plan", "list_plans", "update_plan", "delete_plan",
	}
	for _, name := range knownTools {
		if _, ok := ToolDescriptions[name]; !ok {
			t.Errorf("ToolDescriptions missing key %q", name)
		}
	}
}
