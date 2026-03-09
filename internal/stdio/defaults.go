package stdio

import (
	"io"
	"os"
	"sync"
)

var (
	defaultMu     sync.RWMutex
	defaultIn     io.Reader
	defaultOut    io.Writer
	defaultErrOut io.Writer
)

// Input は互換用途のデフォルト入力先を返す。
// 明示上書きがない場合は process stdin を返す。
func Input() io.Reader {
	defaultMu.RLock()
	in := defaultIn
	defaultMu.RUnlock()
	if in != nil {
		return in
	}
	return os.Stdin
}

// Output は互換用途のデフォルト出力先を返す。
// 明示上書きがない場合は process stdout を返す。
func Output() io.Writer {
	defaultMu.RLock()
	out := defaultOut
	defaultMu.RUnlock()
	if out != nil {
		return out
	}
	return os.Stdout
}

// ErrorOutput は互換用途のデフォルトエラー出力先を返す。
// 明示上書きがない場合は process stderr を返す。
func ErrorOutput() io.Writer {
	defaultMu.RLock()
	errOut := defaultErrOut
	defaultMu.RUnlock()
	if errOut != nil {
		return errOut
	}
	return os.Stderr
}

// SetDefaults は互換用途のデフォルト入出力先を更新する。
// nil を渡した項目は現在の process stdio を参照する。
func SetDefaults(in io.Reader, out, errOut io.Writer) {
	defaultMu.Lock()
	defer defaultMu.Unlock()
	defaultIn = in
	defaultOut = out
	defaultErrOut = errOut
}
