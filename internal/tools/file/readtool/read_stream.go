package readtool

import (
	"bufio"
	"errors"
	"io"
	"strings"
)

func readOutlineSample(r io.Reader, headLimit, tailLimit, maxLineBytes int) (headLines, tailLines []string, totalLines int, hasMore bool, truncated bool, err error) {
	if headLimit <= 0 {
		headLimit = outlineHeadLines
	}

	reader := bufio.NewReader(r)
	for {
		line, ok, lineTruncated, readErr := readNextLine(reader, maxLineBytes)
		if readErr != nil && !errors.Is(readErr, io.EOF) {
			return nil, nil, 0, false, false, readErr
		}
		if !ok {
			return headLines, tailLines, totalLines, totalLines > headLimit, truncated, nil
		}

		totalLines++
		truncated = truncated || lineTruncated
		if totalLines <= headLimit {
			headLines = append(headLines, line)
		}
		if tailLimit > 0 {
			if len(tailLines) < tailLimit {
				tailLines = append(tailLines, line)
			} else {
				copy(tailLines, tailLines[1:])
				tailLines[len(tailLines)-1] = line
			}
		}
		if errors.Is(readErr, io.EOF) {
			return headLines, tailLines, totalLines, totalLines > headLimit, truncated, nil
		}
	}
}

func readWindowLines(r io.Reader, startLine, endLine, maxLineBytes int) (lines []string, totalRead int, err error) {
	window := normalizeRequestedReadLineRange(startLine, endLine)
	reader := bufio.NewReader(r)
	for {
		line, ok, _, readErr := readNextLine(reader, maxLineBytes)
		if readErr != nil && !errors.Is(readErr, io.EOF) {
			return nil, 0, readErr
		}
		if !ok {
			return lines, totalRead, nil
		}
		totalRead++
		if totalRead >= window.startLine && totalRead <= window.endLine {
			lines = append(lines, line)
		}
		if totalRead > window.endLine {
			return lines, totalRead, nil
		}
		if errors.Is(readErr, io.EOF) {
			return lines, totalRead, nil
		}
	}
}

func readNextLine(reader *bufio.Reader, maxLineBytes int) (string, bool, bool, error) {
	var sb strings.Builder
	truncated := false

	for {
		fragment, err := reader.ReadSlice('\n')
		hasDelimiter := len(fragment) > 0 && fragment[len(fragment)-1] == '\n'
		fragment = trimLineEnding(fragment)
		appendLineFragment(&sb, fragment, maxLineBytes, &truncated)

		switch {
		case err == nil && hasDelimiter:
			return finalizeReadLine(sb.String(), truncated), true, truncated, nil
		case errors.Is(err, bufio.ErrBufferFull):
			continue
		case errors.Is(err, io.EOF):
			if sb.Len() == 0 && len(fragment) == 0 {
				return "", false, false, io.EOF
			}
			return finalizeReadLine(sb.String(), truncated), true, truncated, io.EOF
		case err != nil:
			return "", false, false, err
		}
	}
}

func trimLineEnding(fragment []byte) []byte {
	if len(fragment) > 0 && fragment[len(fragment)-1] == '\n' {
		fragment = fragment[:len(fragment)-1]
	}
	if len(fragment) > 0 && fragment[len(fragment)-1] == '\r' {
		fragment = fragment[:len(fragment)-1]
	}
	return fragment
}

func appendLineFragment(sb *strings.Builder, fragment []byte, maxLineBytes int, truncated *bool) {
	if maxLineBytes > 0 {
		remaining := maxLineBytes - sb.Len()
		if remaining <= 0 {
			*truncated = true
			return
		}
		if len(fragment) > remaining {
			sb.Write(fragment[:remaining])
			*truncated = true
			return
		}
	}
	sb.Write(fragment)
}

func finalizeReadLine(line string, truncated bool) string {
	if truncated {
		return line + "..."
	}
	return line
}
