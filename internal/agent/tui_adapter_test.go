package agent

import "testing"

func TestTUICaptureWriter_FlushEmitsTrailingFragment(t *testing.T) {
	var got []string
	w := newTUICaptureWriter(func(text string) {
		got = append(got, text)
	})

	if _, err := w.Write([]byte("partial")); err != nil {
		t.Fatalf("Write() error = %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("expected no flushed lines before Flush(), got %v", got)
	}

	w.Flush()

	if len(got) != 1 {
		t.Fatalf("flushed line count = %d, want 1", len(got))
	}
	if got[0] != "partial" {
		t.Fatalf("flushed text = %q, want %q", got[0], "partial")
	}
}
