package applypatch

import (
	"fmt"
	"strings"
)

const (
	beginPatchMarker        = "*** Begin Patch"
	endPatchMarker          = "*** End Patch"
	addFileMarker           = "*** Add File: "
	deleteFileMarker        = "*** Delete File: "
	updateFileMarker        = "*** Update File: "
	moveToMarker            = "*** Move to: "
	eofMarker               = "*** End of File"
	changeContextMarker     = "@@ "
	emptyChangeContextMaker = "@@"
	parseInStrictMode       = false
)

// ParsedPatch はパース済みのパッチ全体を表す。
type ParsedPatch struct {
	Patch string
	Hunks []Hunk
}

// ParseError は apply_patch の構文エラーを表す。
type ParseError struct {
	Message    string
	LineNumber int
}

func (e *ParseError) Error() string {
	if e == nil {
		return ""
	}
	if e.LineNumber > 0 {
		return fmt.Sprintf("invalid hunk at line %d, %s", e.LineNumber, e.Message)
	}
	return fmt.Sprintf("invalid patch: %s", e.Message)
}

func newInvalidPatchError(message string) *ParseError {
	return &ParseError{Message: message}
}

func newInvalidHunkError(message string, lineNumber int) *ParseError {
	return &ParseError{Message: message, LineNumber: lineNumber}
}

// Hunk はパッチ内の1つのファイル操作を表す。
type Hunk struct {
	Type     string
	Path     string
	MovePath string
	Contents string
	Chunks   []UpdateFileChunk
}

// UpdateFileChunk は Update 内の1つの変更ブロックを表す。
type UpdateFileChunk struct {
	ChangeContext string
	OldLines      []string
	NewLines      []string
	IsEndOfFile   bool
}

type parseMode int

const (
	parseModeStrict parseMode = iota
	parseModeLenient
)

// ParsePatch は apply_patch 形式のテキストをパースする。
func ParsePatch(patch string) (*ParsedPatch, error) {
	mode := parseModeLenient
	if parseInStrictMode {
		mode = parseModeStrict
	}
	return parsePatchText(patch, mode)
}

func parsePatchText(patch string, mode parseMode) (*ParsedPatch, error) {
	lines := splitPatchLines(patch)
	strictErr := checkPatchBoundariesStrict(lines)
	switch {
	case strictErr == nil:
	case mode == parseModeStrict:
		return nil, strictErr
	default:
		var err error
		lines, err = checkPatchBoundariesLenient(lines, strictErr)
		if err != nil {
			return nil, err
		}
	}

	hunks := make([]Hunk, 0)
	lastLineIndex := len(lines) - 1
	remainingLines := lines[1:lastLineIndex]
	lineNumber := 2

	for len(remainingLines) > 0 {
		hunk, hunkLines, err := parseOneHunk(remainingLines, lineNumber)
		if err != nil {
			return nil, err
		}
		hunks = append(hunks, hunk)
		lineNumber += hunkLines
		remainingLines = remainingLines[hunkLines:]
	}

	return &ParsedPatch{
		Patch: strings.Join(lines, "\n"),
		Hunks: hunks,
	}, nil
}

func splitPatchLines(patch string) []string {
	normalized := strings.ReplaceAll(patch, "\r\n", "\n")
	normalized = strings.ReplaceAll(normalized, "\r", "\n")
	trimmed := strings.TrimSpace(normalized)
	if trimmed == "" {
		return nil
	}
	return strings.Split(trimmed, "\n")
}

func checkPatchBoundariesStrict(lines []string) error {
	var firstLine *string
	var lastLine *string

	switch len(lines) {
	case 0:
	case 1:
		firstLine = &lines[0]
		lastLine = &lines[0]
	default:
		firstLine = &lines[0]
		lastLine = &lines[len(lines)-1]
	}

	return checkStartAndEndLinesStrict(firstLine, lastLine)
}

func checkPatchBoundariesLenient(originalLines []string, originalErr error) ([]string, error) {
	if len(originalLines) < 4 {
		return nil, originalErr
	}

	first := originalLines[0]
	last := originalLines[len(originalLines)-1]
	if (first == "<<EOF" || first == "<<'EOF'" || first == "<<\"EOF\"") && strings.HasSuffix(last, "EOF") {
		innerLines := originalLines[1 : len(originalLines)-1]
		if err := checkPatchBoundariesStrict(innerLines); err != nil {
			return nil, err
		}
		return innerLines, nil
	}

	return nil, originalErr
}

func checkStartAndEndLinesStrict(firstLine *string, lastLine *string) error {
	var first string
	var last string
	if firstLine != nil {
		first = strings.TrimSpace(*firstLine)
	}
	if lastLine != nil {
		last = strings.TrimSpace(*lastLine)
	}

	switch {
	case first == beginPatchMarker && last == endPatchMarker:
		return nil
	case first != beginPatchMarker:
		return newInvalidPatchError("The first line of the patch must be '*** Begin Patch'")
	default:
		return newInvalidPatchError("The last line of the patch must be '*** End Patch'")
	}
}

