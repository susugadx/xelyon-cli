package applypatch

import (
	"fmt"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/tools/common"
	"github.com/susugadx/xelyon-cli/internal/uifileview"
)

type patchPreviewBuilder struct {
	readFile func(path string) ([]byte, error)
}

type updatePreviewState struct {
	hunk          Hunk
	originalLines []string
	preview       uifileview.PatchFilePreview
	lineIndex     int
	lineDelta     int
	prevChunkEnd  int
}

// BuildPatchPreview は apply_patch テキストから行番号付きプレビュー情報を構築する。
func BuildPatchPreview(patchText string, readFile func(path string) ([]byte, error)) ([]uifileview.PatchFilePreview, error) {
	parsed, err := ParsePatch(patchText)
	if err != nil {
		return nil, err
	}

	builder := patchPreviewBuilder{readFile: readFile}
	previews := make([]uifileview.PatchFilePreview, 0, len(parsed.Hunks))
	for _, hunk := range parsed.Hunks {
		preview, err := builder.buildHunkPreview(hunk)
		if err != nil {
			return nil, err
		}
		previews = append(previews, preview)
	}

	return previews, nil
}

func (b patchPreviewBuilder) buildHunkPreview(hunk Hunk) (uifileview.PatchFilePreview, error) {
	switch hunk.Type {
	case "add":
		return buildAddFilePreview(hunk), nil
	case "delete":
		return buildDeleteFilePreview(hunk, b.readFile), nil
	case "update":
		return buildUpdateFilePreview(hunk, b.readFile)
	default:
		return uifileview.PatchFilePreview{}, fmt.Errorf("unsupported hunk type: %s", hunk.Type)
	}
}

func buildAddFilePreview(hunk Hunk) uifileview.PatchFilePreview {
	lines := splitPreviewContentLines(hunk.Contents)

	return uifileview.PatchFilePreview{
		Path:   hunk.Path,
		Action: hunk.Type,
		Added:  len(lines),
		Hunks: []uifileview.PatchHunkPreview{{
			StartLine: 1,
			Lines:     buildAddPreviewLines(lines),
		}},
	}
}

func buildAddPreviewLines(lines []string) []uifileview.PatchPreviewLine {
	previewLines := make([]uifileview.PatchPreviewLine, 0, len(lines))
	for i, line := range lines {
		previewLines = append(previewLines, uifileview.PatchPreviewLine{
			Type:    '+',
			LineNum: i + 1,
			Text:    line,
		})
	}
	return previewLines
}

func buildDeleteFilePreview(hunk Hunk, readFile func(path string) ([]byte, error)) uifileview.PatchFilePreview {
	preview := uifileview.PatchFilePreview{Path: hunk.Path, Action: hunk.Type}
	contents, ok := readPreviewFile(hunk.Path, readFile)
	if !ok {
		return preview
	}
	preview.Removed = countPreviewContentLines(string(contents))
	return preview
}

func readPreviewFile(path string, readFile func(path string) ([]byte, error)) ([]byte, bool) {
	if readFile == nil {
		return nil, false
	}

	absPath, err := common.ValidatePath(path)
	if err != nil {
		return nil, false
	}

	contents, err := readFile(absPath)
	if err != nil {
		return nil, false
	}
	return contents, true
}

func buildUpdateFilePreview(hunk Hunk, readFile func(path string) ([]byte, error)) (uifileview.PatchFilePreview, error) {
	originalLines, err := readUpdatePreviewSourceLines(hunk.Path, readFile)
	if err != nil {
		return uifileview.PatchFilePreview{}, err
	}
	state := newUpdatePreviewState(hunk, originalLines)
	return state.build()
}

func readUpdatePreviewSourceLines(path string, readFile func(path string) ([]byte, error)) ([]string, error) {
	if readFile == nil {
		return nil, fmt.Errorf("readFile is required for update preview")
	}
	absPath, err := common.ValidatePath(path)
	if err != nil {
		return nil, err
	}
	contents, err := readFile(absPath)
	if err != nil {
		return nil, err
	}
	return splitPreviewContentLines(string(contents)), nil
}

func newUpdatePreviewState(hunk Hunk, originalLines []string) *updatePreviewState {
	return &updatePreviewState{
		hunk:          hunk,
		originalLines: originalLines,
		preview: uifileview.PatchFilePreview{
			Path:     hunk.Path,
			Action:   hunk.Type,
			MovePath: hunk.MovePath,
			Hunks:    make([]uifileview.PatchHunkPreview, 0, len(hunk.Chunks)),
		},
	}
}

func (s *updatePreviewState) build() (uifileview.PatchFilePreview, error) {
	for _, chunk := range s.hunk.Chunks {
		if err := s.appendChunk(chunk); err != nil {
			return uifileview.PatchFilePreview{}, err
		}
	}
	return s.preview, nil
}

func (s *updatePreviewState) appendChunk(chunk UpdateFileChunk) error {
	result, err := LocateChunk(s.originalLines, s.hunk.Path, chunk, s.lineIndex, s.prevChunkEnd)
	if err != nil {
		return err
	}

	added, removed := countPreviewLines(chunk.previewLines)
	s.preview.Added += added
	s.preview.Removed += removed
	s.preview.Hunks = append(s.preview.Hunks, buildChunkPreview(result.StartIdx, s.lineDelta, chunk.previewLines))

	s.lineIndex = result.NextIndex
	if len(result.Pattern) > 0 {
		s.prevChunkEnd = result.StartIdx + len(result.Pattern)
	}
	s.lineDelta += len(result.NewLines) - len(result.Pattern)
	return nil
}

func buildChunkPreview(startIdx, lineDelta int, lines []patchLine) uifileview.PatchHunkPreview {
	previewLines := make([]uifileview.PatchPreviewLine, 0, len(lines))
	oldLine := startIdx + 1
	newLine := startIdx + 1 + lineDelta

	for _, line := range lines {
		previewLine, nextOld, nextNew := toPreviewLine(line, oldLine, newLine)
		previewLines = append(previewLines, previewLine)
		oldLine = nextOld
		newLine = nextNew
	}

	return uifileview.PatchHunkPreview{
		StartLine: resolveChunkStartLine(startIdx, lineDelta, previewLines),
		Lines:     previewLines,
	}
}

func toPreviewLine(line patchLine, oldLine int, newLine int) (uifileview.PatchPreviewLine, int, int) {
	switch line.Type {
	case '-':
		return uifileview.PatchPreviewLine{
			Type:    '-',
			LineNum: oldLine,
			Text:    line.Text,
		}, oldLine + 1, newLine
	case '+':
		return uifileview.PatchPreviewLine{
			Type:    '+',
			LineNum: newLine,
			Text:    line.Text,
		}, oldLine, newLine + 1
	default:
		return uifileview.PatchPreviewLine{
			Type:    ' ',
			LineNum: newLine,
			Text:    line.Text,
		}, oldLine + 1, newLine + 1
	}
}

func resolveChunkStartLine(startIdx int, lineDelta int, previewLines []uifileview.PatchPreviewLine) int {
	if len(previewLines) > 0 {
		return previewLines[0].LineNum
	}
	return startIdx + 1 + lineDelta
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
