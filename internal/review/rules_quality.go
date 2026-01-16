package review

import (
	"fmt"
	"regexp"
	"strings"
)

// ---- Quality Rules (always applied) ----

func (a *Analyzer) runQualityRules(t Target, opt AnalyzerOptions) []Issue {
	var issues []Issue

	// sql-injection
	issues = append(issues, a.checkSQLInjection(t, opt)...)

	// xss
	issues = append(issues, a.checkXSS(t, opt)...)

	// n-plus-one
	issues = append(issues, a.checkNPlusOne(t, opt)...)

	// magic-number
	issues = append(issues, a.checkMagicNumber(t, opt)...)

	// duplicate-code (simplified heuristic)
	issues = append(issues, a.checkDuplicateCode(t, opt)...)

	// function-size
	issues = append(issues, a.checkFunctionSize(t, opt)...)

	return issues
}

func (a *Analyzer) checkSQLInjection(t Target, opt AnalyzerOptions) []Issue {
	var issues []Issue

	// Check for string concatenation in SQL queries
	patterns := []*regexp.Regexp{
		regexp.MustCompile(`(?i)(SELECT|INSERT|UPDATE|DELETE|FROM|WHERE).*\+.*["']`),
		regexp.MustCompile(`(?i)(SELECT|INSERT|UPDATE|DELETE|FROM|WHERE).*fmt\.Sprintf`),
		regexp.MustCompile(`(?i)db\.(Query|Exec|QueryRow)\s*\([^)]*\+`),
		regexp.MustCompile(`(?i)db\.(Query|Exec|QueryRow)\s*\([^)]*fmt\.Sprintf`),
		regexp.MustCompile(`(?i)(Query|Exec|QueryRow)\s*\(\s*["'].*\$\{`),
	}

	for i, line := range extractAddedLines(t.Diff) {
		for _, pattern := range patterns {
			if pattern.MatchString(line) {
				issues = append(issues, Issue{
					ID:          "sql-injection",
					Title:       "Potential SQL injection",
					Description: "String concatenation in SQL query may allow SQL injection attacks",
					Severity:    SeverityError,
					Path:        t.Path,
					LineStart:   i + 1,
					Snippet:     trimSnippet(line, opt.MaxSnippetLines),
					Suggestions: []string{
						"Use parameterized queries with placeholders ($1, ?, :name)",
						"Use prepared statements",
						"Use an ORM with proper escaping",
					},
					Tags: []string{"security", "sql-injection"},
				})
				break
			}
		}
	}
	return issues
}

func (a *Analyzer) checkXSS(t Target, opt AnalyzerOptions) []Issue {
	var issues []Issue

	// Check for unescaped output in HTML context
	patterns := []*regexp.Regexp{
		regexp.MustCompile(`(?i)innerHTML\s*=`),
		regexp.MustCompile(`(?i)outerHTML\s*=`),
		regexp.MustCompile(`(?i)document\.write\s*\(`),
		regexp.MustCompile(`(?i)\.html\s*\([^)]*\+`),
		regexp.MustCompile(`(?i)template\.HTML\s*\(`),
		regexp.MustCompile(`(?i)dangerouslySetInnerHTML`),
		regexp.MustCompile(`(?i)v-html\s*=`),
	}

	for i, line := range extractAddedLines(t.Diff) {
		for _, pattern := range patterns {
			if pattern.MatchString(line) {
				issues = append(issues, Issue{
					ID:          "xss",
					Title:       "Potential XSS vulnerability",
					Description: "Unescaped HTML output may allow cross-site scripting attacks",
					Severity:    SeverityError,
					Path:        t.Path,
					LineStart:   i + 1,
					Snippet:     trimSnippet(line, opt.MaxSnippetLines),
					Suggestions: []string{
						"Use text content instead of innerHTML",
						"Sanitize user input before rendering",
						"Use template auto-escaping features",
					},
					Tags: []string{"security", "xss"},
				})
				break
			}
		}
	}
	return issues
}

