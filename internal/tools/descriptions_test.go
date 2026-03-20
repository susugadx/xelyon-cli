package tools

import (
	"strings"
	"testing"
)

func TestToolDescriptions_AllKeysNonEmpty(t *testing.T) {
	for name, desc := range ToolDescriptions {
		if desc == "" {
			t.Errorf("ToolDescriptions[%q] is empty", name)
		}
	}
}

func TestToolDescriptions_ExpectedToolCount(t *testing.T) {
	// read_files は read_file の paths パラメータに統合、inspect_symbol 追加で 10 エントリ
	expected := 10
	if len(ToolDescriptions) != expected {
		t.Errorf("ToolDescriptions has %d entries, want %d", len(ToolDescriptions), expected)
	}
}

func TestToolDescriptions_KnownToolsExist(t *testing.T) {
	knownTools := []string{
		"read_file", "write_file", "str_replace", "delete_file", "list_dir",
		"inspect_symbol", "search_code", "web_search",
		"bash",
		"ask_user_question",
	}
	for _, name := range knownTools {
		if _, ok := ToolDescriptions[name]; !ok {
			t.Errorf("ToolDescriptions missing key %q", name)
		}
	}
}

func TestToolDescriptions_InspectSymbolMentionsGoSymbolLookup(t *testing.T) {
	desc := ToolDescriptions["inspect_symbol"]
	if !strings.Contains(desc, "Look up a Go symbol by name") {
		t.Error("inspect_symbol description should describe Go symbol lookup")
	}
	if strings.Contains(desc, "search_code+read_file") {
		t.Error("inspect_symbol description should not compare itself against read_file/search_code")
	}
}

func TestToolDescriptions_ListDirMentionsCompactSummaryAndNextChoice(t *testing.T) {
	desc := ToolDescriptions["list_dir"]
	if !strings.Contains(desc, "Preferred for directory exploration") {
		t.Error("list_dir description should mark it as preferred for directory exploration")
	}
	if !strings.Contains(desc, "compact summary") {
		t.Error("list_dir description should mention compact summary output")
	}
	if !strings.Contains(desc, "next file/subtree") {
		t.Error("list_dir description should mention next file/subtree selection")
	}
}

func TestToolDescriptions_ReadFileAndSearchCodeDescribeCurrentUsage(t *testing.T) {
	if !strings.Contains(ToolDescriptions["read_file"], "Use path for one file, paths for multiple files in one call") {
		t.Error("read_file description should describe path/paths usage")
	}
	if !strings.Contains(ToolDescriptions["read_file"], "Do not re-read files already returned") {
		t.Error("read_file description should discourage rereading returned files")
	}
	if !strings.Contains(ToolDescriptions["search_code"], "prefer one multi-pattern search over serial searches") {
		t.Error("search_code description should prefer multi-pattern searches over serial searches")
	}
}
