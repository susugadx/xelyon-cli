package review

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"
	"time"
)

// FixChoice represents a user's choice for fixing an issue.
type FixChoice string

const (
	FixChoiceYes  FixChoice = "y" // Apply this fix
	FixChoiceNo   FixChoice = "n" // Skip this fix
	FixChoiceAll  FixChoice = "a" // Apply all remaining fixes
	FixChoiceQuit FixChoice = "q" // Stop fixing
)

// FixResult represents the result of an interactive fix session.
type FixResult struct {
	Applied int
	Skipped int
	Failed  int
	Quit    bool
}

// InteractiveFixer handles interactive fix confirmation and application.
type InteractiveFixer struct {
	Reader   *bufio.Reader
	AutoAll  bool // Skip prompts and apply all
	Parallel bool // Enable parallel execution
	Workers  int  // Number of parallel workers (default 4)
}

// NewInteractiveFixer creates a new interactive fixer.
func NewInteractiveFixer() *InteractiveFixer {
	return &InteractiveFixer{
		Reader:  bufio.NewReader(os.Stdin),
		Workers: 4, // Default workers
	}
}

// NewParallelInteractiveFixer creates an interactive fixer with parallel execution enabled.
func NewParallelInteractiveFixer(workers int) *InteractiveFixer {
	if workers <= 0 {
		workers = 4
	}
	return &InteractiveFixer{
		Reader:   bufio.NewReader(os.Stdin),
		Parallel: true,
		Workers:  workers,
	}
}

// PromptFixConfirm displays fix information and prompts for user choice.
func (f *InteractiveFixer) PromptFixConfirm(issue Issue, proposal FixProposal) FixChoice {
	if f.AutoAll {
		return FixChoiceYes
	}

	// Display issue information
	fmt.Println()
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Printf("📌 Issue: %s\n", issue.Title)
	fmt.Printf("   File: %s", issue.Path)
	if issue.LineStart > 0 {
		fmt.Printf(":%d", issue.LineStart)
	}
	fmt.Println()
	fmt.Printf("   Severity: %s\n", issue.Severity)
	fmt.Println()

	// Display fix proposal
	fmt.Printf("🔧 Fix: %s\n", proposal.Title)
	fmt.Printf("   %s\n", proposal.Rationale)
	fmt.Println()

	if proposal.Actionable && proposal.OldCode != "" {
		fmt.Println("   Change:")
		fmt.Printf("   - %s\n", truncateCode(proposal.OldCode, 60))
		fmt.Printf("   + %s\n", truncateCode(proposal.NewCode, 60))
		fmt.Println()
	}

	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Print("Apply fix? [y]es / [n]o / [a]ll / [q]uit: ")

	input, err := f.Reader.ReadString('\n')
	if err != nil {
		return FixChoiceNo
	}

	input = strings.TrimSpace(strings.ToLower(input))
	switch input {
	case "y", "yes":
		return FixChoiceYes
	case "n", "no", "":
		return FixChoiceNo
	case "a", "all":
		return FixChoiceAll
	case "q", "quit":
		return FixChoiceQuit
	default:
		return FixChoiceNo
	}
}

// ApplyFix applies a fix proposal to the file.
func ApplyFix(proposal FixProposal) error {
	if !proposal.Actionable {
		return fmt.Errorf("fix is not actionable (suggestion-only)")
	}

	if proposal.FilePath == "" {
		return fmt.Errorf("fix has no target file path")
	}

	if proposal.OldCode == "" || proposal.NewCode == "" {
		return fmt.Errorf("fix has no replacement code")
	}

	// Read the file
	content, err := os.ReadFile(proposal.FilePath)
	if err != nil {
		return fmt.Errorf("failed to read file %s: %w", proposal.FilePath, err)
	}

	// Check if old code exists
	if !strings.Contains(string(content), proposal.OldCode) {
		return fmt.Errorf("pattern not found in file (may have been modified)")
	}

	// Replace the code
	newContent := strings.Replace(string(content), proposal.OldCode, proposal.NewCode, 1)

	// Write back
	if err := os.WriteFile(proposal.FilePath, []byte(newContent), 0644); err != nil {
		return fmt.Errorf("failed to write file %s: %w", proposal.FilePath, err)
	}

	return nil
}

