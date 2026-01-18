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

		// Also make static detection proposals actionable using AI
		r.makeStaticProposalsActionable(report.Proposals, files)
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

// makeStaticProposalsActionable uses AI to generate fixes for static detection proposals.
func (r *Refactorer) makeStaticProposalsActionable(proposals []RefactorProposal, files []string) {
	if r.LLM == nil {
		return
	}

	// Build file content cache
	fileContents := make(map[string]string)
	for _, file := range files {
		content, err := os.ReadFile(file)
		if err != nil {
			continue
		}
		fileContents[file] = string(content)
	}

	// Process proposals that aren't already actionable
	for i := range proposals {
		p := &proposals[i]
		if p.Actionable {
			continue
		}

		content, ok := fileContents[p.FilePath]
		if !ok {
			continue
		}

		r.makeProposalActionable(p, content)
	}
}

// analyzeWithAI uses LLM to find additional refactoring opportunities.
func (r *Refactorer) analyzeWithAI(files []string) []RefactorProposal {
	if r.LLM == nil {
		return nil
	}

	var proposals []RefactorProposal

	// Filter files suitable for AI analysis (< 500 lines)
	var aiTargets []string
	for _, file := range files {
		content, err := os.ReadFile(file)
		if err != nil {
			continue
		}
		lines := strings.Count(string(content), "\n")
		if lines <= 500 {
			aiTargets = append(aiTargets, file)
		}
	}

	// Apply MaxAIFiles limit
	if r.Config.MaxAIFiles > 0 && len(aiTargets) > r.Config.MaxAIFiles {
		fmt.Printf("⚠️  AI analysis limited to %d files (found %d eligible files)\n", r.Config.MaxAIFiles, len(aiTargets))
		fmt.Printf("   Use --max-ai-files to change the limit\n")
		aiTargets = aiTargets[:r.Config.MaxAIFiles]
	}

	// Show progress for AI analysis
	if len(aiTargets) > 0 {
		fmt.Printf("🤖 Analyzing %d file(s) with AI (this may incur API costs)...\n", len(aiTargets))
	}

	for i, file := range aiTargets {
		content, err := os.ReadFile(file)
		if err != nil {
			continue
		}

		// Show progress
		fmt.Printf("   [%d/%d] %s\n", i+1, len(aiTargets), file)

		aiProposals, err := AnalyzeWithAI(string(content), file, r.LLM)
		if err != nil {
			continue
		}

		// Generate actionable changes for each proposal
		for j := range aiProposals {
			p := &aiProposals[j]
			r.makeProposalActionable(p, string(content))
		}

		proposals = append(proposals, aiProposals...)
	}

	return proposals
}

// makeProposalActionable generates actual code changes for a proposal.
func (r *Refactorer) makeProposalActionable(p *RefactorProposal, fileContent string) {
	if r.LLM == nil {
		return
	}

	switch p.Type {
	case RefactorSplitFile:
		change, err := GenerateSplitFilePlan(fileContent, p.FilePath, r.LLM)
		if err == nil && change != nil && len(change.Changes) > 0 {
			p.Change = change
			p.Actionable = true
		}

	case RefactorExtractMethod:
		if p.FunctionName == "" {
			return
		}
		// Extract the function code from the file
		funcCode := extractFunctionCode(fileContent, p.FilePath, p.FunctionName, p.LineStart, p.LineEnd)
		if funcCode == "" {
			return
		}
		change, err := GenerateExtractMethodCode(funcCode, p.FunctionName, p.FilePath, r.LLM)
		if err == nil && change != nil && len(change.Changes) > 0 {
			p.Change = change
			p.Actionable = true
		}
	}
}

// extractFunctionCode extracts function code from file content.
func extractFunctionCode(content string, filePath string, funcName string, lineStart, lineEnd int) string {
	lines := strings.Split(content, "\n")

	// Use line range if provided
	if lineStart > 0 && lineEnd > 0 && lineEnd <= len(lines) {
		return strings.Join(lines[lineStart-1:lineEnd], "\n")
	}

	// Otherwise try to find the function by name
	ext := strings.ToLower(filepath.Ext(filePath))
	var funcPattern string
	switch ext {
	case ".go":
		funcPattern = "func " + funcName
	case ".py":
		funcPattern = "def " + funcName
	case ".js", ".ts", ".jsx", ".tsx":
		funcPattern = "function " + funcName
	default:
		return ""
	}

	// Find function start
	start := -1
	for i, line := range lines {
		if strings.Contains(line, funcPattern) {
			start = i
			break
		}
	}
	if start == -1 {
		return ""
	}

	// Find function end (brace counting)
	depth := 0
	end := start
	for i := start; i < len(lines); i++ {
		line := lines[i]
		depth += strings.Count(line, "{") - strings.Count(line, "}")
		if depth == 0 && i > start {
			end = i
			break
		}
		end = i
	}

	return strings.Join(lines[start:end+1], "\n")
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
