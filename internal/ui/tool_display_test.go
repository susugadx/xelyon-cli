package ui

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/fatih/color"
)

func init() {
	color.NoColor = true
}

func TestFormatToolLine_SearchCode(t *testing.T) {
	line := FormatToolLine(ToolDisplayInfo{
		ToolName: "search_code",
		Args: map[string]string{
			"pattern": "GetPricingInfo",
			"path":    "internal/agent/",
		},
		Result: "Found 8 match(es) in 4 file(s)\ninternal/agent/stats.go:42: ...",
	})

	want := `🔍 search_code: "GetPricingInfo" in internal/agent/ → 8 matches, 4 files`
	if line != want {
		t.Fatalf("FormatToolLine() = %q, want %q", line, want)
	}
}

func TestFormatToolLine_SearchCodeNoMatches(t *testing.T) {
	line := FormatToolLine(ToolDisplayInfo{
		ToolName: "search_code",
		Args: map[string]string{
			"pattern": "nonExistent",
		},
		Result: "No matches found",
	})

	want := `🔍 search_code: "nonExistent" → No matches found`
	if line != want {
		t.Fatalf("FormatToolLine() = %q, want %q", line, want)
	}
}

func TestFormatToolLine_SearchCodeImpact(t *testing.T) {
	line := FormatToolLine(ToolDisplayInfo{
		ToolName: "search_code",
		Args: map[string]string{
			"pattern": "GetPricingInfo",
			"path":    "internal/agent/",
			"intent":  "impact",
		},
		Result: "Found 8 match(es) in 4 file(s)\ninternal/agent/stats.go:42: ...",
	})

	want := `🔍 search_code: "GetPricingInfo" in internal/agent/ (impact) → 8 matches, 4 files`
	if line != want {
		t.Fatalf("FormatToolLine() = %q, want %q", line, want)
	}
}

func TestFormatToolLine_SearchCodeImpactMultiPatternSummary(t *testing.T) {
	line := FormatToolLine(ToolDisplayInfo{
		ToolName: "search_code",
		Args: map[string]string{
			"pattern": "GetPricingInfo",
			"path":    "internal/agent/",
			"intent":  "impact",
		},
		Result: strings.Join([]string{
			`━━ Pattern 1/3: "GetPricingInfo" ━━`,
			`Found 3 match(es) in 2 file(s)`,
			``,
			`📄 internal/agent/stats.go (2 match(es))`,
			`10: line`,
			`📄 internal/agent/cost.go (1 match(es))`,
			`20: line`,
			``,
			`━━ Pattern 2/3: "GetPricingInfo_test" ━━`,
			`Found 1 match(es) in 1 file(s)`,
			``,
			`📄 internal/agent/stats_test.go (1 match(es))`,
			`30: line`,
			``,
			`━━ Pattern 3/3: "GetPricingInfoImpl" ━━`,
			`No matches found`,
		}, "\n"),
	})

	want := `🔍 search_code: "GetPricingInfo" in internal/agent/ (impact) → 3 matches, 3 files`
	if line != want {
		t.Fatalf("FormatToolLine() = %q, want %q", line, want)
	}
}

func TestFormatToolLine_SearchCodeImpactMultiPatternSummary_DeduplicatesMatchesAndFiles(t *testing.T) {
	line := FormatToolLine(ToolDisplayInfo{
		ToolName: "search_code",
		Args: map[string]string{
			"pattern": "Foo",
			"path":    "internal/agent/",
			"intent":  "impact",
		},
		Result: strings.Join([]string{
			`━━ Pattern 1/2: "Foo" ━━`,
			`Found 2 match(es) in 1 file(s)`,
			``,
			`📄 internal/agent/foo.go (2 match(es))`,
			`10: Foo`,
			`20: FooImpl`,
			``,
			`━━ Pattern 2/2: "FooImpl" ━━`,
			`Found 2 match(es) in 1 file(s)`,
			``,
			`📄 internal/agent/foo.go (2 match(es))`,
			`20: FooImpl`,
			`30: Another FooImpl`,
		}, "\n"),
	})

	want := `🔍 search_code: "Foo" in internal/agent/ (impact) → 3 matches, 1 files`
	if line != want {
		t.Fatalf("FormatToolLine() = %q, want %q", line, want)
	}
}

