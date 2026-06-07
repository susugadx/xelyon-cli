package commandoutputs

import (
	"strconv"
	"strings"
)

func classifyFailure(command, content string, family commandFamily) string {
	lower := strings.ToLower(content)
	switch family {
	case commandFamilyValidation:
		return classifyValidationFailure(command, content, lower)
	case commandFamilyGitStatus, commandFamilyGitDiff, commandFamilyGitShow, commandFamilyGitLog, commandFamilyGitBranch, commandFamilyGitFileList:
		if !looksLikeGitExecutionFailureOutput(content, lower, family) {
			return ""
		}
		return "git_failure"
	case commandFamilySensitive:
		if !looksLikeSideEffectFailureOutput(content, lower) {
			return ""
		}
		return "sensitive_failure"
	case commandFamilyPackage:
		if !looksLikeSideEffectFailureOutput(content, lower) {
			return ""
		}
		return "package_failure"
	case commandFamilyNetwork:
		if !looksLikeExecutionFailureOutput(content, lower) {
			return ""
		}
		return "network_failure"
	case commandFamilyDeploy:
		if !looksLikeSideEffectFailureOutput(content, lower) {
			return ""
		}
		return "deploy_failure"
	case commandFamilyDatabase:
		if !looksLikeExecutionFailureOutput(content, lower) {
			return ""
		}
		return "database_failure"
	default:
		if !looksLikeExecutionFailureOutput(content, lower) {
			return ""
		}
		if containsPermissionFailure(lower) {
			return "permission_failure"
		}
		if containsTimeoutFailure(lower) {
			return "timeout_failure"
		}
		return "unknown_failure"
	}
}

func looksLikeExecutionFailureOutput(content, lower string) bool {
	trimmed := strings.TrimSpace(content)
	return strings.HasPrefix(trimmed, "Error:") ||
		containsNonzeroExitCode(lower) ||
		looksLikeIncomplete(lower)
}

func looksLikeSideEffectFailureOutput(content, lower string) bool {
	return looksLikeExecutionFailureOutput(content, lower) ||
		containsSideEffectFailureText(content, lower)
}

func containsSideEffectFailureText(content, lower string) bool {
	if containsPermissionFailure(lower) || containsTimeoutFailure(lower) {
		return true
	}
	for _, marker := range []string{
		"command failed",
		"command failure",
		"non-zero",
		"nonzero",
		"failed with exit",
		"exited with status",
		"returned exit code",
		"returned non-zero",
		"npm err!",
		"install failed",
		"installation failed",
		"deploy failed",
		"deployment failed",
		"publish failed",
	} {
		if strings.Contains(lower, marker) {
			return true
		}
	}
	for _, line := range outputLines(content) {
		line = strings.ToLower(strings.TrimSpace(line))
		if strings.HasPrefix(line, "error:") || strings.HasPrefix(line, "err:") || strings.HasPrefix(line, "npm err!") {
			return true
		}
	}
	return false
}

func looksLikeGitExecutionFailureOutput(content, lower string, family commandFamily) bool {
	trimmed := strings.TrimSpace(content)
	if (family == commandFamilyGitDiff || family == commandFamilyGitShow) &&
		isParseableGitDiff(content) &&
		!strings.HasPrefix(trimmed, "Error:") {
		return false
	}
	return looksLikeExecutionFailureOutput(content, lower) ||
		containsGitExecutionFailure(content)
}

func looksLikeSuccessfulOutput(content string) bool {
	trimmed := strings.TrimSpace(content)
	lower := strings.ToLower(trimmed)
	if strings.HasPrefix(trimmed, "Error:") || containsNonzeroExitCode(lower) || looksLikeIncomplete(lower) {
		return false
	}
	return true
}

func containsNonzeroExitCode(lower string) bool {
	matches := exitCodePattern.FindAllStringSubmatch(lower, -1)
	for _, match := range matches {
		if len(match) < 2 {
			continue
		}
		code, err := strconv.Atoi(match[1])
		if err == nil && code != 0 {
			return true
		}
	}
	return false
}

