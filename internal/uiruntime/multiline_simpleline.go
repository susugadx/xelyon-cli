package uiruntime

import (
	"errors"
	"os"
	"strings"
	"time"
)

// ReadSimpleLine は simple prompt 向けの1行入力を読み取る。
func (m *MultilineReader) ReadSimpleLine() (string, error) {
	if m.rawModeInit && m.byteChan != nil {
		if m.isTerminalInput() {
			oldState, err := m.rawModeOps().makeRaw(m.fd)
			if err == nil {
				defer func() { _ = m.rawModeOps().restore(m.fd, oldState) }()
				return m.readLineFromChannel()
			}
		}
		return m.readLineFromChannel()
	}

	line, err := m.reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	line = strings.TrimRight(line, "\n\r")
	return StripBracketedPaste(line), nil
}

// ReadSimpleLineWithTimeout は1行入力を読み取り、タイムアウト時に errReadLineTimeout を返す。
func (m *MultilineReader) ReadSimpleLineWithTimeout(timeout time.Duration) (string, error) {
	line, prefix, hasBufferedLine, err := m.consumeBufferedLineForTimedRead()
	if err != nil {
		return "", err
	}
	if hasBufferedLine {
		return line, nil
	}

	if !m.rawModeInit {
		return m.readSimpleLineFromReaderWithTimeout(timeout, prefix)
	}
	if m.byteChan == nil || m.errChan == nil {
		m.initRawModeChannels()
	}

	timer := time.NewTimer(timeout)
	defer timer.Stop()
	timeoutCh := timer.C

	if m.isTerminalInput() {
		oldState, err := m.rawModeOps().makeRaw(m.fd)
		if err == nil {
			defer func() { _ = m.rawModeOps().restore(m.fd, oldState) }()
		}
	}
	return m.readLineFromChannelWithTimeoutAndPrefix(timeoutCh, prefix)
}

func (m *MultilineReader) consumeBufferedLineForTimedRead() (line string, prefix []byte, hasBufferedLine bool, err error) {
	buffered := m.reader.Buffered()
	if buffered == 0 {
		return "", nil, false, nil
	}

	peeked, err := m.reader.Peek(buffered)
	if err != nil {
		return "", nil, false, err
	}

	if consumeLen, found := findBufferedLineConsumeLen(peeked); found {
		lineBytes := append([]byte(nil), peeked[:consumeLen]...)
		if _, err := m.reader.Discard(consumeLen); err != nil {
			return "", nil, false, err
		}
		trimmed := strings.TrimRight(string(lineBytes), "\n\r")
		return StripBracketedPaste(trimmed), nil, true, nil
	}

	prefix = append([]byte(nil), peeked...)
	if _, err := m.reader.Discard(buffered); err != nil {
		return "", nil, false, err
	}
	return "", prefix, false, nil
}

func findBufferedLineConsumeLen(buf []byte) (consumeLen int, found bool) {
	for i, b := range buf {
		switch b {
		case '\n':
			return i + 1, true
		case '\r':
			if i+1 < len(buf) && buf[i+1] == '\n' {
				return i + 2, true
			}
			return i + 1, true
		}
	}
	return 0, false
}

func (m *MultilineReader) readSimpleLineFromReaderWithTimeout(timeout time.Duration, prefix []byte) (string, error) {
	if f, ok := m.input.(*os.File); ok && timeout > 0 {
		if err := f.SetReadDeadline(time.Now().Add(timeout)); err == nil {
			defer func() { _ = f.SetReadDeadline(time.Time{}) }()
			line, err := m.readSimpleLineFromReader(prefix)
			if err != nil {
				if errors.Is(err, os.ErrDeadlineExceeded) {
					return "", errReadLineTimeout
				}
				return "", err
			}
			return line, nil
		}
	}
	return m.readSimpleLineFromReader(prefix)
}

func (m *MultilineReader) readSimpleLineFromReader(prefix []byte) (string, error) {
	line, err := m.reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	if len(prefix) > 0 {
		line = string(prefix) + line
	}
	line = strings.TrimRight(line, "\n\r")
	return StripBracketedPaste(line), nil
}
