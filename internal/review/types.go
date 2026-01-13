package review

import "time"

// types.go centralizes public types for the internal/review package.
//
// Rationale:
// - Avoid type duplication across scanner/analyzer/fixer/reporter/review.
// - Stabilize package-level API for /review command integration.
//
// NOTE: This package is internal; API stability here is for our own maintainability.

// ---- Scanner domain ----

type Target struct {
	Path       string
	OldPath    string
	ChangeType string // "A","M","D","R" etc. Best-effort.
	Diff       string // unified diff (best-effort)
}

// ---- Analyzer domain ----

type ReviewFocus struct {
	Security bool
	Test     bool
}

type Severity string

const (
	SeverityInfo    Severity = "info"
	SeverityWarning Severity = "warning"
	SeverityError   Severity = "error"
)

type Issue struct {
	ID          string
	Title       string
	Description string
	Severity    Severity
	Path        string
	LineStart   int
	LineEnd     int
	Snippet     string
	Suggestions []string
	Tags        []string
}

// severityRank is used for sorting issues in reporter/analyzer.
func severityRank(s Severity) int {
	switch s {
	case SeverityError:
		return 3
	case SeverityWarning:
		return 2
	case SeverityInfo:
		return 1
	default:
		return 0
	}
}

// ---- Fixer domain ----

type FixProposal struct {
	IssueID   string
	Title     string
	Rationale string

	// Example is a code snippet (not necessarily a patch).
	Example string

	// Patch is a best-effort unified diff proposal. It may be empty.
	Patch string

	// Safety notes and constraints.
	Notes []string
}

// ---- Reporter domain ----

type Report struct {
	GeneratedAt time.Time
	Mode        string
	Provider    string
	Model       string

	Targets []Target
	Issues  []Issue
	Fixes   []FixProposal

	Summary string
}
