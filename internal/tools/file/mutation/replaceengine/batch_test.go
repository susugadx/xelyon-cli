package replaceengine

import (
	"strings"
	"testing"
)

func TestBuildBatchOutcome_NormalizedAttemptedEdits(t *testing.T) {
	original := "func main() {\n\tfmt.Println(\"hello\")\n}\nmarker"
	edits := []Edit{
		{
			OldStr: "func main() {\n    fmt.Println(\"hello\")\n}",
			NewStr: "func main() {\n\tfmt.Println(\"world\")\n}",
		},
		{
			OldStr: "marker",
			NewStr: "done",
		},
	}

	outcome := BuildBatchOutcome(original, edits)
	if failure, ok := outcome.Failure(); ok {
		t.Fatalf("expected successful outcome, got failure: %+v", failure)
	}
	normalizedAttemptedEdits := outcome.Plan().NormalizedAttemptedEdits()
	if len(normalizedAttemptedEdits) != 1 || normalizedAttemptedEdits[0] != 0 {
		t.Fatalf("unexpected normalized-attempted edits: %+v", normalizedAttemptedEdits)
	}
	if !strings.Contains(outcome.Plan().NewContent(), "world") || !strings.Contains(outcome.Plan().NewContent(), "done") {
		t.Fatalf("unexpected new content: %q", outcome.Plan().NewContent())
	}
}
