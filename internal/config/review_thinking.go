package config

import "strings"

var reviewThinkingLevels = map[string]struct{}{
	"low":    {},
	"medium": {},
	"high":   {},
	"xhigh":  {},
}

// NormalizeReviewThinkingMode は空白や未設定の /review thinking mode を正規化する。
func NormalizeReviewThinkingMode(mode ReviewThinkingMode) ReviewThinkingMode {
	switch ReviewThinkingMode(strings.ToLower(strings.TrimSpace(string(mode)))) {
	case ReviewThinkingModeOff:
		return ReviewThinkingModeOff
	case ReviewThinkingModeOn:
		return ReviewThinkingModeOn
	case ReviewThinkingModeInherit:
		return ReviewThinkingModeInherit
	default:
		if strings.TrimSpace(string(mode)) == "" {
			return ReviewThinkingModeInherit
		}
		return mode
	}
}

// ReviewThinkingModeValues は /review thinking mode の選択肢を返す。
func ReviewThinkingModeValues() []string {
	return []string{
		string(ReviewThinkingModeInherit),
		string(ReviewThinkingModeOff),
		string(ReviewThinkingModeOn),
	}
}

// ReviewThinkingLevelValues は /review thinking level の選択肢を返す。
func ReviewThinkingLevelValues() []string {
	return []string{"", "low", "medium", "high", "xhigh"}
}

// NormalizeReviewThinkingLevel は /review thinking level override を正規化する。
func NormalizeReviewThinkingLevel(level string) string {
	level = strings.ToLower(strings.TrimSpace(level))
	if !IsValidReviewThinkingLevel(level) {
		return ""
	}
	return level
}

// IsValidReviewThinkingLevel は /review thinking level が有効かを返す。
func IsValidReviewThinkingLevel(level string) bool {
	level = strings.ToLower(strings.TrimSpace(level))
	if level == "" {
		return true
	}
	_, ok := reviewThinkingLevels[level]
	return ok
}

// ResolveReviewThinkingConfig は runtime の ThinkingConfig に /review override を反映した設定を返す。
func ResolveReviewThinkingConfig(base ThinkingConfig, review ReviewThinkingConfig) ThinkingConfig {
	effective := base
	switch NormalizeReviewThinkingMode(review.Mode) {
	case ReviewThinkingModeOff:
		effective.Enabled = false
		return effective
	case ReviewThinkingModeOn:
		effective.Enabled = true
	case ReviewThinkingModeInherit:
	default:
		return effective
	}
	if level := NormalizeReviewThinkingLevel(review.Level); level != "" {
		effective.Level = level
	}
	return effective
}
