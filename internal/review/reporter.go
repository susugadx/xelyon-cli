package review

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type ReporterOptions struct {
	OutputDir string
	// FileName can be specified for tests; when empty, auto-generated.
	FileName string

	// Permissions
	DirPerm  os.FileMode
	FilePerm os.FileMode

	// IncludeDiff controls whether diff content is embedded into report.
	// Default true, but some callers may want to omit for safety.
	IncludeDiff bool
}

type Reporter struct {
	Now func() time.Time
}

func NewReporter() *Reporter {
	return &Reporter{Now: time.Now}
}

func (r *Reporter) WriteMarkdown(rep Report, opt ReporterOptions) (string, error) {
	outDir := opt.OutputDir
	if strings.TrimSpace(outDir) == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("failed to determine home dir: %w", err)
		}
		outDir = filepath.Join(home, ".xelyon", "reviews")
	}

	dirPerm := opt.DirPerm
	if dirPerm == 0 {
		dirPerm = 0o700
	}
	filePerm := opt.FilePerm
	if filePerm == 0 {
		filePerm = 0o600
	}

	if err := os.MkdirAll(outDir, dirPerm); err != nil {
		return "", fmt.Errorf("failed to create reviews dir %s: %w", outDir, err)
	}

	fileName := strings.TrimSpace(opt.FileName)
	if fileName == "" {
		ts := rep.GeneratedAt
		if ts.IsZero() {
			ts = r.Now()
		}
		fileName = fmt.Sprintf("%s.md", ts.Format("20060102_150405"))
	}

	outPath := filepath.Join(outDir, fileName)
	content := buildMarkdown(rep, opt)

	// Write atomically (best-effort): write temp then rename.
	tmp := outPath + ".tmp"
	if err := os.WriteFile(tmp, []byte(content), filePerm); err != nil {
		return "", fmt.Errorf("failed to write review report: %w", err)
	}
	if err := os.Rename(tmp, outPath); err != nil {
		_ = os.Remove(tmp)
		return "", fmt.Errorf("failed to finalize review report: %w", err)
	}

	return outPath, nil
}

