package common

import (
	"fmt"
	"io"

	"github.com/fatih/color"
	"github.com/susugadx/xelyon-cli/internal/uiruntime"
)

// OutputColor は writer 指定可能な色付き出力ラッパー。
type OutputColor struct {
	writer     func() io.Writer
	suppressed func() bool
	attrs      []color.Attribute
}

func newOutputColor(writer func() io.Writer, suppressed func() bool, attrs ...color.Attribute) OutputColor {
	return OutputColor{
		writer:     writer,
		suppressed: suppressed,
		attrs:      attrs,
	}
}

// Print は色付きで出力する。
func (c OutputColor) Print(args ...interface{}) {
	if c.suppressed() {
		return
	}
	_, _ = color.New(c.attrs...).Fprint(c.writer(), args...)
}

// Printf は色付きでフォーマット出力する。
func (c OutputColor) Printf(format string, args ...interface{}) {
	if c.suppressed() {
		return
	}
	_, _ = color.New(c.attrs...).Fprintf(c.writer(), format, args...)
}

// Println は色付きで1行出力する。
func (c OutputColor) Println(args ...interface{}) {
	if c.suppressed() {
		return
	}
	_, _ = color.New(c.attrs...).Fprintln(c.writer(), args...)
}

// Sprint は文字列を返すため抑制しない。
func (c OutputColor) Sprint(args ...interface{}) string {
	return color.New(c.attrs...).Sprint(args...)
}

// Sprintf は文字列を返すため抑制しない。
func (c OutputColor) Sprintf(format string, args ...interface{}) string {
	return color.New(c.attrs...).Sprintf(format, args...)
}

// Output はツール補助出力先を表す。
type Output struct {
	stdout func() io.Writer
	stderr func() io.Writer

	Yellow OutputColor
	Green  OutputColor
	Red    OutputColor
	Cyan   OutputColor
	Dim    OutputColor
}

// NewOutput は writer を束ねた Output を返す。
func NewOutput(stdout, stderr io.Writer) Output {
	stdoutWriter := stdout
	if stdoutWriter == nil {
		stdoutWriter = uiruntime.DefaultRuntime().Output()
	}
	stderrWriter := stderr
	if stderrWriter == nil {
		stderrWriter = uiruntime.DefaultRuntime().ErrorOutput()
	}

	out := Output{
		stdout: func() io.Writer { return stdoutWriter },
		stderr: func() io.Writer { return stderrWriter },
	}

	stdoutSuppressed := func() bool {
		return IsQuietMode() || stdoutWriter == io.Discard
	}

	out.Yellow = newOutputColor(out.stdout, stdoutSuppressed, color.FgYellow)
	out.Green = newOutputColor(out.stdout, stdoutSuppressed, color.FgGreen)
	out.Red = newOutputColor(out.stdout, stdoutSuppressed, color.FgRed)
	out.Cyan = newOutputColor(out.stdout, stdoutSuppressed, color.FgCyan)
	out.Dim = newOutputColor(out.stdout, stdoutSuppressed, color.Faint)

	return out
}

// DefaultOutput は current UI runtime に紐づく Output を返す。
func DefaultOutput() Output {
	runtime := uiruntime.DefaultRuntime()
	return NewOutput(runtime.Output(), runtime.ErrorOutput())
}

// StdoutWriter は stdout writer を返す。
func (o Output) StdoutWriter() io.Writer {
	if o.stdout == nil {
		return uiruntime.DefaultRuntime().Output()
	}
	return o.stdout()
}

// StderrWriter は stderr writer を返す。
func (o Output) StderrWriter() io.Writer {
	if o.stderr == nil {
		return uiruntime.DefaultRuntime().ErrorOutput()
	}
	return o.stderr()
}

// SuppressStdout は stdout 補助表示を抑制すべきか返す。
func (o Output) SuppressStdout() bool {
	return IsQuietMode() || o.StdoutWriter() == io.Discard
}

// SuppressStderr は stderr 補助表示を抑制すべきか返す。
func (o Output) SuppressStderr() bool {
	return IsQuietMode() || o.StderrWriter() == io.Discard
}

// Print は stdout に出力する。
func (o Output) Print(args ...interface{}) {
	if o.SuppressStdout() {
		return
	}
	_, _ = fmt.Fprint(o.StdoutWriter(), args...)
}

// Printf は stdout にフォーマット出力する。
func (o Output) Printf(format string, args ...interface{}) {
	if o.SuppressStdout() {
		return
	}
	_, _ = fmt.Fprintf(o.StdoutWriter(), format, args...)
}

// Println は stdout に 1 行出力する。
func (o Output) Println(args ...interface{}) {
	if o.SuppressStdout() {
		return
	}
	_, _ = fmt.Fprintln(o.StdoutWriter(), args...)
}

// ErrPrint は stderr に出力する。
func (o Output) ErrPrint(args ...interface{}) {
	if o.SuppressStderr() {
		return
	}
	_, _ = fmt.Fprint(o.StderrWriter(), args...)
}

// ErrPrintf は stderr にフォーマット出力する。
func (o Output) ErrPrintf(format string, args ...interface{}) {
	if o.SuppressStderr() {
		return
	}
	_, _ = fmt.Fprintf(o.StderrWriter(), format, args...)
}

// ErrPrintln は stderr に 1 行出力する。
func (o Output) ErrPrintln(args ...interface{}) {
	if o.SuppressStderr() {
		return
	}
	_, _ = fmt.Fprintln(o.StderrWriter(), args...)
}

func defaultStdoutWriter() io.Writer {
	return uiruntime.DefaultRuntime().Output()
}

func defaultStdoutSuppressed() bool {
	return IsQuietMode()
}

// 互換用の global wrappers。
var (
	Yellow = newOutputColor(defaultStdoutWriter, defaultStdoutSuppressed, color.FgYellow)
	Green  = newOutputColor(defaultStdoutWriter, defaultStdoutSuppressed, color.FgGreen)
	Red    = newOutputColor(defaultStdoutWriter, defaultStdoutSuppressed, color.FgRed)
	Cyan   = newOutputColor(defaultStdoutWriter, defaultStdoutSuppressed, color.FgCyan)
	Dim    = newOutputColor(defaultStdoutWriter, defaultStdoutSuppressed, color.Faint)
)

// Print は互換用の stdout 出力。
func Print(args ...interface{}) {
	DefaultOutput().Print(args...)
}

// Printf は互換用の stdout フォーマット出力。
func Printf(format string, args ...interface{}) {
	DefaultOutput().Printf(format, args...)
}

// Println は互換用の stdout 1 行出力。
func Println(args ...interface{}) {
	DefaultOutput().Println(args...)
}
