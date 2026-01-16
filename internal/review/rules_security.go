package review

import (
	"regexp"
	"strings"
)

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
