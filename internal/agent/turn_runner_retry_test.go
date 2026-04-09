package agent

import (
	"fmt"
	"strings"
	"testing"
)

func TestRetryState_DistinctErrors_NeverStalled(t *testing.T) {
	rs := &retryState{}
	for i := 0; i < 10; i++ {
		level := rs.recordFailure(fmt.Sprintf("error %d: something went wrong", i))
		if level != stalledNone {
			t.Fatalf("distinct error at iteration %d returned %d, want stalledNone", i, level)
		}
	}
	if rs.count != 10 {
		t.Errorf("count = %d, want 10", rs.count)
	}
}

func TestRetryState_SameError_SoftThenHard(t *testing.T) {
	rs := &retryState{}
	sameErr := "Error: compilation failed"

	for i := 0; i < stalledRetryThreshold-1; i++ {
		if rs.recordFailure(sameErr) != stalledNone {
			t.Fatalf("iteration %d: expected stalledNone", i)
		}
	}

	for i := 0; i < stalledHardThreshold; i++ {
		level := rs.recordFailure(sameErr)
		if level != stalledSoft {
			t.Fatalf("soft phase iteration %d: got %d, want stalledSoft", i, level)
		}
	}

	if rs.recordFailure(sameErr) != stalledHard {
		t.Fatal("expected stalledHard after soft threshold exhausted")
	}
}

func TestRetryState_DifferentErrorResetsStalledRuns(t *testing.T) {
	rs := &retryState{}

	rs.recordFailure("error A")
	rs.recordFailure("error A")
	if rs.sameCount != 2 {
		t.Fatalf("sameCount = %d, want 2", rs.sameCount)
	}

	rs.recordFailure("error B")
	if rs.sameCount != 1 {
		t.Fatalf("sameCount after different error = %d, want 1", rs.sameCount)
	}
	if rs.stalledRuns != 0 {
		t.Fatalf("stalledRuns after different error = %d, want 0", rs.stalledRuns)
	}
	if rs.count != 3 {
		t.Fatalf("count = %d, want 3", rs.count)
	}
}

func TestRetryState_Reset(t *testing.T) {
	rs := &retryState{}
	rs.recordFailure("error")
	rs.recordFailure("error")
	rs.recordFailure("error")
	rs.reset()
	if rs.count != 0 || rs.sameCount != 0 || rs.stalledRuns != 0 {
		t.Errorf("reset() did not clear state: count=%d sameCount=%d stalledRuns=%d",
			rs.count, rs.sameCount, rs.stalledRuns)
	}
}

func TestErrorFingerprint_Normalize(t *testing.T) {
	tests := []struct {
		name  string
		input string
		check func(string) bool
	}{
		{"short preserved", "short error", func(fp string) bool { return fp == "short error" }},
		{"truncated to 200", strings.Repeat("x", 300), func(fp string) bool { return len(fp) == 200 }},
		{"trim whitespace", "  error msg  \n", func(fp string) bool { return fp == "error msg" }},
		{"compact spaces", "a  b\t\tc\n\nd", func(fp string) bool { return fp == "a b c d" }},
		{"strip ANSI", "\x1b[31mError\x1b[0m: fail", func(fp string) bool { return fp == "Error: fail" }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fp := errorFingerprint(tt.input)
			if !tt.check(fp) {
				t.Errorf("errorFingerprint(%q) = %q", tt.input, fp)
			}
		})
	}
}
