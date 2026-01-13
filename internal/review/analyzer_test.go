package review

import (
	"testing"
)

func TestAnalyzer_Analyze(t *testing.T) {
	analyzer := NewAnalyzer()

	tests := []struct {
		name    string
		targets []Target
		opt     AnalyzerOptions
		// wantIDs are expected to be present somewhere in the issues list.
		wantIDs []string
	}{
		{
			name:    "empty targets",
			targets: []Target{},
			opt:     AnalyzerOptions{},
			wantIDs: nil,
		},
		{
			name: "large diff detection",
			targets: []Target{
				{
					Path:       "large.go",
					ChangeType: "M",
					Diff:       generateLargeDiff(600),
				},
			},
			opt:     AnalyzerOptions{},
			wantIDs: []string{"large-diff"},
		},
		{
			name: "TODO detection",
			targets: []Target{
				{
					Path:       "todo.go",
					ChangeType: "M",
					Diff:       "+// TODO: fix this later\n+func foo() {}",
				},
			},
			opt:     AnalyzerOptions{},
			wantIDs: []string{"todo-added"},
		},
		{
			name: "FIXME detection",
			targets: []Target{
				{
					Path:       "fixme.go",
					ChangeType: "M",
					Diff:       "+// FIXME: broken code\n+func bar() {}",
				},
			},
			opt:     AnalyzerOptions{},
			wantIDs: []string{"todo-added"},
		},
		{
			name: "sensitive file detection",
			targets: []Target{
				{
					Path:       ".env",
					ChangeType: "M",
					Diff:       "+SECRET_KEY=abc123",
				},
			},
			opt:     AnalyzerOptions{},
			wantIDs: []string{"sensitive-file"},
		},
		{
			name: "credentials file detection",
			targets: []Target{
				{
					Path:       "credentials.json",
					ChangeType: "A",
					Diff:       "+{\"api_key\": \"secret\"}",
				},
			},
			opt:     AnalyzerOptions{},
			wantIDs: []string{"sensitive-file"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			issues, err := analyzer.Analyze(tt.targets, tt.opt)
			if err != nil {
				t.Fatalf("Analyze() error = %v", err)
			}

			if tt.wantIDs != nil {
				for _, wantID := range tt.wantIDs {
					found := false
					for _, is := range issues {
						if is.ID == wantID {
							found = true
							break
						}
					}
					if !found {
						t.Errorf("expected issue %q not found", wantID)
					}
				}
			} else {
				// when wantIDs is nil, we expect no issues
				if len(issues) != 0 {
					t.Errorf("Analyze() got %d issues, want 0", len(issues))
				}
			}
		})
	}
}

func TestAnalyzer_SecurityRules(t *testing.T) {
	analyzer := NewAnalyzer()

	tests := []struct {
		name    string
		diff    string
		wantID  string
		wantSev Severity
	}{
		{
			name:    "command injection with concatenation",
			diff:    "+exec.Command(\"sh\", \"-c\", userInput + \" -flag\")",
			wantID:  "cmd-injection",
			wantSev: SeverityError,
		},
		{
			name:    "command injection with fmt.Sprintf",
			diff:    "+exec.Command(\"bash\", \"-c\", fmt.Sprintf(\"echo %s\", input))",
			wantID:  "cmd-injection",
			wantSev: SeverityError,
		},
		{
			name:    "weak crypto MD5",
			diff:    "+h := md5.Sum(data)",
			wantID:  "weak-crypto",
			wantSev: SeverityWarning,
		},
		{
			name:    "weak crypto SHA1 import",
			diff:    "+import \"crypto/sha1\"",
			wantID:  "weak-crypto",
			wantSev: SeverityWarning,
		},
		{
			name:    "HTTP client without timeout",
			diff:    "+client := &http.Client{}",
			wantID:  "http-no-timeout",
			wantSev: SeverityWarning,
		},
		{
			name:    "path traversal with filepath.Join",
			diff:    "+path := filepath.Join(base, r.URL.Path)",
			wantID:  "path-traversal",
			wantSev: SeverityError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			targets := []Target{
				{
					Path:       "test.go",
					ChangeType: "M",
					Diff:       tt.diff,
				},
			}
			opt := AnalyzerOptions{
				Focus: ReviewFocus{Security: true},
			}

			issues, err := analyzer.Analyze(targets, opt)
			if err != nil {
				t.Fatalf("Analyze() error = %v", err)
			}

			found := false
			for _, issue := range issues {
				if issue.ID == tt.wantID {
					found = true
					if issue.Severity != tt.wantSev {
						t.Errorf("issue.Severity = %q, want %q", issue.Severity, tt.wantSev)
					}
					break
				}
			}
			if !found {
				t.Errorf("expected issue %q not found in %v", tt.wantID, issues)
			}
		})
	}
}