func (a *Analyzer) checkNPlusOne(t Target, opt AnalyzerOptions) []Issue {
	var issues []Issue

	// Check for database queries inside loops
	lines := extractAddedLines(t.Diff)
	inLoop := false
	loopStart := 0

	loopPatterns := []*regexp.Regexp{
		regexp.MustCompile(`^\s*for\s+`),
		regexp.MustCompile(`^\s*for\s*\{`),
		regexp.MustCompile(`\.forEach\s*\(`),
		regexp.MustCompile(`\.map\s*\(`),
		regexp.MustCompile(`\.each\s*\(`),
	}

	queryPatterns := []*regexp.Regexp{
		regexp.MustCompile(`(?i)db\.(Query|Exec|QueryRow|Find|First|Get)\s*\(`),
		regexp.MustCompile(`(?i)\.(Select|Insert|Update|Delete)\s*\(`),
		regexp.MustCompile(`(?i)SELECT\s+.*FROM`),
	}

	for i, line := range lines {
		for _, lp := range loopPatterns {
			if lp.MatchString(line) {
				inLoop = true
				loopStart = i
				break
			}
		}

		if inLoop {
			for _, qp := range queryPatterns {
				if qp.MatchString(line) {
					issues = append(issues, Issue{
						ID:          "n-plus-one",
						Title:       "Potential N+1 query",
						Description: "Database query inside a loop may cause performance issues",
						Severity:    SeverityWarning,
						Path:        t.Path,
						LineStart:   loopStart + 1,
						LineEnd:     i + 1,
						Snippet:     trimSnippet(line, opt.MaxSnippetLines),
						Suggestions: []string{
							"Use eager loading (e.g., Preload, Include, JOIN)",
							"Batch queries before the loop",
							"Use IN clause with collected IDs",
						},
						Tags: []string{"performance", "database"},
					})
					break
				}
			}
		}

		// Simple heuristic: closing brace might end loop
		if strings.TrimSpace(line) == "}" || strings.TrimSpace(line) == "})" {
			inLoop = false
		}
	}
	return issues
}

func (a *Analyzer) checkMagicNumber(t Target, opt AnalyzerOptions) []Issue {
	var issues []Issue

	// Skip test files
	if strings.HasSuffix(t.Path, "_test.go") || strings.Contains(t.Path, "test") {
		return nil
	}

	// Match numeric literals (excluding common safe values: 0, 1, 2, 10, 100, etc.)
	pattern := regexp.MustCompile(`[^a-zA-Z0-9_]([3-9]\d{2,}|\d{4,})[^a-zA-Z0-9_]`)
	safePatterns := []string{
		"1000", "10000", "100000", "1000000",
		"1024", "2048", "4096", "8192", "16384",
		"3600", "86400", // seconds
		"0644", "0755", "0600", "0700", // file permissions
	}

	for i, line := range extractAddedLines(t.Diff) {
		// Skip const declarations
		if strings.Contains(line, "const ") || strings.Contains(line, "=") && strings.Contains(line, "const") {
			continue
		}
		// Skip comments
		if strings.HasPrefix(strings.TrimSpace(line), "//") || strings.HasPrefix(strings.TrimSpace(line), "/*") {
			continue
		}

		matches := pattern.FindAllString(line, -1)
		for _, m := range matches {
			m = strings.TrimSpace(m)
			// Check if it's a safe pattern
			isSafe := false
			for _, sp := range safePatterns {
				if strings.Contains(m, sp) {
					isSafe = true
					break
				}
			}
			if isSafe {
				continue
			}

			issues = append(issues, Issue{
				ID:          "magic-number",
				Title:       "Magic number detected",
				Description: "Numeric literals should be extracted to named constants for clarity",
				Severity:    SeverityInfo,
				Path:        t.Path,
				LineStart:   i + 1,
				Snippet:     trimSnippet(line, opt.MaxSnippetLines),
				Suggestions: []string{
					"Extract to a named constant",
					"Add a comment explaining the value",
				},
				Tags: []string{"maintainability"},
			})
			break // One issue per line
		}
	}
	return issues
}

