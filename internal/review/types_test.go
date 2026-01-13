package review

import (
	"testing"
	"time"
)

func TestSeverityConstants(t *testing.T) {
	if SeverityInfo != "info" {
		t.Errorf("SeverityInfo = %q, want %q", SeverityInfo, "info")
	}
	if SeverityWarning != "warning" {
		t.Errorf("SeverityWarning = %q, want %q", SeverityWarning, "warning")
	}
	if SeverityError != "error" {
		t.Errorf("SeverityError = %q, want %q", SeverityError, "error")
	}
}

func TestSeverityRankOrdering(t *testing.T) {
	// Error should be highest, Info lowest
	if severityRank(SeverityError) <= severityRank(SeverityWarning) {
		t.Error("Error should rank higher than Warning")
	}
	if severityRank(SeverityWarning) <= severityRank(SeverityInfo) {
		t.Error("Warning should rank higher than Info")
	}
	if severityRank(SeverityInfo) <= severityRank(Severity("unknown")) {
		t.Error("Info should rank higher than unknown")
	}
}

func TestTargetStruct(t *testing.T) {
	target := Target{
		Path:       "test.go",
		OldPath:    "old.go",
		ChangeType: "R",
		Diff:       "+line",
	}

	if target.Path != "test.go" {
		t.Errorf("Path = %q, want %q", target.Path, "test.go")
	}
	if target.OldPath != "old.go" {
		t.Errorf("OldPath = %q, want %q", target.OldPath, "old.go")
	}
	if target.ChangeType != "R" {
		t.Errorf("ChangeType = %q, want %q", target.ChangeType, "R")
	}
	if target.Diff != "+line" {
		t.Errorf("Diff = %q, want %q", target.Diff, "+line")
	}
}

func TestIssueStruct(t *testing.T) {
	issue := Issue{
		ID:          "test-id",
		Title:       "Test Title",
		Description: "Test Description",
		Severity:    SeverityError,
		Path:        "test.go",
		LineStart:   10,
		LineEnd:     20,
		Snippet:     "code snippet",
		Suggestions: []string{"suggestion1", "suggestion2"},
		Tags:        []string{"security", "test"},
	}

	if issue.ID != "test-id" {
		t.Errorf("ID = %q, want %q", issue.ID, "test-id")
	}
	if issue.Title != "Test Title" {
		t.Errorf("Title = %q, want %q", issue.Title, "Test Title")
	}
	if issue.Severity != SeverityError {
		t.Errorf("Severity = %q, want %q", issue.Severity, SeverityError)
	}
	if issue.LineStart != 10 {
		t.Errorf("LineStart = %d, want %d", issue.LineStart, 10)
	}
	if issue.LineEnd != 20 {
		t.Errorf("LineEnd = %d, want %d", issue.LineEnd, 20)
	}
	if len(issue.Suggestions) != 2 {
		t.Errorf("Suggestions length = %d, want %d", len(issue.Suggestions), 2)
	}
	if len(issue.Tags) != 2 {
		t.Errorf("Tags length = %d, want %d", len(issue.Tags), 2)
	}
}

func TestFixProposalStruct(t *testing.T) {
	fix := FixProposal{
		IssueID:   "issue-1",
		Title:     "Fix Title",
		Rationale: "Fix Rationale",
		Example:   "Example code",
		Patch:     "diff patch",
		Notes:     []string{"note1", "note2"},
	}

	if fix.IssueID != "issue-1" {
		t.Errorf("IssueID = %q, want %q", fix.IssueID, "issue-1")
	}
	if fix.Title != "Fix Title" {
		t.Errorf("Title = %q, want %q", fix.Title, "Fix Title")
	}
	if fix.Rationale != "Fix Rationale" {
		t.Errorf("Rationale = %q, want %q", fix.Rationale, "Fix Rationale")
	}
	if fix.Example != "Example code" {
		t.Errorf("Example = %q, want %q", fix.Example, "Example code")
	}
	if fix.Patch != "diff patch" {
		t.Errorf("Patch = %q, want %q", fix.Patch, "diff patch")
	}
	if len(fix.Notes) != 2 {
		t.Errorf("Notes length = %d, want %d", len(fix.Notes), 2)
	}
}

func TestReportStruct(t *testing.T) {
	now := time.Now()
	report := Report{
		GeneratedAt: now,
		Mode:        "changes",
		Provider:    "deepseek",
		Model:       "deepseek-chat",
		Targets:     []Target{{Path: "test.go"}},
		Issues:      []Issue{{ID: "issue-1"}},
		Fixes:       []FixProposal{{IssueID: "issue-1"}},
		Summary:     "Test summary",
	}

	if report.GeneratedAt != now {
		t.Error("GeneratedAt mismatch")
	}
	if report.Mode != "changes" {
		t.Errorf("Mode = %q, want %q", report.Mode, "changes")
	}
	if report.Provider != "deepseek" {
		t.Errorf("Provider = %q, want %q", report.Provider, "deepseek")
	}
	if report.Model != "deepseek-chat" {
		t.Errorf("Model = %q, want %q", report.Model, "deepseek-chat")
	}
	if len(report.Targets) != 1 {
		t.Errorf("Targets length = %d, want %d", len(report.Targets), 1)
	}
	if len(report.Issues) != 1 {
		t.Errorf("Issues length = %d, want %d", len(report.Issues), 1)
	}
	if len(report.Fixes) != 1 {
		t.Errorf("Fixes length = %d, want %d", len(report.Fixes), 1)
	}
	if report.Summary != "Test summary" {
		t.Errorf("Summary = %q, want %q", report.Summary, "Test summary")
	}
}

func TestReviewFocusStruct(t *testing.T) {
	focus := ReviewFocus{
		Security: true,
		Test:     true,
	}

	if !focus.Security {
		t.Error("Security should be true")
	}
	if !focus.Test {
		t.Error("Test should be true")
	}

	emptyFocus := ReviewFocus{}
	if emptyFocus.Security {
		t.Error("Security should default to false")
	}
	if emptyFocus.Test {
		t.Error("Test should default to false")
	}
}
