package mutation

import (
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/tools/file/mutation/replaceengine"
)

func TestBuildLineRangeReplacementFailure_MissingRange(t *testing.T) {
	failure := replaceengine.BuildLineRangeExecution("one\ntwo", "new", "", "").Failure()
	message := buildLineRangeReplacementFailure("test.txt", failure)
	expected := "Error: old_str is required (or provide both start_line and end_line for line-range replacement)"
	if message != expected {
		t.Fatalf("unexpected failure message:\nwant: %s\n got: %s", expected, message)
	}
}

func TestBuildLineRangeReplacementFailure_InvalidRange(t *testing.T) {
	failure := replaceengine.BuildLineRangeExecution("one\ntwo", "new", "x", "2").Failure()
	message := buildLineRangeReplacementFailure("test.txt", failure)
	if !strings.Contains(message, "Error: invalid line range in test.txt: invalid start_line: strconv.ParseInt: parsing \"x\": invalid syntax") {
		t.Fatalf("unexpected failure headline: %s", message)
	}
	if !strings.Contains(message, "Next: use read_file to confirm start_line/end_line (1-indexed inclusive).") {
		t.Fatalf("unexpected guidance: %s", message)
	}
}

func TestBuildLineRangeReplacementFailure_StartOutOfRange(t *testing.T) {
	failure := replaceengine.BuildLineRangeExecution("one\ntwo\nthree", "new", "10", "10").Failure()
	message := buildLineRangeReplacementFailure("test.txt", failure)
	if !strings.Contains(message, "Error: start_line is out of range in test.txt (start_line=10, file_lines=3).") {
		t.Fatalf("unexpected failure headline: %s", message)
	}
	if !strings.Contains(message, "Next: use read_file to confirm the target range.") {
		t.Fatalf("unexpected guidance: %s", message)
	}
}

func TestBuildLineRangeReplacementFailure_EndOutOfRangeMessage(t *testing.T) {
	failure := replaceengine.BuildLineRangeExecution("one\ntwo\nthree", "new", "2", "20").Failure()
	message := buildLineRangeReplacementFailure("test.txt", failure)
	if !strings.Contains(message, "Error: end_line is out of range in test.txt (end_line=20, file_lines=3).") {
		t.Fatalf("unexpected failure headline: %s", message)
	}
	if !strings.Contains(message, "Next: use read_file to confirm the target range.") {
		t.Fatalf("unexpected guidance: %s", message)
	}
}

func TestBuildAppliedLineRangeStrReplaceResult(t *testing.T) {
	execution := replaceengine.BuildLineRangeExecution("1\n2\n3\n4\n5\n6", "a\nb\nc", "5", "6")
	message := buildAppliedLineRangeStrReplaceResult("test.txt", execution.Plan())
	expected := "Successfully replaced lines 5-6 in test.txt (new range: 5-7)"
	if message != expected {
		t.Fatalf("unexpected success message:\nwant: %s\n got: %s", expected, message)
	}
}
