package ui

import (
	"fmt"
	"io"

	"github.com/fatih/color"
)

// PatchPreviewLine はパッチプレビュー内の1行を表す。
type PatchPreviewLine struct {
	Type    rune
	LineNum int
	Text    string
}

// PatchHunkPreview は1つの hunk の表示情報を表す。
type PatchHunkPreview struct {
	StartLine int
	Lines     []PatchPreviewLine
}

// PatchFilePreview はファイル単位のパッチプレビュー情報を表す。
type PatchFilePreview struct {
	Path     string
	Action   string
	MovePath string
	Added    int
	Removed  int
	Hunks    []PatchHunkPreview
}

// ShowPatchPreview は行番号付きの Codex 風パッチプレビューを出力する。
func ShowPatchPreview(out io.Writer, previews []PatchFilePreview) {
	bold := color.New(color.Bold)
	green := color.New(color.FgGreen)
	red := color.New(color.FgRed)
	dim := color.New(color.Faint)

	for _, preview := range previews {
		showPatchFilePreview(out, bold, green, red, dim, preview)
	}
}

func showPatchFilePreview(out io.Writer, bold, green, red, dim *color.Color, preview PatchFilePreview) {
	showPatchFileHeader(out, bold, dim, preview.Action, preview.Path, preview.MovePath, preview.Added, preview.Removed)

	switch preview.Action {
	case "delete", "deleted":
		fmt.Fprintln(out)
		return
	case "add", "created":
		if len(preview.Hunks) > 0 {
			renderPatchLines(out, green, red, dim, preview.Hunks[0].Lines)
		}
		fmt.Fprintln(out)
		return
	default:
		for i, hunk := range preview.Hunks {
			if i > 0 {
				dim.Fprintf(out, "     :\n")
			}
			renderPatchLines(out, green, red, dim, hunk.Lines)
		}
		fmt.Fprintln(out)
	}
}

func showPatchFileHeader(out io.Writer, bold, dim *color.Color, action, path, movePath string, added, removed int) {
	bold.Fprintf(out, "  • %s %s", formatPatchAction(action), formatPatchTarget(path, movePath))
	if counts := formatPatchCounts(added, removed); counts != "" {
		dim.Fprintf(out, " %s", counts)
	}
	dim.Fprintln(out)
}

func renderPatchLines(out io.Writer, green, red, dim *color.Color, lines []PatchPreviewLine) {
	for _, line := range lines {
		switch line.Type {
		case '-':
			red.Fprintf(out, "  %4d - %s\n", line.LineNum, line.Text)
		case '+':
			green.Fprintf(out, "  %4d + %s\n", line.LineNum, line.Text)
		default:
			dim.Fprintf(out, "  %4d   %s\n", line.LineNum, line.Text)
		}
	}
}

func formatPatchAction(action string) string {
	switch action {
	case "add", "created":
		return "Creating"
	case "delete", "deleted":
		return "Deleting"
	default:
		return "Editing"
	}
}

func formatPatchTarget(path, movePath string) string {
	if movePath != "" && movePath != path {
		return path + " -> " + movePath
	}
	return path
}

func formatPatchCounts(added, removed int) string {
	switch {
	case added > 0 && removed > 0:
		return fmt.Sprintf("(+%d, -%d)", added, removed)
	case added > 0:
		return fmt.Sprintf("(+%d)", added)
	case removed > 0:
		return fmt.Sprintf("(-%d)", removed)
	default:
		return ""
	}
}
