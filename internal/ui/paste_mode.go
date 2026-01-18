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

	fmt.Fprintln(out, "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Fprintln(out, "📝 Paste Mode / ペーストモード")
	fmt.Fprintln(out, "   End: empty line x2, 'END', /end, Ctrl+D")
	fmt.Fprintln(out, "   Cancel: /cancel, /c")
	fmt.Fprintln(out, "   終了: 空行2回, END, /end, Ctrl+D")
	fmt.Fprintln(out, "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Fprintln(out)

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

			if line == "" {
				emptyCount++
				if emptyCount >= 2 {
					goto done
				}
				lines = append(lines, line)
				totalBytes += len(line) + 1
				continue
			}

			if line == "END" || line == "/end" {
				goto done
			}

			if line == "/cancel" || line == "/c" {
				return "", true, nil
			}

			emptyCount = 0
			lines = append(lines, line)
			totalBytes += len(line) + 1

			if maxLines > 0 && len(lines) >= maxLines {
				goto done
			}
			if maxBytes > 0 && totalBytes >= maxBytes {
				goto done
			}

		case <-time.After(timeout):
			goto done
		}
	}

done:
	for len(lines) > 0 && strings.TrimSpace(lines[len(lines)-1]) == "" {
		lines = lines[:len(lines)-1]
	}

	if len(lines) == 0 {
		return "", false, nil
	}

	return strings.Join(lines, "\n"), false, nil
}
