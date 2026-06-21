package uiruntime

import (
	"bytes"
	"io"
	"strings"
	"testing"
	"time"

	"golang.org/x/term"
)

type stubRawModeController struct {
	terminal     bool
	makeRawErr   error
	makeRawCalls int
	restoreCalls int
}

func (s *stubRawModeController) isTerminal(int) bool {
	return s.terminal
}

func (s *stubRawModeController) makeRaw(int) (*term.State, error) {
	s.makeRawCalls++
	if s.makeRawErr != nil {
		return nil, s.makeRawErr
	}
	return &term.State{}, nil
}

func (s *stubRawModeController) restore(int, *term.State) error {
	s.restoreCalls++
	return nil
}

func readByteForTest(t *testing.T, ch <-chan byte) byte {
	t.Helper()
	select {
	case b := <-ch:
		return b
	case <-time.After(200 * time.Millisecond):
		t.Fatal("timed out waiting for byte")
		return 0
	}
}

func readErrForTest(t *testing.T, ch <-chan error) error {
	t.Helper()
	select {
	case err := <-ch:
		return err
	case <-time.After(200 * time.Millisecond):
		t.Fatal("timed out waiting for error")
		return nil
	}
}

func TestInitRawModeChannels_StartsReaderOnlyOnce(t *testing.T) {
	reader := NewMultilineReader(strings.NewReader("ab"))

	reader.initRawModeChannels()
	byteChan := reader.byteChan
	errChan := reader.errChan
	reader.initRawModeChannels()

	if reader.byteChan != byteChan || reader.errChan != errChan {
		t.Fatal("initRawModeChannels() should reuse initialized channels")
	}
	if got := readByteForTest(t, reader.byteChan); got != 'a' {
		t.Fatalf("first byte = %q, want %q", got, 'a')
	}
	if got := readByteForTest(t, reader.byteChan); got != 'b' {
		t.Fatalf("second byte = %q, want %q", got, 'b')
	}
	if err := readErrForTest(t, reader.errChan); err != io.EOF {
		t.Fatalf("reader error = %v, want io.EOF", err)
	}
}

func TestEnableDisableBracketedPaste_Terminal(t *testing.T) {
	var out bytes.Buffer
	reader := NewMultilineReaderWithOutput(strings.NewReader(""), &out)
	reader.fd = 0
	reader.rawMode = &stubRawModeController{terminal: true}

	reader.EnableBracketedPaste()
	if !reader.IsBracketedPasteEnabled() {
		t.Fatal("EnableBracketedPaste() should enable bracketed paste")
	}
	if !strings.Contains(out.String(), bracketedPasteEnable) {
		t.Fatalf("EnableBracketedPaste() output = %q, want enable sequence", out.String())
	}

	reader.DisableBracketedPaste()
	if reader.IsBracketedPasteEnabled() {
		t.Fatal("DisableBracketedPaste() should disable bracketed paste")
	}
	if !strings.Contains(out.String(), bracketedPasteDisable) {
		t.Fatalf("DisableBracketedPaste() output = %q, want disable sequence", out.String())
	}
}

func TestReadInput_UsesRawBracketedPastePath(t *testing.T) {
	var out bytes.Buffer
	ctrl := &stubRawModeController{terminal: true}
	reader, ch, _ := newTestReaderWithChannel()
	reader.fd = 0
	reader.out = &out
	reader.rawMode = ctrl
	reader.bracketedPasteEnabled = true
	go feedBytes(ch, []byte(pasteStart+"line1\r\nline2"+pasteEnd))

	result, err := reader.ReadInput("> ")
	if err != nil {
		t.Fatalf("ReadInput() error = %v", err)
	}
	if result != "line1\nline2" {
		t.Fatalf("ReadInput() = %q, want %q", result, "line1\nline2")
	}
	if ctrl.makeRawCalls != 1 || ctrl.restoreCalls != 1 {
		t.Fatalf("raw mode calls = make:%d restore:%d, want 1/1", ctrl.makeRawCalls, ctrl.restoreCalls)
	}
	if !strings.Contains(out.String(), "> ") {
		t.Fatalf("expected prompt output, got %q", out.String())
	}
}

func TestReadInput_FallsBackToLineModeWhenMakeRawFails(t *testing.T) {
	ctrl := &stubRawModeController{terminal: true, makeRawErr: io.ErrUnexpectedEOF}
	reader := NewMultilineReader(strings.NewReader("^[[200~hello^[[201~\n"))
	reader.fd = 0
	reader.rawMode = ctrl
	reader.bracketedPasteEnabled = true

	result, err := reader.ReadInput("> ")
	if err != nil {
		t.Fatalf("ReadInput() error = %v", err)
	}
	if result != "hello" {
		t.Fatalf("ReadInput() = %q, want %q", result, "hello")
	}
	if ctrl.makeRawCalls != 1 {
		t.Fatalf("makeRawCalls = %d, want 1", ctrl.makeRawCalls)
	}
}

func TestReadWithBracketedPaste_MarkerModeDelegation(t *testing.T) {
	ctrl := &stubRawModeController{terminal: true}
	reader, ch, _ := newTestReaderWithChannel()
	reader.fd = 0
	reader.rawMode = ctrl
	go func() {
		for _, b := range []byte("```\rline 1\r```\r") {
			ch <- b
		}
	}()

	result, err := reader.readWithBracketedPaste()
	if err != nil {
		t.Fatalf("readWithBracketedPaste() error = %v", err)
	}
	if result != "line 1" {
		t.Fatalf("readWithBracketedPaste() = %q, want %q", result, "line 1")
	}
	if ctrl.makeRawCalls != 2 || ctrl.restoreCalls != 3 {
		t.Fatalf("raw mode calls = make:%d restore:%d, want 2/3", ctrl.makeRawCalls, ctrl.restoreCalls)
	}
}

func TestReadWithBracketedPaste_Interrupted(t *testing.T) {
	ctrl := &stubRawModeController{terminal: true}
	reader := NewMultilineReader(strings.NewReader(string([]byte{0x03})))
	reader.fd = 0
	reader.rawMode = ctrl

	_, err := reader.readWithBracketedPaste()
	if err != ErrInterrupted {
		t.Fatalf("readWithBracketedPaste() error = %v, want ErrInterrupted", err)
	}
	if ctrl.restoreCalls != 2 {
		t.Fatalf("restoreCalls = %d, want 2", ctrl.restoreCalls)
	}
}
