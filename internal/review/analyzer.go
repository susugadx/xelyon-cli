package review

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"
)

type AnalyzerOptions struct {
	Focus           ReviewFocus
	MaxIssues       int
	MaxSnippetLines int
}

type Analyzer struct {
	Now func() time.Time
}

func NewAnalyzer() *Analyzer {
	return &Analyzer{Now: time.Now}
}

func (a *Analyzer) Analyze(targets []Target, opt AnalyzerOptions) ([]Issue, error) {
	if opt.MaxSnippetLines <= 0 {
		opt.MaxSnippetLines = 10
	}

	var issues []Issue

	for _, t := range targets {
		// Always apply general rules
		issues = append(issues, a.runGeneralRules(t, opt)...)

		// Apply security rules if Focus.Security is true
		if opt.Focus.Security {
			issues = append(issues, a.runSecurityRules(t, opt)...)
		}

		// Apply test rules if Focus.Test is true
		if opt.Focus.Test {
			issues = append(issues, a.runTestRules(t, opt)...)
		}
	}

	// Sort by severity (error > warning > info)
	sort.Slice(issues, func(i, j int) bool {
		return severityRank(issues[i].Severity) > severityRank(issues[j].Severity)
	})

	// Apply MaxIssues limit
	if opt.MaxIssues > 0 && len(issues) > opt.MaxIssues {
		issues = issues[:opt.MaxIssues]
	}

	return issues, nil
}

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

// ---- Security Rules (Focus.Security = true) ----

func (a *Analyzer) runSecurityRules(t Target, opt AnalyzerOptions) []Issue {
	var issues []Issue

	// cmd-injection
	issues = append(issues, a.checkCmdInjection(t, opt)...)

	// weak-crypto
	issues = append(issues, a.checkWeakCrypto(t, opt)...)

	// http-no-timeout
	issues = append(issues, a.checkHTTPNoTimeout(t, opt)...)

	// path-traversal
	issues = append(issues, a.checkPathTraversal(t, opt)...)

	return issues
}

func (a *Analyzer) checkCmdInjection(t Target, opt AnalyzerOptions) []Issue {
	if !strings.HasSuffix(t.Path, ".go") {
		return nil
	}

	var issues []Issue
	// Match exec.Command or exec.CommandContext with string concatenation or fmt.Sprintf
	patterns := []*regexp.Regexp{
		regexp.MustCompile(`exec\.Command(?:Context)?\s*\([^)]*\+`),
		regexp.MustCompile(`exec\.Command(?:Context)?\s*\([^)]*fmt\.Sprintf`),
		regexp.MustCompile(`exec\.Command(?:Context)?\s*\(\s*[a-z]`), // variable as first arg
	}

	for i, line := range extractAddedLines(t.Diff) {
		for _, pattern := range patterns {
			if pattern.MatchString(line) {
				issues = append(issues, Issue{
					ID:          "cmd-injection",
					Title:       "Potential command injection",
					Description: "Variable interpolation in exec.Command may allow command injection",
					Severity:    SeverityError,
					Path:        t.Path,
					LineStart:   i + 1,
					Snippet:     trimSnippet(line, opt.MaxSnippetLines),
					Suggestions: []string{
						"Use exec.Command with separate arguments instead of concatenation",
						"Validate and sanitize user input before passing to commands",
						"Consider using a whitelist of allowed commands",
					},
					Tags: []string{"security", "injection"},
				})
				break
			}
		}
	}
	return issues
}

func (a *Analyzer) checkWeakCrypto(t Target, opt AnalyzerOptions) []Issue {
	if !strings.HasSuffix(t.Path, ".go") {
		return nil
	}

	var issues []Issue
	weakPatterns := map[string]string{
		`md5\.`:         "MD5 is cryptographically broken",
		`sha1\.`:        "SHA1 is cryptographically weak",
		`des\.`:         "DES is cryptographically broken",
		`rc4\.`:         "RC4 is cryptographically broken",
		`"crypto/md5"`:  "MD5 package imported",
		`"crypto/sha1"`: "SHA1 package imported",
		`"crypto/des"`:  "DES package imported",
		`"crypto/rc4"`:  "RC4 package imported",
	}

	for i, line := range extractAddedLines(t.Diff) {
		for pattern, desc := range weakPatterns {
			if regexp.MustCompile(pattern).MatchString(line) {
				issues = append(issues, Issue{
					ID:          "weak-crypto",
					Title:       "Weak cryptographic algorithm",
					Description: desc,
					Severity:    SeverityWarning,
					Path:        t.Path,
					LineStart:   i + 1,
					Snippet:     trimSnippet(line, opt.MaxSnippetLines),
					Suggestions: []string{
						"Use SHA-256 or stronger for hashing",
						"Use AES-GCM for encryption",
						"Consider using crypto/sha256 or crypto/aes",
					},
					Tags: []string{"security", "crypto"},
				})
				break
			}
		}
	}
	return issues
}

