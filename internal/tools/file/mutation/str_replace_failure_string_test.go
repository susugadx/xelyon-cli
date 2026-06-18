package mutation

import (
	"strings"
	"testing"
)

func TestBuildStringReplacementFailure_MultipleMatchesMessage(t *testing.T) {
	failure := stringReplacementFailure{
		reason:     stringReplacementFailureMultipleMatches,
		exactCount: 3,
	}

	message := buildStringReplacementFailure("test.txt", "foo\nbar\nfoo\nbaz\nfoo", "foo", failure)
	if !strings.Contains(message, "Error: old_str appears 3 times in test.txt (must be unique).") {
		t.Fatalf("unexpected failure headline: %s", message)
	}
	if !strings.Contains(message, "Candidates: 3 total (showing 2)") {
		t.Fatalf("unexpected candidate summary: %s", message)
	}
	if !strings.Contains(message, "- ... 1 more candidates") {
		t.Fatalf("unexpected omitted candidate summary: %s", message)
	}
	if !strings.Contains(message, "Next: use read_file on one candidate and retry with a more specific old_str; use start_line/end_line for a fixed range; use batch edits to replace all matches.") {
		t.Fatalf("unexpected guidance: %s", message)
	}
}

func TestBuildStringReplacementFailure_NotFoundMessage(t *testing.T) {
	failure := stringReplacementFailure{
		reason: stringReplacementFailureNotFound,
	}

	message := buildStringReplacementFailure("test.txt", "first\nsecond\nthird\nfourth", "missing", failure)
	if !strings.Contains(message, "Error: old_str not found in test.txt (tried exact and normalized matching).") {
		t.Fatalf("unexpected failure headline: %s", message)
	}
	if !strings.Contains(message, "Preview: 1:first | 2:second | 3:third | ... +1 more lines") {
		t.Fatalf("unexpected preview summary: %s", message)
	}
	if !strings.Contains(message, "Next: use read_file/search_code to copy the exact text, then retry; use start_line/end_line if you already know the target range.") {
		t.Fatalf("unexpected guidance: %s", message)
	}
}

func TestBuildBatchStringReplacementFailure_MultipleMatchesMessage(t *testing.T) {
	failure := batchStringReplacementFailure{
		editIndex:  2,
		oldContent: "foo\nbar\nfoo\nbaz\nfoo",
		oldStr:     "foo",
		failure: stringReplacementFailure{
			reason:     stringReplacementFailureMultipleMatches,
			exactCount: 3,
		},
	}

	message := buildBatchStringReplacementFailure("test.txt", failure)
	if !strings.Contains(message, "Error: edits[2].old_str appears 3 times in test.txt (must be unique; batch aborted, no changes written).") {
		t.Fatalf("unexpected failure headline: %s", message)
	}
	if !strings.Contains(message, "Candidates: 3 total (showing 2)") {
		t.Fatalf("unexpected candidate summary: %s", message)
	}
	if !strings.Contains(message, "Next: use read_file on one candidate and retry with a more specific edits[2].old_str; use line-range mode for a fixed block.") {
		t.Fatalf("unexpected guidance: %s", message)
	}
}

func TestBuildBatchStringReplacementFailure_NotFoundMessage(t *testing.T) {
	failure := batchStringReplacementFailure{
		editIndex:  1,
		oldContent: "AAA\nbbb\nccc\nddd",
		oldStr:     "missing",
		failure: stringReplacementFailure{
			reason: stringReplacementFailureNotFound,
		},
	}

	message := buildBatchStringReplacementFailure("test.txt", failure)
	if !strings.Contains(message, "Error: edits[1].old_str not found in test.txt (tried exact and normalized matching; batch aborted, no changes written).") {
		t.Fatalf("unexpected failure headline: %s", message)
	}
	if !strings.Contains(message, "Preview: 1:AAA | 2:bbb | 3:ccc | ... +1 more lines") {
		t.Fatalf("unexpected preview summary: %s", message)
	}
	if !strings.Contains(message, "Next: use read_file/search_code to copy the exact text for edits[1].old_str, then retry; split the batch if later edits depend on earlier changes.") {
		t.Fatalf("unexpected guidance: %s", message)
	}
}

func TestBuildDeferredStrReplaceResult(t *testing.T) {
	cancelled := buildDeferredStrReplaceResult("[CANCELLED]", "batch", "test.txt", "")
	if !strings.Contains(cancelled, "[CANCELLED] str_replace (batch) not applied for test.txt.") {
		t.Fatalf("unexpected cancelled headline: %s", cancelled)
	}
	if !strings.Contains(cancelled, "Next: review with read_file before retrying; do not repeat the same replacement unchanged.") {
		t.Fatalf("unexpected cancelled guidance: %s", cancelled)
	}

	withComment := buildDeferredStrReplaceResult("[COMMENT]", "single", "test.txt", "  keep as-is  ")
	if !strings.Contains(withComment, "[COMMENT] str_replace (single) not applied for test.txt.") {
		t.Fatalf("unexpected comment headline: %s", withComment)
	}
	if !strings.Contains(withComment, "Comment: keep as-is") {
		t.Fatalf("unexpected comment payload: %s", withComment)
	}
	if !strings.Contains(withComment, "Next: review with read_file and retry only after user approval.") {
		t.Fatalf("unexpected comment guidance: %s", withComment)
	}
}
