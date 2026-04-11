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
	expected := 12
	if len(ToolDescriptions) != expected {
		t.Errorf("ToolDescriptions has %d entries, want %d", len(ToolDescriptions), expected)
	}
}

func TestToolDescriptions_KnownToolsExist(t *testing.T) {
	knownTools := []string{
		"gather_context",
		"read_file", "write_file", "str_replace", "delete_file", "list_dir",
		"search_code", "web_search",
		"bash",
		"ask_user_question",
		"spawn_agent", "wait_agent",
	}
	for _, name := range knownTools {
		if _, ok := ToolDescriptions[name]; !ok {
			t.Errorf("ToolDescriptions missing key %q", name)
		}
	}
}

func TestToolDescriptions_InspectSymbolNotPublic(t *testing.T) {
	if _, ok := ToolDescriptions["inspect_symbol"]; ok {
		t.Error("inspect_symbol should not be in ToolDescriptions (integrated into search_code)")
	}
}

func TestToolDescriptions_GatherContextIsPrimaryInvestigationTool(t *testing.T) {
	desc := ToolDescriptions["gather_context"]
	if !strings.Contains(desc, "Primary investigation tool") {
		t.Error("gather_context description should mark it as the primary investigation tool")
	}
	if !strings.Contains(desc, "structured impact") {
		t.Error("gather_context description should mention structured impact routing")
	}
	if !strings.Contains(desc, "bounded compact evidence prefetch") {
		t.Error("gather_context description should mention compact evidence prefetch")
	}
}

func TestToolDescriptions_ListDirMentionsCompactSummaryAndOverrideUsage(t *testing.T) {
	desc := ToolDescriptions["list_dir"]
	if !strings.Contains(desc, "Usually prefer gather_context first") {
		t.Error("list_dir description should mention gather_context-first usage")
	}
	if !strings.Contains(desc, "compact summary") {
		t.Error("list_dir description should mention compact summary output")
	}
}

func TestToolDescriptions_ReadFileAndSearchCodeDescribeLowLevelUsage(t *testing.T) {
	if !strings.Contains(ToolDescriptions["read_file"], "Low-level file reader") {
		t.Error("read_file description should position it as low-level")
	}
	if !strings.Contains(ToolDescriptions["read_file"], "Default detail=auto returns full content when feasible") {
		t.Error("read_file description should describe the default auto behavior")
	}
	if !strings.Contains(ToolDescriptions["read_file"], "compact for locator targets or explicit path ranges") {
		t.Error("read_file description should restrict compact to supported reads")
	}
	if !strings.Contains(ToolDescriptions["read_file"], "Do not re-read files already returned") {
		t.Error("read_file description should discourage rereading returned files")
	}
	if !strings.Contains(ToolDescriptions["search_code"], "Low-level code discovery tool") {
		t.Error("search_code description should position it as low-level")
	}
	if !strings.Contains(ToolDescriptions["search_code"], "comma-separated patterns for parallel multi-search") {
		t.Error("search_code description should mention comma-separated parallel multi-search")
	}
	if !strings.Contains(ToolDescriptions["search_code"], "Prefer mode=auto") {
		t.Error("search_code description should recommend mode=auto")
	}
	if !strings.Contains(ToolDescriptions["search_code"], "intent=impact") {
		t.Error("search_code description should mention intent=impact")
	}
	if !strings.Contains(ToolDescriptions["search_code"], "ripgrep-like built-in language aliases") {
		t.Error("search_code description should mention the shared built-in file filter contract")
	}
	if strings.Contains(ToolDescriptions["search_code"], "For Go symbols") {
		t.Error("search_code description should not be Go-specific")
	}
}
