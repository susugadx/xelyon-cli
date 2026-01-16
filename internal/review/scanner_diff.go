package review

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

// runGit executes a git command and returns the output.
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

// parseNameOnly parses git name-only output into a list of paths.
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

// trimDiffToMaxLines trims a diff to a maximum number of lines.
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

// parseDiffGitHeader parses a "diff --git" header line.
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
