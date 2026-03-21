package common

import (
	"bytes"
	"io"
	"strings"
	"testing"
)

func TestNewOutput_WritesToBuffer(t *testing.T) {
	var stdout, stderr bytes.Buffer
	out := NewOutput(&stdout, &stderr)

	out.Print("hello")
	out.Printf(" %s", "world")
	out.Println("!")

	if !strings.Contains(stdout.String(), "hello") {
		t.Errorf("stdout missing 'hello', got: %q", stdout.String())
	}
	if !strings.Contains(stdout.String(), "world") {
		t.Errorf("stdout missing 'world', got: %q", stdout.String())
	}
}

func TestNewOutput_ErrWritesToStderr(t *testing.T) {
	var stdout, stderr bytes.Buffer
	out := NewOutput(&stdout, &stderr)

	out.ErrPrint("err1")
	out.ErrPrintf(" %s", "err2")
	out.ErrPrintln("err3")

	if !strings.Contains(stderr.String(), "err1") {
		t.Errorf("stderr missing 'err1', got: %q", stderr.String())
	}
	if !strings.Contains(stderr.String(), "err2") {
		t.Errorf("stderr missing 'err2', got: %q", stderr.String())
	}
}

func TestNewOutput_DiscardSuppresses(t *testing.T) {
	out := NewOutput(io.Discard, io.Discard)

	// io.Discardの場合、SuppressStdoutがtrueになることを確認
	if !out.SuppressStdout() {
		t.Error("expected SuppressStdout to be true for io.Discard")
	}
	if !out.SuppressStderr() {
		t.Error("expected SuppressStderr to be true for io.Discard")
	}

	// 書き込みがpanicしないことを確認
	out.Print("no panic")
	out.ErrPrint("no panic")
}

func TestOutput_QuietModeSuppresses(t *testing.T) {
	var buf bytes.Buffer
	out := NewOutput(&buf, &buf)

	EnterQuietMode()
	defer ExitQuietMode()

	out.Print("suppressed")
	if buf.Len() != 0 {
		t.Errorf("expected no output in quiet mode, got: %q", buf.String())
	}
}

func TestOutput_StdoutWriter(t *testing.T) {
	var buf bytes.Buffer
	out := NewOutput(&buf, nil)
	if out.StdoutWriter() != &buf {
		t.Error("StdoutWriter should return injected writer")
	}
}

func TestOutput_StderrWriter(t *testing.T) {
	var buf bytes.Buffer
	out := NewOutput(nil, &buf)
	if out.StderrWriter() != &buf {
		t.Error("StderrWriter should return injected writer")
	}
}

func TestOutputColor_Sprint(t *testing.T) {
	var buf bytes.Buffer
	out := NewOutput(&buf, &buf)

	// Sprint は抑制されない
	s := out.Green.Sprint("green text")
	if s == "" {
		t.Error("Sprint should return non-empty string")
	}
}

func TestOutputColor_Sprintf(t *testing.T) {
	var buf bytes.Buffer
	out := NewOutput(&buf, &buf)

	s := out.Red.Sprintf("error: %d", 42)
	if !strings.Contains(s, "42") {
		t.Errorf("Sprintf result missing formatted value, got: %q", s)
	}
}

func TestOutputColor_QuietModeSuppressesPrint(t *testing.T) {
	var buf bytes.Buffer
	out := NewOutput(&buf, &buf)

	EnterQuietMode()
	defer ExitQuietMode()

	out.Yellow.Print("should not appear")
	out.Yellow.Printf("should not appear %s", "either")
	out.Yellow.Println("should not appear too")

	if buf.Len() != 0 {
		t.Errorf("expected no output in quiet mode, got: %q", buf.String())
	}
}

func TestOutput_ZeroValue(t *testing.T) {
	// ゼロ値のOutputでもpanicしないことを確認
	var out Output
	w := out.StdoutWriter()
	if w == nil {
		t.Error("StdoutWriter should return non-nil default writer")
	}
	w = out.StderrWriter()
	if w == nil {
		t.Error("StderrWriter should return non-nil default writer")
	}
}
