package probe

import (
	"io"
	"io/fs"
	"os"
	"path/filepath"
)

type repoSandboxCopyLimits struct {
	maxFiles     int
	maxBytes     int64
	maxFileBytes int64
}

func defaultRepoSandboxCopyLimits() repoSandboxCopyLimits {
	return repoSandboxCopyLimits{
		maxFiles:     defaultRepoSandboxMaxCopyFiles,
		maxBytes:     defaultRepoSandboxMaxCopyBytes,
		maxFileBytes: defaultRepoSandboxMaxCopyFileBytes,
	}
}

type repoSandboxCopyStats struct {
	files int
	bytes int64
}

func copyRepoToSandboxWorktree(repoRoot, worktreeDir string, limits repoSandboxCopyLimits) (repoSandboxCopyStats, error) {
	repoRoot = filepath.Clean(repoRoot)
	worktreeDir = filepath.Clean(worktreeDir)

	var stats repoSandboxCopyStats
	err := filepath.WalkDir(repoRoot, func(srcPath string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return newBlockedCommandErrorf("failed to read repository path %q: %v", srcPath, walkErr)
		}

		name := entry.Name()
		if name == ".git" {
			if entry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		rel, err := filepath.Rel(repoRoot, srcPath)
		if err != nil {
			return newBlockedCommandErrorf("failed to resolve repository path %q: %v", srcPath, err)
		}
		if rel == "." {
			return nil
		}

		dstPath := filepath.Join(worktreeDir, rel)
		info, err := entry.Info()
		if err != nil {
			return newBlockedCommandErrorf("failed to stat repository path %q: %v", srcPath, err)
		}

		if entry.Type()&os.ModeSymlink != 0 {
			if err := accountRepoSandboxCopiedFile(&stats, limits, 0); err != nil {
				return err
			}
			return copyRepoSandboxSymlink(srcPath, dstPath)
		}

		if entry.IsDir() {
			return os.MkdirAll(dstPath, info.Mode().Perm())
		}

		if !info.Mode().IsRegular() {
			return newBlockedCommandErrorf("repository path %q has unsupported file type", srcPath)
		}
		if info.Size() > limits.maxFileBytes {
			return newBlockedCommandErrorf("repository file %q exceeds max copy file bytes", rel)
		}
		if err := accountRepoSandboxCopiedFile(&stats, limits, info.Size()); err != nil {
			return err
		}
		return copyRepoSandboxRegularFile(srcPath, dstPath, info.Mode().Perm())
	})
	if err != nil {
		return repoSandboxCopyStats{}, err
	}
	return stats, nil
}

func accountRepoSandboxCopiedFile(stats *repoSandboxCopyStats, limits repoSandboxCopyLimits, size int64) error {
	stats.files++
	if stats.files > limits.maxFiles {
		return newBlockedCommandErrorf("repository copy exceeds max file count")
	}
	stats.bytes += size
	if stats.bytes > limits.maxBytes {
		return newBlockedCommandErrorf("repository copy exceeds max total bytes")
	}
	return nil
}

func copyRepoSandboxSymlink(srcPath, dstPath string) error {
	target, err := os.Readlink(srcPath)
	if err != nil {
		return newBlockedCommandErrorf("failed to read symlink %q: %v", srcPath, err)
	}
	if err := os.MkdirAll(filepath.Dir(dstPath), 0o755); err != nil {
		return newBlockedCommandErrorf("failed to create symlink parent %q: %v", filepath.Dir(dstPath), err)
	}
	if err := os.Symlink(target, dstPath); err != nil {
		return newBlockedCommandErrorf("failed to copy symlink %q: %v", srcPath, err)
	}
	return nil
}

func copyRepoSandboxRegularFile(srcPath, dstPath string, mode os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(dstPath), 0o755); err != nil {
		return newBlockedCommandErrorf("failed to create file parent %q: %v", filepath.Dir(dstPath), err)
	}

	src, err := os.Open(srcPath)
	if err != nil {
		return newBlockedCommandErrorf("failed to open repository file %q: %v", srcPath, err)
	}
	defer src.Close()

	dst, err := os.OpenFile(dstPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, mode|0o600)
	if err != nil {
		return newBlockedCommandErrorf("failed to create sandbox file %q: %v", dstPath, err)
	}
	if _, err := io.Copy(dst, src); err != nil {
		_ = dst.Close()
		return newBlockedCommandErrorf("failed to copy repository file %q: %v", srcPath, err)
	}
	if err := dst.Close(); err != nil {
		return newBlockedCommandErrorf("failed to close sandbox file %q: %v", dstPath, err)
	}
	if err := os.Chmod(dstPath, mode); err != nil {
		return newBlockedCommandErrorf("failed to apply sandbox file mode %q: %v", dstPath, err)
	}
	return nil
}
