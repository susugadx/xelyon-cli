package agent

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/susugadx/xelyon-cli/internal/pathmatch"
	"github.com/susugadx/xelyon-cli/internal/repomap"
)

func ensureProjectMap(agent *Agent, rootPath string, ignorePatterns []string, ignoreKey string) (*repomap.ProjectMap, bool) {
	if agent == nil {
		return nil, false
	}

	if !agent.projectMapDirty &&
		agent.projectMap != nil &&
		agent.projectMapRootPath == rootPath &&
		agent.projectMapIgnoreKey == ignoreKey {
		if stateKey := currentProjectMapStateKey(agent, rootPath); stateKey != "" && agent.projectMapStateKey == stateKey {
			return agent.projectMap, false
		}
	}

	pm := repomap.NewProjectMap(rootPath, 0, ignorePatterns...)
	if err := pm.Build(); err != nil {
		yellow.Fprintf(agent.output(), "⚠️ ProjectMap build failed: %v\n", err)
		return nil, false
	}

	agent.projectMap = pm
	agent.projectMapRootPath = rootPath
	agent.projectMapIgnoreKey = ignoreKey
	agent.projectMapWatchDirs = nil
	if !isGitProjectMapAvailable(rootPath) {
		agent.projectMapWatchDirs = collectProjectMapWatchDirs(rootPath, ignorePatterns)
	}
	agent.projectMapStateKey = currentProjectMapStateKey(agent, rootPath)
	return pm, true
}

func currentProjectMapStateKey(agent *Agent, rootPath string) string {
	head := gitProjectMapHEAD(rootPath)
	status := gitProjectMapStatusDigest(rootPath)
	if head != "" || status != "" {
		return head + ":" + status
	}

	if digest := nonGitProjectMapWatchDigest(rootPath, projectMapWatchDirs(agent), projectMapIgnorePatterns(agent)); digest != "" {
		return "dirs:" + digest
	}

	info, err := os.Stat(rootPath)
	if err != nil {
		return ""
	}
	return fmt.Sprintf("fs:%d", info.ModTime().UTC().UnixNano())
}

func gitProjectMapHEAD(rootPath string) string {
	return gitProjectMapCommandDigest(rootPath, []string{"rev-parse", "HEAD"})
}

func gitProjectMapStatusDigest(rootPath string) string {
	return gitProjectMapCommandDigest(rootPath, []string{"status", "--porcelain"})
}

func gitProjectMapCommandDigest(rootPath string, args []string) string {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = rootPath

	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	if err := cmd.Run(); err != nil {
		return ""
	}

	sum := sha256.Sum256(bytes.TrimSpace(stdout.Bytes()))
	return hex.EncodeToString(sum[:])
}

func isGitProjectMapAvailable(rootPath string) bool {
	return gitProjectMapHEAD(rootPath) != "" || gitProjectMapStatusDigest(rootPath) != ""
}

func projectMapWatchDirs(agent *Agent) []string {
	if agent == nil || len(agent.projectMapWatchDirs) == 0 {
		return []string{"."}
	}

	dirs := make([]string, len(agent.projectMapWatchDirs))
	copy(dirs, agent.projectMapWatchDirs)
	return dirs
}

func projectMapIgnorePatterns(agent *Agent) []string {
	if agent == nil || agent.projectMapIgnoreKey == "" {
		return nil
	}
	return pathmatch.NormalizePatterns(strings.Split(agent.projectMapIgnoreKey, "\x00"))
}

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
