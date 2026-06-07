package commandoutputs

import (
	"fmt"
	"strings"
)

func classifyValidationFailure(command, content, lower string) string {
	if validationOutputHasSuccessEvidenceForCommand(command, content, lower) {
		return ""
	}
	if !looksLikeValidationFailureOutput(command, content, lower) {
		return ""
	}
	if commandLooksLikeTest(command) || outputHasTestFailure(content, lower) {
		return "validation_failure"
	}
	if outputHasStrongLintFailureForCommand(command, lower) || outputHasLintFailure(lower) {
		return "lint_failure"
	}
	return "build_failure"
}

func looksLikeValidationFailureOutput(command, content, lower string) bool {
	return looksLikeExecutionFailureOutput(content, lower) ||
		outputHasTestFailure(content, lower) ||
		outputHasBuildFailure(lower) ||
		outputHasStrongLintFailureForCommand(command, lower) ||
		outputHasLintFailure(lower)
}

func validationOutputHasSuccessEvidence(command, content, lower string) bool {
	if containsNonzeroExitCode(lower) ||
		outputHasStrongTestFailure(content, lower) ||
		outputHasBuildFailure(lower) ||
		outputHasStrongLintFailureForCommand(command, lower) ||
		looksLikeIncomplete(lower) {
		return false
	}
	return validationOutputHasSuccessMarker(content, lower)
}

func validationOutputHasSuccessEvidenceForCommand(command, content, lower string) bool {
	if validationCommandHasPassingTestEvidence(command, content, lower) {
		return true
	}
	return validationOutputHasSuccessEvidence(command, content, lower)
}

func validationCommandHasPassingTestEvidence(command, content, lower string) bool {
	if !commandLooksLikeTest(command) {
		return false
	}
	if containsNonzeroExitCode(lower) || outputHasStrongTestFailure(content, lower) || looksLikeIncomplete(lower) {
		return false
	}
	return validationOutputHasSuccessMarker(content, lower)
}

func validationOutputHasSuccessMarker(content, lower string) bool {
	return goTestSuccessPattern.MatchString(content) ||
		strings.Contains(lower, "test result: ok") ||
		strings.Contains(lower, "tests passed") ||
		strings.Contains(lower, "test passed") ||
		strings.Contains(lower, " passed in ") ||
		strings.Contains(lower, " passing") ||
		strings.Contains(lower, "build succeeded") ||
		strings.Contains(lower, "built successfully") ||
		strings.Contains(lower, "compiled successfully") ||
		strings.Contains(lower, "build completed successfully") ||
		strings.Contains(lower, "lint clean") ||
		strings.Contains(lower, "lint passed") ||
		strings.Contains(lower, "no lint errors") ||
		strings.Contains(lower, "0 errors") ||
		strings.Contains(lower, "0 problems") ||
		strings.Contains(lower, "process exited with code 0")
}

func validationSummarySuffix(content string) string {
	for _, line := range outputLines(content) {
		lower := strings.ToLower(line)
		if goTestSuccessPattern.MatchString(line) ||
			strings.Contains(lower, "test result: ok") ||
			strings.Contains(lower, "build succeeded") ||
			strings.Contains(lower, "compiled successfully") ||
			strings.Contains(lower, "lint clean") ||
			strings.Contains(lower, "0 errors") ||
			strings.Contains(lower, "0 problems") ||
			strings.Contains(lower, "process exited with code 0") {
			return fmt.Sprintf("; summary=\"%s\"", sanitizeHeaderValue(line))
		}
	}
	return ""
}

func outputHasTestFailure(content, lower string) bool {
	return outputHasStrongTestFailure(content, lower) || outputHasWeakTestFailureMarker(lower)
}

func outputHasStrongTestFailure(content, lower string) bool {
	return strings.Contains(content, "--- FAIL:") ||
		strings.Contains(content, "FAIL\t") ||
		strings.Contains(lower, "test failed") ||
		strings.Contains(lower, "tests failed") ||
		strings.Contains(lower, "test result: failed") ||
		strings.Contains(lower, "failed tests") ||
		testFailedCountPattern.MatchString(lower) ||
		testFailingCountPattern.MatchString(lower)
}

func outputHasWeakTestFailureMarker(lower string) bool {
	return strings.Contains(lower, "failures:") ||
		strings.Contains(lower, "failing")
}

func outputHasBuildFailure(lower string) bool {
	return strings.Contains(lower, "build failed") ||
		strings.Contains(lower, "with errors") ||
		strings.Contains(lower, "has errors") ||
		strings.Contains(lower, "had errors") ||
		strings.Contains(lower, "compile error") ||
		strings.Contains(lower, "compilation error") ||
		strings.Contains(lower, "undefined:") ||
		strings.Contains(lower, "undeclared") ||
		strings.Contains(lower, "cannot find module")
}

func outputHasLintFailure(lower string) bool {
	return outputHasStrongLintFailure(lower) ||
		strings.Contains(lower, "error:")
}

func outputHasStrongLintFailureForCommand(command, lower string) bool {
	return outputHasStrongLintFailure(lower) ||
		commandLooksLikeLint(command) && strings.Contains(lower, "error:")
}

func outputHasStrongLintFailure(lower string) bool {
	return strings.Contains(lower, "lint failed") ||
		strings.Contains(lower, "typecheck failed") ||
		strings.Contains(lower, "error ts") ||
		strings.Contains(lower, "warnings found") ||
		strings.Contains(lower, "issues found") ||
		strings.Contains(lower, "problems found") ||
		lintNonzeroCountPattern.MatchString(lower) ||
		strings.Contains(lower, "eslint") && strings.Contains(lower, "error") ||
		strings.Contains(lower, "ruff") && strings.Contains(lower, "found")
}

func commandLooksLikeLint(command string) bool {
	words := commandWords(command)
	head := wordBase(wordAt(words, 0))
	second := wordAt(words, 1)
	third := wordAt(words, 2)
	return head == "golangci-lint" ||
		head == "eslint" ||
		head == "ruff" && second == "check" ||
		head == "npx" && (second == "eslint" || second == "ruff") ||
		(head == "npm" || head == "pnpm" || head == "yarn") && (second == "lint" || second == "run" && third == "lint") ||
		head == "make" && second == "lint"
}

func commandLooksLikeTest(command string) bool {
	words := commandWords(command)
	head := wordBase(wordAt(words, 0))
	second := wordAt(words, 1)
	third := wordAt(words, 2)
	return head == "go" && second == "test" ||
		head == "cargo" && second == "test" ||
		head == "pytest" ||
		(head == "npm" || head == "pnpm" || head == "yarn") && (second == "test" || second == "run" && third == "test")
}
