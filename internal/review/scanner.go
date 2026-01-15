package review

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
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

// expandPaths expands glob patterns and directories into file paths.
// Uses git ls-files to respect .gitignore when possible.
func expandPaths(ctx context.Context, paths []string) ([]string, error) {
	seen := make(map[string]struct{})
	result := make([]string, 0)

	for _, pattern := range paths {
		expanded, err := expandSinglePath(ctx, pattern)
		if err != nil {
			return nil, fmt.Errorf("invalid path %q: %w", pattern, err)
		}

		for _, p := range expanded {
			if _, ok := seen[p]; !ok {
				seen[p] = struct{}{}
				result = append(result, p)
			}
		}
	}

	sort.Strings(result)
	return result, nil
}

// expandSinglePath expands a single path pattern.
func expandSinglePath(ctx context.Context, pattern string) ([]string, error) {
	// Check if it's a glob pattern
	if containsGlob(pattern) {
		return expandGlobPattern(ctx, pattern)
	}

	// Check if path exists
	info, err := os.Stat(pattern)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("path does not exist: %s", pattern)
		}
		return nil, err
	}

	// If it's a file, return as-is
	if !info.IsDir() {
		return []string{filepath.Clean(pattern)}, nil
	}

	// If it's a directory, get all tracked files recursively
	return getTrackedFilesInDir(ctx, pattern)
}

// containsGlob checks if a pattern contains glob characters.
func containsGlob(pattern string) bool {
	return strings.ContainsAny(pattern, "*?[")
}

// expandGlobPattern expands a glob pattern.
// Supports ** for recursive matching.
func expandGlobPattern(ctx context.Context, pattern string) ([]string, error) {
	// Handle ** pattern (recursive)
	if strings.Contains(pattern, "**") {
		return expandDoubleStarGlob(ctx, pattern)
	}

	// Standard glob
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return nil, err
	}

	// Filter out directories
	files := make([]string, 0, len(matches))
	for _, m := range matches {
		info, err := os.Stat(m)
		if err != nil {
			continue
		}
		if !info.IsDir() {
			files = append(files, filepath.Clean(m))
		}
	}

	return files, nil
}

// expandDoubleStarGlob handles ** patterns for recursive matching.
// Uses git ls-files with pattern matching when possible.
func expandDoubleStarGlob(ctx context.Context, pattern string) ([]string, error) {
	// Try using git ls-files first (respects .gitignore)
	out, err := runGit(ctx, "ls-files", pattern)
	if err == nil && strings.TrimSpace(out) != "" {
		return parseNameOnly(out), nil
	}

	// Fallback: manual recursive walk
	return walkGlobPattern(pattern)
}

// walkGlobPattern walks the filesystem to match a glob pattern with **.
func walkGlobPattern(pattern string) ([]string, error) {
	// Split pattern at **
	parts := strings.SplitN(pattern, "**", 2)
	if len(parts) != 2 {
		return nil, errors.New("invalid ** pattern")
	}

	baseDir := parts[0]
	if baseDir == "" {
		baseDir = "."
	}
	baseDir = strings.TrimSuffix(baseDir, string(filepath.Separator))

	suffix := strings.TrimPrefix(parts[1], string(filepath.Separator))
	suffix = strings.TrimPrefix(suffix, "/")

	// Check if baseDir exists
	if _, err := os.Stat(baseDir); err != nil {
		if os.IsNotExist(err) {
			return nil, nil // No matches
		}
		return nil, err
	}

	var matches []string
	err := filepath.Walk(baseDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip errors
		}
		if info.IsDir() {
			return nil
		}

		// Check if suffix matches
		if suffix == "" {
			matches = append(matches, filepath.Clean(path))
			return nil
		}

		// Match suffix pattern
		matched, err := filepath.Match(suffix, filepath.Base(path))
		if err != nil {
			return nil
		}
		if matched {
			matches = append(matches, filepath.Clean(path))
		}
		return nil
	})

	return matches, err
}

// getTrackedFilesInDir gets all git-tracked files in a directory.
func getTrackedFilesInDir(ctx context.Context, dir string) ([]string, error) {
	// Try git ls-files first
	out, err := runGit(ctx, "ls-files", dir)
	if err == nil && strings.TrimSpace(out) != "" {
		return parseNameOnly(out), nil
	}

	// Fallback: walk directory
	var files []string
	err = filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			// Skip hidden directories
			if strings.HasPrefix(info.Name(), ".") && info.Name() != "." {
				return filepath.SkipDir
			}
			return nil
		}
		// Skip hidden files
		if strings.HasPrefix(info.Name(), ".") {
			return nil
		}
		files = append(files, filepath.Clean(path))
		return nil
	})

	return files, err
}

