package cache

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// RepoMapCacheEntry is a persisted repo map cache.
//
// Key is derived from project path (hashed for filename safety).
// Fingerprint is a best-effort project change detector (max modTime across files).
//
// NOTE: This is intentionally conservative: if fingerprint mismatches, cache is ignored.
// It is not cryptographically strong; it is intended for speed, not security.
type RepoMapCacheEntry struct {
	ProjectPath string `json:"project_path"`
	Fingerprint int64  `json:"fingerprint"` // UnixNano
	RepoMap     string `json:"repo_map"`
}

var ErrRepoMapCacheMiss = errors.New("repomap cache: miss")

// LoadRepoMapFromDisk loads a cached repo map if fingerprint matches.
func LoadRepoMapFromDisk(projectPath string, fingerprint int64) (string, error) {
	path, err := repoMapCacheFilePath(projectPath)
	if err != nil {
		return "", err
	}

	b, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", ErrRepoMapCacheMiss
		}
		return "", fmt.Errorf("repomap cache: read failed: %w", err)
	}

	var entry RepoMapCacheEntry
	if err := json.Unmarshal(b, &entry); err != nil {
		return "", fmt.Errorf("repomap cache: parse failed: %w", err)
	}

	// Ensure the cache belongs to the same project path and fingerprint.
	if entry.ProjectPath != projectPath {
		return "", ErrRepoMapCacheMiss
	}
	if entry.Fingerprint != fingerprint {
		return "", ErrRepoMapCacheMiss
	}
	if entry.RepoMap == "" {
		return "", ErrRepoMapCacheMiss
	}

	return entry.RepoMap, nil
}

// SaveRepoMapToDisk persists repo map cache.
func SaveRepoMapToDisk(projectPath string, fingerprint int64, repoMap string) error {
	path, err := repoMapCacheFilePath(projectPath)
	if err != nil {
		return err
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("repomap cache: mkdir failed: %w", err)
	}

	entry := RepoMapCacheEntry{
		ProjectPath: projectPath,
		Fingerprint: fingerprint,
		RepoMap:     repoMap,
	}
	b, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("repomap cache: marshal failed: %w", err)
	}

	// 0600 to match other sensitive files in this project.
	if err := os.WriteFile(path, b, 0600); err != nil {
		return fmt.Errorf("repomap cache: write failed: %w", err)
	}
	return nil
}

// ComputeProjectFingerprint computes a best-effort fingerprint based on max modTime.
//
// It walks the project directory and finds the latest modification time across files.
// Certain directories are excluded to keep it fast and stable.
func ComputeProjectFingerprint(projectPath string) (int64, error) {
	var maxTime time.Time

	abs, err := filepath.Abs(projectPath)
	if err != nil {
		return 0, fmt.Errorf("repomap cache: abs path failed: %w", err)
	}

	excluded := map[string]bool{
		".git":         true,
		"node_modules": true,
		"vendor":       true,
		".xelyon":      true,
		".idea":        true,
		".vscode":      true,
	}

	err = filepath.WalkDir(abs, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			// Best-effort: skip unreadable entries.
			return nil
		}

		name := d.Name()
		if d.IsDir() {
			if excluded[name] {
				return fs.SkipDir
			}
			return nil
		}

		info, err := d.Info()
		if err != nil {
			return nil
		}

		mt := info.ModTime()
		if mt.After(maxTime) {
			maxTime = mt
		}
		return nil
	})
	if err != nil {
		return 0, fmt.Errorf("repomap cache: walk failed: %w", err)
	}

	if maxTime.IsZero() {
		return 0, nil
	}
	return maxTime.UnixNano(), nil
}

func repoMapCacheFilePath(projectPath string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("repomap cache: home dir failed: %w", err)
	}

	// Hash the project path for filename safety.
	sum := sha256.Sum256([]byte(projectPath))
	key := hex.EncodeToString(sum[:])

	// Keep under ~/.xelyon/cache/repomap/
	base := filepath.Join(home, ".xelyon", "cache", "repomap")
	// .json for readability and debugging.
	file := key + ".json"

	// Defensive: ensure no path traversal.
	if strings.Contains(file, string(os.PathSeparator)) {
		return "", fmt.Errorf("repomap cache: invalid cache file")
	}
	return filepath.Join(base, file), nil
}
