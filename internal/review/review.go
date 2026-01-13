package review

import (
	"context"
	"fmt"
	"time"

	"github.com/susugadx/xelyon-cli/internal/tools"
)

// Options is the orchestrator-level configuration.
// /review command flags will map onto this structure.
//
// MVP scope:
// - Default targets: derived from changeStack.
// - --all: review all changes from git diff.
// - Focus flags: security/test
// - --fix: generate suggestion-only fix proposals.
// - Write Markdown report to ~/.xelyon/reviews

type Options struct {
	All   bool
	Paths []string

	Focus ReviewFocus
	Fix   bool

	Mode     string // "changes" or "diff" (best-effort label)
	Provider string
	Model    string

	OutputDir string

	MaxIssues       int
	MaxSnippetLines int
}

type Orchestrator struct {
	Scanner  *Scanner
	Analyzer *Analyzer
	Fixer    *Fixer
	Reporter *Reporter

	Now func() time.Time
}

func NewOrchestrator() *Orchestrator {
	now := time.Now
	return &Orchestrator{
		Scanner:  NewScanner(),
		Analyzer: NewAnalyzer(),
		Fixer:    NewFixer(),
		Reporter: NewReporter(),
		Now:      now,
	}
}

// Run executes the review pipeline.
//
// changeStack is optional but required when opt.All is false.
func (o *Orchestrator) Run(ctx context.Context, changeStack []tools.FileChange, opt Options) (Report, string, error) {
	if o.Scanner == nil || o.Analyzer == nil || o.Reporter == nil {
		return Report{}, "", fmt.Errorf("review orchestrator is not configured")
	}

	gen := o.Now
	if gen == nil {
		gen = time.Now
	}

	var (
		targets []Target
		err     error
	)

	if opt.All {
		opt.Mode = "diff"
		targets, err = o.Scanner.ScanAllFromGitDiff(ctx, ScanOptions{
			All:             true,
			Paths:           opt.Paths,
			MaxSnippetLines: opt.MaxSnippetLines,
		})
	} else {
		opt.Mode = "changes"
		targets, err = o.Scanner.ScanFromChanges(ctx, changeStack, ScanOptions{
			All:             false,
			Paths:           opt.Paths,
			MaxSnippetLines: opt.MaxSnippetLines,
		})
	}
	if err != nil {
		return Report{}, "", fmt.Errorf("scan failed: %w", err)
	}

	issues, err := o.Analyzer.Analyze(targets, AnalyzerOptions{
		Focus:           opt.Focus,
		MaxIssues:       opt.MaxIssues,
		MaxSnippetLines: opt.MaxSnippetLines,
	})
	if err != nil {
		return Report{}, "", fmt.Errorf("analyze failed: %w", err)
	}

	var fixes []FixProposal
	if opt.Fix {
		if o.Fixer == nil {
			return Report{}, "", fmt.Errorf("--fix requested but fixer is not configured")
		}
		fixes, err = o.Fixer.Propose(issues, FixOptions{})
		if err != nil {
			return Report{}, "", fmt.Errorf("fix proposals failed: %w", err)
		}
	}

	rep := Report{
		GeneratedAt: gen(),
		Mode:        opt.Mode,
		Provider:    opt.Provider,
		Model:       opt.Model,
		Targets:     targets,
		Issues:      issues,
		Fixes:       fixes,
		Summary:     "",
	}

	outPath, err := o.Reporter.WriteMarkdown(rep, ReporterOptions{OutputDir: opt.OutputDir, IncludeDiff: true})
	if err != nil {
		return rep, "", fmt.Errorf("write report failed: %w", err)
	}
	return rep, outPath, nil
}
