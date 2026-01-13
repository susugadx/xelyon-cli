package review

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestReporter_WriteMarkdown(t *testing.T) {
	reporter := NewReporter()

	// Use temp directory
	tmpDir := t.TempDir()

	tests := []struct {
		name    string
		report  Report
		opt     ReporterOptions
		wantErr bool
	}{
		{
			name: "basic report",
			report: Report{
				GeneratedAt: time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC),
				Mode:        "changes",
				Provider:    "deepseek",
				Model:       "deepseek-chat",
				Targets: []Target{
					{Path: "test.go", ChangeType: "M", Diff: "+line"},
				},
				Issues: []Issue{
					{
						ID:          "test-issue",
						Title:       "Test Issue",
						Description: "Description",
						Severity:    SeverityWarning,
						Path:        "test.go",
					},
				},
			},
			opt: ReporterOptions{
				OutputDir:   tmpDir,
				FileName:    "test_report.md",
				IncludeDiff: true,
			},
			wantErr: false,
		},
		{
			name: "empty report",
			report: Report{
				GeneratedAt: time.Now(),
			},
			opt: ReporterOptions{
				OutputDir: tmpDir,
				FileName:  "empty_report.md",
			},
			wantErr: false,
		},
		{
			name: "report with fixes",
			report: Report{
				GeneratedAt: time.Now(),
				Fixes: []FixProposal{
					{
						IssueID:   "test-fix",
						Title:     "Fix Title",
						Rationale: "Fix Rationale",
						Example:   "Example code",
						Notes:     []string{"Note 1", "Note 2"},
					},
				},
			},
			opt: ReporterOptions{
				OutputDir: tmpDir,
				FileName:  "fix_report.md",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path, err := reporter.WriteMarkdown(tt.report, tt.opt)

			if (err != nil) != tt.wantErr {
				t.Fatalf("WriteMarkdown() error = %v, wantErr %v", err, tt.wantErr)
			}

			if !tt.wantErr {
				if path == "" {
					t.Error("WriteMarkdown() returned empty path")
				}

				if _, err := os.Stat(path); os.IsNotExist(err) {
					t.Errorf("WriteMarkdown() file not created: %s", path)
				}
			}
		})
	}
}

func TestReporter_WriteMarkdownAutoFileName(t *testing.T) {
	reporter := NewReporter()
	tmpDir := t.TempDir()

	report := Report{
		GeneratedAt: time.Date(2025, 1, 15, 10, 30, 45, 0, time.UTC),
	}
	opt := ReporterOptions{
		OutputDir: tmpDir,
	}

	path, err := reporter.WriteMarkdown(report, opt)
	if err != nil {
		t.Fatalf("WriteMarkdown() error = %v", err)
	}

	expectedName := "20250115_103045.md"
	if !strings.HasSuffix(path, expectedName) {
		t.Errorf("path = %q, want suffix %q", path, expectedName)
	}
}

func TestReporter_WriteMarkdownDefaultDir(t *testing.T) {
	reporter := NewReporter()

	report := Report{
		GeneratedAt: time.Now(),
	}
	opt := ReporterOptions{
		FileName: "default_dir_test.md",
	}

	path, err := reporter.WriteMarkdown(report, opt)
	if err != nil {
		t.Fatalf("WriteMarkdown() error = %v", err)
	}

	// Should be in ~/.xelyon/reviews/
	homeDir, _ := os.UserHomeDir()
	expectedDir := filepath.Join(homeDir, ".xelyon", "reviews")
	if !strings.HasPrefix(path, expectedDir) {
		t.Errorf("path = %q, want prefix %q", path, expectedDir)
	}

	// Cleanup
	os.Remove(path)
}

func TestBuildMarkdown(t *testing.T) {
	report := Report{
		GeneratedAt: time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC),
		Mode:        "changes",
		Provider:    "deepseek",
		Model:       "deepseek-chat",
		Targets: []Target{
			{Path: "test.go", ChangeType: "M", Diff: "+added line"},
		},
		Issues: []Issue{
			{
				ID:          "test-issue",
				Title:       "Test Issue",
				Description: "Test description",
				Severity:    SeverityError,
				Path:        "test.go",
				LineStart:   10,
				Tags:        []string{"test"},
				Suggestions: []string{"Fix it"},
			},
		},
		Summary: "Test summary",
	}

	opt := ReporterOptions{IncludeDiff: true}
	md := buildMarkdown(report, opt)

	// Check header
	if !strings.Contains(md, "# XELYON Review Report") {
		t.Error("missing header")
	}

	// Check metadata
	if !strings.Contains(md, "Generated:") {
		t.Error("missing Generated timestamp")
	}
	if !strings.Contains(md, "Mode: changes") {
		t.Error("missing Mode")
	}
	if !strings.Contains(md, "deepseek") {
		t.Error("missing Provider")
	}

	// Check summary
	if !strings.Contains(md, "Test summary") {
		t.Error("missing Summary")
	}

	// Check targets table
	if !strings.Contains(md, "test.go") {
		t.Error("missing target file")
	}

	// Check issues
	if !strings.Contains(md, "[error] Test Issue") {
		t.Error("missing issue")
	}
	if !strings.Contains(md, "Line: 10") {
		t.Error("missing line number")
	}
	if !strings.Contains(md, "Fix it") {
		t.Error("missing suggestion")
	}

	// Check appendix
	if !strings.Contains(md, "## Appendix") {
		t.Error("missing appendix")
	}
}

