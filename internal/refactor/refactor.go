package refactor

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/review"
)

// Refactorer performs code refactoring analysis and application.
type Refactorer struct {
	Config RefactorConfig
	LLM    review.LLMClient // Optional, for AI-powered analysis
}

// NewRefactorer creates a new Refactorer with default config.
func NewRefactorer() *Refactorer {
	return &Refactorer{
		Config: DefaultConfig(),
	}
}

// NewRefactorerWithConfig creates a new Refactorer with custom config.
func NewRefactorerWithConfig(config RefactorConfig) *Refactorer {
	return &Refactorer{
		Config: config,
	}
}

// Analyze performs refactoring analysis on the given paths.
func (r *Refactorer) Analyze(paths []string) (*RefactorReport, error) {
	if len(paths) == 0 {
		return nil, fmt.Errorf("no paths specified")
	}

	// Expand paths and collect files
	files, err := r.collectFiles(paths)
	if err != nil {
		return nil, fmt.Errorf("failed to collect files: %w", err)
	}

	report := &RefactorReport{
		Proposals: []RefactorProposal{},
		Stats: RefactorStats{
			FilesAnalyzed: len(files),
		},
	}

	// Run detectors
	largeFileProposals := DetectLargeFiles(files, r.Config.MaxFileLines)
	report.Proposals = append(report.Proposals, largeFileProposals...)
	report.Stats.LargeFiles = len(largeFileProposals)

	longFuncProposals := DetectLongFunctions(files, r.Config.MaxFunctionLines)
	report.Proposals = append(report.Proposals, longFuncProposals...)
	report.Stats.LongFunctions = len(longFuncProposals)

	duplicateProposals := DetectDuplicateCode(files, r.Config.MinDuplicateLines)
	report.Proposals = append(report.Proposals, duplicateProposals...)
	report.Stats.DuplicateBlocks = len(duplicateProposals)

	namingProposals := DetectPoorNaming(files)
	report.Proposals = append(report.Proposals, namingProposals...)
	report.Stats.NamingIssues = len(namingProposals)

	// AI analysis if enabled
	if r.Config.UseAI && r.LLM != nil {
		aiProposals := r.analyzeWithAI(files)
		report.Proposals = append(report.Proposals, aiProposals...)
	}

	// Update stats
	report.Stats.TotalProposals = len(report.Proposals)
	for _, p := range report.Proposals {
		if p.Actionable {
			report.Stats.ActionableCount++
		}
	}

	return report, nil
}

// collectFiles expands paths (directories, globs) into a list of files.
func (r *Refactorer) collectFiles(paths []string) ([]string, error) {
	var files []string
	seen := make(map[string]bool)

	for _, path := range paths {
		// Check if it's a glob pattern
		if strings.ContainsAny(path, "*?[") {
			matches, err := filepath.Glob(path)
			if err != nil {
				return nil, fmt.Errorf("invalid glob pattern %s: %w", path, err)
			}
			for _, m := range matches {
				if !seen[m] && isSourceFile(m) {
					files = append(files, m)
					seen[m] = true
				}
			}
			continue
		}

		info, err := os.Stat(path)
		if err != nil {
			return nil, fmt.Errorf("path not found: %s", path)
		}

		if info.IsDir() {
			// Walk directory
			err = filepath.Walk(path, func(p string, fi os.FileInfo, err error) error {
				if err != nil {
					return err
				}
				if fi.IsDir() {
					// Skip common non-source directories
					base := filepath.Base(p)
					if base == ".git" || base == "node_modules" || base == "vendor" || base == "dist" || base == "build" {
						return filepath.SkipDir
					}
					return nil
				}
				if !seen[p] && isSourceFile(p) {
					files = append(files, p)
					seen[p] = true
				}
				return nil
			})
			if err != nil {
				return nil, fmt.Errorf("failed to walk directory %s: %w", path, err)
			}
		} else {
			if !seen[path] && isSourceFile(path) {
				files = append(files, path)
				seen[path] = true
			}
		}
	}

	return files, nil
}

// isSourceFile checks if a file is a source code file.
func isSourceFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".go", ".py", ".js", ".ts", ".tsx", ".jsx", ".java", ".rs", ".rb", ".php", ".c", ".cpp", ".h", ".hpp":
		return true
	default:
		return false
	}
}

// analyzeWithAI uses LLM to find additional refactoring opportunities.
func (r *Refactorer) analyzeWithAI(files []string) []RefactorProposal {
	if r.LLM == nil {
		return nil
	}

	var proposals []RefactorProposal

	for _, file := range files {
		content, err := os.ReadFile(file)
		if err != nil {
			continue
		}

		// Only analyze reasonably sized files
		lines := strings.Count(string(content), "\n")
		if lines > 500 {
			continue
		}

		aiProposals, err := AnalyzeWithAI(string(content), file, r.LLM)
		if err != nil {
			continue
		}

		proposals = append(proposals, aiProposals...)
	}

	return proposals
}

// Apply applies a refactoring proposal using the multi-file applier.
func (r *Refactorer) Apply(proposal RefactorProposal) error {
	if !proposal.Actionable || proposal.Change == nil {
		return fmt.Errorf("proposal is not actionable")
	}

	applier := review.NewMultiFileApplier()
	return applier.ApplyMultiFileChange(proposal.Change)
}

// FilterByType filters proposals by refactor type.
func FilterByType(proposals []RefactorProposal, t RefactorType) []RefactorProposal {
	if t == "" {
		return proposals
	}

	var filtered []RefactorProposal
	for _, p := range proposals {
		if p.Type == t {
			filtered = append(filtered, p)
		}
	}
	return filtered
}

// ConvertToMultiFileChanges converts actionable proposals to MultiFileChanges.
func ConvertToMultiFileChanges(proposals []RefactorProposal) []*review.MultiFileChange {
	var changes []*review.MultiFileChange
	for _, p := range proposals {
		if p.Actionable && p.Change != nil {
			changes = append(changes, p.Change)
		}
	}
	return changes
}
