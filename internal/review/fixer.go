package review

import (
	"fmt"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

// Fixer generates fix proposals for issues.
// IMPORTANT: This is suggestion-only. It does not modify any files.
//
// In MVP we implement a rule-based fixer that provides:
// - short rationale
// - example snippet (when possible)
// - best-effort unified diff patch (optional)
//
// Future: LLM-based fixer can be added behind the same interface.

type Fixer struct {
	Now func() time.Time
}

func NewFixer() *Fixer {
	return &Fixer{Now: time.Now}
}

type FixOptions struct {
	// MaxFixes caps proposals.
	// 0 means default (50).
	MaxFixes int
}

func (f *Fixer) Propose(issues []Issue, opt FixOptions) ([]FixProposal, error) {
	maxFixes := opt.MaxFixes
	if maxFixes <= 0 {
		maxFixes = 50
	}

	proposals := make([]FixProposal, 0, min(len(issues), maxFixes))
	for _, is := range issues {
		if len(proposals) >= maxFixes {
			break
		}

		p := proposeForIssue(is)
		if p.IssueID == "" {
			// skip unknown/unhandled
			continue
		}
		proposals = append(proposals, p)
	}

	// deterministic order
	sort.Slice(proposals, func(i, j int) bool {
		if proposals[i].IssueID != proposals[j].IssueID {
			return proposals[i].IssueID < proposals[j].IssueID
		}
		return proposals[i].Title < proposals[j].Title
	})

	return proposals, nil
}

func proposeForIssue(is Issue) FixProposal {
	// We match either by known rule IDs (preferred) or tags.
	// Our analyzer uses issueID("rule", path, line) -> rule:path[:line]
	// So we can detect rule prefix before the first ':'
	rule := is.ID
	if idx := strings.Index(rule, ":"); idx > 0 {
		rule = rule[:idx]
	}

	switch rule {
	case "general-large-diff":
		return FixProposal{
			IssueID:   is.ID,
			Title:     "Split change into smaller commits/PRs",
			Rationale: "Large diffs are difficult to review and increase the chance of regressions.",
			Example:   "Suggested approach:\n- Split refactor vs behavior change\n- Add/adjust tests per logical change\n- Add a short PR summary and verification steps",
			Notes:     []string{"No automated patch for this item", "Consider interactive rebase/squash"},
		}

	case "general-todo-added":
		return FixProposal{
			IssueID:   is.ID,
			Title:     "Track TODO/FIXME and add context",
			Rationale: "Untracked TODOs tend to become permanent. Adding context and a link helps maintenance.",
			Example:   "Example:\n// TODO(#123): Explain why this is needed and what blocks it\n// FIXME(#123): ...",
			Notes:     []string{"Prefer linking to GitHub Issues", "Avoid committing temporary hacks without follow-up"},
		}

	case "general-go-exported-doc":
		sym := guessGoExportedSymbolFromDiff(is.Snippet)
		if sym == "" {
			sym = "<SymbolName>"
		}
		return FixProposal{
			IssueID:   is.ID,
			Title:     "Add doc comment for exported Go symbol",
			Rationale: "Doc comments improve readability and help generated documentation; they also align with Go conventions.",
			Example:   fmt.Sprintf("// %s ...\n%s", sym, ""),
			Notes:     []string{"Doc comment should start with the symbol name", "Keep it concise and behavior-focused"},
		}

	case "sec-shell-exec":
		return FixProposal{
			IssueID:   is.ID,
			Title:     "Harden command execution (avoid injection)",
			Rationale: "Shell execution is a common injection vector when any part of the command is influenced by external input.",
			Example: strings.TrimSpace(`Prefer:
exec.Command("git", "diff", "--", file)

Avoid:
exec.Command("sh", "-c", userInput)

If inputs are variable:
- validate with allowlist (regex)
- avoid concatenation
- pass args as separate parameters`),
			Notes: []string{"Never pass user-controlled strings to 'sh -c'", "Add validation and tests for malicious payloads"},
		}

	case "sec-crypto-weak":
		return FixProposal{
			IssueID:   is.ID,
			Title:     "Use crypto-safe primitives",
			Rationale: "MD5/SHA1 and math/rand are not suitable for security-sensitive use.",
			Example: strings.TrimSpace(`For tokens/IDs:
- use crypto/rand
- encode with hex/base64

For hashes:
- use SHA-256+ when security matters

Go example:
import "crypto/rand"

b := make([]byte, 32)
if _, err := rand.Read(b); err != nil { /* handle */ }`),
			Notes: []string{"If used only for non-security (e.g., checksum), document that clearly"},
		}

	case "sec-http-timeout":
		proposal := FixProposal{
			IssueID:   is.ID,
			Title:     "Set HTTP client timeout",
			Rationale: "Timeouts prevent hangs and reduce risk of resource exhaustion.",
			Example: strings.TrimSpace(`client := &http.Client{
	Timeout: 30 * time.Second,
}

req, err := http.NewRequestWithContext(ctx, method, url, body)
// ...`),
			Notes:    []string{"Also consider per-request context deadlines", "Align timeout with expected API behavior"},
			FilePath: is.Path,
		}
		// Try to generate actionable fix for &http.Client{} -> &http.Client{Timeout: 30*time.Second}
		if oldCode, newCode := generateHTTPTimeoutFix(is.Snippet); oldCode != "" {
			proposal.Actionable = true
			proposal.OldCode = oldCode
			proposal.NewCode = newCode
		}
		return proposal

	case "sec-path-traversal":
		return FixProposal{
			IssueID:   is.ID,
			Title:     "Add strict path validation (prevent traversal)",
			Rationale: "User-influenced paths can escape intended directories if not validated.",
			Example: strings.TrimSpace(`Example approach:
- resolve to absolute path
- ensure it stays under allowed root
- reject '..' segments and absolute paths

Go snippet:
clean := filepath.Clean(input)
if strings.Contains(clean, "..") { return err }
abs := filepath.Join(root, clean)
if !strings.HasPrefix(abs, root+string(os.PathSeparator)) { return err }`),
			Notes: []string{"Prefer central ValidatePath() if project already has it", "Add tests for traversal attempts"},
		}

	case "test-coverage-check":
		return FixProposal{
			IssueID:   is.ID,
			Title:     "Add or update tests for changed behavior",
			Rationale: "Tests reduce regressions and clarify intended behavior.",
			Example:   "Suggested steps:\n- Identify affected functions\n- Add table-driven tests\n- Add error-path tests\n- Run: go test ./...",
			Notes:     []string{"No automated patch for this item"},
		}

	case "test-no-assertions":
		return FixProposal{
			IssueID:   is.ID,
			Title:     "Ensure tests assert expected behavior",
			Rationale: "Tests without assertions can pass without validating anything.",
			Example: strings.TrimSpace(`Examples:
- if got != want { t.Fatalf("got %v want %v", got, want) }
- require.NoError(t, err)
- assert.Equal(t, want, got)`),
			Notes: []string{"Prefer require/assert helpers where already used"},
		}
	}

	// Tag-based fallbacks
	if hasTag(is.Tags, "security") {
		return FixProposal{
			IssueID:   is.ID,
			Title:     "Review security impact and add validation",
			Rationale: "Security-related changes benefit from explicit validation and tests.",
			Example:   "Checklist:\n- Validate inputs\n- Add allowlists\n- Avoid logging secrets\n- Add negative tests",
			Notes:     []string{"Suggestion-only"},
		}
	}

	return FixProposal{}
}

func guessGoExportedSymbolFromDiff(snippet string) string {
	re := regexp.MustCompile(`(?m)^\+\s*(func|type)\s+([A-Z]\w*)`)
	m := re.FindStringSubmatch(snippet)
	if len(m) >= 3 {
		return m[2]
	}
	return ""
}

// generateHTTPTimeoutFix generates an actionable fix for &http.Client{} patterns.
// Returns (oldCode, newCode) if a fix can be generated, empty strings otherwise.
func generateHTTPTimeoutFix(snippet string) (string, string) {
	// Match patterns like:
	// &http.Client{}
	// http.Client{}
	// &http.Client{Transport: ...} (no Timeout field)
	lines := strings.Split(snippet, "\n")
	for _, line := range lines {
		// Remove diff prefix (+/-)
		clean := strings.TrimPrefix(strings.TrimPrefix(line, "+"), "-")
		clean = strings.TrimSpace(clean)

		// Pattern 1: &http.Client{} or http.Client{}
		if strings.Contains(clean, "http.Client{}") {
			oldCode := "http.Client{}"
			newCode := "http.Client{Timeout: 30 * time.Second}"
			return oldCode, newCode
		}

		// Pattern 2: &http.Client{...} without Timeout
		// This is more complex and we skip for now to avoid breaking existing code
	}
	return "", ""
}

func hasTag(tags []string, want string) bool {
	for _, t := range tags {
		if t == want {
			return true
		}
	}
	return false
}

// BuildPatchSkeleton produces a minimal patch skeleton. Not applied.
// This helper is currently unused but kept for future extension.
func BuildPatchSkeleton(path string, note string) string {
	p := filepath.ToSlash(filepath.Clean(path))
	return fmt.Sprintf("diff --git a/%[1]s b/%[1]s\n--- a/%[1]s\n+++ b/%[1]s\n@@\n// %[2]s\n", p, note)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// AIFixer extends Fixer with AI-powered fix generation.
type AIFixer struct {
	*Fixer
	AIAnalyzer *AIAnalyzer
}

// NewAIFixer creates a new AI-powered fixer.
func NewAIFixer(llm LLMClient) *AIFixer {
	return &AIFixer{
		Fixer:      NewFixer(),
		AIAnalyzer: NewAIAnalyzer(llm),
	}
}

// ProposeWithAI generates fix proposals, using AI for issues that
// don't have static rule-based fixes.
func (f *AIFixer) ProposeWithAI(issues []Issue, fileContents map[string]string, opt FixOptions) ([]FixProposal, error) {
	maxFixes := opt.MaxFixes
	if maxFixes <= 0 {
		maxFixes = 50
	}

	proposals := make([]FixProposal, 0, min(len(issues), maxFixes))

	for _, is := range issues {
		if len(proposals) >= maxFixes {
			break
		}

		// First try static rule-based fix
		p := proposeForIssue(is)
		if p.IssueID != "" {
			proposals = append(proposals, p)
			continue
		}

		// If static fix not available and AI is configured, try AI fix
		if f.AIAnalyzer != nil && f.AIAnalyzer.LLM != nil {
			code, ok := fileContents[is.Path]
			if !ok {
				continue
			}

			aiProposal, err := f.AIAnalyzer.GenerateAIFix(is, code)
			if err != nil {
				// AI fix failed, skip this issue
				continue
			}

			proposals = append(proposals, *aiProposal)
		}
	}

	// Sort by IssueID for deterministic order
	sort.Slice(proposals, func(i, j int) bool {
		if proposals[i].IssueID != proposals[j].IssueID {
			return proposals[i].IssueID < proposals[j].IssueID
		}
		return proposals[i].Title < proposals[j].Title
	})

	return proposals, nil
}
