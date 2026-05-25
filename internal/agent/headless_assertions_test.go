package agent

import (
	"fmt"
	"strings"
	"testing"
)

func requireHeadlessToolLoopLimitError(t *testing.T, result *HeadlessResult, wantLimit int) {
	t.Helper()

	if result == nil {
		t.Fatal("result = nil, want headless result")
	}
	if result.Status != HeadlessStatusError {
		t.Fatalf("result.Status = %q, want %q", result.Status, HeadlessStatusError)
	}
	if result.Error == nil {
		t.Fatal("result.Error = nil, want tool loop limit error")
	}
	if result.Error.Type != HeadlessErrorTypeToolLoopLimit {
		t.Fatalf("result.Error.Type = %q, want %q", result.Error.Type, HeadlessErrorTypeToolLoopLimit)
	}
	if result.Response != "" {
		t.Fatalf("result.Response = %q, want empty error response", result.Response)
	}
	wantFragment := fmt.Sprintf("%d iterations", wantLimit)
	if !strings.Contains(result.Error.Message, wantFragment) {
		t.Fatalf("result.Error.Message = %q, want %q", result.Error.Message, wantFragment)
	}
}
