package uifileview

import (
	"fmt"
	"io"

	"github.com/susugadx/xelyon-cli/internal/uistyle"
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
	pal := uistyle.NewFileOpPalette(out)

	for _, preview := range previews {
		showPatchFilePreview(out, pal, preview)
	}
}

// ShowSinglePatchPreview は単一ファイルのパッチプレビューを出力する。
func ShowSinglePatchPreview(out io.Writer, preview PatchFilePreview) {
	pal := uistyle.NewFileOpPalette(out)
	showPatchFilePreview(out, pal, preview)
}

func showPatchFilePreview(out io.Writer, pal uistyle.FileOpPalette, preview PatchFilePreview) {
	showPatchFileHeader(out, pal, preview.Action, preview.Path, preview.MovePath, preview.Added, preview.Removed)

	switch preview.Action {
	case "delete", "deleted":
		if len(preview.Hunks) > 0 {
			renderPatchLines(out, pal, preview.Hunks[0].Lines)
		}
		fmt.Fprintln(out)
		return
	case "add", "created":
		if len(preview.Hunks) > 0 {
			renderPatchLines(out, pal, preview.Hunks[0].Lines)
		}
		fmt.Fprintln(out)
		return
	default:
		for i, hunk := range preview.Hunks {
			if i > 0 {
				pal.Muted(out, "     :\n")
			}
			renderPatchLines(out, pal, hunk.Lines)
		}
		fmt.Fprintln(out)
	}
}

func showPatchFileHeader(out io.Writer, pal uistyle.FileOpPalette, action, path, movePath string, added, removed int) {
	pal.Accent(out, fmt.Sprintf("  • %s %s", formatPatchAction(action), formatPatchTarget(path, movePath)))
	if counts := formatPatchCounts(added, removed); counts != "" {
		pal.Muted(out, fmt.Sprintf(" %s", counts))
	}
	fmt.Fprintln(out)
}

func renderPatchLines(out io.Writer, pal uistyle.FileOpPalette, lines []PatchPreviewLine) {
	for _, line := range lines {
		switch line.Type {
		case '-':
			pal.DelLine(out, fmt.Sprintf("  %4d - %s\n", line.LineNum, line.Text))
		case '+':
			pal.AddLine(out, fmt.Sprintf("  %4d + %s\n", line.LineNum, line.Text))
		default:
			pal.Context(out, fmt.Sprintf("  %4d   %s\n", line.LineNum, line.Text))
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