// RunInteractiveFixes runs the interactive fix session for all actionable proposals.
func RunInteractiveFixes(issues []Issue, proposals []FixProposal, autoApprove bool) FixResult {
	fixer := NewInteractiveFixer()
	fixer.AutoAll = autoApprove

	result := FixResult{}

	// Build issue lookup map
	issueMap := make(map[string]Issue)
	for _, issue := range issues {
		issueMap[issue.ID] = issue
	}

	for _, proposal := range proposals {
		if !proposal.Actionable {
			// Skip non-actionable (suggestion-only) proposals
			continue
		}

		issue, ok := issueMap[proposal.IssueID]
		if !ok {
			result.Skipped++
			continue
		}

		choice := fixer.PromptFixConfirm(issue, proposal)

		switch choice {
		case FixChoiceYes:
			if err := ApplyFix(proposal); err != nil {
				fmt.Printf("❌ Failed to apply fix: %v\n", err)
				result.Failed++
			} else {
				fmt.Printf("✅ Applied: %s\n", proposal.Title)
				result.Applied++
			}
		case FixChoiceNo:
			result.Skipped++
		case FixChoiceAll:
			fixer.AutoAll = true
			if err := ApplyFix(proposal); err != nil {
				fmt.Printf("❌ Failed to apply fix: %v\n", err)
				result.Failed++
			} else {
				fmt.Printf("✅ Applied: %s\n", proposal.Title)
				result.Applied++
			}
		case FixChoiceQuit:
			result.Quit = true
			return result
		}
	}

	return result
}

// truncateCode truncates code for display, adding ellipsis if needed.
func truncateCode(code string, maxLen int) string {
	// Remove newlines for single-line display
	code = strings.ReplaceAll(code, "\n", " ")
	code = strings.ReplaceAll(code, "\t", " ")

	if len(code) <= maxLen {
		return code
	}
	return code[:maxLen-3] + "..."
}

// MultiFixChoice represents a user's choice for multi-file fixes.
type MultiFixChoice string

const (
	MultiFixChoiceApply    MultiFixChoice = "a" // Apply all changes
	MultiFixChoicePreview  MultiFixChoice = "p" // Preview changes
	MultiFixChoiceSkip     MultiFixChoice = "s" // Skip this batch
	MultiFixChoiceQuit     MultiFixChoice = "q" // Quit
	MultiFixChoiceRollback MultiFixChoice = "r" // Rollback after apply
)

// MultiFixResult represents the result of a multi-file fix session.
type MultiFixResult struct {
	Applied    int
	Skipped    int
	Failed     int
	RolledBack bool
	Quit       bool
}

// PromptMultiFileConfirm displays multi-file change information and prompts for user choice.
func (f *InteractiveFixer) PromptMultiFileConfirm(change *MultiFileChange) MultiFixChoice {
	if f.AutoAll {
		return MultiFixChoiceApply
	}

	fmt.Println()
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Printf("📦 Multi-file Change: %s\n", change.Description)
	fmt.Printf("   ID: %s\n", change.ID)
	fmt.Printf("   Files: %d\n", len(change.Changes))
	fmt.Println()

	// Show file list
	for _, fc := range change.Changes {
		fmt.Printf("   • %s (%d patch(es))\n", fc.FilePath, len(fc.Patches))
	}
	fmt.Println()

	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Print("[a]pply / [p]review / [s]kip / [q]uit: ")

	input, err := f.Reader.ReadString('\n')
	if err != nil {
		return MultiFixChoiceSkip
	}

	input = strings.TrimSpace(strings.ToLower(input))
	switch input {
	case "a", "apply":
		return MultiFixChoiceApply
	case "p", "preview":
		return MultiFixChoicePreview
	case "s", "skip", "":
		return MultiFixChoiceSkip
	case "q", "quit":
		return MultiFixChoiceQuit
	default:
		return MultiFixChoiceSkip
	}
}

// PreviewMultiFileChange displays detailed preview of all changes.
func PreviewMultiFileChange(change *MultiFileChange) {
	fmt.Println()
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Printf("📋 Preview: %s\n", change.Description)
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	for _, fc := range change.Changes {
		fmt.Printf("\n📄 %s\n", fc.FilePath)
		for i, patch := range fc.Patches {
			fmt.Printf("   Patch %d:\n", i+1)
			if patch.StartLine > 0 {
				fmt.Printf("   Lines %d-%d:\n", patch.StartLine, patch.EndLine)
			}
			if patch.OldCode != "" {
				fmt.Printf("   - %s\n", truncateCode(patch.OldCode, 60))
			}
			fmt.Printf("   + %s\n", truncateCode(patch.NewCode, 60))
		}
	}
	fmt.Println()
}

