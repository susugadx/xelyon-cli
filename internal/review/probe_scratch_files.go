package review

import (
	"os"
	"path/filepath"
	"strings"
)

type scratchOnlyFile struct {
	absPath string
	content string
}

func validateAndBuildScratchFiles(scratchDir string, files []ReviewProbeFile) ([]scratchOnlyFile, error) {
	planned := make([]scratchOnlyFile, 0, len(files))
	seen := make(map[string]struct{}, len(files))

	for _, file := range files {
		absPath, err := resolveScratchRelativePath(scratchDir, file.Path)
		if err != nil {
			return nil, err
		}
		if _, ok := seen[absPath]; ok {
			return nil, newBlockedCommandErrorf("duplicate scratch file path %q", file.Path)
		}
		seen[absPath] = struct{}{}
		planned = append(planned, scratchOnlyFile{
			absPath: absPath,
			content: file.Content,
		})
	}

	return planned, nil
}

func writeScratchFiles(files []scratchOnlyFile) error {
	for _, file := range files {
		if err := os.MkdirAll(filepath.Dir(file.absPath), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(file.absPath, []byte(file.content), 0o644); err != nil {
			return err
		}
	}
	return nil
}

func resolveScratchRelativePath(scratchDir, value string) (string, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "", newBlockedCommandErrorf("scratch path is empty")
	}
	if filepath.IsAbs(trimmed) {
		return "", newBlockedCommandErrorf("scratch path %q must be relative", value)
	}
	if hasTrailingPathSeparator(trimmed) {
		return "", newBlockedCommandErrorf("scratch path %q must be a file path", value)
	}

	resolved, err := resolvePathWithinRepoRoot(scratchDir, scratchDir, trimmed)
	if err != nil {
		if isOutsideRepoPathError(err) {
			return "", newBlockedCommandErrorf("scratch path %q escapes scratch directory", value)
		}
		return "", newBlockedCommandErrorf("scratch path %q is invalid: %v", value, err)
	}
	if resolved == scratchDir {
		return "", newBlockedCommandErrorf("scratch path %q must be a file path", value)
	}
	return resolved, nil
}

func hasTrailingPathSeparator(value string) bool {
	return strings.HasSuffix(value, "/") || strings.HasSuffix(value, "\\")
}
