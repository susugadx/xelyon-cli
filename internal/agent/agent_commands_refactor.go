package agent

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/refactor"
	"github.com/susugadx/xelyon-cli/internal/review"
)

// handleRefactorCommand handles the /refactor command for code refactoring analysis.
// Flags: --fix, --yes, --type, --ai, --max-file-lines, --max-func-lines
// Usage: /refactor [flags] [paths...]
func handleRefactorCommand(agent *Agent, args []string) bool {
	opt := refactor.RefactorOptions{
		Config: refactor.DefaultConfig(),
	}

	for i := 0; i < len(args); i++ {
		arg := args[i]

		if strings.HasPrefix(arg, "-") {
			switch arg {
			case "--fix", "-f":
				opt.Fix = true
			case "--yes", "-y":
				opt.AutoApprove = true
			case "--ai":
				opt.Config.UseAI = true
			case "--type", "-t":
				if i+1 < len(args) {
					i++
					opt.TypeFilter = refactor.RefactorType(args[i])
				}
			case "--max-file-lines":
				if i+1 < len(args) {
					i++
					if n, err := strconv.Atoi(args[i]); err == nil {
						opt.Config.MaxFileLines = n
					}
				}
			case "--max-func-lines":
				if i+1 < len(args) {
					i++
					if n, err := strconv.Atoi(args[i]); err == nil {
						opt.Config.MaxFunctionLines = n
					}
				}
			default:
				yellow.Printf("Unknown flag: %s\n", arg)
			}
		} else {
			opt.Paths = append(opt.Paths, arg)
		}
	}

	// Default to current directory if no paths specified
	if len(opt.Paths) == 0 {
		opt.Paths = []string{"."}
	}

	// Confirm before running
	if !opt.AutoApprove {
		cyan.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		cyan.Println("🔧 Code Refactoring Analysis")
		cyan.Printf("   Paths: %s\n", strings.Join(opt.Paths, ", "))
		cyan.Printf("   Max file lines: %d\n", opt.Config.MaxFileLines)
		cyan.Printf("   Max function lines: %d\n", opt.Config.MaxFunctionLines)
		if opt.TypeFilter != "" {
			cyan.Printf("   Filter: %s\n", opt.TypeFilter)
		}
		if opt.Config.UseAI {
			cyan.Println("   AI analysis: enabled")
		}
		cyan.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

		if !promptConfirm("Run refactoring analysis? [y/N]: ") {
			yellow.Println("❌ Analysis cancelled")
			return true
		}
	}

	// Create refactorer
	r := refactor.NewRefactorerWithConfig(opt.Config)

	// Set up LLM client if AI is enabled
	if opt.Config.UseAI {
		r.LLM = &providerLLMAdapter{
			provider:     agent.CurrentProvider,
			model:        agent.CurrentModel,
			systemPrompt: "You are a code refactoring assistant. Analyze code for improvement opportunities. Always respond in valid JSON format.",
		}
		cyan.Println("🤖 AI analysis enabled")
	}

	// Run analysis
	report, err := r.Analyze(opt.Paths)
	if err != nil {
		red.Printf("Analysis failed: %v\n", err)
		return true
	}

	// Filter by type if specified
	if opt.TypeFilter != "" {
		report.Proposals = refactor.FilterByType(report.Proposals, opt.TypeFilter)
	}

	// Display results
	green.Println("\n━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	green.Println("✅ Refactoring Analysis Complete")
	green.Printf("   Files analyzed: %d\n", report.Stats.FilesAnalyzed)
	green.Printf("   Total proposals: %d\n", len(report.Proposals))
	green.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	// Show stats by category
	if report.Stats.LargeFiles > 0 {
		yellow.Printf("   📦 Large files: %d\n", report.Stats.LargeFiles)
	}
	if report.Stats.LongFunctions > 0 {
		yellow.Printf("   📏 Long functions: %d\n", report.Stats.LongFunctions)
	}
	if report.Stats.DuplicateBlocks > 0 {
		yellow.Printf("   🔄 Duplicate blocks: %d\n", report.Stats.DuplicateBlocks)
	}
	if report.Stats.NamingIssues > 0 {
		yellow.Printf("   📝 Naming issues: %d\n", report.Stats.NamingIssues)
	}

	// Show proposals
	if len(report.Proposals) > 0 {
		fmt.Println("\n📋 Proposals:")
		maxShow := 10
		if len(report.Proposals) < maxShow {
			maxShow = len(report.Proposals)
		}

		for i := 0; i < maxShow; i++ {
			p := report.Proposals[i]
			icon := getRefactorIcon(p.Type)
			confidence := fmt.Sprintf("%.0f%%", p.Confidence*100)

			fmt.Printf("\n   %s [%s] %s (%s)\n", icon, p.Type, p.Description, confidence)
			if p.FilePath != "" {
				fmt.Printf("      📄 %s", p.FilePath)
				if p.LineStart > 0 {
					fmt.Printf(":%d", p.LineStart)
					if p.LineEnd > 0 && p.LineEnd != p.LineStart {
						fmt.Printf("-%d", p.LineEnd)
					}
				}
				fmt.Println()
			}
			if p.FunctionName != "" {
				fmt.Printf("      🔹 Function: %s\n", p.FunctionName)
			}
		}

		if len(report.Proposals) > maxShow {
			fmt.Printf("\n   ... and %d more proposals\n", len(report.Proposals)-maxShow)
		}
	} else {
		green.Println("\n✨ No refactoring suggestions - code looks good!")
	}

	// Interactive fix application
	if opt.Fix && report.Stats.ActionableCount > 0 {
		fmt.Println()
		cyan.Printf("🔧 %d actionable fix(es) available\n", report.Stats.ActionableCount)

		changes := refactor.ConvertToMultiFileChanges(report.Proposals)
		if len(changes) > 0 {
			if !opt.AutoApprove {
				fmt.Println("   Press Enter to start interactive fix session, or 'n' to skip:")
				if !promptConfirm("") {
					yellow.Println("   Skipped fix application")
					return true
				}
			}

			result := review.RunInteractiveMultiFixes(changes, opt.AutoApprove)
			fmt.Println()
			green.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
			green.Println("📊 Refactoring Results:")
			green.Printf("   Applied: %d\n", result.Applied)
			if result.Skipped > 0 {
				yellow.Printf("   Skipped: %d\n", result.Skipped)
			}
			if result.Failed > 0 {
				red.Printf("   Failed: %d\n", result.Failed)
			}
			if result.RolledBack {
				yellow.Println("   (Some changes were rolled back)")
			}
			if result.Quit {
				yellow.Println("   (Session ended early)")
			}
			green.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

			// Run tests if changes were applied
			if result.Applied > 0 {
				runRefactorVerification(agent, changes)
			}
		}
	} else if opt.Fix && len(report.Proposals) > 0 {
		fmt.Println()
		yellow.Println("ℹ️  No actionable fixes available (proposals require manual refactoring or AI assistance)")
		yellow.Println("   Try: /refactor --ai --fix to enable AI-powered refactoring")
	}

	return true
}

// runRefactorVerification runs tests after refactoring changes are applied.
func runRefactorVerification(agent *Agent, changes []*review.MultiFileChange) {
	// Collect changed Go files
	var goFiles []string
	seenDirs := make(map[string]bool)

	for _, change := range changes {
		for _, fc := range change.Changes {
			if strings.HasSuffix(fc.FilePath, ".go") {
				dir := filepath.Dir(fc.FilePath)
				if !seenDirs[dir] {
					goFiles = append(goFiles, fc.FilePath)
					seenDirs[dir] = true
				}
			}
		}
	}

	if len(goFiles) == 0 {
		return // No Go files changed
	}

	// Check if this is a Go project
	if !CheckGoModExists() {
		return
	}

	fmt.Println()
	cyan.Println("🧪 Run tests to verify refactoring?")
	fmt.Printf("   Changed Go packages: %d\n", len(seenDirs))
	yellow.Print("   Run go test? [y/N]: ")

	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		return
	}

	input = strings.TrimSpace(strings.ToLower(input))
	if input != "y" && input != "yes" {
		yellow.Println("   Skipped test verification")
		return
	}

	// Run tests for each directory
	allPassed := true
	for dir := range seenDirs {
		cyan.Printf("\n   Testing %s...\n", dir)
		output, passed, _ := RunGoTest(dir + "/test.go") // Use any file in dir

		// Show summary
		lines := strings.Split(output, "\n")
		for _, line := range lines {
			if strings.Contains(line, "PASS") || strings.Contains(line, "ok") {
				green.Println("      " + line)
			} else if strings.Contains(line, "FAIL") {
				red.Println("      " + line)
			}
		}

		if !passed {
			allPassed = false
		}
	}

	fmt.Println()
	if allPassed {
		green.Println("✅ All tests passed - refactoring verified!")
	} else {
		red.Println("❌ Some tests failed")
		yellow.Println("   Consider using /undo to rollback changes")
		agent.suggestRollback()
	}
}

// getRefactorIcon returns an icon for the refactor type.
func getRefactorIcon(t refactor.RefactorType) string {
	switch t {
	case refactor.RefactorSplitFile:
		return "📦"
	case refactor.RefactorExtractMethod:
		return "📏"
	case refactor.RefactorDRY:
		return "🔄"
	case refactor.RefactorRename:
		return "📝"
	default:
		return "🔧"
	}
}
