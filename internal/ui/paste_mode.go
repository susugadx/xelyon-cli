package ui

import (
	"bufio"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/susugadx/xelyon-cli/internal/config"
)

// PasteMode captures multiline input for environments where bracketed paste mode is unreliable.
// It is also reusable from other input contexts (e.g., comment input during confirmations).
//
// End conditions:
// - empty line x2
// - "END" or "/end"
// - Ctrl+D (EOF)
//
// Cancel:
// - "/cancel" or "/c" (content is discarded)
//
// Limits:
// - max lines / max bytes
// - idle timeout
//
// Note: This does NOT read from the OS clipboard. It is an interactive capture mode.
type PasteMode struct {
	cfg config.PasteConfig
}

func NewPasteMode(cfg config.PasteConfig) *PasteMode {
	return &PasteMode{cfg: cfg}
}

// Capture reads multiline input from in and writes prompts/help to out.
// Returns captured content, cancelled=true when user cancelled, and error when I/O fails.
// Deprecated: Use CaptureWithReader for better buffer sharing.
func (p *PasteMode) Capture(in io.Reader, out io.Writer) (content string, cancelled bool, err error) {
	return p.CaptureWithReader(bufio.NewReader(in), out)
}

// CaptureWithReader reads multiline input using an existing bufio.Reader.
// This avoids buffer conflicts when sharing stdin with other readers.
// Returns captured content, cancelled=true when user cancelled, and error when I/O fails.
func (p *PasteMode) CaptureWithReader(reader *bufio.Reader, out io.Writer) (content string, cancelled bool, err error) {
	maxLines := p.cfg.MaxLines
	maxBytes := p.cfg.MaxBytes
	timeout := time.Duration(p.cfg.TimeoutSeconds) * time.Second

	p.printBanner(out)

	var lines []string
	emptyCount := 0
	totalBytes := 0

	type readResult struct {
		line string
		err  error
	}
	inputChan := make(chan readResult, 1)

	for {
		go func() {
			line, e := reader.ReadString('\n')
			inputChan <- readResult{line: line, err: e}
		}()

		select {
		case result := <-inputChan:
			if result.err == io.EOF {
				goto done
			}
			if result.err != nil {
				return "", false, result.err
			}

			line := strings.TrimRight(result.line, "\r\n")
			line = StripBracketedPaste(line)

			action := p.processLine(line, &lines, &emptyCount, &totalBytes, maxLines, maxBytes)
			switch action {
			case pasteActionDone:
				goto done
			case pasteActionCancel:
				return "", true, nil
			}

		case <-time.After(timeout):
			goto done
		}
	}

done:
	return p.finalize(lines), false, nil
}

// CaptureWithMultilineReader reads multiline input using a MultilineReader.
// When the raw mode goroutine is active, this uses ReadSimpleLine() which enters
// raw mode to suppress terminal echo of paste markers.
func (p *PasteMode) CaptureWithMultilineReader(mlReader *MultilineReader, out io.Writer) (content string, cancelled bool, err error) {
	maxLines := p.cfg.MaxLines
	maxBytes := p.cfg.MaxBytes
	timeout := time.Duration(p.cfg.TimeoutSeconds) * time.Second

	p.printBanner(out)

	var lines []string
	emptyCount := 0
	totalBytes := 0

	type readResult struct {
		line string
		err  error
	}
	inputChan := make(chan readResult, 1)

	for {
		go func() {
			line, e := mlReader.ReadSimpleLine()
			inputChan <- readResult{line: line, err: e}
		}()

		select {
		case result := <-inputChan:
			if result.err == io.EOF {
				goto done
			}
			if result.err != nil {
				return "", false, result.err
			}

			line := result.line // ReadSimpleLine already strips paste markers

			action := p.processLine(line, &lines, &emptyCount, &totalBytes, maxLines, maxBytes)
			switch action {
			case pasteActionDone:
				goto done
			case pasteActionCancel:
				return "", true, nil
			}

		case <-time.After(timeout):
			goto done
		}
	}

done:
	return p.finalize(lines), false, nil
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
