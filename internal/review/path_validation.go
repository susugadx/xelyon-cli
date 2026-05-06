package review

import (
	"path"
	"path/filepath"
	"strings"
)

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
