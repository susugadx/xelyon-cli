package review

import (
	"context"
	"fmt"
	"os"
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

	// AI analysis options
	UseAI     bool      // Enable AI-powered analysis
	LLMClient LLMClient // LLM client for AI analysis (optional)
}

type Orchestrator struct {
	Scanner    *Scanner
	Analyzer   *Analyzer
	Fixer      *Fixer
	AIAnalyzer *AIAnalyzer
	Reporter   *Reporter

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

	if len(opt.Paths) > 0 && !opt.All {
		// Path-based scan (new mode)
		opt.Mode = "paths"
		targets, err = o.Scanner.ScanFromPaths(ctx, opt.Paths, ScanOptions{
			Paths:           opt.Paths,
			MaxSnippetLines: opt.MaxSnippetLines,
		})
	} else if opt.All {
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

	// AI analysis (if enabled)
	var fileContents map[string]string
	if opt.UseAI && opt.LLMClient != nil {
		if o.AIAnalyzer == nil {
			o.AIAnalyzer = NewAIAnalyzer(opt.LLMClient)
		} else {
			o.AIAnalyzer.LLM = opt.LLMClient
		}

		// Read file contents for AI analysis
		fileContents = make(map[string]string)
		for _, t := range targets {
			content, readErr := readFileContent(t.Path)
			if readErr == nil {
				fileContents[t.Path] = content

				// AI analysis per file
				insights, aiErr := o.AIAnalyzer.AnalyzeWithAI(content, t.Path)
				if aiErr == nil {
					aiIssues := ConvertInsightsToIssues(insights, t.Path)
					issues = append(issues, aiIssues...)
				}
			}
		}
	}

	var fixes []FixProposal
	if opt.Fix {
		if o.Fixer == nil {
			return Report{}, "", fmt.Errorf("--fix requested but fixer is not configured")
		}

		// Use AI fixer if AI is enabled
		if opt.UseAI && opt.LLMClient != nil {
			aiFixer := NewAIFixer(opt.LLMClient)
			fixes, err = aiFixer.ProposeWithAI(issues, fileContents, FixOptions{})
		} else {
			fixes, err = o.Fixer.Propose(issues, FixOptions{})
		}
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

// readFileContent reads file content for AI analysis.
func readFileContent(path string) (string, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(content), nil
}