func TestFormatToolLine_SearchCodeImpactMultiPatternSummary_CountsGroupedSymbolBundleItems(t *testing.T) {
	line := FormatToolLine(ToolDisplayInfo{
		ToolName: "search_code",
		Args: map[string]string{
			"pattern": "Close",
			"path":    "internal/agent/",
			"intent":  "impact",
		},
		Result: strings.Join([]string{
			`━━ Symbol Bundle: "Close" ━━`,
			`── func Close (L10-L20) in internal/agent/agent.go ──`,
			`Definition:`,
			`  10: func Close() {}`,
			``,
			`Callers (2):`,
			`  - internal/agent/run.go:40 | Close()`,
			`  - internal/agent/run.go:40 | Close()`,
			``,
			`Tests (1):`,
			`  - internal/agent/agent_test.go:15 | func TestClose`,
		}, "\n"),
	})

	want := `🔍 search_code: "Close" in internal/agent/ (impact) → 3 matches, 3 files`
	if line != want {
		t.Fatalf("FormatToolLine() = %q, want %q", line, want)
	}
}

func TestFormatToolLine_SearchCodeImpactMultiPatternSummary_CountsGroupedSymbolBundleItems_WindowsPath(t *testing.T) {
	line := FormatToolLine(ToolDisplayInfo{
		ToolName: "search_code",
		Args: map[string]string{
			"pattern": "Close",
			"path":    ".",
			"intent":  "impact",
		},
		Result: strings.Join([]string{
			`━━ Symbol Bundle: "Close" ━━`,
			`── func Close (L10-L20) in C:\repo\agent.go ──`,
			`Definition:`,
			`  10: func Close() {}`,
			``,
			`Callers (2):`,
			`  - C:\repo\run.go:40 | Close()`,
			`  - C:\repo\run.go:40 | Close()`,
			``,
			`Tests (1):`,
			`  - C:\repo\agent_test.go:15 | func TestClose`,
		}, "\n"),
	})

	want := `🔍 search_code: "Close" in . (impact) → 3 matches, 3 files`
	if line != want {
		t.Fatalf("FormatToolLine() = %q, want %q", line, want)
	}
}

func TestFormatToolLine_SearchCodeImpactMultiPatternSummary_CountsGroupedSymbolBundleItems_MixedWindowsPathSeparators(t *testing.T) {
	line := FormatToolLine(ToolDisplayInfo{
		ToolName: "search_code",
		Args: map[string]string{
			"pattern": "Close",
			"path":    ".",
			"intent":  "impact",
		},
		Result: strings.Join([]string{
			`━━ Symbol Bundle: "Close" ━━`,
			`── func Close (L10-L20) in C:\repo\agent.go ──`,
			`Definition:`,
			`  10: func Close() {}`,
			``,
			`Callers (2):`,
			`  - C:\repo\run.go:40 | Close()`,
			`  - C:/repo/run.go:40 | Close()`,
			``,
			`Tests (1):`,
			`  - C:\repo\agent_test.go:15 | func TestClose`,
		}, "\n"),
	})

	want := `🔍 search_code: "Close" in . (impact) → 3 matches, 3 files`
	if line != want {
		t.Fatalf("FormatToolLine() = %q, want %q", line, want)
	}
}

