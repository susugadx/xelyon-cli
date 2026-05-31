package report

import (
	"fmt"
	"path"
	"path/filepath"
	"strings"
	"unicode"
)

type reviewRelativePathValidationPolicy struct {
	pathKind         string
	rejectNullByte   bool
	rejectWhitespace bool
}

func validateReviewCanonicalRelativePath(field, candidate string, policy reviewRelativePathValidationPolicy) error {
	if strings.TrimSpace(candidate) == "" {
		return fmt.Errorf("%s must be non-empty", field)
	}
	if strings.TrimSpace(candidate) != candidate {
		return fmt.Errorf("%s must be canonical %s without leading/trailing whitespace: got %q", field, policy.pathKind, candidate)
	}
	if policy.rejectWhitespace && containsAnyWhitespace(candidate) {
		return fmt.Errorf("%s must not include whitespace: got %q", field, candidate)
	}
	if policy.rejectNullByte && strings.ContainsRune(candidate, '\x00') {
		return fmt.Errorf("%s must not contain null byte", field)
	}
	if isReviewAbsolutePathLike(candidate) {
		return fmt.Errorf("%s must be %s: got absolute path %q", field, policy.pathKind, candidate)
	}
	if strings.Contains(candidate, `\`) {
		return fmt.Errorf("%s must use '/' separators: got %q", field, candidate)
	}
	for _, segment := range strings.Split(candidate, "/") {
		if segment == "." || segment == ".." {
			return fmt.Errorf("%s must not contain %q segment: got %q", field, segment, candidate)
		}
	}

	cleaned := path.Clean(candidate)
	if cleaned != candidate {
		return fmt.Errorf("%s must be canonical %s: got %q (canonical: %q)", field, policy.pathKind, candidate, cleaned)
	}
	return nil
}

func isReviewAbsolutePathLike(candidate string) bool {
	if path.IsAbs(candidate) || filepath.IsAbs(candidate) {
		return true
	}
	if strings.HasPrefix(candidate, `\\`) || strings.HasPrefix(candidate, `\`) {
		return true
	}
	if len(candidate) >= 2 && isReviewASCIIAlpha(candidate[0]) && candidate[1] == ':' {
		return true
	}
	return false
}

func isReviewASCIIAlpha(ch byte) bool {
	return (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z')
}

func containsAnyWhitespace(candidate string) bool {
	for _, r := range candidate {
		if unicode.IsSpace(r) {
			return true
		}
	}
	return false
}
