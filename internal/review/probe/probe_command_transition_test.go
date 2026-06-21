package probe

import (
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/review/domain"
)

func TestHostReadOnlyResultReducer_BlockedCommandResult(t *testing.T) {
	reducer := newHostReadOnlyResultReducer(ReviewProbeRequest{
		ID:   "host-transition-blocked",
		Mode: domain.ReviewProbeHostReadOnly,
	})

	stop := reducer.applyCommandResult(hostReadOnlyCommand{
		command: "git",
		args:    []string{"status", "--short"},
	}, ReviewProbeCommandResult{
		Status: domain.ReviewProbeBlocked,
	})

	result := reducer.resultValue()
	if !stop {
		t.Fatal("stop = false, want true")
	}
	if result.Status != domain.ReviewProbeBlocked {
		t.Fatalf("Status = %q, want %q", result.Status, domain.ReviewProbeBlocked)
	}
	if len(result.CommandResults) != 1 {
		t.Fatalf("len(CommandResults) = %d, want 1", len(result.CommandResults))
	}
	if !strings.Contains(result.Error, "probe command blocked") {
		t.Fatalf("Error = %q, want to contain %q", result.Error, "probe command blocked")
	}
}

func TestScratchOnlyCommandTransition_BlockedCommandResult(t *testing.T) {
	result := newScratchOnlyProbeResult(ReviewProbeRequest{
		ID:   "scratch-transition-blocked",
		Mode: domain.ReviewProbeScratchOnly,
	})

	stop := applyScratchOnlyCommandTransition(&result, scratchOnlyCommand{
		command: "cat",
		args:    []string{"check.txt"},
	}, ReviewProbeCommandResult{
		Status: domain.ReviewProbeBlocked,
	})

	if !stop {
		t.Fatal("stop = false, want true")
	}
	if result.Status != domain.ReviewProbeBlocked {
		t.Fatalf("Status = %q, want %q", result.Status, domain.ReviewProbeBlocked)
	}
	if len(result.CommandResults) != 1 {
		t.Fatalf("len(CommandResults) = %d, want 1", len(result.CommandResults))
	}
	if !strings.Contains(result.Error, "probe command blocked") {
		t.Fatalf("Error = %q, want to contain %q", result.Error, "probe command blocked")
	}
}
