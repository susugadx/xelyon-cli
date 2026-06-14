package openairesponses

import "testing"

func TestReasoningEffortFromThinkingLevel(t *testing.T) {
	tests := []struct {
		level string
		want  string
	}{
		{level: "low", want: "low"},
		{level: "medium", want: "medium"},
		{level: "high", want: "high"},
		{level: "xhigh", want: "xhigh"},
		{level: "", want: "medium"},
		{level: "unknown", want: "medium"},
	}

	for _, tt := range tests {
		if got := ReasoningEffortFromThinkingLevel(tt.level); got != tt.want {
			t.Fatalf("ReasoningEffortFromThinkingLevel(%q) = %q, want %q", tt.level, got, tt.want)
		}
	}
}
