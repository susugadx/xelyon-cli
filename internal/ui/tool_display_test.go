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
