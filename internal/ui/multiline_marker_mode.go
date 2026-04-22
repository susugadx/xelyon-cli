package ui

import (
	"io"
)

func (m *MultilineReader) markerModeUsesChannel() bool {
	return m.rawModeInit && m.byteChan != nil
}

func (m *MultilineReader) beginMarkerModeRawInput(useChannel bool) func() {
	if !useChannel || !m.isTerminalInput() {
		return func() {}
	}

	oldState, err := m.rawModeOps().makeRaw(m.fd)
	if err != nil {
		return func() {}
	}

	return func() {
		_ = m.rawModeOps().restore(m.fd, oldState)
	}
}

func (m *MultilineReader) readMarkerModeLines(useChannel bool) ([]string, error) {
	lines := make([]string, 0, 8)
	lineNum := 1

	for {
		m.printf("%3d | ", lineNum)

		line, err := m.readMarkerModeLine(useChannel)
		if err != nil {
			if err == io.EOF && len(lines) > 0 {
				return lines, nil
			}
			return nil, err
		}

		if line == "```" {
			return lines, nil
		}

		lines = append(lines, line)
		lineNum++
	}
}

func (m *MultilineReader) readMarkerModeLine(useChannel bool) (string, error) {
	if useChannel {
		return m.readLineFromChannel()
	}
	return m.readLineFromReaderBuffer()
}

func (m *MultilineReader) readLineFromReaderBuffer() (string, error) {
	line, err := m.reader.ReadString('\n')
	if err != nil {
		return "", err
	}

	return normalizeBufferedInputLine(line), nil
}

func normalizeBufferedInputLine(line string) string {
	line = trimLineBreak(line)
	line = stripAllBracketedPasteMarkers(line)
	return line
}

func trimLineBreak(line string) string {
	for len(line) > 0 {
		last := line[len(line)-1]
		if last != '\n' && last != '\r' {
			break
		}
		line = line[:len(line)-1]
	}
	return line
}
