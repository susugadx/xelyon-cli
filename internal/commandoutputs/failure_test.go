package commandoutputs

import (
	"strings"
	"testing"
)

func TestBuildReplacementNonValidationExecutionSignalStillFails(t *testing.T) {
	tests := []struct {
		name       string
		command    string
		output     string
		classifier string
	}{
		{
			name:       "observation nonzero",
			command:    `rg "missing" internal`,
			output:     strings.Repeat("Error: exit status 2\nOutput: no matches\n", 120),
			classifier: "unknown_failure",
		},
		{
			name:       "git fatal",
			command:    "git status",
			output:     strings.Repeat("fatal: not a git repository\n", 120),
			classifier: "git_failure",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			replacement, reason, ok := BuildReplacement(NewRequest(tt.command, tt.output))

			if !ok || reason != "" {
				t.Fatalf("BuildReplacement() = (%#v, %q, %v), want failure compact", replacement, reason, ok)
			}
			if replacement.Classifier() != tt.classifier {
				t.Fatalf("Classifier = %q, want %q", replacement.Classifier(), tt.classifier)
			}
			if !strings.Contains(replacement.Text(), "[compacted old failed command output") {
				t.Fatalf("failure compact missing failed header:\n%s", replacement.Text())
			}
		})
	}
}

func TestBuildReplacementSideEffectTextualFailureStillFails(t *testing.T) {
	tests := []struct {
		name       string
		command    string
		output     string
		classifier string
	}{
		{
			name:       "package manager textual failure",
			command:    "npm install",
			output:     strings.Repeat("error: dependency resolution failed\ncommand failed: npm install\n", 120),
			classifier: "package_failure",
		},
		{
			name:       "deploy textual failure",
			command:    "vercel deploy",
			output:     strings.Repeat("deployment failed: permission denied\n", 120),
			classifier: "deploy_failure",
		},
		{
			name:       "github release upload textual failure",
			command:    "gh release upload v1.0.0 dist/app.tar.gz",
			output:     strings.Repeat("deployment failed: permission denied\n", 120),
			classifier: "deploy_failure",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			replacement, reason, ok := BuildReplacement(NewRequest(tt.command, tt.output))

			if !ok || reason != "" {
				t.Fatalf("BuildReplacement() = (%#v, %q, %v), want failure compact", replacement, reason, ok)
			}
			if replacement.Classifier() != tt.classifier {
				t.Fatalf("Classifier = %q, want %q", replacement.Classifier(), tt.classifier)
			}
			if strings.Contains(replacement.Text(), "successful side-effect command output") ||
				strings.Contains(replacement.Text(), "successful sensitive command output") {
				t.Fatalf("textual failure was mislabeled as success:\n%s", replacement.Text())
			}
		})
	}
}

func TestBuildReplacementSensitiveFailureKeepsRawOutputOutsideArtifactOptimization(t *testing.T) {
	output := strings.Repeat("permission denied reading environment\n", 120)

	replacement, reason, ok := BuildReplacement(NewRequest("env", output))

	if ok || reason != sensitiveOutputKeepReason {
		t.Fatalf("BuildReplacement() = (%#v, %q, %v), want sensitive raw keep reason", replacement, reason, ok)
	}
}

func TestBuildReplacementValidationSuccessWithExpectedExceptionLogs(t *testing.T) {
	output := strings.Join([]string{
		"=== RUN   TestExpectedExceptionPath",
		"expected exception: sentinel value",
		"Traceback (most recent call last):",
		"ValueError: expected sentinel",
		"panic: expected sentinel recovered by test",
		"ok\tgithub.com/acme/project\t0.001s",
		"Process exited with code 0",
		strings.Repeat("verbose passing test detail\n", 90),
	}, "\n")

	for _, command := range []string{"go test ./...", "make ci-check"} {
		t.Run(command, func(t *testing.T) {
			replacement, reason, ok := BuildReplacement(NewRequest(command, output))

			if !ok || reason != "" {
				t.Fatalf("BuildReplacement() = (%#v, %q, %v), want validation success", replacement, reason, ok)
			}
			if replacement.Reason() != "validation_success" || replacement.Classifier() != "validation" {
				t.Fatalf("replacement reason/classifier = %q/%q, want validation_success/validation", replacement.Reason(), replacement.Classifier())
			}
			for _, reject := range []string{"validation_failure", "lint_failure", "build_failure"} {
				if strings.Contains(replacement.Text(), reject) {
					t.Fatalf("validation success was mislabeled with %q:\n%s", reject, replacement.Text())
				}
			}
		})
	}
}

