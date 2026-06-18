package tui

import (
	"strings"

	"github.com/susugadx/xelyon-cli/internal/review"
	"github.com/susugadx/xelyon-cli/internal/tui/reviewscreen"
)

// ReviewRunResult は /review 実行結果と、その実行単位の表示用 usage summary を表す。
type ReviewRunResult struct {
	Report review.ReviewReport
	Usage  ReviewRunUsageSummary
}

// ReviewRunUsageSummary は /review 1 回分の token/cost summary を TUI 表示用に整形して保持する。
type ReviewRunUsageSummary struct {
	Tokens string
	Cost   string
}

func (s ReviewRunUsageSummary) inlineText() string {
	parts := make([]string, 0, 2)
	if tokens := strings.TrimSpace(s.Tokens); tokens != "" {
		parts = append(parts, tokens)
	}
	if cost := strings.TrimSpace(s.Cost); cost != "" {
		parts = append(parts, cost)
	}
	return strings.Join(parts, " · ")
}

func (s ReviewRunUsageSummary) statusText() string {
	inline := s.inlineText()
	if inline == "" {
		return ""
	}
	return "Review: " + inline
}

func reviewRunTimelineMessage(result ReviewRunResult) string {
	lines := reviewscreen.PlainLines(result.Report)
	if usage := result.Usage.inlineText(); usage != "" {
		lines = append(lines[:1], append([]string{"Usage: " + usage}, lines[1:]...)...)
	}
	return strings.Join(lines, "\n")
}
