package uiruntime

import (
	"io"
	"time"
)

// initRawModeChannels initializes the raw mode channels and goroutine (once)
func (m *MultilineReader) initRawModeChannels() {
	if m.rawModeInit {
		return
	}
	m.byteChan = make(chan byte, 4096)
	m.errChan = make(chan error, 1)
	m.rawModeInit = true

	go func() {
		b := make([]byte, 1)
		input := m.input
		if input == nil {
			input = DefaultRuntime().Input()
		}
		for {
			_, err := input.Read(b)
			if err != nil {
				m.errChan <- err
				return
			}
			m.byteChan <- b[0]
		}
	}()
}

// readWithBracketedPaste reads input using raw mode with paste marker detection
func (m *MultilineReader) readWithBracketedPaste() (string, error) {
	pasteDebugf(m.errorWriter(), "[DEBUG] Entering raw mode...\n")

	oldState, err := m.rawModeOps().makeRaw(m.fd)
	if err != nil {
		pasteDebugf(m.errorWriter(), "[DEBUG] MakeRaw FAILED: %v\n", err)
		return m.readLine()
	}
	defer func() { _ = m.rawModeOps().restore(m.fd, oldState) }()

	pasteDebugWriteString(m.errorWriter(), "[DEBUG] Raw mode OK\r\n")

	var line lineAssembler
	var paste pasteAssembler

	m.initRawModeChannels()

	readByteTimeout := func(timeout time.Duration) (byte, bool) {
		select {
		case b := <-m.byteChan:
			return b, true
		case <-time.After(timeout):
			return 0, false
		}
	}

	for {
		select {
		case b := <-m.byteChan:
			if b == 0x03 {
				return "", m.interruptRawInput(oldState)
			}

			if m.isEscapeLeadByte(b) {
				escBuf, marker, err := m.readEscapeSequence(b, readByteTimeout, func(next byte) error {
					if next == 0x03 {
						return m.interruptRawInput(oldState)
					}
					return nil
				})
				if err != nil {
					return "", err
				}
				switch marker {
				case pasteMarkerStart:
					if !paste.active {
						paste.start()
						pasteDebugWriteString(m.errorWriter(), "[DEBUG] Paste START\r\n")
					}
				case pasteMarkerEnd:
					content := paste.finish()
					pasteDebugf(m.errorWriter(), "[DEBUG] Paste END, %d bytes\r\n", len(content))
					line.appendString(content)
					m.echoPastedContent(content)
				default:
					if len(escBuf) > 0 {
						if paste.active {
							paste.appendBytes(escBuf)
						} else {
							line.appendBytes(escBuf)
						}
					}
				}
				continue
			}

			if paste.active {
				paste.appendByte(b)
				continue
			}

			if b == 0x04 {
				if line.len() == 0 {
					return "", io.EOF
				}
				m.print("\r\n")
				return line.string(), nil
			}

			if b == '\r' || b == '\n' {
				m.print("\r\n")
				content := line.string()
				content = stripAllBracketedPasteMarkers(content)
				if content == "```" {
					_ = m.rawModeOps().restore(m.fd, oldState)
					return m.readMultilineWithMarker()
				}
				return content, nil
			}

			if b == 0x7f || b == 0x08 {
				m.handleBackspace(&line)
				continue
			}

			line.appendByte(b)
			m.echoByte(line.bytes(), b)

		case err := <-m.errChan:
			if len(m.byteChan) > 0 {
				continue
			}
			if err == io.EOF {
				return line.string(), nil
			}
			return "", err
		}
	}
}

// readByteTimeoutFromChannel reads a byte from the channel with timeout
func (m *MultilineReader) readByteTimeoutFromChannel(timeout time.Duration) (byte, bool) {
	select {
	case b := <-m.byteChan:
		return b, true
	case <-time.After(timeout):
		return 0, false
	}
}

// readLineFromChannel reads a line from the byte channel (when goroutine is active)
func (m *MultilineReader) readLineFromChannel() (string, error) {
	return m.readLineFromChannelWithTimeout(nil)
}

func (m *MultilineReader) readLineFromChannelWithTimeout(timeout <-chan time.Time) (string, error) {
	return m.readLineFromChannelWithTimeoutAndPrefix(timeout, nil)
}

func (m *MultilineReader) readLineFromChannelWithTimeoutAndPrefix(timeout <-chan time.Time, prefix []byte) (string, error) {
	var line lineAssembler
	if len(prefix) > 0 {
		line.appendBytes(prefix)
	}
	for {
		select {
		case <-timeout:
			return "", errReadLineTimeout
		case b := <-m.byteChan:
			if b == '\n' || b == '\r' {
				m.print("\r\n")
				return StripBracketedPaste(line.string()), nil
			}

			if m.isEscapeLeadByte(b) {
				escBuf, marker, err := m.readEscapeSequence(b, m.readByteTimeoutFromChannel, nil)
				if err != nil {
					return "", err
				}
				if marker == pasteMarkerNone && len(escBuf) > 0 {
					line.appendBytes(escBuf)
					m.echoPrintableASCII(escBuf)
				}
				continue
			}

			if b == 0x7f || b == 0x08 {
				m.handleBackspace(&line)
				continue
			}

			line.appendByte(b)
			m.echoByte(line.bytes(), b)
		case err := <-m.errChan:
			if len(m.byteChan) > 0 {
				continue
			}
			if line.len() > 0 {
				return StripBracketedPaste(line.string()), nil
			}
			return "", err
		}
	}
}
