package review

import (
	"bufio"
	"fmt"
	"os"
	"strings"
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
	Reader  *bufio.Reader
	AutoAll bool // Skip prompts and apply all
}

// NewInteractiveFixer creates a new interactive fixer.
func NewInteractiveFixer() *InteractiveFixer {
	return &InteractiveFixer{
		Reader: bufio.NewReader(os.Stdin),
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
