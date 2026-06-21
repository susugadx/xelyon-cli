package uiruntime

import (
	"bufio"
	"context"
	"io"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/uiprompt"
)

// PromptIO は対話 UI の入出力先を表す。
type PromptIO struct {
	In      io.Reader
	Out     io.Writer
	Err     io.Writer
	Reader  *MultilineReader
	Context context.Context

	simpleReader *bufio.Reader
	runtime      *Runtime
	prompter     uiprompt.Prompter
}

// NewPromptIO は入出力先を明示した PromptIO を返す。
func NewPromptIO(in io.Reader, out, err io.Writer, reader *MultilineReader) PromptIO {
	return NormalizePromptIO(PromptIO{
		In:     in,
		Out:    out,
		Err:    err,
		Reader: reader,
	})
}

// NewPromptIOWithRuntime は Runtime を紐づけた PromptIO を返す。
// stopSpinnerForPromptIO が正しい Runtime のスピナーを停止できるようにする。
func NewPromptIOWithRuntime(in io.Reader, out, err io.Writer, reader *MultilineReader, runtime *Runtime) PromptIO {
	return NormalizePromptIO(PromptIO{
		In:      in,
		Out:     out,
		Err:     err,
		Reader:  reader,
		runtime: runtime,
	})
}

// DefaultPromptIO は process stdio を使う PromptIO を返す。
func DefaultPromptIO() PromptIO {
	return DefaultRuntime().PromptIO()
}

// NormalizePromptIO は不足している入出力を補い、共有 reader を初期化する。
func NormalizePromptIO(p PromptIO) PromptIO {
	return normalizePromptIO(p)
}

// ReadSimpleLine は PromptIO に紐づく入力から1行読む。
func (p *PromptIO) ReadSimpleLine() (string, error) {
	*p = NormalizePromptIO(*p)
	if p.Reader != nil {
		return p.Reader.ReadSimpleLine()
	}

	line, err := p.BufioReader().ReadString('\n')
	if err != nil {
		if err == io.EOF && line != "" {
			line = strings.TrimRight(line, "\n\r")
			return StripBracketedPaste(line), nil
		}
		return "", err
	}

	line = strings.TrimRight(line, "\n\r")
	return StripBracketedPaste(line), nil
}

// BufioReader は共有可能な bufio.Reader を返す。
func (p *PromptIO) BufioReader() *bufio.Reader {
	*p = NormalizePromptIO(*p)
	if p.Reader != nil {
		return p.Reader.GetBufioReader()
	}
	return p.simpleReader
}

// Prompter は PromptIO に紐づく prompt 実装を返す。
func (p *PromptIO) Prompter() uiprompt.Prompter {
	*p = NormalizePromptIO(*p)
	return p.prompter
}

// PromptContext は prompt 実行に使う context を返す。
func (p *PromptIO) PromptContext() context.Context {
	*p = NormalizePromptIO(*p)
	if p.Context != nil {
		return p.Context
	}
	return context.Background()
}

func normalizePromptIO(p PromptIO) PromptIO {
	defaultRuntime := DefaultRuntime()
	if p.runtime != nil {
		if p.In == nil {
			p.In = p.runtime.Input()
		}
		if p.Out == nil {
			p.Out = p.runtime.Output()
		}
		if p.Err == nil {
			p.Err = p.runtime.ErrorOutput()
		}
		if p.Reader == nil {
			p.Reader = p.runtime.PromptReader()
		}
		if p.prompter == nil {
			p.prompter = p.runtime.Prompter()
		}
	}
	if p.In == nil {
		p.In = defaultRuntime.Input()
	}
	if p.Out == nil {
		p.Out = defaultRuntime.Output()
	}
	if p.Err == nil {
		p.Err = defaultRuntime.ErrorOutput()
	}
	if p.Reader == nil && p.simpleReader == nil {
		p.simpleReader = bufio.NewReader(p.In)
	}
	if p.prompter == nil {
		p.prompter = defaultRuntime.Prompter()
	}
	return p
}

func stopSpinnerForPromptIO(promptIO PromptIO) {
	if promptIO.runtime != nil {
		promptIO.runtime.StopSpinner()
		return
	}
	DefaultRuntime().StopSpinner()
}
