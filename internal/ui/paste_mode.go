package ui

import (
	"bufio"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/susugadx/xelyon-cli/internal/config"
)

// PasteMode は bracketed paste が不安定な環境向けに複数行入力を対話的に収集する。
//
// 終了条件:
// - 空行2回
// - "END" または "/end"
// - Ctrl+D (EOF)
//
// キャンセル:
// - "/cancel" または "/c"（入力内容は破棄）
//
// 制約:
// - 最大行数 / 最大バイト数
// - アイドルタイムアウト
//
// 注記: OSクリップボードは参照しない。
type PasteMode struct {
	cfg config.PasteConfig
}

type pasteLineReader func() (string, error)

type pasteReadResult struct {
	line string
	err  error
}

// NewPasteMode は設定付きの PasteMode を生成する。
func NewPasteMode(cfg config.PasteConfig) *PasteMode {
	return &PasteMode{cfg: cfg}
}

// Capture は in から複数行入力を読み取り、案内文を out へ出力する。
// キャンセル時は cancelled=true を返す。
// Deprecated: バッファ共有のため CaptureWithReader を使うこと。
func (p *PasteMode) Capture(in io.Reader, out io.Writer) (content string, cancelled bool, err error) {
	return p.CaptureWithReader(bufio.NewReader(in), out)
}

// CaptureWithReader は既存の bufio.Reader で複数行入力を収集する。
// stdin を他 reader と共有する際のバッファ競合を避ける。
func (p *PasteMode) CaptureWithReader(reader *bufio.Reader, out io.Writer) (content string, cancelled bool, err error) {
	return p.captureLoop(out, func() (string, error) {
		line, err := reader.ReadString('\n')
		if err != nil {
			return "", err
		}
		line = strings.TrimRight(line, "\r\n")
		return StripBracketedPaste(line), nil
	})
}

// CaptureWithMultilineReader は MultilineReader を使って複数行入力を収集する。
// raw mode goroutine が有効な場合は ReadSimpleLine() 側で raw mode に入り、
// paste marker の端末エコーを抑制する。
func (p *PasteMode) CaptureWithMultilineReader(mlReader *MultilineReader, out io.Writer) (content string, cancelled bool, err error) {
	return p.captureLoop(out, mlReader.ReadSimpleLine)
}

func (p *PasteMode) captureLoop(out io.Writer, readLine pasteLineReader) (content string, cancelled bool, err error) {
	maxLines := p.cfg.MaxLines
	maxBytes := p.cfg.MaxBytes
	timeout := time.Duration(p.cfg.TimeoutSeconds) * time.Second

	p.printBanner(out)

	var lines []string
	emptyCount := 0
	totalBytes := 0

	for {
		line, timedOut, err := p.readLineWithTimeout(readLine, timeout)
		if timedOut {
			goto done
		}
		if err == io.EOF {
			goto done
		}
		if err != nil {
			return "", false, err
		}

		action := p.processLine(line, &lines, &emptyCount, &totalBytes, maxLines, maxBytes)
		switch action {
		case pasteActionDone:
			goto done
		case pasteActionCancel:
			return "", true, nil
		}
	}

done:
	return p.finalize(lines), false, nil
}

func (p *PasteMode) readLineWithTimeout(readLine pasteLineReader, timeout time.Duration) (line string, timedOut bool, err error) {
	inputChan := make(chan pasteReadResult, 1)
	go func() {
		line, err := readLine()
		inputChan <- pasteReadResult{line: line, err: err}
	}()

	select {
	case result := <-inputChan:
		return result.line, false, result.err
	case <-time.After(timeout):
		return "", true, nil
	}
}

type pasteAction int

const (
	pasteActionContinue pasteAction = iota
	pasteActionDone
	pasteActionCancel
)

func (p *PasteMode) printBanner(out io.Writer) {
	fmt.Fprintln(out, "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Fprintln(out, "📝 Paste Mode / ペーストモード")
	fmt.Fprintln(out, "   End: empty line x2, 'END', /end, Ctrl+D")
	fmt.Fprintln(out, "   Cancel: /cancel, /c")
	fmt.Fprintln(out, "   終了: 空行2回, END, /end, Ctrl+D")
	fmt.Fprintln(out, "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Fprintln(out)
}

func (p *PasteMode) processLine(line string, lines *[]string, emptyCount *int, totalBytes *int, maxLines, maxBytes int) pasteAction {
	if line == "" {
		*emptyCount++
		if *emptyCount >= 2 {
			return pasteActionDone
		}
		*lines = append(*lines, line)
		*totalBytes += len(line) + 1
		return pasteActionContinue
	}

	if line == "END" || line == "/end" {
		return pasteActionDone
	}

	if line == "/cancel" || line == "/c" {
		return pasteActionCancel
	}

	*emptyCount = 0
	*lines = append(*lines, line)
	*totalBytes += len(line) + 1

	if maxLines > 0 && len(*lines) >= maxLines {
		return pasteActionDone
	}
	if maxBytes > 0 && *totalBytes >= maxBytes {
		return pasteActionDone
	}

	return pasteActionContinue
}

func (p *PasteMode) finalize(lines []string) string {
	for len(lines) > 0 && strings.TrimSpace(lines[len(lines)-1]) == "" {
		lines = lines[:len(lines)-1]
	}

	if len(lines) == 0 {
		return ""
	}

	return strings.Join(lines, "\n")
}
