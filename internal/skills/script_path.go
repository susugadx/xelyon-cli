package skills

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ResolveScriptPath は skill scripts 配下の実行対象を安全に解決する。
// path traversal と scripts root 外 symlink を拒否する。
func ResolveScriptPath(skill ParsedSkill, scriptPath string) (string, error) {
	scriptPath = strings.TrimSpace(scriptPath)
	if scriptPath == "" {
		return "", fmt.Errorf("script path is required")
	}
	if filepath.IsAbs(scriptPath) {
		return "", fmt.Errorf("absolute script path is not allowed")
	}

	scriptRoot := filepath.Join(cleanAbsPathOrFallback(skill.Directory), "scripts")
	cleanScript := filepath.Clean(scriptPath)
	if cleanScript == "." || cleanScript == string(filepath.Separator) {
		return "", fmt.Errorf("script path is invalid")
	}

	candidate := cleanAbsPathOrFallback(filepath.Join(scriptRoot, cleanScript))
	if !isSubPath(candidate, cleanAbsPathOrFallback(scriptRoot)) {
		return "", fmt.Errorf("script path escapes scripts directory")
	}

	info, err := os.Lstat(candidate)
	if err != nil {
		return "", fmt.Errorf("script not found: %w", err)
	}
	if info.IsDir() {
		return "", fmt.Errorf("script path points to a directory")
	}

	realRoot, err := filepath.EvalSymlinks(scriptRoot)
	if err != nil {
		realRoot = cleanAbsPathOrFallback(scriptRoot)
	}
	realRoot = cleanAbsPathOrFallback(realRoot)

	realPath, err := filepath.EvalSymlinks(candidate)
	if err != nil {
		return "", fmt.Errorf("failed to resolve script symlink: %w", err)
	}
	realPath = cleanAbsPathOrFallback(realPath)
	if !isSubPath(realPath, realRoot) {
		return "", fmt.Errorf("script symlink escapes skill scripts directory")
	}

	return realPath, nil
}

func isSubPath(path, base string) bool {
	if path == base {
		return true
	}
	return strings.HasPrefix(path, base+string(filepath.Separator))
}
