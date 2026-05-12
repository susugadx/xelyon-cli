package ledger

import (
	"regexp"
	"strconv"
	"strings"
)

var bashExitStatusRe = regexp.MustCompile(`(?m)^Error:\s+exit status\s+(-?\d+)`)

func isTestLikeCommand(command string) bool {
	command = strings.TrimSpace(command)
	if command == "" {
		return false
	}
	needles := []string{
		"go test", "go vet",
		"npm test", "npm run test", "pnpm test", "yarn test",
		"pytest", "python -m pytest",
		"cargo test", "cargo clippy",
		"make test", "make check", "make ci-check", "make verify",
		"mvn test", "gradle test",
	}
	for _, needle := range needles {
		if commandContainsTestCommand(command, needle) {
			return true
		}
	}
	return false
}

func commandContainsTestCommand(command, needle string) bool {
	for offset := 0; offset < len(command); {
		idx := strings.Index(command[offset:], needle)
		if idx < 0 {
			return false
		}
		start := offset + idx
		end := start + len(needle)
		if hasCommandBoundaryBefore(command, start) && hasCommandBoundaryAfter(command, end) {
			return true
		}
		offset = start + 1
	}
	return false
}

func hasCommandBoundaryBefore(command string, idx int) bool {
	return idx == 0 || isShellCommandBoundary(command[idx-1])
}

func hasCommandBoundaryAfter(command string, idx int) bool {
	return idx == len(command) || isShellCommandBoundary(command[idx])
}

func isShellCommandBoundary(ch byte) bool {
	return ch <= ' ' || strings.ContainsRune(";&|()", rune(ch))
}

func bashExitCode(output string, isError bool) int {
	if match := bashExitStatusRe.FindStringSubmatch(output); match != nil {
		exitCode, _ := strconv.Atoi(match[1])
		return exitCode
	}
	if isError || strings.HasPrefix(strings.TrimSpace(output), "Error:") || bashOutputWasCancelled(output) {
		return -1
	}
	return 0
}

func bashOutputWasCancelled(output string) bool {
	normalized := strings.ToLower(strings.TrimSpace(output))
	return strings.Contains(normalized, "cancelled by user") ||
		strings.Contains(normalized, "command interrupted") ||
		strings.HasPrefix(normalized, "[cancelled]")
}

func testResultFromObservation(observation TestObservation) (TestResult, bool) {
	command := strings.TrimSpace(observation.Command)
	if command == "" {
		return TestResult{}, false
	}
	status := normalizeTestStatus(observation.Status, observation.ExitCode)
	if status == "" {
		return TestResult{}, false
	}
	return NewTestResultWithExitCode(command, observation.ExitCode, status, observation.Output), true
}

func normalizeTestStatus(status string, exitCode int) string {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "pass", "passed", "ok", "success", "successful":
		return "passed"
	case "fail", "failed", "failure", "error":
		return "failed"
	case "":
		if exitCode == 0 {
			return "passed"
		}
		return "failed"
	default:
		if exitCode == 0 {
			return "passed"
		}
		return "failed"
	}
}
