package readtool

import (
	"errors"
	"fmt"
	"io"
	"reflect"
	"strings"
	"testing"
)

type errReader struct {
	err error
}

func (r errReader) Read(_ []byte) (int, error) {
	return 0, r.err
}

func TestReadWindowLines_StopsBeforeFailingTail(t *testing.T) {
	t.Parallel()

	reader := io.MultiReader(
		strings.NewReader("line1\nline2\nline3\n"),
		errReader{err: errors.New("unexpected tail read")},
	)

	lines, totalRead, err := readWindowLines(reader, 1, 2, 0)
	if err != nil {
		t.Fatalf("readWindowLines() error = %v", err)
	}
	if totalRead != 3 {
		t.Fatalf("readWindowLines() totalRead = %d, want 3", totalRead)
	}
	if want := []string{"line1", "line2"}; !reflect.DeepEqual(lines, want) {
		t.Fatalf("readWindowLines() lines = %#v, want %#v", lines, want)
	}
}

func TestReadWindowLines_AllowsLongLines(t *testing.T) {
	t.Parallel()

	longLine := strings.Repeat("x", 2*1024*1024)
	reader := strings.NewReader(longLine + "\nsecond\n")

	lines, totalRead, err := readWindowLines(reader, 2, 2, 0)
	if err != nil {
		t.Fatalf("readWindowLines() error = %v", err)
	}
	if totalRead != 2 {
		t.Fatalf("readWindowLines() totalRead = %d, want 2", totalRead)
	}
	if want := []string{"second"}; !reflect.DeepEqual(lines, want) {
		t.Fatalf("readWindowLines() lines = %#v, want %#v", lines, want)
	}
}

func TestReadWindowLines_TruncatesLongLinesWhenCapped(t *testing.T) {
	t.Parallel()

	longLine := strings.Repeat("x", previewMaxLineBytes+128)
	reader := strings.NewReader(longLine + "\n")

	lines, totalRead, err := readWindowLines(reader, 1, 1, previewMaxLineBytes)
	if err != nil {
		t.Fatalf("readWindowLines() error = %v", err)
	}
	if totalRead != 1 {
		t.Fatalf("readWindowLines() totalRead = %d, want 1", totalRead)
	}
	if len(lines) != 1 {
		t.Fatalf("readWindowLines() lines len = %d, want 1", len(lines))
	}
	if !strings.HasSuffix(lines[0], "...") {
		t.Fatalf("readWindowLines() should truncate long lines, got suffixless line")
	}
	if len(lines[0]) != previewMaxLineBytes+3 {
		t.Fatalf("readWindowLines() truncated line len = %d, want %d", len(lines[0]), previewMaxLineBytes+3)
	}
}

func TestReadOutlineSample_PropagatesTailReadErrors(t *testing.T) {
	t.Parallel()

	reader := io.MultiReader(
		strings.NewReader("line1\nline2\nline3\n"),
		errReader{err: errors.New("unexpected tail read")},
	)

	_, _, _, _, _, err := readOutlineSample(reader, 2, 1, previewMaxLineBytes)
	if err == nil || !strings.Contains(err.Error(), "unexpected tail read") {
		t.Fatalf("readOutlineSample() error = %v, want unexpected tail read", err)
	}
}

func TestReadOutlineSample_PreservesTailForShortFiles(t *testing.T) {
	t.Parallel()

	reader := strings.NewReader("line1\nline2\nline3\nline4\n")

	_, tailLines, totalLines, hasMore, truncated, err := readOutlineSample(reader, 10, 2, previewMaxLineBytes)
	if err != nil {
		t.Fatalf("readOutlineSample() error = %v", err)
	}
	if hasMore {
		t.Fatalf("readOutlineSample() hasMore = true, want false")
	}
	if truncated {
		t.Fatalf("readOutlineSample() truncated = true, want false")
	}
	if totalLines != 4 {
		t.Fatalf("readOutlineSample() totalLines = %d, want 4", totalLines)
	}
	if want := []string{"line3", "line4"}; !reflect.DeepEqual(tailLines, want) {
		t.Fatalf("readOutlineSample() tailLines = %#v, want %#v", tailLines, want)
	}
}

func TestReadOutlineSample_KeepsActualTailAndTotalForLongFiles(t *testing.T) {
	t.Parallel()

	var sb strings.Builder
	for i := 1; i <= 2200; i++ {
		fmt.Fprintf(&sb, "line%d\n", i)
	}

	headLines, tailLines, totalLines, hasMore, truncated, err := readOutlineSample(strings.NewReader(sb.String()), MaxReadLines, outlineTailLines, previewMaxLineBytes)
	if err != nil {
		t.Fatalf("readOutlineSample() error = %v", err)
	}
	if !hasMore {
		t.Fatalf("readOutlineSample() hasMore = false, want true")
	}
	if truncated {
		t.Fatalf("readOutlineSample() truncated = true, want false")
	}
	if totalLines != 2200 {
		t.Fatalf("readOutlineSample() totalLines = %d, want 2200", totalLines)
	}
	if len(headLines) != MaxReadLines {
		t.Fatalf("readOutlineSample() headLines len = %d, want %d", len(headLines), MaxReadLines)
	}
	if want := []string{"line2191", "line2192", "line2193", "line2194", "line2195", "line2196", "line2197", "line2198", "line2199", "line2200"}; !reflect.DeepEqual(tailLines, want) {
		t.Fatalf("readOutlineSample() tailLines = %#v, want %#v", tailLines, want)
	}
}