// RunInteractiveMultiFixes runs an interactive session for multi-file changes.
func RunInteractiveMultiFixes(changes []*MultiFileChange, autoApprove bool) MultiFixResult {
	fixer := NewInteractiveFixer()
	fixer.AutoAll = autoApprove
	applier := NewMultiFileApplier()

	result := MultiFixResult{}

	for _, change := range changes {
		if len(change.Changes) == 0 {
			continue
		}

	promptLoop:
		for {
			choice := fixer.PromptMultiFileConfirm(change)

			switch choice {
			case MultiFixChoiceApply:
				if err := applier.ApplyMultiFileChange(change); err != nil {
					fmt.Printf("❌ Failed to apply: %v\n", err)
					result.Failed++
				} else {
					fmt.Printf("✅ Applied %d file(s)\n", len(change.Changes))
					result.Applied += len(change.Changes)

					// Offer rollback option
					if !fixer.AutoAll {
						fmt.Print("   [r]ollback / [Enter] to continue: ")
						input, _ := fixer.Reader.ReadString('\n')
						if strings.TrimSpace(strings.ToLower(input)) == "r" {
							if err := applier.RollbackMultiFileChange(change); err != nil {
								fmt.Printf("❌ Rollback failed: %v\n", err)
							} else {
								fmt.Println("↩️  Rolled back")
								result.Applied -= len(change.Changes)
								result.RolledBack = true
							}
						} else {
							// Cleanup backups on success
							applier.CleanupBackups(change)
						}
					} else {
						applier.CleanupBackups(change)
					}
				}
				break promptLoop

			case MultiFixChoicePreview:
				PreviewMultiFileChange(change)
				// Loop back to prompt

			case MultiFixChoiceSkip:
				result.Skipped += len(change.Changes)
				break promptLoop

			case MultiFixChoiceQuit:
				result.Quit = true
				return result
			}
		}
	}

	return result
}

// RunParallelFixes runs fixes in parallel for faster execution.
// autoApprove: if true, skip all prompts and apply all fixes
// workers: number of parallel workers (0 = default 4)
func RunParallelFixes(issues []Issue, proposals []FixProposal, autoApprove bool, workers int) FixResult {
	if workers <= 0 {
		workers = 4
	}

	result := FixResult{}

	// Filter actionable proposals
	actionable := make([]FixProposal, 0)
	for _, p := range proposals {
		if p.Actionable {
			actionable = append(actionable, p)
		}
	}

	if len(actionable) == 0 {
		return result
	}

	// Confirmation prompt for parallel mode
	if !autoApprove {
		fmt.Println()
		fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		fmt.Printf("🚀 Parallel Fix Mode: %d fixes to apply\n", len(actionable))
		fmt.Printf("   Workers: %d\n", workers)
		fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

		// Show file summary
		fileCount := make(map[string]int)
		for _, p := range actionable {
			fileCount[p.FilePath]++
		}
		fmt.Println()
		fmt.Println("📁 Files to modify:")
		for path, count := range fileCount {
			fmt.Printf("   • %s (%d fix(es))\n", path, count)
		}
		fmt.Println()

		fmt.Print("[a]pply all / [q]uit: ")
		reader := bufio.NewReader(os.Stdin)
		input, err := reader.ReadString('\n')
		if err != nil || strings.TrimSpace(strings.ToLower(input)) != "a" {
			result.Quit = true
			return result
		}
	}

	// Group by file
	groups := GroupByFile(actionable)

	// Create parallel executor
	executor := NewParallelExecutor(workers)

	// Apply function
	applyFn := func(ctx context.Context, proposal FixProposal) error {
		return ApplyFix(proposal)
	}

	// Execute in parallel
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	start := time.Now()
	parallelResults := executor.Execute(ctx, groups, applyFn)
	duration := time.Since(start)

	// Convert results
	for _, r := range parallelResults {
		if r.Success {
			result.Applied++
		} else {
			result.Failed++
		}
	}

	// Show summary
	fmt.Println()
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("📊 Parallel Execution Summary")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Printf("   ✅ Applied: %d\n", result.Applied)
	fmt.Printf("   ❌ Failed: %d\n", result.Failed)
	fmt.Printf("   ⏱️  Duration: %v\n", duration.Round(time.Millisecond))
	fmt.Printf("   👷 Workers: %d\n", workers)
	fmt.Println()

	return result
}