func TestFormatToolLine_SearchCodeImpactMultiPatternSummary_CountsFormattedTextSearchMatches(t *testing.T) {
	line := FormatToolLine(ToolDisplayInfo{
		ToolName: "search_code",
		Args: map[string]string{
			"pattern": "helper",
			"path":    ".",
			"intent":  "impact",
		},
		Result: strings.Join([]string{
			`━━ Pattern 1/3: "helper" ━━`,
			`Found 2 match(es) in 1 file(s)`,
			``,
			`📄 pkg/helper.go (2 match(es)) @loc1`,
			`[def] >  42 │ func helper() {}`,
			`[call] >  84 │ helper()`,
			``,
			`━━ Pattern 2/3: "helperImpl" ━━`,
			`Found 1 match(es) in 1 file(s)`,
			``,
			`📄 pkg/helper.go (1 match(es)) @loc2`,
			`[def] >  84 │ func helperImpl() {}`,
			``,
			`━━ Pattern 3/3: "TestHelper" ━━`,
			`Found 1 match(es) in 1 file(s)`,
			``,
			`📄 pkg/helper_test.go (1 match(es)) @loc3`,
			`[def] >  12 │ func TestHelper(t *testing.T) {}`,
		}, "\n"),
	})

	want := `🔍 search_code: "helper" in . (impact) → 3 matches, 2 files`
	if line != want {
		t.Fatalf("FormatToolLine() = %q, want %q", line, want)
	}
}

func TestFormatToolLine_ReadFile(t *testing.T) {
	line := FormatToolLine(ToolDisplayInfo{
		ToolName: "read_file",
		Args: map[string]string{
			"paths": `["example.txt"]`,
		},
		Result: "1: line1\n2: line2\n3: line3\n",
	})

	want := "📄 read_file: example.txt (3 lines)"
	if line != want {
		t.Fatalf("FormatToolLine() = %q, want %q", line, want)
	}
}

func TestFormatToolLine_GatherContext(t *testing.T) {
	line := FormatToolLine(ToolDisplayInfo{
		ToolName: "gather_context",
		Args: map[string]string{
			"query":       "Run",
			"path":        "internal/agent",
			"file_filter": "go",
		},
		Result: "Route: Structured impact\n\nSearch / Discovery\n...",
	})

	want := `🧭 gather_context: "Run" in internal/agent [go]`
	if line != want {
		t.Fatalf("FormatToolLine() = %q, want %q", line, want)
	}
}

func TestFormatToolLine_ReadFile_GoOutline(t *testing.T) {
	line := FormatToolLine(ToolDisplayInfo{
		ToolName: "read_file",
		Args: map[string]string{
			"paths": `["server.go"]`,
		},
		Result: "1: package main\n\n--- Signatures ---\n  L50  func Build\n\n(200 lines total. For specific sections: paths=[\"server.go:start-end\"])\n",
	})

	want := `📄 read_file: server.go (outline of 200 lines)`
	if line != want {
		t.Fatalf("FormatToolLine() = %q, want %q", line, want)
	}
}

func TestFormatToolLine_ReadFile_NonGoOutline(t *testing.T) {
	line := FormatToolLine(ToolDisplayInfo{
		ToolName: "read_file",
		Args: map[string]string{
			"paths": `["data.txt"]`,
		},
		Result: "1: header\n\n--- Last lines ---\n150: end\n\n(150 lines total. For specific sections: paths=[\"data.txt:start-end\"])\n",
	})

	want := `📄 read_file: data.txt (outline of 150 lines)`
	if line != want {
		t.Fatalf("FormatToolLine() = %q, want %q", line, want)
	}
}

func TestFormatToolLine_ReadFile_MultiPaths(t *testing.T) {
	line := FormatToolLine(ToolDisplayInfo{
		ToolName: "read_file",
		Args: map[string]string{
			"paths": `["a.go","b.go"]`,
		},
		Result: "📄 File: a.go\n1: package main\n\n📄 File: b.go\n1: package util\n",
	})

	want := "📄 read_file: a.go, b.go"
	if line != want {
		t.Fatalf("FormatToolLine() = %q, want %q", line, want)
	}
}

func TestFormatToolLine_ReadFiles_MultiPaths(t *testing.T) {
	line := FormatToolLine(ToolDisplayInfo{
		ToolName: "read_files",
		Args: map[string]string{
			"paths": `["internal/tools/search/web.go","internal/tools/search/register.go","internal/api/websearch/registry.go"]`,
		},
		Result: "",
	})

	want := "📄 read_files: web.go, register.go, registry.go"
	if line != want {
		t.Fatalf("FormatToolLine() = %q, want %q", line, want)
	}
}

