package applypatch

import (
	"fmt"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/tools"
	"github.com/susugadx/xelyon-cli/internal/tools/common"
	"github.com/susugadx/xelyon-cli/internal/uifileview"
)

func showCodexStyleResult(out common.Output, result *ApplyResult) {
	if out.SuppressStdout() {
		return
	}

	showApplyPatchHeader(out, len(result.details))
	uifileview.ShowCodexStyleDiff(out.StdoutWriter(), buildCodexStyleFiles(result.details))
}

func buildCodexStyleFiles(details []applyResultDetail) []uifileview.PatchFileDisplay {
	files := make([]uifileview.PatchFileDisplay, 0, len(details))
	for _, d := range details {
		files = append(files, uifileview.PatchFileDisplay{
			Path:       d.Path,
			Action:     d.Action,
			OldContent: d.OldContent,
			NewContent: d.NewContent,
		})
	}
	return files
}

func showApplyPatchPreview(out common.Output, patchText string, hunks []Hunk) {
	if out.SuppressStdout() {
		return
	}

	showApplyPatchHeader(out, len(hunks))
	showApplyPatchPreviewBody(out, patchText, hunks)
}

func showApplyPatchHeader(out common.Output, operations int) {
	uifileview.FileOpHeader(out.StdoutWriter(), "apply_patch", fmt.Sprintf("%d files", operations))
}

func showApplyPatchPreviewBody(out common.Output, patchText string, hunks []Hunk) {
	uifileview.ShowPatchToWriter(out.StdoutWriter(), patchText)

	lineStats := countLinesPerPath(patchText)
	w := out.StdoutWriter()
	for _, hunk := range hunks {
		status, target, lineInfo := formatPreviewHunkSummary(hunk, lineStats[hunk.Path])
		uifileview.FileOpPathLine(w, status, target, lineInfo)
	}
	out.Println()
}

func formatPreviewHunkSummary(hunk Hunk, lineStats [2]int) (status string, target string, lineInfo string) {
	lineInfo = ""
	if lineStats[0] > 0 || lineStats[1] > 0 {
		lineInfo = fmt.Sprintf("(+%d, -%d)", lineStats[0], lineStats[1])
	}

	switch hunk.Type {
	case "add":
		return "A", hunk.Path, lineInfo
	case "delete":
		return "D", hunk.Path, ""
	default:
		target = hunk.Path
		if hunk.MovePath != "" {
			target = hunk.Path + " -> " + hunk.MovePath
		}
		return "M", target, lineInfo
	}
}

func countLinesPerPath(patchText string) map[string][2]int {
	counts := make(map[string][2]int) // path -> [added, removed]
	lines := strings.Split(patchText, "\n")
	currentPath := ""

	for _, line := range lines {
		switch {
		case strings.HasPrefix(line, "*** Add File: "):
			currentPath = strings.TrimSpace(strings.TrimPrefix(line, "*** Add File: "))
		case strings.HasPrefix(line, "*** Update File: "):
			currentPath = strings.TrimSpace(strings.TrimPrefix(line, "*** Update File: "))
		case strings.HasPrefix(line, "*** Delete File: "):
			currentPath = strings.TrimSpace(strings.TrimPrefix(line, "*** Delete File: "))
		case strings.HasPrefix(line, "*** Move to: "):
		case strings.HasPrefix(line, "+") && currentPath != "":
			c := counts[currentPath]
			c[0]++
			counts[currentPath] = c
		case strings.HasPrefix(line, "-") && currentPath != "":
			c := counts[currentPath]
			c[1]++
			counts[currentPath] = c
		}
	}
	return counts
}

func formatApplyResult(result *ApplyResult) string {
	return fmt.Sprintf(
		"✓ Patch applied successfully.\nAdded: %s\nModified: %s\nDeleted: %s",
		formatApplyPaths(result.Added),
		formatApplyPaths(result.Modified),
		formatApplyPaths(result.Deleted),
	)
}

func formatApplyPaths(paths []string) string {
	if len(paths) == 0 {
		return "(none)"
	}
	return strings.Join(paths, ", ")
}

func buildApplyPatchFileChange(result *ApplyResult) *tools.FileChange {
	if result == nil || len(result.details) == 0 {
		return nil
	}

	change := &tools.FileChange{
		Timestamp: common.GetCurrentTime(),
		Tool:      "apply_patch",
		Details:   buildFileChangeDetails(result.details),
		FilePath:  result.details[0].Path,
	}
	change.Description = buildApplyPatchChangeDescription(result.details)
	return change
}

func buildFileChangeDetails(details []applyResultDetail) []tools.FileChangeDetail {
	fileDetails := make([]tools.FileChangeDetail, 0, len(details))
	for _, detail := range details {
		fileDetails = append(fileDetails, tools.FileChangeDetail{
			FilePath:     detail.Path,
			Action:       detail.Action,
			LinesAdded:   detail.LinesAdded,
			LinesRemoved: detail.LinesRemoved,
		})
	}
	return fileDetails
}

func buildApplyPatchChangeDescription(details []applyResultDetail) string {
	if len(details) == 1 {
		return "Patched file " + details[0].Path
	}
	return fmt.Sprintf("Patched %d files", len(details))
}
