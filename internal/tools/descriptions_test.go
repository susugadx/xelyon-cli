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
	// read_files は read_file の paths パラメータに統合されたため 9 エントリ
	expected := 9
	if len(ToolDescriptions) != expected {
		t.Errorf("ToolDescriptions has %d entries, want %d", len(ToolDescriptions), expected)
	}
}

func TestToolDescriptions_KnownToolsExist(t *testing.T) {
	knownTools := []string{
		"read_file", "write_file", "str_replace", "delete_file", "list_dir",
		"search_code", "web_search",
		"bash",
		"ask_user_question",
	}
	for _, name := range knownTools {
		if _, ok := ToolDescriptions[name]; !ok {
			t.Errorf("ToolDescriptions missing key %q", name)
		}
	}
}
