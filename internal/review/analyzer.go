package review

import (
	"sort"
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

		// Always apply quality rules (new rules)
		issues = append(issues, a.runQualityRules(t, opt)...)

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
