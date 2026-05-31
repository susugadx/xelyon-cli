package probe

import (
	"path/filepath"
	"strconv"
	"strings"
)

type scratchOnlyFile = probeGeneratedFile

func validateAndBuildScratchFiles(scratchDir string, files []ReviewProbeFile) ([]scratchOnlyFile, error) {
	if len(files) > defaultScratchOnlyMaxFiles {
		return nil, newBlockedCommandErrorf("scratch_only allows at most %d files", defaultScratchOnlyMaxFiles)
	}

	planned := make([]scratchOnlyFile, 0, len(files))
	seen := make(map[string]struct{}, len(files))
	var totalBytes int64

	for _, file := range files {
		contentBytes := int64(len(file.Content))
		if contentBytes > defaultScratchOnlyMaxFileBytes {
			return nil, newBlockedCommandErrorf("scratch file %q exceeds max file bytes", file.Path)
		}
		totalBytes += contentBytes
		if totalBytes > defaultScratchOnlyMaxTotalFileBytes {
			return nil, newBlockedCommandErrorf("scratch files exceed max total bytes")
		}

		absPath, err := resolveScratchRelativePath(scratchDir, file.Path)
		if err != nil {
			return nil, err
		}
		label := "scratch file " + strconv.Quote(file.Path)
		if err := validateProbeGeneratedFileTarget(probeGeneratedFileValidationSpec{
			modeName:  "scratch",
			rootLabel: "scratch directory",
			rootDir:   scratchDir,
		}, absPath, label); err != nil {
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
	return writeProbeGeneratedFiles(files)
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
