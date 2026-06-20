package review

import (
	"testing"

	reviewdomain "github.com/susugadx/xelyon-cli/internal/review/domain"
)

func TestNewCurrentChangesRequest(t *testing.T) {
	got := NewCurrentChangesRequest("focus on correctness")
	if got.TargetKind != reviewdomain.TargetCurrentChanges {
		t.Fatalf("TargetKind = %q, want %q", got.TargetKind, reviewdomain.TargetCurrentChanges)
	}
	if got.CustomInstructions != "focus on correctness" {
		t.Fatalf("CustomInstructions = %q", got.CustomInstructions)
	}
}