// guessChangeTypeFromGit guesses change type from git status.
func guessChangeTypeFromGit(ctx context.Context, path string) string {
	out, err := runGit(ctx, "status", "--porcelain", "--", path)
	if err != nil || strings.TrimSpace(out) == "" {
		return "M" // Default to modified
	}

	// Parse porcelain output: XY path
	line := strings.TrimSpace(out)
	if len(line) < 2 {
		return "M"
	}

	status := line[:2]
	switch {
	case strings.Contains(status, "A") || strings.Contains(status, "?"):
		return "A" // Added or untracked
	case strings.Contains(status, "D"):
		return "D" // Deleted
	case strings.Contains(status, "R"):
		return "R" // Renamed
	default:
		return "M" // Modified
	}
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

// ---- helpers ----

func uniqueFilePaths(changes []tools.FileChange) []string {
	seen := map[string]struct{}{}
	paths := make([]string, 0, len(changes))
	for _, c := range changes {
		p := strings.TrimSpace(c.FilePath)
		if p == "" {
			continue
		}
		// normalize to slash style to match git output more often
		p = filepath.Clean(p)
		if _, ok := seen[p]; ok {
			continue
		}
		seen[p] = struct{}{}
		paths = append(paths, p)
	}
	sort.Strings(paths)
	return paths
}

func guessChangeTypeFromTool(changes []tools.FileChange, path string) string {
	// Newest change wins
	for i := len(changes) - 1; i >= 0; i-- {
		c := changes[i]
		if filepath.Clean(strings.TrimSpace(c.FilePath)) != filepath.Clean(path) {
			continue
		}
		switch c.Tool {
		case "write_file":
			return "A"
		case "delete_file":
			return "D"
		case "move_file":
			return "R"
		default:
			return "M"
		}
	}
	return "M"
}

func runGit(ctx context.Context, args ...string) (string, error) {
	if len(args) == 0 {
		return "", errors.New("no git args")
	}
	cmd := exec.CommandContext(ctx, "git", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		// include output for debugging, but keep it bounded
		msg := strings.TrimSpace(string(out))
		if len(msg) > 2000 {
			msg = msg[:2000] + "..."
		}
		return "", fmt.Errorf("git %s failed: %w: %s", strings.Join(args, " "), err, msg)
	}
	return string(out), nil
}

func parseNameOnly(out string) []string {
	lines := strings.Split(out, "\n")
	paths := make([]string, 0, len(lines))
	seen := map[string]struct{}{}
	for _, line := range lines {
		p := strings.TrimSpace(line)
		if p == "" {
			continue
		}
		p = filepath.Clean(p)
		if _, ok := seen[p]; ok {
			continue
		}
		seen[p] = struct{}{}
		paths = append(paths, p)
	}
	sort.Strings(paths)
	return paths
}

func trimDiffToMaxLines(diff string, maxLines int) string {
	if maxLines <= 0 {
		return diff
	}
	scanner := bufio.NewScanner(strings.NewReader(diff))
	lines := make([]string, 0, maxLines)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
		if len(lines) >= maxLines {
			break
		}
	}
	if len(lines) == 0 {
		return strings.TrimSpace(diff)
	}
	trimmed := strings.Join(lines, "\n")
	// If we truncated, annotate.
	if scanner.Scan() {
		trimmed += "\n... (truncated)"
	}
	return strings.TrimSpace(trimmed)
}

// splitUnifiedDiffByFile splits a unified diff into per-file chunks.
// It is best-effort and relies on standard headers: "diff --git a/... b/...".
func splitUnifiedDiffByFile(patch string) []Target {
	lines := strings.Split(patch, "\n")
	var (
		current     []string
		currentPath string
		currentOld  string
		currentType string
		out         []Target
	)

	flush := func() {
		if currentPath == "" || len(current) == 0 {
			return
		}
		out = append(out, Target{
			Path:       currentPath,
			OldPath:    currentOld,
			ChangeType: currentType,
			Diff:       strings.TrimSpace(strings.Join(current, "\n")),
		})
	}

	for _, line := range lines {
		if strings.HasPrefix(line, "diff --git ") {
			// flush previous
			flush()
			current = []string{line}
			currentPath, currentOld = parseDiffGitHeader(line)
			currentType = "M" // default
			continue
		}
		if current != nil {
			current = append(current, line)
			// best-effort change type hints
			if strings.HasPrefix(line, "new file mode ") {
				currentType = "A"
			} else if strings.HasPrefix(line, "deleted file mode ") {
				currentType = "D"
			} else if strings.HasPrefix(line, "rename from ") {
				currentType = "R"
				currentOld = strings.TrimSpace(strings.TrimPrefix(line, "rename from "))
			} else if strings.HasPrefix(line, "rename to ") {
				currentPath = strings.TrimSpace(strings.TrimPrefix(line, "rename to "))
			}
		}
	}
	flush()
	return out
}

func parseDiffGitHeader(line string) (newPath string, oldPath string) {
	// format: diff --git a/foo b/foo
	parts := strings.Fields(line)
	if len(parts) < 4 {
		return "", ""
	}
	a := strings.TrimPrefix(parts[2], "a/")
	b := strings.TrimPrefix(parts[3], "b/")
	if b != "" {
		newPath = filepath.Clean(b)
	}
	if a != "" {
		oldPath = filepath.Clean(a)
	}
	return newPath, oldPath
}
