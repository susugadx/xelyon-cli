package ui

import (
	"bufio"
	"errors"
	"io"
	"os"
	"strings"

	"golang.org/x/term"
)

// ErrInterrupted は Ctrl+C で入力が中断されたことを表す。
var ErrInterrupted = errors.New("interrupted")
var errReadLineTimeout = errors.New("read line timeout")

// Bracketed Paste Mode escape sequences
const (
	bracketedPasteEnable  = "\x1b[?2004h" // Enable bracketed paste mode
	bracketedPasteDisable = "\x1b[?2004l" // Disable bracketed paste mode
	pasteStart            = "\x1b[200~"   // Paste start marker
	pasteEnd              = "\x1b[201~"   // Paste end marker
)

// MultilineReader は複数行入力と bracketed paste を扱う入力リーダー。
type MultilineReader struct {
	reader                *bufio.Reader
	input                 io.Reader
	out                   io.Writer
	err                   io.Writer
	rawMode               rawModeController
	bracketedPasteEnabled bool
	fd                    int // file descriptor for stdin (for raw mode)
	// Raw mode channels (initialized lazily, reused across calls)
	byteChan    chan byte
	errChan     chan error
	rawModeInit bool
}

type rawModeController interface {
	isTerminal(fd int) bool
	makeRaw(fd int) (*term.State, error)
	restore(fd int, state *term.State) error
}

type systemRawModeController struct{}

func (systemRawModeController) isTerminal(fd int) bool {
	return term.IsTerminal(fd)
}

func (systemRawModeController) makeRaw(fd int) (*term.State, error) {
	return term.MakeRaw(fd)
}

func (systemRawModeController) restore(fd int, state *term.State) error {
	return term.Restore(fd, state)
}

// NewMultilineReader は標準 runtime 出力先で multiline reader を生成する。
func NewMultilineReader(r io.Reader) *MultilineReader {
	return NewMultilineReaderWithOutput(r, DefaultRuntime().Output())
}

// NewMultilineReaderWithOutput は出力先を指定して multiline reader を生成する。
func NewMultilineReaderWithOutput(r io.Reader, out io.Writer) *MultilineReader {
	return newMultilineReader(r, out, DefaultRuntime().ErrorOutput())
}

// NewMultilineReaderWithRuntime は runtime の入出力に紐づく multiline reader を返す。
func NewMultilineReaderWithRuntime(runtime *Runtime) *MultilineReader {
	rt := runtimeOrDefault(runtime)
	return newMultilineReader(rt.Input(), rt.Output(), rt.ErrorOutput())
}

func newMultilineReader(r io.Reader, out, errOut io.Writer) *MultilineReader {
	if r == nil {
		r = DefaultRuntime().Input()
	}
	fd := -1
	if f, ok := r.(*os.File); ok {
		fd = int(f.Fd())
	}
	if out == nil {
		out = DefaultRuntime().Output()
	}
	if errOut == nil {
		errOut = DefaultRuntime().ErrorOutput()
	}
	return &MultilineReader{
		reader:                bufio.NewReaderSize(r, 1024*1024), // 1MB buffer
		input:                 r,
		out:                   out,
		err:                   errOut,
		rawMode:               systemRawModeController{},
		bracketedPasteEnabled: false,
		fd:                    fd,
		byteChan:              nil,
		errChan:               nil,
		rawModeInit:           false,
	}
}

func (m *MultilineReader) rawModeOps() rawModeController {
	if m != nil && m.rawMode != nil {
		return m.rawMode
	}
	return systemRawModeController{}
}

func (m *MultilineReader) isTerminalInput() bool {
	return m != nil && m.fd >= 0 && m.rawModeOps().isTerminal(m.fd)
}

// EnableBracketedPaste は端末に bracketed paste 有効化シーケンスを送る。
func (m *MultilineReader) EnableBracketedPaste() {
	pasteDebugf(m.errorWriter(), "[DEBUG] EnableBracketedPaste: fd=%d, IsTerminal=%v\n", m.fd, m.isTerminalInput())

	if m.isTerminalInput() {
		m.writeString(bracketedPasteEnable)
		m.bracketedPasteEnabled = true
		pasteDebugf(m.errorWriter(), "[DEBUG] Sent: \\x1b[?2004h (bracketed paste enable)\n")
	} else {
		pasteDebugf(m.errorWriter(), "[DEBUG] Skipped: not a terminal\n")
	}
}

// DisableBracketedPaste は端末の bracketed paste を無効化する。
func (m *MultilineReader) DisableBracketedPaste() {
	if m.bracketedPasteEnabled {
		m.writeString(bracketedPasteDisable)
		m.bracketedPasteEnabled = false
	}
}

// StripBracketedPaste は入力中の bracketed paste マーカーを除去する。
func StripBracketedPaste(input string) string {
	input = strings.ReplaceAll(input, "\x1b[200~", "")
	input = strings.ReplaceAll(input, "\x1b[201~", "")
	input = strings.ReplaceAll(input, "^[[200~", "")
	input = strings.ReplaceAll(input, "^[[201~", "")
	return input
}

// stripAllBracketedPasteMarkers is an alias for backward compatibility within this package.
func stripAllBracketedPasteMarkers(input string) string {
	return StripBracketedPaste(input)
}

// ReadInput は1行/複数行の入力を読み取り、必要に応じて raw mode に切り替える。
func (m *MultilineReader) ReadInput(prompt string) (string, error) {
	m.print(prompt)

	if m.bracketedPasteEnabled && m.isTerminalInput() {
		return m.readWithBracketedPaste()
	}

	return m.readLine()
}

func (m *MultilineReader) readLine() (string, error) {
	line, err := m.readLineFromReaderBuffer()
	if err != nil {
		return "", err
	}
	if line == "```" {
		return m.readMultilineWithMarker()
	}
	return line, nil
}

func (m *MultilineReader) readMultilineWithMarker() (string, error) {
	m.println("📝 Multiline input mode (end with ``` on a new line)")

	useChannel := m.markerModeUsesChannel()
	restore := m.beginMarkerModeRawInput(useChannel)
	defer restore()

	lines, err := m.readMarkerModeLines(useChannel)
	if err != nil {
		return "", err
	}
	result := strings.Join(lines, "\n")
	m.printf("✅ Captured %d lines\n", len(lines))
	return result, nil
}

// IsMultilineMarker は複数行入力開始マーカーかどうかを判定する。
func IsMultilineMarker(input string) bool {
	return input == "```"
}

// TrimBracketedPasteMarkers は入力中の bracketed paste マーカーを除去する。
func TrimBracketedPasteMarkers(input string) string {
	return stripAllBracketedPasteMarkers(input)
}

// GetBufioReader は内部の bufio.Reader を返す。
func (m *MultilineReader) GetBufioReader() *bufio.Reader {
	return m.reader
}

// FlushInput はバッファ済み入力を破棄する。
func (m *MultilineReader) FlushInput() {
	if _, err := m.reader.Discard(m.reader.Buffered()); err != nil {
		return
	}
}

// Reader は共有用の bufio.Reader を返す。
func (m *MultilineReader) Reader() *bufio.Reader {
	return m.reader
}

// IsBracketedPasteEnabled は bracketed paste が有効かどうかを返す。
func (m *MultilineReader) IsBracketedPasteEnabled() bool {
	return m.bracketedPasteEnabled
}
