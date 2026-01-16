package review

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/susugadx/xelyon-cli/internal/tools"
)

// ScanOptions is a minimal set of options used by the scanner.
// (The full /review Options will be introduced in subsequent steps.)
//
// Design goals:
// - Default: derive targets from agent changeStack (file paths), then obtain per-file diff snippets.
// - --all: use `git diff` to extract full patch (and optionally name-only list).
// - Keep the scanner self-contained: it should not depend on internal/agent.
//
// NOTE: This package is intentionally decoupled from agent internals.
// Agent passes changeStack (as []tools.FileChange) into the scanner.
// This keeps review package reusable from both interactive command and future cobra subcommand.

type ScanOptions struct {
	All bool
	// Paths filters targets when All==true (git diff). If empty, all changed files.
	Paths []string

	// GitArgs allows customizing the git diff command (mainly for tests or future extension).
	// Example: []string{"diff", "--cached"}
	GitArgs []string

	// MaxSnippetLines limits diff snippet lines for changeStack-based scanning.
	// 0 means default (200).
	MaxSnippetLines int
}

type Scanner struct {
	// If provided, used for timestamps in outputs and testability.
	Now func() time.Time
}

func NewScanner() *Scanner {
	return &Scanner{Now: time.Now}
}

// ScanFromPaths scans files from specified paths (files, directories, or glob patterns).
//
// Behavior:
// - Supports file paths, directory paths (recursive), and glob patterns (e.g., **/*.go)
// - Respects .gitignore via `git ls-files` when possible
// - For each file, tries to get diff from git
// - Returns error if no valid paths found
func (s *Scanner) ScanFromPaths(ctx context.Context, paths []string, opt ScanOptions) ([]Target, error) {
	if len(paths) == 0 {
		return nil, errors.New("no paths specified")
	}

	// Expand paths (glob patterns, directories)
	expandedPaths, err := expandPaths(ctx, paths)
	if err != nil {
		return nil, fmt.Errorf("failed to expand paths: %w", err)
	}

	if len(expandedPaths) == 0 {
		return nil, errors.New("no files found matching the specified paths")
	}

	maxLines := opt.MaxSnippetLines
	if maxLines <= 0 {
		maxLines = 200
	}

	targets := make([]Target, 0, len(expandedPaths))
	for _, p := range expandedPaths {
		// Get diff for this file (best-effort)
		diff, _ := runGit(ctx, "diff", "--", p)
		if strings.TrimSpace(diff) == "" {
			// Try staged diff
			diff, _ = runGit(ctx, "diff", "--cached", "--", p)
		}
		if strings.TrimSpace(diff) == "" {
			// Try HEAD diff
			diff, _ = runGit(ctx, "diff", "HEAD", "--", p)
		}
		diff = trimDiffToMaxLines(diff, maxLines)

		// Guess change type from git status
		ct := guessChangeTypeFromGit(ctx, p)

		targets = append(targets, Target{
			Path:       p,
			ChangeType: ct,
			Diff:       diff,
		})
	}

	return targets, nil
}

// ScanFromChanges derives target file paths from changeStack and tries to gather per-file diff snippets.
//
// Behavior:
// - Targets are unique by path.
// - For each file, tries: `git diff -- <file>`.
// - If that is empty, tries: `git diff HEAD -- <file>` (covers newly created but unstaged states in some cases).
// - If still empty, Diff is left empty (caller can decide how to handle).
func (s *Scanner) ScanFromChanges(ctx context.Context, changeStack []tools.FileChange, opt ScanOptions) ([]Target, error) {
	paths := uniqueFilePaths(changeStack)
	if len(paths) == 0 {
		return nil, nil
	}

	maxLines := opt.MaxSnippetLines
	if maxLines <= 0 {
		maxLines = 200
	}

	targets := make([]Target, 0, len(paths))
	for _, p := range paths {
		// Best-effort change type from tool name
		ct := guessChangeTypeFromTool(changeStack, p)

		diff, _ := runGit(ctx, "diff", "--", p)
		if strings.TrimSpace(diff) == "" {
			// fallback (best-effort)
			diff, _ = runGit(ctx, "diff", "HEAD", "--", p)
		}
		diff = trimDiffToMaxLines(diff, maxLines)

		targets = append(targets, Target{
			Path:       p,
			ChangeType: ct,
			Diff:       diff,
		})
	}

	return targets, nil
}

// ScanAllFromGitDiff uses git diff to obtain patch and extract targets.
//
// Implementation notes:
// - Uses `git diff --name-only` to list files when available (fast).
// - Uses `git diff` to obtain patch; then slices per-file patch by "diff --git" headers.
// - If parsing fails, returns a single Target with Diff containing the entire patch.
func (s *Scanner) ScanAllFromGitDiff(ctx context.Context, opt ScanOptions) ([]Target, error) {
	args := opt.GitArgs
	if len(args) == 0 {
		args = []string{"diff"}
	}

	// name-only list (best-effort)
	nameOnlyArgs := append([]string{}, args...)
	nameOnlyArgs = append(nameOnlyArgs, "--name-only")
	if len(opt.Paths) > 0 {
		nameOnlyArgs = append(nameOnlyArgs, "--")
		nameOnlyArgs = append(nameOnlyArgs, opt.Paths...)
	}
	nameOnly, _ := runGit(ctx, nameOnlyArgs...)
	paths := parseNameOnly(nameOnly)

	// full patch
	patchArgs := append([]string{}, args...)
	if len(opt.Paths) > 0 {
		patchArgs = append(patchArgs, "--")
		patchArgs = append(patchArgs, opt.Paths...)
	}
	patch, err := runGit(ctx, patchArgs...)
	if err != nil {
		return nil, fmt.Errorf("git diff failed: %w", err)
	}
	if strings.TrimSpace(patch) == "" {
		return nil, nil
	}

	parsed := splitUnifiedDiffByFile(patch)
	if len(parsed) == 0 {
		// fallback: single target
		return []Target{{
			Path: "(git diff)",
			Diff: patch,
		}}, nil
	}

	// If name-only list exists, order and filter parsed targets accordingly
	if len(paths) > 0 {
		idx := make(map[string]Target, len(parsed))
		for _, t := range parsed {
			idx[t.Path] = t
		}
		ordered := make([]Target, 0, len(paths))
		for _, p := range paths {
			if t, ok := idx[p]; ok {
				ordered = append(ordered, t)
			}
		}
		if len(ordered) > 0 {
			return ordered, nil
		}
	}

	// deterministic order
	sort.Slice(parsed, func(i, j int) bool { return parsed[i].Path < parsed[j].Path })
	return parsed, nil
}