func buildMarkdown(rep Report, opt ReporterOptions) string {
	includeDiff := opt.IncludeDiff
	// default true
	if !opt.IncludeDiff {
		includeDiff = false
	} else {
		includeDiff = true
	}

	gen := rep.GeneratedAt
	if gen.IsZero() {
		gen = time.Now()
	}

	sevCounts := map[Severity]int{}
	for _, is := range rep.Issues {
		sevCounts[is.Severity]++
	}

	b := &strings.Builder{}
	b.WriteString("# XELYON Review Report\n\n")
	b.WriteString(fmt.Sprintf("- Generated: %s\n", gen.UTC().Format(time.RFC3339)))
	if rep.Mode != "" {
		b.WriteString(fmt.Sprintf("- Mode: %s\n", rep.Mode))
	}
	if rep.Provider != "" || rep.Model != "" {
		b.WriteString(fmt.Sprintf("- Provider/Model: %s / %s\n", blankIf(rep.Provider, "(unknown)"), blankIf(rep.Model, "(unknown)")))
	}
	b.WriteString(fmt.Sprintf("- Targets: %d files\n", len(rep.Targets)))
	b.WriteString(fmt.Sprintf("- Issues: %d (error: %d, warning: %d, info: %d)\n\n",
		len(rep.Issues), sevCounts[SeverityError], sevCounts[SeverityWarning], sevCounts[SeverityInfo]))

	b.WriteString("## Summary\n")
	if strings.TrimSpace(rep.Summary) == "" {
		b.WriteString("(No summary)\n\n")
	} else {
		b.WriteString(rep.Summary)
		if !strings.HasSuffix(rep.Summary, "\n") {
			b.WriteString("\n")
		}
		b.WriteString("\n")
	}

	b.WriteString("## Targets\n")
	b.WriteString("| File | Type | Issues | Diff hash |\n")
	b.WriteString("|---|---:|---:|---:|\n")

	// issues per file
	issueCount := map[string]int{}
	for _, is := range rep.Issues {
		issueCount[is.Path]++
	}

	targets := make([]Target, 0, len(rep.Targets))
	targets = append(targets, rep.Targets...)
	sort.Slice(targets, func(i, j int) bool { return targets[i].Path < targets[j].Path })
	for _, t := range targets {
		h := diffHash(t.Diff)
		b.WriteString(fmt.Sprintf("| %s | %s | %d | %s |\n",
			escapeMD(t.Path), escapeMD(blankIf(t.ChangeType, "-")), issueCount[t.Path], h))
	}
	b.WriteString("\n")

	b.WriteString("## Priority\n")
	b.WriteString("- [error] Fix before merge\n")
	b.WriteString("- [warning] Should address or document rationale\n")
	b.WriteString("- [info] Nice-to-have / follow-up\n\n")

	b.WriteString("## Issues\n\n")
	if len(rep.Issues) == 0 {
		b.WriteString("No issues found.\n\n")
	} else {
		issues := make([]Issue, 0, len(rep.Issues))
		issues = append(issues, rep.Issues...)
		sort.Slice(issues, func(i, j int) bool {
			if severityRank(issues[i].Severity) != severityRank(issues[j].Severity) {
				return severityRank(issues[i].Severity) > severityRank(issues[j].Severity)
			}
			if issues[i].Path != issues[j].Path {
				return issues[i].Path < issues[j].Path
			}
			return issues[i].ID < issues[j].ID
		})

		for _, is := range issues {
			b.WriteString(fmt.Sprintf("### [%s] %s\n", is.Severity, escapeMD(is.Title)))
			b.WriteString(fmt.Sprintf("- ID: %s\n", escapeMD(is.ID)))
			b.WriteString(fmt.Sprintf("- File: %s\n", escapeMD(is.Path)))
			if is.LineStart > 0 {
				if is.LineEnd > 0 && is.LineEnd != is.LineStart {
					b.WriteString(fmt.Sprintf("- Lines: %d-%d\n", is.LineStart, is.LineEnd))
				} else {
					b.WriteString(fmt.Sprintf("- Line: %d\n", is.LineStart))
				}
			}
			if len(is.Tags) > 0 {
				b.WriteString(fmt.Sprintf("- Tags: %s\n", escapeMD(strings.Join(is.Tags, ", "))))
			}
			b.WriteString("\n")

			if strings.TrimSpace(is.Description) != "" {
				b.WriteString("**What / Why**\n\n")
				b.WriteString(is.Description)
				if !strings.HasSuffix(is.Description, "\n") {
					b.WriteString("\n")
				}
				b.WriteString("\n")
			}

			if strings.TrimSpace(is.Snippet) != "" {
				b.WriteString("**Snippet**\n\n")
				b.WriteString("```\n")
				b.WriteString(strings.TrimSpace(is.Snippet))
				b.WriteString("\n```\n\n")
			} else if includeDiff {
				// If snippet is empty and we allow diff, try to include the target diff (bounded)
				if td := findTargetDiff(targets, is.Path); strings.TrimSpace(td) != "" {
					b.WriteString("**Diff (context)**\n\n")
					b.WriteString("```diff\n")
					b.WriteString(trimLines(td, 200))
					b.WriteString("\n```\n\n")
				}
			}

			if len(is.Suggestions) > 0 {
				b.WriteString("**Suggestions**\n\n")
				for _, s := range is.Suggestions {
					b.WriteString(fmt.Sprintf("- %s\n", escapeMD(s)))
				}
				b.WriteString("\n")
			}

			b.WriteString("---\n\n")
		}
	}

	if len(rep.Fixes) > 0 {
		b.WriteString("## Fix Proposals (--fix)\n\n")
		fixes := make([]FixProposal, 0, len(rep.Fixes))
		fixes = append(fixes, rep.Fixes...)
		sort.Slice(fixes, func(i, j int) bool { return fixes[i].IssueID < fixes[j].IssueID })

		for _, fx := range fixes {
			b.WriteString(fmt.Sprintf("### %s\n", escapeMD(fx.Title)))
			b.WriteString(fmt.Sprintf("- Issue: %s\n\n", escapeMD(fx.IssueID)))

			if strings.TrimSpace(fx.Rationale) != "" {
				b.WriteString("**Rationale**\n\n")
				b.WriteString(fx.Rationale)
				if !strings.HasSuffix(fx.Rationale, "\n") {
					b.WriteString("\n")
				}
				b.WriteString("\n")
			}

			if strings.TrimSpace(fx.Example) != "" {
				b.WriteString("**Example**\n\n")
				b.WriteString("```\n")
				b.WriteString(strings.TrimSpace(fx.Example))
				b.WriteString("\n```\n\n")
			}

			if strings.TrimSpace(fx.Patch) != "" {
				b.WriteString("**Patch (proposal)**\n\n")
				b.WriteString("```diff\n")
				b.WriteString(strings.TrimSpace(fx.Patch))
				b.WriteString("\n```\n\n")
			}

			if len(fx.Notes) > 0 {
				b.WriteString("**Notes**\n\n")
				for _, n := range fx.Notes {
					b.WriteString(fmt.Sprintf("- %s\n", escapeMD(n)))
				}
				b.WriteString("\n")
			}

			b.WriteString("---\n\n")
		}
	}

	b.WriteString("## Appendix\n\n")
	b.WriteString("Generated by XELYON CLI.\n")
	return b.String()
}

func findTargetDiff(targets []Target, path string) string {
	for _, t := range targets {
		if t.Path == path {
			return t.Diff
		}
	}
	return ""
}

func trimLines(s string, max int) string {
	if max <= 0 {
		max = 200
	}
	lines := strings.Split(s, "\n")
	if len(lines) <= max {
		return strings.TrimSpace(s)
	}
	return strings.TrimSpace(strings.Join(lines[:max], "\n")) + "\n... (truncated)"
}

func diffHash(diff string) string {
	if strings.TrimSpace(diff) == "" {
		return "-"
	}
	sum := sha256.Sum256([]byte(diff))
	// short hash for table
	return hex.EncodeToString(sum[:])[:10]
}

func escapeMD(s string) string {
	// minimal escaping for tables/inline values
	s = strings.ReplaceAll(s, "|", "\\|")
	s = strings.ReplaceAll(s, "\n", " ")
	return s
}

func blankIf(s, fallback string) string {
	if strings.TrimSpace(s) == "" {
		return fallback
	}
	return s
}
