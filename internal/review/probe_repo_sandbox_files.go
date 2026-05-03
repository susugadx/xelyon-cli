package review

import (
	"path/filepath"
	"strconv"
	"strings"
)

type repoSandboxFile = probeGeneratedFile

func validateAndBuildRepoSandboxFiles(worktreeDir string, files []ReviewProbeFile) ([]repoSandboxFile, error) {
	if len(files) > defaultRepoSandboxMaxGeneratedFiles {
		return nil, newBlockedCommandErrorf("repo_sandbox allows at most %d generated files", defaultRepoSandboxMaxGeneratedFiles)
	}

	planned := make([]repoSandboxFile, 0, len(files))
	seen := make(map[string]struct{}, len(files))
	var totalBytes int64
	for _, file := range files {
		size := int64(len([]byte(file.Content)))
		if size > defaultRepoSandboxMaxGeneratedFileBytes {
			return nil, newBlockedCommandErrorf("repo_sandbox generated file %q exceeds max file bytes", file.Path)
		}
		totalBytes += size
		if totalBytes > defaultRepoSandboxMaxGeneratedBytes {
			return nil, newBlockedCommandErrorf("repo_sandbox generated files exceed max total bytes")
		}

		absPath, err := resolveRepoSandboxRelativePath(worktreeDir, file.Path, "generated file path")
		if err != nil {
			return nil, err
		}
		label := "repo_sandbox generated file " + strconv.Quote(file.Path)
		if err := validateProbeGeneratedFileTarget(probeGeneratedFileValidationSpec{
			modeName:  "repo_sandbox",
			rootLabel: "sandbox worktree",
			rootDir:   worktreeDir,
		}, absPath, label); err != nil {
			return nil, err
		}
		key := filepath.Clean(absPath)
		if _, ok := seen[key]; ok {
			return nil, newBlockedCommandErrorf("duplicate repo_sandbox generated file path %q", file.Path)
		}
		seen[key] = struct{}{}

		planned = append(planned, repoSandboxFile{
			absPath: absPath,
			content: file.Content,
		})
	}

	return planned, nil
}

func writeRepoSandboxFiles(files []repoSandboxFile) error {
	return writeProbeGeneratedFiles(files)
}

func resolveRepoSandboxRelativePath(worktreeDir, value, label string) (string, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "", newBlockedCommandErrorf("repo_sandbox %s is empty", label)
	}
	if filepath.IsAbs(trimmed) {
		return "", newBlockedCommandErrorf("repo_sandbox %s %q must be relative", label, value)
	}
	if trimmed == "." || trimmed == string(filepath.Separator) {
		return "", newBlockedCommandErrorf("repo_sandbox %s %q must be a file path", label, value)
	}

	resolved, err := resolvePathWithinRepoRoot(worktreeDir, worktreeDir, trimmed)
	if err != nil {
		if isOutsideRepoPathError(err) {
			return "", newBlockedCommandErrorf("repo_sandbox %s %q escapes sandbox worktree", label, value)
		}
		return "", newBlockedCommandErrorf("repo_sandbox %s %q is invalid: %v", label, value, err)
	}
	if resolved == filepath.Clean(worktreeDir) {
		return "", newBlockedCommandErrorf("repo_sandbox %s %q must be a file path", label, value)
	}
	return resolved, nil
}
