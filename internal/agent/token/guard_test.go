package token

import (
	"errors"
	"testing"
)

func TestTruncateToLastLines(t *testing.T) {
	tests := []struct {
		name           string
		text           string
		lastN          int
		expectedText   string
		expectedTrunc  bool
	}{
		{
			name:           "no truncation needed",
			text:           "line1\nline2\nline3",
			lastN:          5,
			expectedText:   "line1\nline2\nline3",
			expectedTrunc:  false,
		},
		{
			name:           "exact match",
			text:           "line1\nline2\nline3",
			lastN:          3,
			expectedText:   "line1\nline2\nline3",
			expectedTrunc:  false,
		},
		{
			name:           "truncate to last 2",
			text:           "line1\nline2\nline3\nline4\nline5",
			lastN:          2,
			expectedText:   "line4\nline5",
			expectedTrunc:  true,
		},
		{
			name:           "truncate to last 1",
			text:           "line1\nline2\nline3",
			lastN:          1,
			expectedText:   "line3",
			expectedTrunc:  true,
		},
		{
			name:           "empty text",
			text:           "",
			lastN:          5,
			expectedText:   "",
			expectedTrunc:  false,
		},
		{
			name:           "single line",
			text:           "single line",
			lastN:          5,
			expectedText:   "single line",
			expectedTrunc:  false,
		},
		{
			name:           "lastN is zero",
			text:           "line1\nline2",
			lastN:          0,
			expectedText:   "line1\nline2",
			expectedTrunc:  false,
		},
		{
			name:           "lastN is negative",
			text:           "line1\nline2",
			lastN:          -1,
			expectedText:   "line1\nline2",
			expectedTrunc:  false,
		},
		{
			name:           "multiline with empty lines",
			text:           "line1\n\nline3\n\nline5",
			lastN:          2,
			expectedText:   "\nline5",
			expectedTrunc:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, truncated := TruncateToLastLines(tt.text, tt.lastN)

			if result != tt.expectedText {
				t.Errorf("TruncateToLastLines() text = %q, want %q", result, tt.expectedText)
			}
			if truncated != tt.expectedTrunc {
				t.Errorf("TruncateToLastLines() truncated = %v, want %v", truncated, tt.expectedTrunc)
			}
		})
	}
}

func TestIsTokenLimitError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
		{
			name:     "generic error",
			err:      errors.New("something went wrong"),
			expected: false,
		},
		{
			name:     "OpenAI input tokens exceed",
			err:      errors.New("input tokens exceed maximum context length"),
			expected: true,
		},
		{
			name:     "context length error",
			err:      errors.New("This model's maximum context length is 4096 tokens"),
			expected: true,
		},
		{
			name:     "maximum context error",
			err:      errors.New("Request exceeds maximum context window"),
			expected: true,
		},
		{
			name:     "case insensitive - INPUT TOKENS",
			err:      errors.New("INPUT TOKENS EXCEED limit"),
			expected: true,
		},
		{
			name:     "case insensitive - Context Length",
			err:      errors.New("Context Length exceeded"),
			expected: true,
		},
		{
			name:     "network error",
			err:      errors.New("connection refused"),
			expected: false,
		},
		{
			name:     "rate limit error",
			err:      errors.New("rate limit exceeded"),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsTokenLimitError(tt.err)
			if result != tt.expected {
				t.Errorf("IsTokenLimitError(%v) = %v, want %v", tt.err, result, tt.expected)
			}
		})
	}
}
