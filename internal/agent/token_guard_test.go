package agent

import (
	"errors"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/agent/token"
	"github.com/susugadx/xelyon-cli/internal/api"
)

func TestTrimToolOutputsForSend_NonToolMessage(t *testing.T) {
	history := []api.Message{
		{Role: "user", Content: "Hello"},
		{Role: "assistant", Content: "Hi there!"},
		{Role: "user", Content: "How are you?"},
	}

	result := trimToolOutputsForSend(history, 100)

	if len(result) != len(history) {
		t.Errorf("Expected %d messages, got %d", len(history), len(result))
	}

	for i, msg := range result {
		if msg.Content != history[i].Content {
			t.Errorf("Message %d content changed: got %q, want %q", i, msg.Content, history[i].Content)
		}
	}
}

func TestTrimToolOutputsForSend_ShortToolOutput(t *testing.T) {
	shortOutput := "[Tool Result for read_file]\nline1\nline2\nline3"
	history := []api.Message{
		{Role: "user", Content: shortOutput},
	}

	result := trimToolOutputsForSend(history, 100)

	if len(result) != 1 {
		t.Fatalf("Expected 1 message, got %d", len(result))
	}

	// Short output should not be truncated
	if result[0].Content != shortOutput {
		t.Errorf("Short tool output was modified: got %q, want %q", result[0].Content, shortOutput)
	}
}

func TestTrimToolOutputsForSend_LongToolOutput(t *testing.T) {
	// Create a tool output with 100 lines
	var lines []string
	lines = append(lines, "[Tool Result for bash]")
	for i := 1; i <= 100; i++ {
		lines = append(lines, "output line "+string(rune('0'+i%10)))
	}
	longOutput := strings.Join(lines, "\n")

	history := []api.Message{
		{Role: "user", Content: longOutput},
	}

	// Truncate to last 50 lines
	result := trimToolOutputsForSend(history, 50)

	if len(result) != 1 {
		t.Fatalf("Expected 1 message, got %d", len(result))
	}

	// Should contain truncation note
	if !strings.Contains(result[0].Content, "[SYSTEM NOTE] Tool output was truncated") {
		t.Error("Expected truncation note in output")
	}

	// Should have approximately 50 lines (plus the note)
	resultLines := strings.Split(result[0].Content, "\n")
	// 50 lines + note line
	if len(resultLines) < 50 {
		t.Errorf("Expected at least 50 lines, got %d", len(resultLines))
	}
}

func TestTrimToolOutputsForSend_DefaultTailLines(t *testing.T) {
	// Create output longer than default
	var lines []string
	lines = append(lines, "[Tool Result for test]")
	for i := 0; i < 400; i++ {
		lines = append(lines, "line")
	}
	longOutput := strings.Join(lines, "\n")

	history := []api.Message{
		{Role: "user", Content: longOutput},
	}

	// Pass 0 to use default
	result := trimToolOutputsForSend(history, 0)

	// Should be truncated using default (300 lines)
	if !strings.Contains(result[0].Content, "[SYSTEM NOTE] Tool output was truncated to the last 300 lines") {
		t.Error("Expected truncation to default 300 lines")
	}
}

func TestTrimToolOutputsForSend_MinTailLines(t *testing.T) {
	var lines []string
	lines = append(lines, "[Tool Result for test]")
	for i := 0; i < 100; i++ {
		lines = append(lines, "line")
	}
	longOutput := strings.Join(lines, "\n")

	history := []api.Message{
		{Role: "user", Content: longOutput},
	}

	// Pass value less than minimum (50)
	result := trimToolOutputsForSend(history, 10)

	// Should be truncated to minimum (50 lines), not 10
	if !strings.Contains(result[0].Content, "[SYSTEM NOTE] Tool output was truncated to the last 50 lines") {
		t.Error("Expected truncation to minimum 50 lines")
	}
}

func TestTruncateToLastLines_BelowLimit(t *testing.T) {
	tests := []struct {
		name   string
		text   string
		lastN  int
		change bool
	}{
		{
			name:   "fewer lines than limit",
			text:   "line1\nline2\nline3",
			lastN:  10,
			change: false,
		},
		{
			name:   "exactly at limit",
			text:   "line1\nline2\nline3",
			lastN:  3,
			change: false,
		},
		{
			name:   "single line",
			text:   "single line",
			lastN:  5,
			change: false,
		},
		{
			name:   "empty text",
			text:   "",
			lastN:  5,
			change: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, changed := token.TruncateToLastLines(tt.text, tt.lastN)

			if changed != tt.change {
				t.Errorf("token.TruncateToLastLines() changed = %v, want %v", changed, tt.change)
			}

			if result != tt.text {
				t.Errorf("token.TruncateToLastLines() result = %q, want %q", result, tt.text)
			}
		})
	}
}

func TestTruncateToLastLines_AboveLimit(t *testing.T) {
	text := "line1\nline2\nline3\nline4\nline5"

	result, changed := token.TruncateToLastLines(text, 3)

	if !changed {
		t.Error("Expected changed to be true")
	}

	expected := "line3\nline4\nline5"
	if result != expected {
		t.Errorf("token.TruncateToLastLines() = %q, want %q", result, expected)
	}
}

func TestTruncateToLastLines_ZeroLimit(t *testing.T) {
	text := "line1\nline2\nline3"

	result, changed := token.TruncateToLastLines(text, 0)

	if changed {
		t.Error("Expected changed to be false for zero limit")
	}

	if result != text {
		t.Errorf("token.TruncateToLastLines() = %q, want %q", result, text)
	}
}

func TestTruncateToLastLines_NegativeLimit(t *testing.T) {
	text := "line1\nline2\nline3"

	result, changed := token.TruncateToLastLines(text, -5)

	if changed {
		t.Error("Expected changed to be false for negative limit")
	}

	if result != text {
		t.Errorf("token.TruncateToLastLines() = %q, want %q", result, text)
	}
}

func TestIsTokenLimitError_Patterns(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "nil error",
			err:  nil,
			want: false,
		},
		{
			name: "OpenAI token limit",
			err:  errors.New("input tokens exceed the maximum allowed"),
			want: true,
		},
		{
			name: "context length error",
			err:  errors.New("This model's maximum context length is 8192 tokens"),
			want: true,
		},
		{
			name: "maximum context error",
			err:  errors.New("maximum context reached"),
			want: true,
		},
		{
			name: "unrelated error",
			err:  errors.New("network timeout"),
			want: false,
		},
		{
			name: "rate limit error (not token)",
			err:  errors.New("rate limit exceeded"),
			want: false,
		},
		{
			name: "case insensitive - uppercase",
			err:  errors.New("INPUT TOKENS EXCEED the limit"),
			want: true,
		},
		{
			name: "case insensitive - mixed",
			err:  errors.New("Context Length exceeded"),
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := token.IsTokenLimitError(tt.err)
			if got != tt.want {
				t.Errorf("token.IsTokenLimitError() = %v, want %v", got, tt.want)
			}
		})
	}
}
