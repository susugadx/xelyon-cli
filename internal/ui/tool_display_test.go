package ui

import (
	"bytes"
	"strings"
	"testing"
	"time"
)

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
			"path": "example.txt",
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
			"path": "server.go",
		},
		Result: "1: package main\n\n--- Signatures ---\n  L50  func Build\n\n(200 lines total. Use start_line/end_line or symbol=\"Name\" to read details)\n",
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
			"path": "data.txt",
		},
		Result: "1: header\n\n--- Last lines ---\n150: end\n\n(150 lines total. Use start_line/end_line to read specific sections)\n",
	})

	want := `📄 read_file: data.txt (outline of 150 lines)`
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