func (a *Analyzer) checkDuplicateCode(t Target, opt AnalyzerOptions) []Issue {
	var issues []Issue

	lines := extractAddedLines(t.Diff)
	if len(lines) < 5 {
		return nil
	}

	// Simple heuristic: look for repeated patterns of 3+ lines
	seen := make(map[string]int)
	windowSize := 3

	for i := 0; i <= len(lines)-windowSize; i++ {
		// Create a normalized key from window
		window := make([]string, windowSize)
		for j := 0; j < windowSize; j++ {
			window[j] = strings.TrimSpace(lines[i+j])
		}
		key := strings.Join(window, "|")

		// Skip if window contains mostly whitespace or braces
		meaningful := 0
		for _, w := range window {
			if len(w) > 5 && w != "{" && w != "}" && w != "})," {
				meaningful++
			}
		}
		if meaningful < 2 {
			continue
		}

		if firstIdx, ok := seen[key]; ok {
			if i-firstIdx > windowSize { // Not adjacent
				issues = append(issues, Issue{
					ID:          "duplicate-code",
					Title:       "Potential duplicate code",
					Description: fmt.Sprintf("Similar code pattern found at lines %d and %d", firstIdx+1, i+1),
					Severity:    SeverityInfo,
					Path:        t.Path,
					LineStart:   i + 1,
					Snippet:     strings.Join(window, "\n"),
					Suggestions: []string{
						"Extract common code to a shared function",
						"Use a loop or helper method",
					},
					Tags: []string{"maintainability", "dry"},
				})
				// Only report once per pattern
				delete(seen, key)
			}
		} else {
			seen[key] = i
		}
	}
	return issues
}

func (a *Analyzer) checkFunctionSize(t Target, opt AnalyzerOptions) []Issue {
	var issues []Issue

	lines := extractAddedLines(t.Diff)
	if len(lines) < 50 {
		return nil
	}

	// Track function starts and sizes
	funcPattern := regexp.MustCompile(`^\s*func\s+(\w+)?\s*\(`)
	braceCount := 0
	funcStart := -1
	funcName := ""

	for i, line := range lines {
		if match := funcPattern.FindStringSubmatch(line); match != nil {
			if funcStart >= 0 && braceCount == 0 {
				// Previous function ended
				size := i - funcStart
				if size > 50 {
					issues = append(issues, Issue{
						ID:          "function-size",
						Title:       fmt.Sprintf("Large function: %s (%d lines)", funcName, size),
						Description: "Functions over 50 lines are harder to understand and test",
						Severity:    SeverityWarning,
						Path:        t.Path,
						LineStart:   funcStart + 1,
						LineEnd:     i,
						Suggestions: []string{
							"Extract logical sections into smaller helper functions",
							"Consider using early returns to reduce nesting",
							"Apply single responsibility principle",
						},
						Tags: []string{"maintainability", "complexity"},
					})
				}
			}
			funcStart = i
			funcName = match[1]
			if funcName == "" {
				funcName = "(anonymous)"
			}
			braceCount = 0
		}

		braceCount += strings.Count(line, "{") - strings.Count(line, "}")
	}

	// Check last function
	if funcStart >= 0 {
		size := len(lines) - funcStart
		if size > 50 {
			issues = append(issues, Issue{
				ID:          "function-size",
				Title:       fmt.Sprintf("Large function: %s (%d lines)", funcName, size),
				Description: "Functions over 50 lines are harder to understand and test",
				Severity:    SeverityWarning,
				Path:        t.Path,
				LineStart:   funcStart + 1,
				LineEnd:     len(lines),
				Suggestions: []string{
					"Extract logical sections into smaller helper functions",
					"Consider using early returns to reduce nesting",
					"Apply single responsibility principle",
				},
				Tags: []string{"maintainability", "complexity"},
			})
		}
	}

	return issues
}