func TestBuildMarkdown_WithFixes(t *testing.T) {
	report := Report{
		GeneratedAt: time.Now(),
		Fixes: []FixProposal{
			{
				IssueID:   "fix-1",
				Title:     "Fix Title",
				Rationale: "Rationale text",
				Example:   "Example code",
				Patch:     "diff patch",
				Notes:     []string{"Note 1"},
			},
		},
	}

	opt := ReporterOptions{}
	md := buildMarkdown(report, opt)

	if !strings.Contains(md, "## Fix Proposals") {
		t.Error("missing fix proposals section")
	}
	if !strings.Contains(md, "Fix Title") {
		t.Error("missing fix title")
	}
	if !strings.Contains(md, "Rationale text") {
		t.Error("missing rationale")
	}
	if !strings.Contains(md, "Example code") {
		t.Error("missing example")
	}
	if !strings.Contains(md, "diff patch") {
		t.Error("missing patch")
	}
}

func TestDiffHash(t *testing.T) {
	tests := []struct {
		name string
		diff string
		want string
	}{
		{
			name: "empty diff",
			diff: "",
			want: "-",
		},
		{
			name: "whitespace only",
			diff: "   \n\t",
			want: "-",
		},
		{
			name: "normal diff",
			diff: "+added line",
			want: "", // Will be 10-char hex hash
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := diffHash(tt.diff)

			if tt.want == "-" {
				if got != "-" {
					t.Errorf("diffHash() = %q, want %q", got, "-")
				}
			} else {
				if len(got) != 10 {
					t.Errorf("diffHash() length = %d, want 10", len(got))
				}
			}
		})
	}
}

func TestEscapeMD(t *testing.T) {
	tests := []struct {
		name string
		s    string
		want string
	}{
		{
			name: "no escaping needed",
			s:    "normal text",
			want: "normal text",
		},
		{
			name: "escape pipe",
			s:    "a|b",
			want: "a\\|b",
		},
		{
			name: "newlines to spaces",
			s:    "line1\nline2",
			want: "line1 line2",
		},
		{
			name: "combined",
			s:    "a|b\nc",
			want: "a\\|b c",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := escapeMD(tt.s)
			if got != tt.want {
				t.Errorf("escapeMD() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestBlankIf(t *testing.T) {
	tests := []struct {
		s        string
		fallback string
		want     string
	}{
		{"value", "default", "value"},
		{"", "default", "default"},
		{"  ", "default", "default"},
		{"\t", "default", "default"},
	}

	for _, tt := range tests {
		got := blankIf(tt.s, tt.fallback)
		if got != tt.want {
			t.Errorf("blankIf(%q, %q) = %q, want %q", tt.s, tt.fallback, got, tt.want)
		}
	}
}

func TestFindTargetDiff(t *testing.T) {
	targets := []Target{
		{Path: "a.go", Diff: "diff a"},
		{Path: "b.go", Diff: "diff b"},
	}

	tests := []struct {
		path string
		want string
	}{
		{"a.go", "diff a"},
		{"b.go", "diff b"},
		{"c.go", ""},
	}

	for _, tt := range tests {
		got := findTargetDiff(targets, tt.path)
		if got != tt.want {
			t.Errorf("findTargetDiff(%q) = %q, want %q", tt.path, got, tt.want)
		}
	}
}

func TestTrimLines(t *testing.T) {
	tests := []struct {
		name string
		s    string
		max  int
		want string
	}{
		{
			name: "under limit",
			s:    "line1\nline2\nline3",
			max:  10,
			want: "line1\nline2\nline3",
		},
		{
			name: "over limit",
			s:    "line1\nline2\nline3\nline4\nline5",
			max:  3,
			want: "line1\nline2\nline3\n... (truncated)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := trimLines(tt.s, tt.max)
			if got != tt.want {
				t.Errorf("trimLines() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestSeverityRank(t *testing.T) {
	tests := []struct {
		sev  Severity
		want int
	}{
		{SeverityError, 3},
		{SeverityWarning, 2},
		{SeverityInfo, 1},
		{Severity("unknown"), 0},
	}

	for _, tt := range tests {
		got := severityRank(tt.sev)
		if got != tt.want {
			t.Errorf("severityRank(%q) = %d, want %d", tt.sev, got, tt.want)
		}
	}
}
