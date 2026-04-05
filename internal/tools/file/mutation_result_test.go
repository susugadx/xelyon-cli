package file

import "testing"

func TestFileMutationResult_ShouldRecordChange(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		result fileMutationResult
		want   bool
	}{
		{name: "applied", result: newAppliedMutationResult("ok"), want: true},
		{name: "noop", result: newNoopMutationResult("no changes"), want: false},
		{name: "cancelled", result: newCancelledMutationResult("Cancelled by user"), want: false},
		{name: "comment", result: newCommentMutationResult("[COMMENT] feedback"), want: false},
		{name: "error", result: newErrorMutationResult("Error: failed"), want: false},
		{name: "zero value", result: fileMutationResult{}, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.result.ShouldRecordChange(); got != tt.want {
				t.Fatalf("ShouldRecordChange() = %v, want %v", got, tt.want)
			}
		})
	}
}
