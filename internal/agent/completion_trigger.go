package agent

import (
	"regexp"
	"strings"
)

var completionNegativePatternsEnglish = []*regexp.Regexp{
	regexp.MustCompile(`(?i)\b(?:not|isn't|is not|wasn't|was not|aren't|are not|weren't|were not)\s+(?:done|completed|finished|complete)\b`),
	regexp.MustCompile(`(?i)\b(?:haven't|have not|hasn't|has not)\s+(?:done|completed|finished)\b`),
	regexp.MustCompile(`(?i)\b(?:have|has)\s+not\s+been\s+(?:done|completed|finished)\b`),
	regexp.MustCompile(`(?i)\b(?:needs?|needed|still needs?)\s+to\s+be\s+done\b`),
}

var completionContinuationPatternsEnglish = []*regexp.Regexp{
	regexp.MustCompile(`(?i)\bnext\s+i(?:'ll|\s+will)\b`),
	regexp.MustCompile(`(?i)\bthen\s+i(?:'ll|\s+will)\b`),
	regexp.MustCompile(`(?i)\bstill\s+(?:need|needs|have|has)\b`),
	regexp.MustCompile(`(?i)\bremaining\s+(?:work|changes?|steps?|tasks?)\b`),
	regexp.MustCompile(`(?i)\bleft\s+to\s+do\b`),
}

var completionPositivePatternsEnglish = []*regexp.Regexp{
	regexp.MustCompile(`(?i)^\s*done\b`),
	regexp.MustCompile(`(?i)^\s*completed(?:\s+successfully)?(?:\s*[.!?])?\s*$`),
	regexp.MustCompile(`(?i)^\s*finished(?:\s*[.!?])?\s*$`),
	regexp.MustCompile(`(?i)\b(?:i am|i'm|we are|we're)\s+(?:done|finished)\b`),
	regexp.MustCompile(`(?i)\b(?:i have|i've|we have|we've)\s+(?:finished|completed)\b`),
	regexp.MustCompile(`(?i)\b(?:all done|all set|that's it)\b`),
	regexp.MustCompile(`(?i)\b(?:task(?: is)? complete(?:d)?|changes are complete|implementation is complete)\b`),
	regexp.MustCompile(`(?i)\b(?:the\s+)?(?:requested\s+changes|changes|implementation|work|task)\s+(?:is|are|was|were|has been|have been)\s+(?:done|completed|finished|complete)\b`),
}

var completionNegativePatternsJapanese = []string{
	"完了していません",
	"完了してません",
	"まだ完了",
	"未完了",
	"実装していません",
	"修正していません",
	"対応していません",
}

var completionContinuationPatternsJapanese = []string{
	"次に",
	"次は",
	"続いて",
	"引き続き",
	"このあと",
	"これから",
	"残り",
	"残って",
	"続けて",
}

var completionPositivePatternsJapanese = []string{
	"完了しました",
	"修正しました",
	"以上です",
	"実装しました",
	"対応しました",
	"修正完了",
	"作業は以上",
	"変更は以上",
}

// isCompletionTriggerResponse は AI 応答を completion finalization に進めてよいかを返す。
// final_checks.commands と no-tool disposition の completion 分岐はこの契約を共有する。
func isCompletionTriggerResponse(response string) bool {
	response = strings.TrimSpace(response)
	if response == "" {
		return false
	}

	lowered := strings.ToLower(response)

	for _, pattern := range completionNegativePatternsEnglish {
		if pattern.MatchString(lowered) {
			return false
		}
	}

	for _, pattern := range completionNegativePatternsJapanese {
		if strings.Contains(response, pattern) {
			return false
		}
	}

	for _, pattern := range completionContinuationPatternsEnglish {
		if pattern.MatchString(lowered) {
			return false
		}
	}

	for _, pattern := range completionContinuationPatternsJapanese {
		if strings.Contains(response, pattern) {
			return false
		}
	}

	for _, pattern := range completionPositivePatternsEnglish {
		if pattern.MatchString(lowered) {
			return true
		}
	}

	for _, pattern := range completionPositivePatternsJapanese {
		if strings.Contains(response, pattern) {
			return true
		}
	}

	return false
}
