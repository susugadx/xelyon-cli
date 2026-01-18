package agent

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/refactor"
)

// handleRefactorCommand handles the /refactor command for code refactoring analysis.
// Flags: --type, --max-file-lines, --max-func-lines
// Usage: /refactor [flags] [paths...]
func handleRefactorCommand(agent *Agent, args []string) bool {
	opt := refactor.RefactorOptions{
		Config: refactor.DefaultConfig(),
	}

	for i := 0; i < len(args); i++ {
		arg := args[i]

		if strings.HasPrefix(arg, "-") {
			switch arg {
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

	// Show info and confirm before running
	cyan.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	cyan.Println("🔧 Code Refactoring Analysis")
	cyan.Printf("   Paths: %s\n", strings.Join(opt.Paths, ", "))
	cyan.Printf("   Max file lines: %d\n", opt.Config.MaxFileLines)
	cyan.Printf("   Max function lines: %d\n", opt.Config.MaxFunctionLines)
	if opt.TypeFilter != "" {
		cyan.Printf("   Filter: %s\n", opt.TypeFilter)
	}
	cyan.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	if !promptConfirm("Run refactoring analysis? [y/N]: ") {
		yellow.Println("❌ Analysis cancelled")
		return true
	}

	// Create refactorer
	r := refactor.NewRefactorerWithConfig(opt.Config)

	// Show progress
	cyan.Println("\n🔍 Scanning files...")

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

	return true
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
