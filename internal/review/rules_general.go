package review

import (
	"fmt"
	"regexp"
	"strings"
)

// ---- General Rules (always applied) ----

func (a *Analyzer) runGeneralRules(t Target, opt AnalyzerOptions) []Issue {
	var issues []Issue

	// large-diff
	if issue := a.checkLargeDiff(t, opt); issue != nil {
		issues = append(issues, *issue)
	}

	// todo-added
	issues = append(issues, a.checkTodoAdded(t, opt)...)

	// go-export-missing-doc
	issues = append(issues, a.checkGoExportMissingDoc(t, opt)...)

	// sensitive-file
	if issue := a.checkSensitiveFile(t, opt); issue != nil {
		issues = append(issues, *issue)
	}

	return issues
}

func (a *Analyzer) checkLargeDiff(t Target, opt AnalyzerOptions) *Issue {
	added := countAddedLines(t.Diff)
	if added > 500 {
		return &Issue{
			ID:          "large-diff",
			Title:       "Large diff detected",
			Description: fmt.Sprintf("%d lines added - consider splitting into smaller changes", added),
			Severity:    SeverityWarning,
			Path:        t.Path,
			Tags:        []string{"maintainability"},
		}
	}
	return nil
}

func (a *Analyzer) checkTodoAdded(t Target, opt AnalyzerOptions) []Issue {
	var issues []Issue
	pattern := regexp.MustCompile(`(?i)(TODO|FIXME)`)

	for i, line := range extractAddedLines(t.Diff) {
		if pattern.MatchString(line) {
			issues = append(issues, Issue{
				ID:          "todo-added",
				Title:       "TODO/FIXME added",
				Description: "New TODO or FIXME comment detected",
				Severity:    SeverityInfo,
				Path:        t.Path,
				LineStart:   i + 1,
				Snippet:     trimSnippet(line, opt.MaxSnippetLines),
				Tags:        []string{"maintainability"},
			})
		}
	}
	return issues
}

func (a *Analyzer) checkGoExportMissingDoc(t Target, opt AnalyzerOptions) []Issue {
	if !strings.HasSuffix(t.Path, ".go") || strings.HasSuffix(t.Path, "_test.go") {
		return nil
	}

	var issues []Issue
	pattern := regexp.MustCompile(`^(func|type|var|const)\s+([A-Z]\w*)`)
	lines := extractAddedLines(t.Diff)

	for i, line := range lines {
		if pattern.MatchString(line) {
			// Check if previous line is a comment
			hasDoc := false
			if i > 0 {
				prevLine := lines[i-1]
				if strings.HasPrefix(strings.TrimSpace(prevLine), "//") {
					hasDoc = true
				}
			}
			if !hasDoc {
				match := pattern.FindStringSubmatch(line)
				if len(match) >= 3 {
					issues = append(issues, Issue{
						ID:          "go-export-missing-doc",
						Title:       fmt.Sprintf("Exported %s %s missing documentation", match[1], match[2]),
						Description: "Exported symbols should have documentation comments",
						Severity:    SeverityInfo,
						Path:        t.Path,
						LineStart:   i + 1,
						Snippet:     trimSnippet(line, opt.MaxSnippetLines),
						Suggestions: []string{
							fmt.Sprintf("Add a comment: // %s ...", match[2]),
						},
						Tags: []string{"documentation"},
					})
				}
			}
		}
	}
	return issues
}

func (a *Analyzer) checkSensitiveFile(t Target, opt AnalyzerOptions) *Issue {
	sensitivePatterns := []string{
		".env", "credentials", "secret", "password", "apikey", "api_key",
		"private_key", "id_rsa", ".pem", ".key",
	}

	lowerPath := strings.ToLower(t.Path)
	for _, pattern := range sensitivePatterns {
		if strings.Contains(lowerPath, pattern) {
			return &Issue{
				ID:          "sensitive-file",
				Title:       "Sensitive file modified",
				Description: fmt.Sprintf("File %s may contain sensitive information", t.Path),
				Severity:    SeverityWarning,
				Path:        t.Path,
				Suggestions: []string{
					"Ensure secrets are not committed",
					"Consider using environment variables",
					"Check .gitignore configuration",
				},
				Tags: []string{"security"},
			}
		}
	}
	return nil
}
