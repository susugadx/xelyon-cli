package agent

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/pathmatch"
)

func nonGitProjectMapWatchDigest(rootPath string, watchDirs []string, ignorePatterns []string) string {
	if len(watchDirs) == 0 {
		return ""
	}

	matcher := pathmatch.NewMatcher(ignorePatterns)
	var state strings.Builder
	for _, relDir := range watchDirs {
		relDir = filepath.Clean(filepath.ToSlash(strings.TrimSpace(relDir)))
		if relDir == "" {
			relDir = "."
		}

		absDir := rootPath
		if relDir != "." {
			absDir = filepath.Join(rootPath, relDir)
		}

		entries, err := os.ReadDir(absDir)
		switch {
		case err != nil:
			state.WriteString(relDir)
			state.WriteString(":missing\n")
		default:
			filtered := 0
			var entryState strings.Builder
			for _, entry := range entries {
				entryRelPath := entry.Name()
				if relDir != "." {
					entryRelPath = filepath.ToSlash(filepath.Join(relDir, entry.Name()))
				}
				if matcher.Match(entryRelPath, entry.IsDir()) {
					continue
				}
				filtered++
				entryState.WriteString(entry.Name())
				if entry.IsDir() {
					entryState.WriteByte('/')
				}
				entryState.WriteByte('\n')
			}
			_, _ = fmt.Fprintf(&state, "%s:%d\n", relDir, filtered)
			state.WriteString(entryState.String())
		}
	}

	sum := sha256.Sum256([]byte(state.String()))
	return hex.EncodeToString(sum[:])
}

func collectProjectMapWatchDirs(rootPath string, ignorePatterns []string) []string {
	matcher := pathmatch.NewMatcher(ignorePatterns)
	dirs := []string{"."}

	_ = filepath.WalkDir(rootPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			if d != nil && d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if !d.IsDir() {
			return nil
		}

		relPath, relErr := filepath.Rel(rootPath, path)
		if relErr != nil {
			return nil
		}
		relPath = filepath.Clean(filepath.ToSlash(relPath))
		if relPath == "." {
			return nil
		}
		if matcher.Match(relPath, true) {
			return filepath.SkipDir
		}

		dirs = append(dirs, relPath)
		return nil
	})

	slices.Sort(dirs)
	return slices.Compact(dirs)
}
