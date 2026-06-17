package tools

import (
	"fmt"
	"io"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/tools/common"
)

type toolArgPreviewPrinter func(io.Writer, *ToolCall)

var toolArgPreviewPrinters = buildToolArgPreviewPrinters()

func buildToolArgPreviewPrinters() map[string]toolArgPreviewPrinter {
	printers := map[string]toolArgPreviewPrinter{
		"gather_context": printGatherContextArgs,
		"read_file":      printReadFileArgs,
		"write_file":     printWriteFileArgs,
		"apply_patch":    printApplyPatchArgs,
		"str_replace":    printStrReplaceArgs,
		"bash":           printBashArgs,
		"list_dir":       printListDirArgs,
		"copy_file":      printCopyFileArgs,
		"delete_file":    printDeleteFileArgs,
		"lint":           printLintArgs,
		"search_code":    printSearchCodeArgs,
		"web_search":     printWebSearchArgs,
	}

	for _, name := range gitToolNamesForPreview {
		printers[name] = printGitToolArgs
	}
	return printers
}

var gitToolNamesForPreview = []string{
	"git_add", "git_commit", "git_push", "git_status", "git_diff", "git_log",
	"git_branch", "git_checkout", "git_stash",
}

func printGatherContextArgs(w io.Writer, tc *ToolCall) {
	_, _ = fmt.Fprintf(w, "   Query: %s\n", common.Truncate(tc.Args["query"], 60))
	if tc.Args["path"] != "" {
		_, _ = fmt.Fprintf(w, "   Path: %s\n", tc.Args["path"])
	}
	if tc.Args["file_filter"] != "" {
		_, _ = fmt.Fprintf(w, "   File filter: %s\n", tc.Args["file_filter"])
	}
}

func printReadFileArgs(w io.Writer, tc *ToolCall) {
	_, _ = fmt.Fprintf(w, "   %s\n", formatReadFilePreviewArg(tc.Args))
}

func printWriteFileArgs(w io.Writer, tc *ToolCall) {
	lines := strings.Split(tc.Args["content"], "\n")
	_, _ = fmt.Fprintf(w, "   File: %s (%d lines)\n", tc.Args["path"], len(lines))
}

func printApplyPatchArgs(w io.Writer, tc *ToolCall) {
	lines := strings.Split(tc.Args["patch"], "\n")
	_, _ = fmt.Fprintf(w, "   Patch: %d lines\n", len(lines))
}

func printStrReplaceArgs(w io.Writer, tc *ToolCall) {
	_, _ = fmt.Fprintf(w, "   File: %s\n", tc.Args["path"])
}

func printBashArgs(w io.Writer, tc *ToolCall) {
	_, _ = fmt.Fprintf(w, "   Command: %s\n", common.Truncate(tc.Args["command"], 60))
}

func printListDirArgs(w io.Writer, tc *ToolCall) {
	path := tc.Args["path"]
	if path == "" {
		path = "."
	}
	_, _ = fmt.Fprintf(w, "   Directory: %s\n", path)
}

func printGitToolArgs(w io.Writer, tc *ToolCall) {
	for k, v := range tc.Args {
		if v != "" {
			_, _ = fmt.Fprintf(w, "   %s: %s\n", k, common.Truncate(v, 60))
		}
	}
}

func printCopyFileArgs(w io.Writer, tc *ToolCall) {
	_, _ = fmt.Fprintf(w, "   Source: %s\n", tc.Args["src"])
	_, _ = fmt.Fprintf(w, "   Destination: %s\n", tc.Args["dest"])
}

func printDeleteFileArgs(w io.Writer, tc *ToolCall) {
	_, _ = fmt.Fprintf(w, "   File: %s\n", tc.Args["path"])
}

func printLintArgs(w io.Writer, tc *ToolCall) {
	path := tc.Args["path"]
	if path == "" {
		path = "."
	}
	_, _ = fmt.Fprintf(w, "   Path: %s\n", path)
	if tc.Args["auto_fix"] == "true" {
		_, _ = fmt.Fprintf(w, "   Auto-fix: enabled\n")
	}
}

func printSearchCodeArgs(w io.Writer, tc *ToolCall) {
	_, _ = fmt.Fprintf(w, "   Pattern: %s\n", tc.Args["pattern"])
	if tc.Args["path"] != "" {
		_, _ = fmt.Fprintf(w, "   Path: %s\n", tc.Args["path"])
	}
	if tc.Args["file_filter"] != "" {
		_, _ = fmt.Fprintf(w, "   File filter: %s\n", tc.Args["file_filter"])
	} else if tc.Args["file_pattern"] != "" {
		_, _ = fmt.Fprintf(w, "   File pattern: %s\n", tc.Args["file_pattern"])
	}
}

func printWebSearchArgs(w io.Writer, tc *ToolCall) {
	_, _ = fmt.Fprintf(w, "   Query: %s\n", tc.Args["query"])
}

func printGenericToolArgs(w io.Writer, tc *ToolCall) {
	if tc == nil || len(tc.Args) == 0 {
		return
	}
	for k, v := range tc.Args {
		_, _ = fmt.Fprintf(w, "   %s: %s\n", k, common.Truncate(v, 60))
	}
}
