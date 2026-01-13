package review

import (
	"strings"
	"testing"
)

func TestFixer_Propose(t *testing.T) {
	fixer := NewFixer()

	tests := []struct {
		name       string
		issues     []Issue
		opt        FixOptions
		wantCount  int
		wantIDs    []string
		wantTitles []string
	}{
		{
			name:      "empty issues",
			issues:    []Issue{},
			opt:       FixOptions{},
			wantCount: 0,
		},
		{
			name: "large diff proposal",
			issues: []Issue{
				{ID: "general-large-diff", Title: "Large diff"},
			},
			opt:        FixOptions{},
			wantCount:  1,
			wantTitles: []string{"Split change into smaller commits/PRs"},
		},
		{
			name: "todo added proposal",
			issues: []Issue{
				{ID: "general-todo-added", Title: "TODO added"},
			},
			opt:        FixOptions{},
			wantCount:  1,
			wantTitles: []string{"Track TODO/FIXME and add context"},
		},
		{
			name: "security proposal",
			issues: []Issue{
				{ID: "sec-shell-exec", Title: "Shell exec"},
			},
			opt:        FixOptions{},
			wantCount:  1,
			wantTitles: []string{"Harden command execution (avoid injection)"},
		},
		{
			name: "security tagged fallback",
			issues: []Issue{
				{ID: "unknown-security", Title: "Unknown", Tags: []string{"security"}},
			},
			opt:        FixOptions{},
			wantCount:  1,
			wantTitles: []string{"Review security impact and add validation"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			proposals, err := fixer.Propose(tt.issues, tt.opt)
			if err != nil {
				t.Fatalf("Propose() error = %v", err)
			}

			if len(proposals) != tt.wantCount {
				t.Fatalf("Propose() returned %d proposals, want %d", len(proposals), tt.wantCount)
			}

			for i, want := range tt.wantTitles {
				if i < len(proposals) && proposals[i].Title != want {
					t.Errorf("proposals[%d].Title = %q, want %q", i, proposals[i].Title, want)
				}
			}
		})
	}
}

func TestFixer_ProposeMaxFixes(t *testing.T) {
	fixer := NewFixer()

	issues := make([]Issue, 20)
	for i := range issues {
		issues[i] = Issue{ID: "general-todo-added", Title: "TODO"}
	}

	opt := FixOptions{MaxFixes: 5}
	proposals, err := fixer.Propose(issues, opt)
	if err != nil {
		t.Fatalf("Propose() error = %v", err)
	}

	if len(proposals) > 5 {
		t.Errorf("Propose() returned %d proposals, want <= 5", len(proposals))
	}
}

func TestProposeForIssue(t *testing.T) {
	tests := []struct {
		name         string
		issue        Issue
		wantTitle    string
		wantHasNotes bool
	}{
		{
			name:         "large diff",
			issue:        Issue{ID: "general-large-diff"},
			wantTitle:    "Split change into smaller commits/PRs",
			wantHasNotes: true,
		},
		{
			name:         "todo added",
			issue:        Issue{ID: "general-todo-added"},
			wantTitle:    "Track TODO/FIXME and add context",
			wantHasNotes: true,
		},
		{
			name:         "go exported doc",
			issue:        Issue{ID: "general-go-exported-doc"},
			wantTitle:    "Add doc comment for exported Go symbol",
			wantHasNotes: true,
		},
		{
			name:         "shell exec",
			issue:        Issue{ID: "sec-shell-exec"},
			wantTitle:    "Harden command execution (avoid injection)",
			wantHasNotes: true,
		},
		{
			name:         "weak crypto",
			issue:        Issue{ID: "sec-crypto-weak"},
			wantTitle:    "Use crypto-safe primitives",
			wantHasNotes: true,
		},
		{
			name:         "http timeout",
			issue:        Issue{ID: "sec-http-timeout"},
			wantTitle:    "Set HTTP client timeout",
			wantHasNotes: true,
		},
		{
			name:         "path traversal",
			issue:        Issue{ID: "sec-path-traversal"},
			wantTitle:    "Add strict path validation (prevent traversal)",
			wantHasNotes: true,
		},
		{
			name:         "test coverage",
			issue:        Issue{ID: "test-coverage-check"},
			wantTitle:    "Add or update tests for changed behavior",
			wantHasNotes: true,
		},
		{
			name:         "no assertions",
			issue:        Issue{ID: "test-no-assertions"},
			wantTitle:    "Ensure tests assert expected behavior",
			wantHasNotes: true,
		},
		{
			name:         "unknown issue",
			issue:        Issue{ID: "unknown-rule"},
			wantTitle:    "",
			wantHasNotes: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			proposal := proposeForIssue(tt.issue)

			if proposal.Title != tt.wantTitle {
				t.Errorf("Title = %q, want %q", proposal.Title, tt.wantTitle)
			}

			if tt.wantHasNotes && len(proposal.Notes) == 0 {
				t.Error("expected Notes to be non-empty")
			}

			if !tt.wantHasNotes && len(proposal.Notes) > 0 {
				t.Error("expected Notes to be empty")
			}
		})
	}
}

func TestGuessGoExportedSymbolFromDiff(t *testing.T) {
	tests := []struct {
		name    string
		snippet string
		want    string
	}{
		{
			name:    "exported func",
			snippet: "+func ExportedFunc() {}",
			want:    "ExportedFunc",
		},
		{
			name:    "exported type",
			snippet: "+type ExportedType struct {}",
			want:    "ExportedType",
		},
		{
			name:    "unexported func",
			snippet: "+func unexported() {}",
			want:    "",
		},
		{
			name:    "no match",
			snippet: "+var x = 1",
			want:    "",
		},
		{
			name:    "empty",
			snippet: "",
			want:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := guessGoExportedSymbolFromDiff(tt.snippet)
			if got != tt.want {
				t.Errorf("guessGoExportedSymbolFromDiff() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestHasTag(t *testing.T) {
	tests := []struct {
		name string
		tags []string
		want string
		has  bool
	}{
		{
			name: "found",
			tags: []string{"security", "test"},
			want: "security",
			has:  true,
		},
		{
			name: "not found",
			tags: []string{"test"},
			want: "security",
			has:  false,
		},
		{
			name: "empty tags",
			tags: []string{},
			want: "security",
			has:  false,
		},
		{
			name: "nil tags",
			tags: nil,
			want: "security",
			has:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := hasTag(tt.tags, tt.want)
			if got != tt.has {
				t.Errorf("hasTag() = %v, want %v", got, tt.has)
			}
		})
	}
}

func TestBuildPatchSkeleton(t *testing.T) {
	patch := BuildPatchSkeleton("internal/foo.go", "Add timeout")

	if !strings.Contains(patch, "diff --git") {
		t.Error("patch should contain 'diff --git'")
	}
	if !strings.Contains(patch, "internal/foo.go") {
		t.Error("patch should contain file path")
	}
	if !strings.Contains(patch, "Add timeout") {
		t.Error("patch should contain note")
	}
}

func TestMin(t *testing.T) {
	tests := []struct {
		a, b, want int
	}{
		{1, 2, 1},
		{2, 1, 1},
		{5, 5, 5},
		{0, 1, 0},
		{-1, 1, -1},
	}

	for _, tt := range tests {
		got := min(tt.a, tt.b)
		if got != tt.want {
			t.Errorf("min(%d, %d) = %d, want %d", tt.a, tt.b, got, tt.want)
		}
	}
}
