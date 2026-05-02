package review

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

type worktreeSnapshot struct {
	entries map[string]worktreeSnapshotEntry
}

type worktreeSnapshotEntry struct {
	statusCode  string
	fingerprint string
}

func captureWorktreeSnapshot(ctx context.Context, repoRoot string) (worktreeSnapshot, error) {
	statusPorcelain, err := gitStatusPorcelainV1Z(ctx, repoRoot)
	if err != nil {
		return worktreeSnapshot{}, err
	}

	statusEntries := parseGitStatusEntriesPorcelainV1Z(statusPorcelain)
	entries := make(map[string]worktreeSnapshotEntry, len(statusEntries))
	for path, statusCode := range statusEntries {
		absPath, err := resolveSnapshotPath(repoRoot, path)
		if err != nil {
			return worktreeSnapshot{}, err
		}
		fingerprint, err := buildWorktreeFingerprint(absPath)
		if err != nil {
			return worktreeSnapshot{}, fmt.Errorf("failed to fingerprint %q: %w", path, err)
		}
		entries[path] = worktreeSnapshotEntry{
			statusCode:  statusCode,
			fingerprint: fingerprint,
		}
	}

	return worktreeSnapshot{
		entries: entries,
	}, nil
}

func diffWorktreeSnapshots(before, after worktreeSnapshot) []string {
	changed := make(map[string]struct{}, len(before.entries)+len(after.entries))

	for path, afterEntry := range after.entries {
		beforeEntry, ok := before.entries[path]
		if !ok || beforeEntry.statusCode != afterEntry.statusCode || beforeEntry.fingerprint != afterEntry.fingerprint {
			changed[path] = struct{}{}
		}
	}

	for path := range before.entries {
		if _, ok := after.entries[path]; !ok {
			changed[path] = struct{}{}
		}
	}

	paths := make([]string, 0, len(changed))
	for path := range changed {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	return paths
}

func gitStatusPorcelainV1Z(ctx context.Context, repoRoot string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, "git", "status", "--porcelain=v1", "-z", "--untracked-files=all")
	cmd.Dir = repoRoot
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("git status --porcelain=v1 -z failed: %w: %s", err, strings.TrimSpace(string(out)))
	}
	return out, nil
}

func parseGitStatusEntriesPorcelainV1Z(status []byte) map[string]string {
	parts := bytes.Split(status, []byte{0})
	entries := make(map[string]string, len(parts))

	for i := 0; i < len(parts); i++ {
		part := parts[i]
		if len(part) == 0 {
			continue
		}
		if len(part) < 4 || part[2] != ' ' {
			continue
		}
		statusCode := string(part[:2])
		path := string(part[3:])
		if path == "" {
			continue
		}

		entries[path] = statusCode

		if isRenameOrCopyStatus(statusCode) && i+1 < len(parts) {
			i++
		}
	}

	return entries
}

func isRenameOrCopyStatus(statusCode string) bool {
	if len(statusCode) < 2 {
		return false
	}
	return statusCode[0] == 'R' || statusCode[0] == 'C' || statusCode[1] == 'R' || statusCode[1] == 'C'
}

func resolveSnapshotPath(repoRoot, path string) (string, error) {
	candidate := filepath.Clean(filepath.Join(repoRoot, path))
	rel, err := filepath.Rel(repoRoot, candidate)
	if err != nil {
		return "", fmt.Errorf("invalid snapshot path %q: %w", path, err)
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
		return "", fmt.Errorf("invalid snapshot path %q: outside repository root", path)
	}
	return candidate, nil
}

func buildWorktreeFingerprint(absPath string) (string, error) {
	info, err := os.Lstat(absPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "missing", nil
		}
		return "", err
	}

	mode := info.Mode()
	switch {
	case mode.IsRegular():
		sum, err := hashFileSHA256(absPath)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("file:%s:%s", mode.String(), sum), nil
	case mode&os.ModeSymlink != 0:
		target, err := os.Readlink(absPath)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("symlink:%s:%s", mode.String(), target), nil
	default:
		return fmt.Sprintf("other:%s:%d", mode.String(), info.Size()), nil
	}
}

func hashFileSHA256(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}
