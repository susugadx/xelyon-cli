package review

import (
	"os"
	"path/filepath"
	"strings"
)

func validateProbeHostReadOnlyDir(candidate, repoRoot string) (string, bool) {
	trimmed := strings.TrimSpace(candidate)
	if trimmed == "" || !filepath.IsAbs(trimmed) {
		return "", false
	}

	cleaned := filepath.Clean(trimmed)
	info, err := os.Lstat(cleaned)
	if err != nil || !info.IsDir() {
		return "", false
	}
	if probePathIsInsideRoot(cleaned, repoRoot) {
		return "", false
	}

	evaluated, err := filepath.EvalSymlinks(cleaned)
	if err != nil {
		return "", false
	}
	evaluated = filepath.Clean(evaluated)
	info, err = os.Stat(evaluated)
	if err != nil || !info.IsDir() {
		return "", false
	}
	if probePathIsInsideRoot(evaluated, repoRoot) {
		return "", false
	}
	return evaluated, true
}
