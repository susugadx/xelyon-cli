package review

import (
	"context"
	"path/filepath"
	"sort"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/tools"
)

// uniqueFilePaths extracts unique file paths from a change stack.
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

// guessChangeTypeFromTool guesses change type from the tool name in change stack.
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
