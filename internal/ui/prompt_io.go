package ui

import (
	"bufio"
	"io"
	"os"
	"strings"
)

// PromptIO は対話 UI の入出力先を表す。
type PromptIO struct {
	In     io.Reader
	Out    io.Writer
	Err    io.Writer
	Reader *MultilineReader
}

// NewPromptIO は入出力先を明示した PromptIO を返す。
func NewPromptIO(in io.Reader, out, err io.Writer, reader *MultilineReader) PromptIO {
	return normalizePromptIO(PromptIO{
		In:     in,
		Out:    out,
		Err:    err,
		Reader: reader,
	})
}

// DefaultPromptIO は process stdio を使う PromptIO を返す。
func DefaultPromptIO() PromptIO {
	return normalizePromptIO(PromptIO{})
}

// ReadSimpleLine は PromptIO に紐づく入力から1行読む。
func (p PromptIO) ReadSimpleLine() (string, error) {
	p = normalizePromptIO(p)
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
func (p PromptIO) BufioReader() *bufio.Reader {
	p = normalizePromptIO(p)
	if p.Reader != nil {
		return p.Reader.GetBufioReader()
	}
	return bufio.NewReader(p.In)
}

func normalizePromptIO(p PromptIO) PromptIO {
	if p.In == nil {
		p.In = os.Stdin
	}
	if p.Out == nil {
		p.Out = os.Stdout
	}
	if p.Err == nil {
		p.Err = os.Stderr
	}
	return p
}
