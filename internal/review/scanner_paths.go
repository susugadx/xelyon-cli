package review

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

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
