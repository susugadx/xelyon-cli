package agent

import (
	"testing"
)

func TestContainsFailure(t *testing.T) {
	tests := []struct {
		name       string
		result     string
		wantFailed bool
		wantReason string
	}{
		// Go test failures
		{
			name:       "Go test FAIL line",
			result:     "--- FAIL: TestSomething (0.01s)",
			wantFailed: true,
			wantReason: "Go test failed",
		},
		{
			name:       "Go test FAIL with tab",
			result:     "FAIL\tgithub.com/user/repo/pkg",
			wantFailed: true,
			wantReason: "Go test failed",
		},
		// Panic
		{
			name:       "panic detected",
			result:     "panic: runtime error: invalid memory address",
			wantFailed: true,
			wantReason: "Panic detected",
		},
		// Fatal error
		{
			name:       "fatal error",
			result:     "fatal error: concurrent map writes",
			wantFailed: true,
			wantReason: "Fatal error",
		},
		// Build/compile errors
		{
			name:       "compile error",
			result:     "compile error: syntax error",
			wantFailed: true,
			wantReason: "Compilation error",
		},
		{
			name:       "build failed",
			result:     "build failed: missing dependency",
			wantFailed: true,
			wantReason: "Build failed",
		},
		{
			name:       "undefined symbol",
			result:     "undefined: someFunction",
			wantFailed: true,
			wantReason: "Undefined symbol",
		},
		{
			name:       "undeclared identifier",
			result:     "undeclared name: x",
			wantFailed: true,
			wantReason: "Undeclared identifier",
		},
		{
			name:       "cannot find module",
			result:     "cannot find module for path",
			wantFailed: true,
			wantReason: "Build error",
		},
		{
			name:       "does not exist",
			result:     "file does not exist: /path/to/file",
			wantFailed: true,
			wantReason: "File or module not found",
		},
		// Exit status
		{
			name:       "exit status 1",
			result:     "command failed with exit status 1",
			wantFailed: true,
			wantReason: "Command failed with exit code 1",
		},
		// npm errors
		{
			name:       "npm error",
			result:     "npm ERR! code ENOENT",
			wantFailed: true,
			wantReason: "npm error",
		},
		// JavaScript errors
		{
			name:       "SyntaxError",
			result:     "SyntaxError: Unexpected token",
			wantFailed: true,
			wantReason: "Syntax error",
		},
		{
			name:       "TypeError",
			result:     "TypeError: Cannot read property of undefined",
			wantFailed: true,
			wantReason: "Type error",
		},
		{
			name:       "ReferenceError",
			result:     "ReferenceError: x is not defined",
			wantFailed: true,
			wantReason: "Reference error",
		},
		// Python errors
		{
			name:       "Python traceback",
			result:     "Traceback (most recent call last):\n  File \"test.py\"",
			wantFailed: true,
			wantReason: "Python exception",
		},
		{
			name:       "AssertionError",
			result:     "AssertionError: values do not match",
			wantFailed: true,
			wantReason: "Assertion failed",
		},
		// Rust errors
		{
			name:       "Rust compilation error",
			result:     "error[E0308]: mismatched types",
			wantFailed: true,
			wantReason: "Rust compilation error",
		},
		// Negative cases - should NOT trigger
		{
			name:       "normal output",
			result:     "all tests passed successfully",
			wantFailed: false,
			wantReason: "",
		},
		{
			name:       "empty string",
			result:     "",
			wantFailed: false,
			wantReason: "",
		},
		{
			name:       "error in code search result",
			result:     "found in test.go:42: t.Errorf(\"expected error\")",
			wantFailed: false,
			wantReason: "",
		},
		{
			name:       "Error word in log output",
			result:     "Error handling was successfully implemented",
			wantFailed: false,
			wantReason: "",
		},
		{
			name:       "successful build",
			result:     "Build completed successfully in 2.3s",
			wantFailed: false,
			wantReason: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotFailed, gotReason := containsFailure(tt.result)
			if gotFailed != tt.wantFailed {
				t.Errorf("containsFailure() failed = %v, want %v", gotFailed, tt.wantFailed)
			}
			if gotReason != tt.wantReason {
				t.Errorf("containsFailure() reason = %q, want %q", gotReason, tt.wantReason)
			}
		})
	}
}
