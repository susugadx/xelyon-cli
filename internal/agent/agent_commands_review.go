package agent

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/review"
	"github.com/susugadx/xelyon-cli/internal/tools"
)

// handleReviewCommand handles the /review command for code review (static analysis only)
// Flags: --all, --security, --test, --max-issues
// Usage: /review [flags] [paths...]
//
//	paths can be files, directories, or glob patterns (e.g., **/*.go)
func handleReviewCommand(agent *Agent, args []string) bool {
	// Parse flags and paths
	opt := review.Options{
		Provider: agent.CurrentProvider.Name(),
		Model:    agent.CurrentModel,
	}

	var paths []string

	for i := 0; i < len(args); i++ {
		arg := args[i]

		// Check if it's a flag
		if strings.HasPrefix(arg, "-") {
			switch arg {
			case "--all", "-a":
				opt.All = true
			case "--security", "-s":
				opt.Focus.Security = true
			case "--test", "-t":
				opt.Focus.Test = true
			case "--max-issues":
				if i+1 < len(args) {
					i++
					if n, err := strconv.Atoi(args[i]); err == nil {
						opt.MaxIssues = n
					}
				}
			default:
				yellow.Printf("Unknown flag: %s\n", arg)
			}
		} else {
			// It's a path argument
			paths = append(paths, arg)
		}
	}

	opt.Paths = paths

	// Check for changes or paths
	if !opt.All && len(paths) == 0 && len(agent.changeStack) == 0 {
		yellow.Println("⚠️  No changes to review.")
		yellow.Println("Usage:")
		yellow.Println("  /review                      - Review session changes")
		yellow.Println("  /review <path>               - Review specific file/directory")
		yellow.Println("  /review **/*.go              - Review files matching pattern")
		yellow.Println("  /review --all                - Review all git changes")
		return true
	}

	// Confirm before running
	cyan.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	cyan.Println("🔍 Code Review (Static Analysis)")
	if len(paths) > 0 {
		cyan.Printf("   Target: %d path(s)\n", len(paths))
		for _, p := range paths {
			cyan.Printf("     - %s\n", p)
		}
	} else if opt.All {
		cyan.Println("   Target: All git changes")
	} else {
		cyan.Printf("   Target: %d changed files\n", len(agent.changeStack))
	}
	if opt.Focus.Security {
		cyan.Println("   Focus: Security")
	}
	if opt.Focus.Test {
		cyan.Println("   Focus: Test coverage")
	}
	cyan.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	if !promptConfirm("Run review? [y/N]: ") {
		yellow.Println("❌ Review cancelled")
		return true
	}

	// Run review
	ctx := context.Background()
	orchestrator := review.NewOrchestrator()

	// Convert changeStack to []tools.FileChange
	var changes []tools.FileChange
	changes = append(changes, agent.changeStack...)

	report, outPath, err := orchestrator.Run(ctx, changes, opt)
	if err != nil {
		red.Printf("Review failed: %v\n", err)
		return true
	}

	// Summary
	green.Println("\n━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	green.Println("✅ Review Complete")
	green.Printf("   Files: %d\n", len(report.Targets))
	green.Printf("   Issues: %d\n", len(report.Issues))
	if outPath != "" {
		green.Printf("   Report: %s\n", outPath)
	}
	green.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	// Print issues summary
	if len(report.Issues) > 0 {
		fmt.Println("\n📋 Issues:")
		errorCount := 0
		warningCount := 0
		infoCount := 0
		for _, issue := range report.Issues {
			switch issue.Severity {
			case review.SeverityError:
				errorCount++
			case review.SeverityWarning:
				warningCount++
			case review.SeverityInfo:
				infoCount++
			}
		}
		if errorCount > 0 {
			red.Printf("   🔴 Errors: %d\n", errorCount)
		}
		if warningCount > 0 {
			yellow.Printf("   🟡 Warnings: %d\n", warningCount)
		}
		if infoCount > 0 {
			cyan.Printf("   🔵 Info: %d\n", infoCount)
		}

		// Show first few issues
		maxShow := 5
		if len(report.Issues) < maxShow {
			maxShow = len(report.Issues)
		}
		fmt.Println()
		for i := 0; i < maxShow; i++ {
			issue := report.Issues[i]
			var icon string
			switch issue.Severity {
			case review.SeverityError:
				icon = "🔴"
			case review.SeverityWarning:
				icon = "🟡"
			default:
				icon = "🔵"
			}
			fmt.Printf("   %s [%s] %s\n", icon, issue.ID, issue.Title)
			if issue.Path != "" {
				fmt.Printf("      %s", issue.Path)
				if issue.LineStart > 0 {
					fmt.Printf(":%d", issue.LineStart)
				}
				fmt.Println()
			}
		}
		if len(report.Issues) > maxShow {
			fmt.Printf("   ... and %d more (see report)\n", len(report.Issues)-maxShow)
		}
	}

	return true
}
