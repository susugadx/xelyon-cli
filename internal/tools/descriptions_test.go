package tools

import (
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/toolmeta"
)

func TestToolDescriptions_AllKeysNonEmpty(t *testing.T) {
	for name, desc := range ToolDescriptions {
		if desc == "" {
			t.Errorf("ToolDescriptions[%q] is empty", name)
		}
	}
}

func TestToolDescriptions_KnownToolsExist(t *testing.T) {
	knownTools := make([]string, 0, len(toolmeta.BuiltinSpecs()))
	for _, spec := range toolmeta.BuiltinSpecs() {
		knownTools = append(knownTools, spec.Name)
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
	if !strings.Contains(desc, "Natural-language searches like A or B in docs/ stay search queries") {
		t.Error("gather_context description should describe natural-language search routing")
	}
}

func TestToolDescriptions_ListDirMentionsCompactSummaryAndOverrideUsage(t *testing.T) {
	desc := ToolDescriptions["list_dir"]
	if !strings.Contains(desc, "gather_context is the default front door") {
		t.Error("list_dir description should mention gather_context as the directory front door")
	}
	if !strings.Contains(desc, "stays hidden on current gather_context-first agent surfaces") {
		t.Error("list_dir description should describe its hidden-by-default agent surface")
	}
	if !strings.Contains(desc, "compact summary") {
		t.Error("list_dir description should mention compact summary output")
	}
}

func TestToolDescriptions_ReadFileAndSearchCodeDescribeLowLevelUsage(t *testing.T) {
	if !strings.Contains(ToolDescriptions["read_file"], "gather_context remains the default investigation front door") {
		t.Error("read_file description should keep gather_context as the default front door")
	}
	if !strings.Contains(ToolDescriptions["read_file"], "edit exact-control override") {
		t.Error("read_file description should describe the apply_patch exact-control contract")
	}
	if !strings.Contains(ToolDescriptions["read_file"], "legacy surfaces it remains a low-level expert override") {
		t.Error("read_file description should describe the legacy override contract")
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
	if !strings.Contains(ToolDescriptions["search_code"], "gather_context remains the default investigation front door") {
		t.Error("search_code description should keep gather_context as the default front door")
	}
	if !strings.Contains(ToolDescriptions["search_code"], "When a legacy surface exposes search_code") {
		t.Error("search_code description should describe the legacy exposure contract")
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
	if !strings.Contains(ToolDescriptions["search_code"], "targeted TSX .tsx symbols") {
		t.Error("search_code description should mention targeted TSX structured impact")
	}
	if !strings.Contains(ToolDescriptions["search_code"], "file_filter=typescript remains a broad fallback scope") {
		t.Error("search_code description should preserve the broad TypeScript fallback contract")
	}
	if !strings.Contains(ToolDescriptions["search_code"], "ripgrep-like built-in language aliases") {
		t.Error("search_code description should mention the shared built-in file filter contract")
	}
	if strings.Contains(ToolDescriptions["search_code"], "For Go symbols") {
		t.Error("search_code description should not be Go-specific")
	}
}

func TestToolDescriptions_StrReplaceEvidenceFirstContract(t *testing.T) {
	desc := ToolDescriptions["str_replace"]
	for _, want := range []string{
		"Copy exact old_str and existing context",
		"actual gather_context, read_file, or search_code output",
		"do not invent old_str",
		"Write new_str as the intended replacement based on that verified context",
		"same-file edits=[{old_str,new_str},...]",
		"advanced fallback",
		"fresh tool output provides an exact range",
		"do not guess line ranges",
		"avoid evidence",
	} {
		if !strings.Contains(desc, want) {
			t.Fatalf("str_replace description missing %q:\n%s", want, desc)
		}
	}
	for _, forbidden := range []string{
		"PREFERRED",
		"no read_file needed",
		"old_str mode requires read_file first",
		"old_str/new_str copied from actual",
	} {
		if strings.Contains(desc, forbidden) {
			t.Fatalf("str_replace description should not contain legacy guidance %q:\n%s", forbidden, desc)
		}
	}
}