func TestAnalyzer_TestRules(t *testing.T) {
	analyzer := NewAnalyzer()

	tests := []struct {
		name    string
		target  Target
		wantID  string
		wantSev Severity
	}{
		{
			name: "new file without tests",
			target: Target{
				Path:       "newfile.go",
				ChangeType: "A",
				Diff:       "+package main\n+func NewFunc() {}",
			},
			wantID:  "test-coverage",
			wantSev: SeverityInfo,
		},
		{
			name: "test file without assertions",
			target: Target{
				Path:       "foo_test.go",
				ChangeType: "M",
				Diff:       "+func TestFoo(t *testing.T) {\n+\tfoo()\n+}",
			},
			wantID:  "assertion-missing",
			wantSev: SeverityWarning,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opt := AnalyzerOptions{
				Focus: ReviewFocus{Test: true},
			}

			issues, err := analyzer.Analyze([]Target{tt.target}, opt)
			if err != nil {
				t.Fatalf("Analyze() error = %v", err)
			}

			found := false
			for _, issue := range issues {
				if issue.ID == tt.wantID {
					found = true
					if issue.Severity != tt.wantSev {
						t.Errorf("issue.Severity = %q, want %q", issue.Severity, tt.wantSev)
					}
					break
				}
			}
			if !found {
				t.Errorf("expected issue %q not found", tt.wantID)
			}
		})
	}
}

func TestAnalyzer_MaxIssues(t *testing.T) {
	analyzer := NewAnalyzer()

	// Generate multiple TODOs
	diff := ""
	for i := 0; i < 20; i++ {
		diff += "+// TODO: item\n"
	}

	targets := []Target{
		{Path: "test.go", ChangeType: "M", Diff: diff},
	}

	opt := AnalyzerOptions{MaxIssues: 5}
	issues, err := analyzer.Analyze(targets, opt)
	if err != nil {
		t.Fatalf("Analyze() error = %v", err)
	}

	if len(issues) > 5 {
		t.Errorf("Analyze() returned %d issues, want <= 5", len(issues))
	}
}

func TestAnalyzer_GoExportMissingDoc(t *testing.T) {
	analyzer := NewAnalyzer()

	tests := []struct {
		name      string
		diff      string
		wantIssue bool
	}{
		{
			name:      "exported func without doc",
			diff:      "+func ExportedFunc() {}",
			wantIssue: true,
		},
		{
			name:      "exported type without doc",
			diff:      "+type ExportedType struct {}",
			wantIssue: true,
		},
		{
			name:      "unexported func",
			diff:      "+func unexportedFunc() {}",
			wantIssue: false,
		},
		{
			name:      "exported func with doc",
			diff:      "+// ExportedFunc does something\n+func ExportedFunc() {}",
			wantIssue: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			targets := []Target{
				{Path: "test.go", ChangeType: "M", Diff: tt.diff},
			}

			issues, err := analyzer.Analyze(targets, AnalyzerOptions{})
			if err != nil {
				t.Fatalf("Analyze() error = %v", err)
			}

			found := false
			for _, issue := range issues {
				if issue.ID == "go-export-missing-doc" {
					found = true
					break
				}
			}

			if found != tt.wantIssue {
				t.Errorf("go-export-missing-doc found = %v, want %v", found, tt.wantIssue)
			}
		})
	}
}

// Helper functions

func generateLargeDiff(lines int) string {
	diff := ""
	for i := 0; i < lines; i++ {
		diff += "+// line\n"
	}
	return diff
}

func TestCountAddedLines(t *testing.T) {
	tests := []struct {
		name string
		diff string
		want int
	}{
		{
			name: "simple additions",
			diff: "+line1\n+line2\n+line3",
			want: 3,
		},
		{
			name: "mixed diff",
			diff: "+added\n-removed\n context\n+added2",
			want: 2,
		},
		{
			name: "with +++ header",
			diff: "+++ b/file.go\n+line1\n+line2",
			want: 2,
		},
		{
			name: "empty diff",
			diff: "",
			want: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := countAddedLines(tt.diff)
			if got != tt.want {
				t.Errorf("countAddedLines() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestExtractAddedLines(t *testing.T) {
	diff := "+line1\n-removed\n context\n+line2\n+++ header"
	got := extractAddedLines(diff)

	if len(got) != 2 {
		t.Fatalf("extractAddedLines() returned %d lines, want 2", len(got))
	}
	if got[0] != "line1" {
		t.Errorf("got[0] = %q, want %q", got[0], "line1")
	}
	if got[1] != "line2" {
		t.Errorf("got[1] = %q, want %q", got[1], "line2")
	}
}

func TestTrimSnippet(t *testing.T) {
	tests := []struct {
		name   string
		line   string
		maxLen int
		want   string
	}{
		{
			name:   "short line",
			line:   "short",
			maxLen: 100,
			want:   "short",
		},
		{
			name:   "long line truncated",
			line:   "this is a very long line that exceeds the limit",
			maxLen: 20,
			want:   "this is a very long ...",
		},
		{
			name:   "whitespace trimmed",
			line:   "  trimmed  ",
			maxLen: 100,
			want:   "trimmed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := trimSnippet(tt.line, tt.maxLen)
			if got != tt.want {
				t.Errorf("trimSnippet() = %q, want %q", got, tt.want)
			}
		})
	}
}