func TestFormatToolLine_StrReplace(t *testing.T) {
	line := FormatToolLine(ToolDisplayInfo{
		ToolName: "str_replace",
		Args: map[string]string{
			"path": "auto_compress.go",
		},
		Result: "Successfully replaced text in auto_compress.go (lines 30-30 → 30-32)",
	})

	want := "✏️ str_replace: auto_compress.go:30"
	if line != want {
		t.Fatalf("FormatToolLine() = %q, want %q", line, want)
	}
}

func TestFormatToolLine_SpawnAgent(t *testing.T) {
	line := FormatToolLine(ToolDisplayInfo{
		ToolName: "spawn_agent",
		Args: map[string]string{
			"message": "register.goとweb.goを読んで差分を報告しろ",
		},
		Result: `{"agent_id":"sub-001","status":"running"}`,
	})

	want := "🚀 spawn_agent: register.goとweb.goを読んで差分を報告しろ"
	if line != want {
		t.Fatalf("FormatToolLine() = %q, want %q", line, want)
	}
}

func TestFormatToolLine_SpawnAgent_FirstLineAndLongerLimit(t *testing.T) {
	message := strings.Repeat("a", 80) + "\n" + strings.Repeat("b", 80)
	line := FormatToolLine(ToolDisplayInfo{
		ToolName: "spawn_agent",
		Args: map[string]string{
			"message": message,
		},
		Result: `{"agent_id":"sub-001","status":"running"}`,
	})

	want := "🚀 spawn_agent: " + strings.Repeat("a", 80)
	if line != want {
		t.Fatalf("FormatToolLine() = %q, want %q", line, want)
	}
}

func TestFormatToolLine_WaitAgent(t *testing.T) {
	line := FormatToolLine(ToolDisplayInfo{
		ToolName: "wait_agent",
		Args: map[string]string{
			"ids": `["sub-001","sub-002"]`,
		},
		Result: `{"status":"completed","results":[]}`,
	})

	want := "⏳ wait_agent: 2 agents"
	if line != want {
		t.Fatalf("FormatToolLine() = %q, want %q", line, want)
	}
}

func TestFormatParallelElapsed(t *testing.T) {
	if got := FormatParallelElapsed(250 * time.Millisecond); got != "250ms" {
		t.Fatalf("FormatParallelElapsed(250ms) = %q, want %q", got, "250ms")
	}
	if got := FormatParallelElapsed(1200 * time.Millisecond); got != "1.2s" {
		t.Fatalf("FormatParallelElapsed(1.2s) = %q, want %q", got, "1.2s")
	}
}

func TestPrintParallelGroupToWriter_UsesInjectedWriter(t *testing.T) {
	var buf bytes.Buffer

	PrintParallelGroupStartToWriter(&buf, 2)
	PrintParallelGroupLineToWriter(&buf, "📄 read_file: file.go")
	PrintParallelGroupEndToWriter(&buf, "Done: 2 tools")

	output := buf.String()
	for _, want := range []string{
		"┌ Parallel (2 calls)",
		"│ 📄 read_file: file.go",
		"└ Done: 2 tools",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("expected injected output to contain %q, got %q", want, output)
		}
	}
}

