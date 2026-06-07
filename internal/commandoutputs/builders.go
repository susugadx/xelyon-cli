package commandoutputs

import (
	"fmt"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/token"
)

func buildValidationSuccessPlaceholder(command, content string) (Replacement, string, bool) {
	lower := strings.ToLower(content)
	if !validationOutputHasSuccessEvidenceForCommand(command, content, lower) {
		return Replacement{}, "validation_success_without_evidence", false
	}
	text := fmt.Sprintf(
		"[omitted old successful validation command output; command=\"%s\"; exit=0; classifier=validation%s]",
		sanitizeCommandForHeader(command),
		validationSummarySuffix(content),
	)
	return replacement("omit_successful_validation_command_output", "validation_success", "validation", content, text), "", true
}

func buildSafePlaceholder(command, content, kind, reason, classifier, label string) (Replacement, string, bool) {
	text := fmt.Sprintf(
		"[omitted old successful %s command output; command_family=%s; exit=0]",
		label,
		classifier,
	)
	return replacement(kind, reason, classifier, content, text), "", true
}

func buildFailureCompact(command, content string, family commandFamily, classifier string) (Replacement, bool) {
	lines := outputLines(content)
	if len(lines) < largeGenericLineThreshold && len(content) < largeGenericByteThreshold {
		return Replacement{}, false
	}
	firstLimit, lastLimit, keyLimit := failureCaps(family, classifier)
	keyLines := errorFocusedLines(lines, keyLimit)
	if keyLimit == 0 {
		keyLines = nil
	}
	parts := []string{fmt.Sprintf(
		"[compacted old failed command output; command=\"%s\"; classifier=%s; exit=%s; lines=%d; bytes=%d]",
		sanitizeCommandForHeader(command),
		classifier,
		exitCodeSummary(content),
		len(lines),
		len(content),
	)}
	parts = append(parts, "summary:")
	parts = append(parts, "  category: "+classifier)
	parts = append(parts, fmt.Sprintf("  matched_error_lines: %d", len(keyLines)))
	if len(keyLines) > 0 {
		parts = append(parts, "key error lines:")
		parts = append(parts, indentLines(redactLines(keyLines))...)
	}
	if firstLimit > 0 {
		first := redactLines(uniqueLines(lines[:minInt(len(lines), firstLimit)], keyLines))
		if len(first) > 0 {
			parts = append(parts, "first lines:")
			parts = append(parts, indentLines(first)...)
		}
	}
	if lastLimit > 0 {
		start := maxInt(0, len(lines)-lastLimit)
		last := redactLines(uniqueLines(lines[start:], append([]string{}, keyLines...)))
		if len(last) > 0 {
			parts = append(parts, "last lines:")
			parts = append(parts, indentLines(last)...)
		}
	}
	parts = append(parts, fmt.Sprintf("[omitted %d lines]", omittedFailureLineCount(len(lines), len(parts))))
	text := strings.Join(parts, "\n")
	return replacement("compact_"+classifier+"_command_output", classifier, classifier, content, text), true
}

func failureCaps(family commandFamily, classifier string) (int, int, int) {
	switch family {
	case commandFamilySensitive:
		return 0, 0, sensitiveFailureKeyErrorLineLimit
	case commandFamilyPackage, commandFamilyNetwork, commandFamilyDeploy:
		return strictFailureFirstLineLimit, strictFailureLastLineLimit, strictFailureKeyErrorLineLimit
	case commandFamilyDatabase:
		return 0, strictFailureLastLineLimit, strictFailureKeyErrorLineLimit
	default:
		if classifier == "permission_failure" || classifier == "timeout_failure" {
			return strictFailureFirstLineLimit, strictFailureLastLineLimit, strictFailureKeyErrorLineLimit
		}
		return failureFirstLineLimit, failureLastLineLimit, failureKeyErrorLineLimit
	}
}

func replacement(kind, reason, classifier, original, text string) Replacement {
	return Replacement{
		kind:        kind,
		reason:      reason,
		classifier:  classifier,
		text:        text,
		savedBytes:  savedBytes(len(original), len(text)),
		savedTokens: savedTokens(token.EstimateTokenCount(original), token.EstimateTokenCount(text)),
	}
}
