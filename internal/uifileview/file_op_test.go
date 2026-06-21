package uifileview

import (
	"bytes"
	"strings"
	"testing"
)

func TestFileOpHelpers_PlainOutput(t *testing.T) {
	var buf bytes.Buffer
	FileOpHeader(&buf, "write", "test.txt")
	FileOpDivider(&buf, 4)
	FileOpStatsLine(&buf, 2, 5)
	FileOpStatsLine(&buf, 5, 2)
	FileOpStatsLine(&buf, 3, 3)
	FileOpPathLine(&buf, "M", "file.txt", "(+5, -2)")

	got := buf.String()
	for _, want := range []string{"write", "test.txt", "────", "+5", "-5", "net +3", "net -3", "net 0", "M file.txt"} {
		if !strings.Contains(got, want) {
			t.Fatalf("file op output missing %q:\n%s", want, got)
		}
	}
}

func TestFileOpHelpers_NilWriter(t *testing.T) {
	FileOpHeader(nil, "write", "test.txt")
	FileOpDivider(nil, 5)
	FileOpStatsLine(nil, 1, 2)
	FileOpPathLine(nil, "M", "file.txt", "")
}
