package agent

import (
	"testing"
)

func TestContainsFailure(t *testing.T) {
	tests := []struct {
		name           string
		result         string
		expectFailure  bool
		expectReason   string
	}{
		{
			name:           "Go test failure with FAIL prefix",
			result:         "--- FAIL: TestSomething (0.01s)",
			expectFailure:  true,
			expectReason:   "Go test failed",
		},
		{
			name:           "Go test failure with FAIL tab",
			result:         "FAIL\tgithub.com/example/pkg",
			expectFailure:  true,
			expectReason:   "Go test failed",
		},
		{
			name:           "exit status 1",
			result:         "error: command failed\nexit status 1",
			expectFailure:  true,
			expectReason:   "Command failed with exit code 1",
		},
		{
			name:           "exit status other",
			result:         "exit status 2",
			expectFailure:  true,
			expectReason:   "Command failed",
		},
		{
			name:           "panic detected",
			result:         "panic: runtime error: index out of range",
			expectFailure:  true,
			expectReason:   "Panic detected",
		},
		{
			name:           "fatal error",
			result:         "fatal error: all goroutines are asleep",
			expectFailure:  true,
			expectReason:   "Fatal error",
		},
		{
			name:           "assertion error",
			result:         "AssertionError: expected true but got false",
			expectFailure:  true,
			expectReason:   "Assertion failed",
		},
		{
			name:           "FAILED keyword",
			result:         "Test FAILED",
			expectFailure:  true,
			expectReason:   "Test failed",
		},
		{
			name:           "npm error",
			result:         "npm ERR! code ENOENT",
			expectFailure:  true,
			expectReason:   "npm error",
		},
		{
			name:           "SyntaxError",
			result:         "SyntaxError: Unexpected token",
			expectFailure:  true,
			expectReason:   "Syntax error",
		},
		{
			name:           "TypeError",
			result:         "TypeError: Cannot read property 'x' of undefined",
			expectFailure:  true,
			expectReason:   "Type error",
		},
		{
			name:           "ReferenceError",
			result:         "ReferenceError: x is not defined",
			expectFailure:  true,
			expectReason:   "Reference error",
		},
		{
			name:           "compile error",
			result:         "compile error: undefined variable",
			expectFailure:  true,
			expectReason:   "Compilation error",
		},
		{
			name:           "build failed",
			result:         "build failed: missing dependency",
			expectFailure:  true,
			expectReason:   "Build failed",
		},
		{
			name:           "error with lowercase",
			result:         "error: something went wrong",
			expectFailure:  true,
			expectReason:   "Error occurred",
		},
		{
			name:           "Error with uppercase",
			result:         "Error: Something went wrong",
			expectFailure:  true,
			expectReason:   "Error occurred",
		},
		{
			name:           "successful output",
			result:         "ok  \tgithub.com/example/pkg\t0.123s",
			expectFailure:  false,
			expectReason:   "",
		},
		{
			name:           "empty output",
			result:         "",
			expectFailure:  false,
			expectReason:   "",
		},
		{
			name:           "normal log output",
			result:         "INFO: Starting process\nDEBUG: Processing item 1\nINFO: Done",
			expectFailure:  false,
			expectReason:   "",
		},
		{
			name:           "warning but not error",
			result:         "warning: deprecated function used",
			expectFailure:  false,
			expectReason:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotFailure, gotReason := containsFailure(tt.result)
			if gotFailure != tt.expectFailure {
				t.Errorf("containsFailure() failure = %v, want %v", gotFailure, tt.expectFailure)
			}
			if gotReason != tt.expectReason {
				t.Errorf("containsFailure() reason = %q, want %q", gotReason, tt.expectReason)
			}
		})
	}
}

func TestContainsFailure_Priority(t *testing.T) {
	// Test that more specific patterns are matched first
	tests := []struct {
		name         string
		result       string
		expectReason string
	}{
		{
			name:         "FAIL prefix takes priority over generic error",
			result:       "--- FAIL: TestX\nerror: test failed",
			expectReason: "Go test failed",
		},
		{
			name:         "panic takes priority over exit status",
			result:       "panic: nil pointer\nexit status 2",
			expectReason: "Panic detected",
		},
		{
			name:         "build failed takes priority over generic error",
			result:       "build failed\nerror: missing module",
			expectReason: "Build failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, gotReason := containsFailure(tt.result)
			if gotReason != tt.expectReason {
				t.Errorf("containsFailure() reason = %q, want %q", gotReason, tt.expectReason)
			}
		})
	}
}

func TestFailureAction_Constants(t *testing.T) {
	// Test that FailureAction constants have expected values
	if FailureActionRetry != "retry" {
		t.Errorf("FailureActionRetry = %q, want %q", FailureActionRetry, "retry")
	}
	if FailureActionSkip != "skip" {
		t.Errorf("FailureActionSkip = %q, want %q", FailureActionSkip, "skip")
	}
	if FailureActionAbort != "abort" {
		t.Errorf("FailureActionAbort = %q, want %q", FailureActionAbort, "abort")
	}
}
