package analysis

import "strings"

func uniqueReviewProbePlanEvidencePaths(paths []string) []string {
	seen := make(map[string]struct{}, len(paths))
	result := make([]string, 0, len(paths))
	for _, path := range paths {
		path = normalizeReviewProbePlanEvidencePath(path)
		if path == "" {
			continue
		}
		if _, exists := seen[path]; exists {
			continue
		}
		seen[path] = struct{}{}
		result = append(result, path)
	}
	return result
}

func normalizeReviewProbePlanEvidencePath(path string) string {
	return strings.TrimSpace(strings.ReplaceAll(path, "\\", "/"))
}

func reviewProbePlanSurfaceTextContainsToken(text, token string) bool {
	text = strings.ToLower(text)
	token = strings.ToLower(strings.TrimSpace(token))
	if token == "" {
		return true
	}
	return reviewProbePlanSurfaceTextContainsDelimitedLiteral(text, token, isReviewProbePlanTokenContinuationByte)
}

func reviewProbePlanSurfaceTextContainsPath(text, path string) bool {
	text = strings.ReplaceAll(text, "\\", "/")
	path = normalizeReviewProbePlanEvidencePath(path)
	if path == "" {
		return false
	}
	return reviewProbePlanSurfaceTextContainsDelimitedLiteral(text, path, isReviewProbePlanPathContinuationByte)
}

func reviewProbePlanSurfaceTextContainsDelimitedLiteral(text, literal string, isContinuation func(byte) bool) bool {
	start := 0
	for {
		index := strings.Index(text[start:], literal)
		if index < 0 {
			return false
		}
		index += start
		beforeOK := index == 0 || !isContinuation(text[index-1])
		afterIndex := index + len(literal)
		afterOK := afterIndex >= len(text) || !isContinuation(text[afterIndex])
		if beforeOK && afterOK {
			return true
		}
		start = afterIndex
	}
}

func isReviewProbePlanTokenContinuationByte(ch byte) bool {
	return ch == '_' || ch == '-' || ch == '/' ||
		(ch >= 'a' && ch <= 'z') ||
		(ch >= 'A' && ch <= 'Z') ||
		(ch >= '0' && ch <= '9')
}

func isReviewProbePlanPathContinuationByte(ch byte) bool {
	return ch == '_' || ch == '-' || ch == '/' ||
		(ch >= 'a' && ch <= 'z') ||
		(ch >= 'A' && ch <= 'Z') ||
		(ch >= '0' && ch <= '9')
}
