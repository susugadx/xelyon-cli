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

func TestToolDescription_UsesToolMetaSourceOfTruth(t *testing.T) {
	original, ok := ToolDescriptions["read_file"]
	if !ok {
		t.Fatal("ToolDescriptions missing read_file")
	}
	ToolDescriptions["read_file"] = "mutated compatibility snapshot"
	defer func() {
		ToolDescriptions["read_file"] = original
	}()

	if got := ToolDescription("read_file"); got != original {
		t.Fatalf("ToolDescription(read_file) = %q, want toolmeta source %q", got, original)
	}
	if got := ToolDescription("unknown_tool"); got != "" {
		t.Fatalf("ToolDescription(unknown_tool) = %q, want empty", got)
	}
}

func TestToolDescriptions_GatherContextIsPrimaryInvestigationTool(t *testing.T) {
	desc := mustToolDescription(t, "gather_context")
	if !strings.Contains(desc, "Primary repository investigation tool") {
		t.Error("gather_context description should mark it as the primary investigation tool")
	}
	if !strings.Contains(desc, "files, ranges, symbols, callers, tests, or directory context") {
		t.Error("gather_context description should summarize the repository context it can retrieve")
	}
	if !strings.Contains(desc, "Prefer it when the target is not already available as exact content") {
		t.Error("gather_context description should keep the default investigation positioning")
	}
}

func TestToolDescriptions_ListDirMentionsCompactSummaryAndOverrideUsage(t *testing.T) {
	desc := mustToolDescription(t, "list_dir")
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
	readFileDesc := mustToolDescription(t, "read_file")
	searchCodeDesc := mustToolDescription(t, "search_code")
	if !strings.Contains(readFileDesc, "Read exact content from known files or line ranges") {
		t.Error("read_file description should describe exact-content reads")
	}
	if !strings.Contains(readFileDesc, "avoid repeating content already available") {
		t.Error("read_file description should discourage rereading returned content")
	}
	if !strings.Contains(searchCodeDesc, "Low-level repository search") {
		t.Error("search_code description should position it as low-level repository search")
	}
	if !strings.Contains(searchCodeDesc, "exact symbols, literals, regex patterns, references, or impact analysis") {
		t.Error("search_code description should summarize its exact-search scope")
	}
	if !strings.Contains(searchCodeDesc, "Use when gather_context is insufficient or exact search control is needed") {
		t.Error("search_code description should keep the low-level override positioning")
	}
	if !strings.Contains(searchCodeDesc, "Combine related independent patterns when practical") {
		t.Error("search_code description should keep compact multi-pattern guidance")
	}
	for _, forbidden := range []string{
		"file_filter=go",
		"file_filter=typescript",
		"ripgrep-like built-in language aliases",
		".mjs/.cjs",
		"targeted TSX",
	} {
		if strings.Contains(searchCodeDesc, forbidden) {
			t.Errorf("search_code description should not keep long parameter encyclopaedia %q", forbidden)
		}
	}
}

func TestToolDescriptions_BashDoesNotAdvertiseAutoApprovedInvestigation(t *testing.T) {
	desc := mustToolDescription(t, "bash")
	for _, want := range []string{
		"Run local commands for build, test, format, lint, git, package tooling",
		"Runtime policy determines approval or denial",
		"Prefer dedicated repository tools for reading and searching",
	} {
		if !strings.Contains(desc, want) {
			t.Fatalf("bash description missing %q:\n%s", want, desc)
		}
	}
	for _, forbidden := range []string{
		"cat/ls/grep auto-approve",
		"Dangerous commands require confirmation",
	} {
		if strings.Contains(desc, forbidden) {
			t.Fatalf("bash description should not expose approval policy fragment %q:\n%s", forbidden, desc)
		}
	}
}

func TestToolDescriptions_StrReplaceEvidenceFirstContract(t *testing.T) {
	desc := mustToolDescription(t, "str_replace")
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

func mustToolDescription(t *testing.T, name string) string {
	t.Helper()
	desc := ToolDescription(name)
	if desc == "" {
		t.Fatalf("ToolDescription(%q) returned empty", name)
	}
	return desc
}
