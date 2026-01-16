package review

import (
	"fmt"
	"regexp"
	"strings"
)

// ---- Test Rules (Focus.Test = true) ----

func (a *Analyzer) runTestRules(t Target, opt AnalyzerOptions) []Issue {
	var issues []Issue

	// test-coverage (only for new .go files)
	if issue := a.checkTestCoverage(t, opt); issue != nil {
		issues = append(issues, *issue)
	}

	// assertion-missing (only for _test.go files)
	if issue := a.checkAssertionMissing(t, opt); issue != nil {
		issues = append(issues, *issue)
	}

	return issues
}

func (a *Analyzer) checkTestCoverage(t Target, opt AnalyzerOptions) *Issue {
	// Only check new Go files (not test files)
	if t.ChangeType != "A" || !strings.HasSuffix(t.Path, ".go") || strings.HasSuffix(t.Path, "_test.go") {
		return nil
	}

	// Check if corresponding test file might exist (heuristic)
	testPath := strings.TrimSuffix(t.Path, ".go") + "_test.go"

	return &Issue{
		ID:          "test-coverage",
		Title:       "New file without tests",
		Description: fmt.Sprintf("Consider adding tests in %s", testPath),
		Severity:    SeverityInfo,
		Path:        t.Path,
		Suggestions: []string{
			fmt.Sprintf("Create %s with unit tests", testPath),
			"Aim for at least 80%% code coverage",
		},
		Tags: []string{"testing"},
	}
}

func (a *Analyzer) checkAssertionMissing(t Target, opt AnalyzerOptions) *Issue {
	if !strings.HasSuffix(t.Path, "_test.go") {
		return nil
	}

	lines := extractAddedLines(t.Diff)
	if len(lines) == 0 {
		return nil
	}

	// Check for function definitions and assertions
	funcPattern := regexp.MustCompile(`func\s+Test\w+\s*\(`)
	assertPatterns := []*regexp.Regexp{
		regexp.MustCompile(`t\.Error`),
		regexp.MustCompile(`t\.Fatal`),
		regexp.MustCompile(`t\.Fail`),
		regexp.MustCompile(`assert\.`),
		regexp.MustCompile(`require\.`),
		regexp.MustCompile(`if\s+.*!=.*\{`),
		regexp.MustCompile(`if\s+.*==.*\{`),
	}

	hasTestFunc := false
	hasAssertion := false

	for _, line := range lines {
		if funcPattern.MatchString(line) {
			hasTestFunc = true
		}
		for _, ap := range assertPatterns {
			if ap.MatchString(line) {
				hasAssertion = true
				break
			}
		}
	}

	if hasTestFunc && !hasAssertion {
		return &Issue{
			ID:          "assertion-missing",
			Title:       "Test function without assertions",
			Description: "Test functions should contain assertions to verify behavior",
			Severity:    SeverityWarning,
			Path:        t.Path,
			Suggestions: []string{
				"Add t.Error, t.Fatal, or assertion library calls",
				"Verify expected vs actual values",
			},
			Tags: []string{"testing"},
		}
	}
	return nil
}
