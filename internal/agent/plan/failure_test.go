package plan

import (
	"testing"
)

func TestContainsFailure(t *testing.T) {
	tests := []struct {
		name           string
		result         string
		expectFailure  bool
		expectContains string // Partial match for reason
	}{
		// Go test failures
		{
			name:           "go test FAIL",
			result:         "--- FAIL: TestExample (0.00s)",
			expectFailure:  true,
			expectContains: "Go test failed",
		},
		{
			name:           "go test FAIL tab",
			result:         "FAIL\tgithub.com/example/pkg",
			expectFailure:  true,
			expectContains: "Go test failed",
		},
		// Command failures
		{
			name:           "exit status 1",
			result:         "Error: exit status 1",
			expectFailure:  true,
			expectContains: "exit code 1",
		},
		// Panics
		{
			name:           "panic detected",
			result:         "panic: runtime error: index out of range",
			expectFailure:  true,
			expectContains: "Panic",
		},
		// Fatal errors
		{
			name:           "fatal error",
			result:         "fatal error: concurrent map writes",
			expectFailure:  true,
			expectContains: "Fatal",
		},
		// Build errors
		{
			name:           "compile error",
			result:         "compile error: cannot convert",
			expectFailure:  true,
			expectContains: "Compilation",
		},
		{
			name:           "build failed",
			result:         "build failed: missing dependency",
			expectFailure:  true,
			expectContains: "Build failed",
		},
		{
			name:           "undefined symbol",
			result:         "undefined: myFunction",
			expectFailure:  true,
			expectContains: "Undefined",
		},
		{
			name:           "undeclared identifier",
			result:         "undeclared name: x",
			expectFailure:  true,
			expectContains: "Undeclared",
		},
		{
			name:           "cannot find",
			result:         "cannot find package github.com/missing/pkg",
			expectFailure:  true,
			expectContains: "Build error",
		},
		{
			name:           "does not exist",
			result:         "package does not exist",
			expectFailure:  true,
			expectContains: "not found",
		},
		// npm errors
		{
			name:           "npm error",
			result:         "npm ERR! code E404",
			expectFailure:  true,
			expectContains: "npm",
		},
		// JavaScript errors
		{
			name:           "syntax error",
			result:         "SyntaxError: Unexpected token",
			expectFailure:  true,
			expectContains: "Syntax",
		},
		{
			name:           "type error",
			result:         "TypeError: Cannot read property 'x' of undefined",
			expectFailure:  true,
			expectContains: "Type error",
		},
		{
			name:           "reference error",
			result:         "ReferenceError: x is not defined",
			expectFailure:  true,
			expectContains: "Reference",
		},
		// Python errors
		{
			name:           "python traceback",
			result:         "Traceback (most recent call last):\n  File \"test.py\", line 1",
			expectFailure:  true,
			expectContains: "Python",
		},
		{
			name:           "assertion error",
			result:         "AssertionError: expected True",
			expectFailure:  true,
			expectContains: "Assertion",
		},
		// Rust errors
		{
			name:           "rust compilation error",
			result:         "error[E0308]: mismatched types",
			expectFailure:  true,
			expectContains: "Rust",
		},
		// No failure cases
		{
			name:          "success output",
			result:        "ok  	github.com/example/pkg	0.001s",
			expectFailure: false,
		},
		{
			name:          "empty output",
			result:        "",
			expectFailure: false,
		},
		{
			name:          "normal log output",
			result:        "Processing files... done",
			expectFailure: false,
		},
		// False positive prevention
		{
			name:          "code containing t.Errorf",
			result:        "func TestExample(t *testing.T) {\n\tt.Errorf(\"expected %v\")\n}",
			expectFailure: false,
		},
		{
			name:          "log message with Error",
			result:        "logger.Error(\"something failed\")",
			expectFailure: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			failed, reason := ContainsFailure(tt.result)

			if failed != tt.expectFailure {
				t.Errorf("ContainsFailure() failed = %v, want %v", failed, tt.expectFailure)
			}

			if tt.expectFailure && tt.expectContains != "" {
				if !containsIgnoreCase(reason, tt.expectContains) {
					t.Errorf("ContainsFailure() reason = %q, should contain %q", reason, tt.expectContains)
				}
			}
		})
	}
}

func TestContainsFailure_PriorityOrder(t *testing.T) {
	// Test that more specific patterns are matched first
	result := "panic: runtime error\nexit status 1"

	failed, reason := ContainsFailure(result)

	if !failed {
		t.Error("Expected failure to be detected")
	}

	// "panic:" should be matched before "exit status 1"
	if reason != "Panic detected" {
		t.Errorf("Expected 'Panic detected' (higher priority), got %q", reason)
	}
}

func TestFailureActionConstants(t *testing.T) {
	// Verify constants are defined correctly
	if FailureActionRetry != "retry" {
		t.Errorf("FailureActionRetry = %q, want 'retry'", FailureActionRetry)
	}
	if FailureActionComment != "comment" {
		t.Errorf("FailureActionComment = %q, want 'comment'", FailureActionComment)
	}
	if FailureActionSkip != "skip" {
		t.Errorf("FailureActionSkip = %q, want 'skip'", FailureActionSkip)
	}
	if FailureActionAbort != "abort" {
		t.Errorf("FailureActionAbort = %q, want 'abort'", FailureActionAbort)
	}
}

func containsIgnoreCase(s, substr string) bool {
	sLower := toLower(s)
	substrLower := toLower(substr)
	for i := 0; i <= len(sLower)-len(substrLower); i++ {
		if sLower[i:i+len(substrLower)] == substrLower {
			return true
		}
	}
	return false
}

func toLower(s string) string {
	result := make([]byte, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			result[i] = c + 32
		} else {
			result[i] = c
		}
	}
	return string(result)
}
