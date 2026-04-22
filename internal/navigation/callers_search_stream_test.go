package navigation

import (
	"errors"
	"io"
	"strconv"
	"strings"
	"testing"
)

type stagedErrorReader struct {
	chunks   [][]byte
	finalErr error
}

func (r *stagedErrorReader) Read(p []byte) (int, error) {
	if len(r.chunks) > 0 {
		chunk := r.chunks[0]
		n := copy(p, chunk)
		if n == len(chunk) {
			r.chunks = r.chunks[1:]
		} else {
			r.chunks[0] = r.chunks[0][n:]
		}
		return n, nil
	}
	if r.finalErr != nil {
		err := r.finalErr
		r.finalErr = nil
		return 0, err
	}
	return 0, io.EOF
}

func makeRipgrepOutput(symbol string, count int) string {
	var sb strings.Builder
	for i := range count {
		sb.WriteString("file.go:")
		sb.WriteString(strconv.Itoa(i + 1))
		sb.WriteString(":")
		sb.WriteString(symbol)
		sb.WriteString("()\n")
	}
	return sb.String()
}

func TestCollectReferenceSearchResult_LongLineDoesNotSilentlyDrop(t *testing.T) {
	symbol := "Target"
	longLine := "file.go:1:" + strings.Repeat("x", 70*1024) + symbol + "()\n"

	result := collectReferenceSearchResult(strings.NewReader(longLine), symbol)
	if result.Incomplete {
		t.Fatal("expected complete result for line larger than default scanner buffer")
	}
	if result.Truncated {
		t.Fatal("expected no truncation for single long line")
	}
	if len(result.Refs) != 1 {
		t.Fatalf("expected 1 ref, got %d", len(result.Refs))
	}
	if !strings.Contains(result.Refs[0].Snippet, symbol+"()") {
		t.Fatalf("expected snippet to contain symbol call, got: %q", result.Refs[0].Snippet)
	}
}

func TestCollectReferenceSearchResult_ReaderErrorMarksIncomplete(t *testing.T) {
	symbol := "Target"
	reader := &stagedErrorReader{
		chunks:   [][]byte{[]byte("file.go:1:Target()\n")},
		finalErr: errors.New("read failed"),
	}

	result := collectReferenceSearchResult(reader, symbol)
	if !result.Incomplete {
		t.Fatal("expected incomplete result when reader returns an error")
	}
	if result.Truncated {
		t.Fatal("expected no truncation on reader error")
	}
	if len(result.Refs) != 1 {
		t.Fatalf("expected first parsed ref to be preserved, got %d", len(result.Refs))
	}
}

func TestRunReferenceSearch_WaitErrorMarksIncomplete(t *testing.T) {
	refs, truncated, incomplete := runReferenceSearch(
		strings.NewReader("file.go:1:Target()\n"),
		"Target",
		nil,
		func() error { return errors.New("rg exit status 2") },
	)

	if truncated {
		t.Fatal("expected no truncation")
	}
	if !incomplete {
		t.Fatal("expected incomplete result when wait returns an error")
	}
	if len(refs) != 1 {
		t.Fatalf("expected 1 ref, got %d", len(refs))
	}
}

func TestFindReferences_ExactlyLimitNotTruncated(t *testing.T) {
	refs, truncated, incomplete := runReferenceSearch(
		strings.NewReader(makeRipgrepOutput("Target", maxRipgrepResults)),
		"Target",
		nil,
		func() error { return nil },
	)

	if truncated {
		t.Fatalf("expected truncated=false for exactly %d refs", maxRipgrepResults)
	}
	if incomplete {
		t.Fatal("expected complete result for exactly-on-limit refs")
	}
	if len(refs) != maxRipgrepResults {
		t.Fatalf("expected %d refs, got %d", maxRipgrepResults, len(refs))
	}
}

func TestFindReferences_OverLimitTruncated(t *testing.T) {
	canceled := false
	refs, truncated, incomplete := runReferenceSearch(
		strings.NewReader(makeRipgrepOutput("Target", maxRipgrepResults+1)),
		"Target",
		func() { canceled = true },
		func() error { return errors.New("signal: killed") },
	)

	if !truncated {
		t.Fatalf("expected truncated=true for %d refs", maxRipgrepResults+1)
	}
	if incomplete {
		t.Fatal("expected intentional cancellation to remain complete")
	}
	if len(refs) != maxRipgrepResults {
		t.Fatalf("expected %d refs after truncation, got %d", maxRipgrepResults, len(refs))
	}
	if !canceled {
		t.Fatalf("expected cancel after detecting the %dth result", maxRipgrepResults+1)
	}
}
