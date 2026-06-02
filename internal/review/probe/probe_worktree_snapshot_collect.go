package probe

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"strings"
)

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

func gitStatusPorcelainV1Z(ctx context.Context, repoRoot string) ([]byte, error) {
	result, err := runReviewProbeGitProcess(ctx, reviewProbeGitProcessRequest{
		repoRoot: repoRoot,
		args:     []string{"status", "--porcelain=v1", "-z", "--untracked-files=all"},
	})
	if err != nil {
		return nil, fmt.Errorf("git status --porcelain=v1 -z failed: %w: %s", err, strings.TrimSpace(result.diagnostics))
	}
	return []byte(result.parsedOutput), nil
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
	resolved, err := resolvePathWithinRepoRoot(repoRoot, repoRoot, path)
	if err != nil {
		if errors.Is(err, ErrHostReadOnlyOutsideRepoPath) {
			return "", fmt.Errorf("invalid snapshot path %q: outside repository root", path)
		}
		return "", fmt.Errorf("invalid snapshot path %q: %v", path, err)
	}
	return resolved, nil
}