func parseOneHunk(lines []string, lineNumber int) (Hunk, int, error) {
	firstLine := strings.TrimSpace(lines[0])

	if path, ok := strings.CutPrefix(firstLine, addFileMarker); ok {
		var contents strings.Builder
		parsedLines := 1
		for _, addLine := range lines[1:] {
			lineToAdd, ok := strings.CutPrefix(addLine, "+")
			if !ok {
				break
			}
			contents.WriteString(lineToAdd)
			contents.WriteByte('\n')
			parsedLines++
		}
		return Hunk{
			Type:     "add",
			Path:     path,
			Contents: contents.String(),
		}, parsedLines, nil
	}

	if path, ok := strings.CutPrefix(firstLine, deleteFileMarker); ok {
		return Hunk{
			Type: "delete",
			Path: path,
		}, 1, nil
	}

	if path, ok := strings.CutPrefix(firstLine, updateFileMarker); ok {
		remainingLines := lines[1:]
		parsedLines := 1

		movePath := ""
		if len(remainingLines) > 0 {
			if nextPath, ok := strings.CutPrefix(remainingLines[0], moveToMarker); ok {
				movePath = nextPath
				remainingLines = remainingLines[1:]
				parsedLines++
			}
		}

		chunks := make([]UpdateFileChunk, 0)
		for len(remainingLines) > 0 {
			if strings.TrimSpace(remainingLines[0]) == "" {
				parsedLines++
				remainingLines = remainingLines[1:]
				continue
			}
			if strings.HasPrefix(remainingLines[0], "***") {
				break
			}

			chunk, chunkLines, err := parseUpdateFileChunk(remainingLines, lineNumber+parsedLines, len(chunks) == 0)
			if err != nil {
				return Hunk{}, 0, err
			}
			chunks = append(chunks, chunk)
			parsedLines += chunkLines
			remainingLines = remainingLines[chunkLines:]
		}

		if len(chunks) == 0 {
			return Hunk{}, 0, newInvalidHunkError(
				fmt.Sprintf("Update file hunk for path '%s' is empty", path),
				lineNumber,
			)
		}

		return Hunk{
			Type:     "update",
			Path:     path,
			MovePath: movePath,
			Chunks:   chunks,
		}, parsedLines, nil
	}

	return Hunk{}, 0, newInvalidHunkError(
		fmt.Sprintf(
			"'%s' is not a valid hunk header. Valid hunk headers: '*** Add File: {path}', '*** Delete File: {path}', '*** Update File: {path}'",
			firstLine,
		),
		lineNumber,
	)
}

func parseUpdateFileChunk(lines []string, lineNumber int, allowMissingContext bool) (UpdateFileChunk, int, error) {
	if len(lines) == 0 {
		return UpdateFileChunk{}, 0, newInvalidHunkError("Update hunk does not contain any lines", lineNumber)
	}

	changeContext := ""
	startIndex := 0
	switch {
	case lines[0] == emptyChangeContextMaker:
		startIndex = 1
	case strings.HasPrefix(lines[0], changeContextMarker):
		changeContext = strings.TrimPrefix(lines[0], changeContextMarker)
		startIndex = 1
	default:
		if !allowMissingContext {
			return UpdateFileChunk{}, 0, newInvalidHunkError(
				fmt.Sprintf("Expected update hunk to start with a @@ context marker, got: '%s'", lines[0]),
				lineNumber,
			)
		}
	}

	if startIndex >= len(lines) {
		return UpdateFileChunk{}, 0, newInvalidHunkError("Update hunk does not contain any lines", lineNumber+1)
	}

	chunk := UpdateFileChunk{
		ChangeContext: changeContext,
		OldLines:      make([]string, 0),
		NewLines:      make([]string, 0),
	}
	parsedLines := 0

	for _, line := range lines[startIndex:] {
		switch line {
		case eofMarker:
			if parsedLines == 0 {
				return UpdateFileChunk{}, 0, newInvalidHunkError("Update hunk does not contain any lines", lineNumber+1)
			}
			chunk.IsEndOfFile = true
			parsedLines++
			return chunk, parsedLines + startIndex, nil
		case "":
			chunk.OldLines = append(chunk.OldLines, "")
			chunk.NewLines = append(chunk.NewLines, "")
			parsedLines++
		default:
			switch line[0] {
			case ' ':
				chunk.OldLines = append(chunk.OldLines, line[1:])
				chunk.NewLines = append(chunk.NewLines, line[1:])
			case '+':
				chunk.NewLines = append(chunk.NewLines, line[1:])
			case '-':
				chunk.OldLines = append(chunk.OldLines, line[1:])
			default:
				if parsedLines == 0 {
					return UpdateFileChunk{}, 0, newInvalidHunkError(
						fmt.Sprintf(
							"Unexpected line found in update hunk: '%s'. Every line should start with ' ' (context line), '+' (added line), or '-' (removed line)",
							line,
						),
						lineNumber+1,
					)
				}
				return chunk, parsedLines + startIndex, nil
			}
			parsedLines++
		}
	}

	return chunk, parsedLines + startIndex, nil
}
