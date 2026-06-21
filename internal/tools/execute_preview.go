package tools

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/fatih/color"
	"github.com/susugadx/xelyon-cli/internal/uiruntime"
)

// printToolArgs はツールの引数を簡潔に表示する（Execute/PreviewToolCallで共通使用）
func printToolArgs(w io.Writer, tc *ToolCall) {
	if tc != nil {
		if printer, ok := toolArgPreviewPrinters[tc.Tool]; ok {
			printer(w, tc)
			_, _ = fmt.Fprintln(w)
			return
		}
	}
	printGenericToolArgs(w, tc)
	_, _ = fmt.Fprintln(w)
}

func formatReadFilePreviewArg(args map[string]string) string {
	paths := previewReadFilePaths(args)
	switch len(paths) {
	case 0:
		return "Files: (none)"
	case 1:
		return "File: " + paths[0]
	default:
		return fmt.Sprintf("Files: %d", len(paths))
	}
}

func previewReadFilePaths(args map[string]string) []string {
	if rawPaths := strings.TrimSpace(args["paths"]); rawPaths != "" {
		var paths []string
		if err := json.Unmarshal([]byte(rawPaths), &paths); err == nil {
			return paths
		}
	}
	if path := strings.TrimSpace(args["path"]); path != "" {
		return []string{path}
	}
	return nil
}

// PreviewToolCallWithWriter は指定 writer にツール情報を表示する（実行はしない）。
func PreviewToolCallWithWriter(w io.Writer, tc *ToolCall) {
	if w == nil {
		w = uiruntime.DefaultRuntime().Output()
	}
	color.New(color.FgCyan).Fprintf(w, "🔧 Tool: %s (Dry Run)\n", tc.Tool)
	printToolArgs(w, tc)
}

// PreviewToolCall displays tool information without executing it
func PreviewToolCall(tc *ToolCall) {
	PreviewToolCallWithWriter(uiruntime.DefaultRuntime().Output(), tc)
}