func (a *Analyzer) checkHTTPNoTimeout(t Target, opt AnalyzerOptions) []Issue {
	if !strings.HasSuffix(t.Path, ".go") {
		return nil
	}

	var issues []Issue
	// Match http.Client{} without Timeout
	clientPattern := regexp.MustCompile(`http\.Client\s*\{`)
	timeoutPattern := regexp.MustCompile(`Timeout\s*:`)

	lines := extractAddedLines(t.Diff)
	for i, line := range lines {
		if clientPattern.MatchString(line) {
			// Check if Timeout is set (simple heuristic: look at next few lines)
			hasTimeout := timeoutPattern.MatchString(line)
			for j := i + 1; j < len(lines) && j < i+5; j++ {
				if strings.Contains(lines[j], "}") {
					break
				}
				if timeoutPattern.MatchString(lines[j]) {
					hasTimeout = true
					break
				}
			}
			if !hasTimeout {
				issues = append(issues, Issue{
					ID:          "http-no-timeout",
					Title:       "HTTP client without timeout",
					Description: "http.Client without Timeout may cause resource leaks",
					Severity:    SeverityWarning,
					Path:        t.Path,
					LineStart:   i + 1,
					Snippet:     trimSnippet(line, opt.MaxSnippetLines),
					Suggestions: []string{
						"Add Timeout: 30 * time.Second or appropriate value",
						"Consider using context.WithTimeout for request-level timeouts",
					},
					Tags: []string{"security", "reliability"},
				})
			}
		}
	}
	return issues
}

func (a *Analyzer) checkPathTraversal(t Target, opt AnalyzerOptions) []Issue {
	if !strings.HasSuffix(t.Path, ".go") {
		return nil
	}

	var issues []Issue
	// Match filepath.Join with potential user input
	patterns := []*regexp.Regexp{
		regexp.MustCompile(`filepath\.Join\s*\([^)]*\+`),
		regexp.MustCompile(`filepath\.Join\s*\([^)]*r\.URL`),
		regexp.MustCompile(`filepath\.Join\s*\([^)]*r\.FormValue`),
		regexp.MustCompile(`filepath\.Join\s*\([^)]*c\.Param`),
		regexp.MustCompile(`os\.Open\s*\([^)]*\+`),
		regexp.MustCompile(`ioutil\.ReadFile\s*\([^)]*\+`),
		regexp.MustCompile(`os\.ReadFile\s*\([^)]*\+`),
	}

	for i, line := range extractAddedLines(t.Diff) {
		for _, pattern := range patterns {
			if pattern.MatchString(line) {
				issues = append(issues, Issue{
					ID:          "path-traversal",
					Title:       "Potential path traversal vulnerability",
					Description: "User input in file path operations may allow directory traversal",
					Severity:    SeverityError,
					Path:        t.Path,
					LineStart:   i + 1,
					Snippet:     trimSnippet(line, opt.MaxSnippetLines),
					Suggestions: []string{
						"Use filepath.Clean to normalize the path",
						"Validate that the resolved path is within expected directory",
						"Consider using a whitelist of allowed paths",
					},
					Tags: []string{"security", "path-traversal"},
				})
				break
			}
		}
	}
	return issues
}

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

// ---- Diff Helper Functions ----

// countAddedLines counts lines starting with '+' (excluding +++ header)
func countAddedLines(diff string) int {
	count := 0
	for _, line := range strings.Split(diff, "\n") {
		if strings.HasPrefix(line, "+") && !strings.HasPrefix(line, "+++") {
			count++
		}
	}
	return count
}

// extractAddedLines extracts lines starting with '+' (excluding +++ header)
func extractAddedLines(diff string) []string {
	var lines []string
	for _, line := range strings.Split(diff, "\n") {
		if strings.HasPrefix(line, "+") && !strings.HasPrefix(line, "+++") {
			// Remove the leading '+' for analysis
			lines = append(lines, strings.TrimPrefix(line, "+"))
		}
	}
	return lines
}

// trimSnippet trims a snippet to maxLines (single line case)
func trimSnippet(line string, maxLen int) string {
	if maxLen <= 0 {
		maxLen = 100
	}
	line = strings.TrimSpace(line)
	if len(line) > maxLen {
		return line[:maxLen] + "..."
	}
	return line
}

// extractSnippet extracts lines around a target line number
func extractSnippet(diff string, lineHint int, maxLines int) string {
	if maxLines <= 0 {
		maxLines = 10
	}

	lines := strings.Split(diff, "\n")
	if lineHint < 0 || lineHint >= len(lines) {
		return ""
	}

	start := lineHint - maxLines/2
	if start < 0 {
		start = 0
	}
	end := start + maxLines
	if end > len(lines) {
		end = len(lines)
	}

	return strings.Join(lines[start:end], "\n")
}

// findLineInDiff finds line numbers matching a pattern in diff
func findLineInDiff(diff string, pattern *regexp.Regexp) []int {
	var matches []int
	for i, line := range strings.Split(diff, "\n") {
		if pattern.MatchString(line) {
			matches = append(matches, i)
		}
	}
	return matches
}