func TestBuildReplacementValidationSuccessWithWeakFailureWordsInTestName(t *testing.T) {
	output := strings.Join([]string{
		"=== RUN   TestHandlesFailingInput",
		"--- PASS: TestHandlesFailingInput (0.00s)",
		"ok\tgithub.com/acme/project\t0.001s",
		"Process exited with code 0",
		strings.Repeat("verbose passing test detail\n", 90),
	}, "\n")

	replacement, reason, ok := BuildReplacement(NewRequest("go test ./...", output))

	if !ok || reason != "" {
		t.Fatalf("BuildReplacement() = (%#v, %q, %v), want validation success", replacement, reason, ok)
	}
	if replacement.Reason() != "validation_success" || replacement.Classifier() != "validation" {
		t.Fatalf("replacement reason/classifier = %q/%q, want validation_success/validation", replacement.Reason(), replacement.Classifier())
	}
	for _, reject := range []string{"validation_failure", "lint_failure", "build_failure"} {
		if strings.Contains(replacement.Text(), reject) {
			t.Fatalf("validation success was mislabeled with %q:\n%s", reject, replacement.Text())
		}
	}
}

func TestBuildReplacementDirectLintErrorWithZeroExitStillFailure(t *testing.T) {
	output := strings.Join([]string{
		"error: unexpected console statement",
		"Process exited with code 0",
		strings.Repeat("lint detail\n", 90),
	}, "\n")

	replacement, reason, ok := BuildReplacement(NewRequest("npm run lint", output))

	if !ok || reason != "" {
		t.Fatalf("BuildReplacement() = (%#v, %q, %v), want lint failure compact", replacement, reason, ok)
	}
	if replacement.Classifier() != "lint_failure" {
		t.Fatalf("Classifier = %q, want lint_failure", replacement.Classifier())
	}
}

func TestBuildReplacementValidationFailureWithTracebackAndNonzeroExit(t *testing.T) {
	output := strings.Join([]string{
		"Traceback (most recent call last):",
		"ValueError: broken input",
		"Process exited with code 1",
		strings.Repeat("failing detail\n", 90),
	}, "\n")

	replacement, reason, ok := BuildReplacement(NewRequest("pytest", output))

	if !ok || reason != "" {
		t.Fatalf("BuildReplacement() = (%#v, %q, %v), want validation failure compact", replacement, reason, ok)
	}
	if replacement.Classifier() != "validation_failure" {
		t.Fatalf("Classifier = %q, want validation_failure", replacement.Classifier())
	}
	if !strings.Contains(replacement.Text(), "Traceback") || !strings.Contains(replacement.Text(), "exit=1") {
		t.Fatalf("failure compact missing traceback/nonzero evidence:\n%s", replacement.Text())
	}
}

func TestBuildReplacementFailureCompactRedactsSecrets(t *testing.T) {
	output := strings.Repeat("Error: exit status 1\nAuthorization: Bearer abcdef\nhttps://example.com/path?token=secret\nmain.go:12: undefined: value\n", 120)

	replacement, reason, ok := BuildReplacement(NewRequest("go test ./...", output))

	if !ok || reason != "" {
		t.Fatalf("BuildReplacement() = (%#v, %q, %v), want failure compact", replacement, reason, ok)
	}
	text := replacement.Text()
	for _, want := range []string{"classifier=validation_failure", "key error lines:", "main.go:12"} {
		if !strings.Contains(text, want) {
			t.Fatalf("failure compact missing %q:\n%s", want, text)
		}
	}
	for _, reject := range []string{"Bearer abcdef", "token=secret"} {
		if strings.Contains(text, reject) {
			t.Fatalf("failure compact leaked secret %q:\n%s", reject, text)
		}
	}
}