func TestFormatMultiplePathNames(t *testing.T) {
	tests := []struct {
		name  string
		paths []string
		want  string
	}{
		{
			name:  "no duplicate basenames",
			paths: []string{"internal/tools/search/web.go", "internal/tools/search/register.go"},
			want:  "web.go, register.go",
		},
		{
			name:  "duplicate basenames use parent directory",
			paths: []string{"internal/tools/search/web.go", "internal/api/web.go"},
			want:  "search/web.go, api/web.go",
		},
		{
			name:  "more than five entries are truncated",
			paths: []string{"a.go", "b.go", "c.go", "d.go", "e.go", "f.go"},
			want:  "a.go, b.go, c.go, d.go, e.go ... +1 more",
		},
		{
			name:  "line ranges are stripped before display",
			paths: []string{"internal/ui/tool_display.go:100-200", "internal/ui/colors.go:1-10"},
			want:  "tool_display.go, colors.go",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := formatMultiplePathNames(tt.paths); got != tt.want {
				t.Fatalf("formatMultiplePathNames() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestFormatApplyPatchSummary(t *testing.T) {
	tests := []struct {
		name   string
		result string
		want   string
	}{
		{
			name:   "no files changed",
			result: "",
			want:   "No files changed",
		},
		{
			name:   "modified files only",
			result: "Modified: a.go, b.go\n",
			want:   "2 files (2 modified)",
		},
		{
			name:   "added files only",
			result: "Added: new1.txt, new2.txt, new3.txt\n",
			want:   "3 files (3 added)",
		},
		{
			name:   "deleted files only",
			result: "Deleted: old.txt\n",
			want:   "1 files (1 deleted)",
		},
		{
			name:   "mixed changes",
			result: "Added: new.txt\nModified: a.go, b.go\nDeleted: old.txt\n",
			want:   "4 files (2 modified, 1 added, 1 deleted)",
		},
		{
			name:   "none for each type",
			result: "Added: (none)\nModified: (none)\nDeleted: (none)\n",
			want:   "No files changed",
		},
		{
			name:   "empty paths string",
			result: "Added: \nModified: \nDeleted: \n",
			want:   "No files changed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatApplyPatchSummary(nil, tt.result)
			if got != tt.want {
				t.Errorf("formatApplyPatchSummary() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestFormatToolLine_ErrorIncludesTarget(t *testing.T) {
	line := FormatToolLine(ToolDisplayInfo{
		ToolName: "read_file",
		Args: map[string]string{
			"path": "internal/lsp/server.go",
		},
		Result: "Error: permission denied\nsecond line",
	})

	want := "❌ read_file: internal/lsp/server.go → Error: permission denied"
	if line != want {
		t.Fatalf("FormatToolLine() = %q, want %q", line, want)
	}
}

func TestFormatWriteFileSummary(t *testing.T) {
	got := formatWriteFileSummary(map[string]string{"path": "tmp/out.txt"}, "Successfully wrote 42 bytes (3 lines) to tmp/out.txt")
	if got != "tmp/out.txt (3 lines)" {
		t.Fatalf("formatWriteFileSummary() = %q, want %q", got, "tmp/out.txt (3 lines)")
	}
}

func TestFormatCopyFileSummary_FallsBackToSinglePath(t *testing.T) {
	got := formatCopyFileSummary(map[string]string{"dest": "tmp/out.txt"})
	if got != "tmp/out.txt" {
		t.Fatalf("formatCopyFileSummary() = %q, want %q", got, "tmp/out.txt")
	}
}

func TestFormatGitSummary_PrefersPathOrder(t *testing.T) {
	got := formatGitSummary(map[string]string{
		"message": "commit message",
		"branch":  "feature/test",
		"path":    "internal/api",
	})
	if got != "internal/api" {
		t.Fatalf("formatGitSummary() = %q, want %q", got, "internal/api")
	}
}

func TestReadFileDisplayTarget_FallsBackToPath(t *testing.T) {
	got := readFileDisplayTarget(map[string]string{"path": "internal/api/base.go"})
	if got != "internal/api/base.go" {
		t.Fatalf("readFileDisplayTarget() = %q, want %q", got, "internal/api/base.go")
	}
}

func TestFormatToolSummary_UnknownToolUsesSortedArgs(t *testing.T) {
	got := formatToolSummary(ToolDisplayInfo{
		ToolName: "custom_tool",
		Args: map[string]string{
			"z": "last",
			"a": "first",
		},
	}, "")
	if got != "first" {
		t.Fatalf("formatToolSummary() = %q, want %q", got, "first")
	}
}