func exitCodeSummary(content string) string {
	lower := strings.ToLower(content)
	matches := exitCodePattern.FindAllStringSubmatch(lower, -1)
	for _, match := range matches {
		if len(match) >= 2 {
			return match[1]
		}
	}
	return "unknown"
}

func looksLikeIncomplete(lower string) bool {
	return strings.Contains(lower, "command interrupted") ||
		strings.Contains(lower, "partial output") ||
		strings.Contains(lower, "context canceled") ||
		strings.Contains(lower, "context cancelled") ||
		strings.Contains(lower, "operation canceled") ||
		strings.Contains(lower, "operation cancelled") ||
		strings.Contains(lower, "signal: interrupt") ||
		strings.Contains(lower, "signal: killed")
}

func containsFatalGitError(lower string) bool {
	return strings.Contains(lower, "fatal:") ||
		strings.Contains(lower, "not a git repository") ||
		strings.Contains(lower, "ambiguous argument") ||
		strings.Contains(lower, "unknown revision") ||
		strings.Contains(lower, "pathspec") && strings.Contains(lower, "did not match")
}

func containsGitExecutionFailure(content string) bool {
	for _, line := range outputLines(content) {
		lower := strings.ToLower(strings.TrimSpace(line))
		if strings.HasPrefix(lower, "fatal:") {
			return true
		}
		if strings.HasPrefix(lower, "error:") && containsFatalGitError(lower) {
			return true
		}
	}
	return false
}

func containsPermissionFailure(lower string) bool {
	return strings.Contains(lower, "permission denied") ||
		strings.Contains(lower, "access denied") ||
		strings.Contains(lower, "unsafe path") ||
		strings.Contains(lower, "invalid path") ||
		strings.Contains(lower, "outside repo") ||
		strings.Contains(lower, "not allowed") ||
		strings.Contains(lower, "blocked by policy") ||
		strings.Contains(lower, "approval required")
}

func containsTimeoutFailure(lower string) bool {
	return strings.Contains(lower, "timed out") ||
		strings.Contains(lower, "timeout") ||
		strings.Contains(lower, "killed") ||
		strings.Contains(lower, "cancelled") ||
		strings.Contains(lower, "canceled") ||
		strings.Contains(lower, "interrupted") ||
		strings.Contains(lower, "context deadline exceeded")
}

func errorFocusedLines(lines []string, limit int) []string {
	if limit <= 0 {
		return nil
	}
	out := make([]string, 0, minInt(limit, len(lines)))
	seen := map[string]struct{}{}
	for _, line := range lines {
		if !looksLikeErrorLine(line) {
			continue
		}
		clean := sanitizeHeaderValue(line)
		if clean == "" {
			continue
		}
		if _, ok := seen[clean]; ok {
			continue
		}
		seen[clean] = struct{}{}
		out = append(out, clean)
		if len(out) >= limit {
			break
		}
	}
	return out
}

func looksLikeErrorLine(line string) bool {
	lower := strings.ToLower(line)
	return strings.Contains(line, "Error:") ||
		strings.Contains(lower, "error:") ||
		strings.Contains(line, "ERROR") ||
		strings.Contains(lower, "fatal:") ||
		strings.Contains(lower, "panic:") ||
		strings.Contains(line, "FAIL") ||
		strings.Contains(lower, "failed") ||
		strings.Contains(line, "Exception") ||
		strings.Contains(line, "Traceback") ||
		strings.Contains(line, "AssertionError") ||
		strings.Contains(lower, "undefined") ||
		strings.Contains(lower, "cannot find") ||
		strings.Contains(lower, "not found") ||
		strings.Contains(lower, "permission denied") ||
		strings.Contains(lower, "access denied") ||
		strings.Contains(lower, "timed out") ||
		strings.Contains(lower, "timeout") ||
		strings.Contains(lower, "killed") ||
		strings.Contains(lower, "cancelled") ||
		strings.Contains(lower, "canceled") ||
		strings.Contains(lower, "interrupted") ||
		strings.Contains(lower, "exit status") ||
		strings.Contains(lower, "exit code") ||
		locationLinePattern.MatchString(line)
}
