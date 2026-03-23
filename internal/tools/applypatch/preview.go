package applypatch

import (
	"fmt"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/tools/common"
	"github.com/susugadx/xelyon-cli/internal/ui"
)

// BuildPatchPreview は apply_patch テキストから行番号付きプレビュー情報を構築する。
func BuildPatchPreview(patchText string, readFile func(path string) ([]byte, error)) ([]ui.PatchFilePreview, error) {
	parsed, err := ParsePatch(patchText)
	if err != nil {
		return nil, err
	}

	previews := make([]ui.PatchFilePreview, 0, len(parsed.Hunks))
	for _, hunk := range parsed.Hunks {
		switch hunk.Type {
		case "add":
			previews = append(previews, buildAddFilePreview(hunk))
		case "delete":
			previews = append(previews, buildDeleteFilePreview(hunk, readFile))
		case "update":
			preview, err := buildUpdateFilePreview(hunk, readFile)
			if err != nil {
				return nil, err
			}
			previews = append(previews, preview)
		default:
			return nil, fmt.Errorf("unsupported hunk type: %s", hunk.Type)
		}
	}

	return previews, nil
}

func buildAddFilePreview(hunk Hunk) ui.PatchFilePreview {
	lines := splitPreviewContentLines(hunk.Contents)
	previewLines := make([]ui.PatchPreviewLine, 0, len(lines))
	for i, line := range lines {
		previewLines = append(previewLines, ui.PatchPreviewLine{
			Type:    '+',
			LineNum: i + 1,
			Text:    line,
		})
	}

	return ui.PatchFilePreview{
		Path:   hunk.Path,
		Action: hunk.Type,
		Added:  len(lines),
		Hunks: []ui.PatchHunkPreview{{
			StartLine: 1,
			Lines:     previewLines,
		}},
	}
}

func buildDeleteFilePreview(hunk Hunk, readFile func(path string) ([]byte, error)) ui.PatchFilePreview {
	preview := ui.PatchFilePreview{
		Path:   hunk.Path,
		Action: hunk.Type,
	}

	if readFile == nil {
		return preview
	}

	absPath, err := common.ValidatePath(hunk.Path)
	if err != nil {
		return preview
	}

	contents, err := readFile(absPath)
	if err != nil {
		return preview
	}
	preview.Removed = countPreviewContentLines(string(contents))
	return preview
}

func buildUpdateFilePreview(hunk Hunk, readFile func(path string) ([]byte, error)) (ui.PatchFilePreview, error) {
	if readFile == nil {
		return ui.PatchFilePreview{}, fmt.Errorf("readFile is required for update preview")
	}

	absPath, err := common.ValidatePath(hunk.Path)
	if err != nil {
		return ui.PatchFilePreview{}, err
	}

	contents, err := readFile(absPath)
	if err != nil {
		return ui.PatchFilePreview{}, err
	}

	originalLines := splitPreviewContentLines(string(contents))
	preview := ui.PatchFilePreview{
		Path:     hunk.Path,
		Action:   hunk.Type,
		MovePath: hunk.MovePath,
		Hunks:    make([]ui.PatchHunkPreview, 0, len(hunk.Chunks)),
	}

	lineIndex := 0
	lineDelta := 0
	prevChunkEnd := 0
	for _, chunk := range hunk.Chunks {
		result, err := LocateChunk(originalLines, hunk.Path, chunk, lineIndex, prevChunkEnd)
		if err != nil {
			return ui.PatchFilePreview{}, err
		}

		added, removed := countPreviewLines(chunk.previewLines)
		preview.Added += added
		preview.Removed += removed
		preview.Hunks = append(preview.Hunks, buildChunkPreview(result.StartIdx, lineDelta, chunk.previewLines))

		lineIndex = result.NextIndex
		if len(result.Pattern) > 0 {
			prevChunkEnd = result.StartIdx + len(result.Pattern)
		}
		lineDelta += len(result.NewLines) - len(result.Pattern)
	}

	return preview, nil
}

func buildChunkPreview(startIdx, lineDelta int, lines []patchLine) ui.PatchHunkPreview {
	oldLine := startIdx + 1
	newLine := startIdx + 1 + lineDelta
	previewLines := make([]ui.PatchPreviewLine, 0, len(lines))

	for _, line := range lines {
		switch line.Type {
		case '-':
			previewLines = append(previewLines, ui.PatchPreviewLine{
				Type:    '-',
				LineNum: oldLine,
				Text:    line.Text,
			})
			oldLine++
		case '+':
			previewLines = append(previewLines, ui.PatchPreviewLine{
				Type:    '+',
				LineNum: newLine,
				Text:    line.Text,
			})
			newLine++
		default:
			previewLines = append(previewLines, ui.PatchPreviewLine{
				Type:    ' ',
				LineNum: newLine,
				Text:    line.Text,
			})
			oldLine++
			newLine++
		}
	}

	startLine := startIdx + 1 + lineDelta
	if len(previewLines) > 0 {
		startLine = previewLines[0].LineNum
	}

	return ui.PatchHunkPreview{
		StartLine: startLine,
		Lines:     previewLines,
	}
}

func countPreviewLines(lines []patchLine) (added, removed int) {
	for _, line := range lines {
		switch line.Type {
		case '+':
			added++
		case '-':
			removed++
		}
	}
	return added, removed
}

func splitPreviewContentLines(content string) []string {
	if content == "" {
		return nil
	}
	parts := strings.Split(content, "\n")
	if len(parts) > 0 && parts[len(parts)-1] == "" {
		parts = parts[:len(parts)-1]
	}
	return parts
}

func countPreviewContentLines(content string) int {
	return len(splitPreviewContentLines(content))
}
