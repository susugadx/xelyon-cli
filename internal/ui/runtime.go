package ui

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"sync"

	"github.com/susugadx/xelyon-cli/internal/stdio"
	"golang.org/x/term"
)

// Runtime は UI/terminal の入出力状態を session 単位で保持する。
type Runtime struct {
	mu           sync.RWMutex
	in           io.Reader
	out          io.Writer
	err          io.Writer
	promptReader *MultilineReader
	simpleReader *bufio.Reader
	spinner      *Spinner
	logLevel     LogLevel
	logLevelSet  bool
}

// NewRuntime は UI/terminal 用の runtime を作成する。
func NewRuntime(in io.Reader, out, err io.Writer) *Runtime {
	if in == nil {
		in = stdio.Input()
	}
	if out == nil {
		out = stdio.Output()
	}
	if err == nil {
		err = stdio.ErrorOutput()
	}
	return &Runtime{
		in:  in,
		out: out,
		err: err,
	}
}

var defaultRuntime = NewRuntime(stdio.Input(), stdio.Output(), stdio.ErrorOutput())

// DefaultRuntime は互換用途の process-global UI runtime を返す。
func DefaultRuntime() *Runtime {
	defaultRuntime.mu.Lock()
	currentIn := stdio.Input()
	currentOut := stdio.Output()
	currentErr := stdio.ErrorOutput()
	if defaultRuntime.in != currentIn {
		defaultRuntime.simpleReader = nil
	}
	defaultRuntime.in = currentIn
	defaultRuntime.out = currentOut
	defaultRuntime.err = currentErr
	defaultRuntime.mu.Unlock()
	return defaultRuntime
}

func runtimeOrDefault(runtime *Runtime) *Runtime {
	if runtime == nil || runtime == defaultRuntime {
		return DefaultRuntime()
	}
	return runtime
}

// Input は runtime が使用する標準入力を返す。
func (r *Runtime) Input() io.Reader {
	if r == nil {
		return DefaultRuntime().Input()
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.in
}

// Output は runtime が使用する標準出力を返す。
func (r *Runtime) Output() io.Writer {
	if r == nil {
		return DefaultRuntime().Output()
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.out
}

// ErrorOutput は runtime が使用する標準エラー出力を返す。
func (r *Runtime) ErrorOutput() io.Writer {
	if r == nil {
		return DefaultRuntime().ErrorOutput()
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.err
}

// SetLogLevel は runtime のログレベルを設定する。
func (r *Runtime) SetLogLevel(level LogLevel) {
	if r == nil {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.logLevel = level
	r.logLevelSet = true
}

// LogLevel は runtime のログレベルを返す。未設定の場合は default runtime のレベルを使う。
func (r *Runtime) LogLevel() LogLevel {
	if r == nil {
		return DefaultRuntime().LogLevel()
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	if !r.logLevelSet {
		if r != defaultRuntime {
			return DefaultRuntime().LogLevel()
		}
		return defaultLogLevel()
	}
	return r.logLevel
}

// SetPromptReader は runtime に紐づく共有 MultilineReader を設定する。
func (r *Runtime) SetPromptReader(reader *MultilineReader) {
	if r == nil {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.promptReader = reader
	if reader != nil {
		r.simpleReader = nil
	}
}

// PromptReader は runtime に紐づく共有 MultilineReader を返す。
func (r *Runtime) PromptReader() *MultilineReader {
	if r == nil {
		return DefaultRuntime().PromptReader()
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.promptReader
}

// PromptIO は runtime の入出力から PromptIO を構築する。
func (r *Runtime) PromptIO() PromptIO {
	if r == nil {
		return DefaultRuntime().PromptIO()
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.promptReader == nil && r.simpleReader == nil {
		r.simpleReader = bufio.NewReader(r.in)
	}
	return PromptIO{
		In:           r.in,
		Out:          r.out,
		Err:          r.err,
		Reader:       r.promptReader,
		simpleReader: r.simpleReader,
		runtime:      r,
	}
}

// NewSpinner は runtime の出力先へ紐づく spinner を作成する。
func (r *Runtime) NewSpinner() *Spinner {
	if r == nil {
		return DefaultRuntime().NewSpinner()
	}
	return NewSpinnerWithWriter(r.Output())
}

// SetSpinner は runtime に紐づく現在の spinner を設定する。
func (r *Runtime) SetSpinner(spinner *Spinner) {
	if r == nil {
		DefaultRuntime().SetSpinner(spinner)
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.spinner != nil && r.spinner != spinner {
		r.spinner.Stop()
	}
	r.spinner = spinner
}

// CurrentSpinner は runtime に紐づく現在の spinner を返す。
func (r *Runtime) CurrentSpinner() *Spinner {
	if r == nil {
		return DefaultRuntime().CurrentSpinner()
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.spinner
}

// StopSpinner は runtime に紐づく spinner を停止する。
func (r *Runtime) StopSpinner() {
	if r == nil {
		DefaultRuntime().StopSpinner()
		return
	}
	r.mu.Lock()
	spinner := r.spinner
	r.spinner = nil
	r.mu.Unlock()
	if spinner != nil {
		spinner.Stop()
	}
}

// ResetTerminalState は runtime に紐づく terminal 表示をリセットする。
func (r *Runtime) ResetTerminalState() {
	if r == nil {
		DefaultRuntime().ResetTerminalState()
		return
	}
	out := r.Output()
	_, _ = fmt.Fprint(out, "\033[?25h")
	_, _ = fmt.Fprint(out, "\r\033[K")
}

// IsTerminal は runtime の出力先が端末かどうかを返す。
func (r *Runtime) IsTerminal() bool {
	if r == nil {
		return DefaultRuntime().IsTerminal()
	}
	return isFileTerminal(r.Output())
}

// isFileTerminal は writer が *os.File かつ端末であれば true を返す。
func isFileTerminal(w io.Writer) bool {
	if f, ok := w.(*os.File); ok {
		return term.IsTerminal(int(f.Fd()))
	}
	return false
}

type runtimeContextKey struct{}

// WithRuntime は request context に UI runtime を埋め込む。
func WithRuntime(ctx context.Context, runtime *Runtime) context.Context {
	if runtime == nil {
		return ctx
	}
	return context.WithValue(ctx, runtimeContextKey{}, runtime)
}

// RuntimeFromContext は request context から UI runtime を取得する。
func RuntimeFromContext(ctx context.Context) *Runtime {
	if runtime, ok := LookupRuntime(ctx); ok {
		return runtime
	}
	return DefaultRuntime()
}

// LookupRuntime は request context に明示注入された UI runtime のみを取得する。
func LookupRuntime(ctx context.Context) (*Runtime, bool) {
	if ctx == nil {
		return nil, false
	}
	runtime, ok := ctx.Value(runtimeContextKey{}).(*Runtime)
	if !ok || runtime == nil {
		return nil, false
	}
	return runtime, true
}

func pasteDebugEnabled() bool {
	return os.Getenv("XELYON_DEBUG_PASTE") == "1"
}

func pasteDebugf(out io.Writer, format string, args ...interface{}) {
	if !pasteDebugEnabled() {
		return
	}
	if out == nil {
		out = DefaultRuntime().ErrorOutput()
	}
	_, _ = fmt.Fprintf(out, format, args...)
}

func pasteDebugWriteString(out io.Writer, s string) {
	if !pasteDebugEnabled() {
		return
	}
	if out == nil {
		out = DefaultRuntime().ErrorOutput()
	}
	_, _ = io.WriteString(out, s)
}
