package probe

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

func buildResolverCommandCandidates(command, pathExtValue string) []string {
	if runtime.GOOS != "windows" {
		return []string{command}
	}

	if filepath.Ext(command) != "" {
		return []string{command}
	}

	exts := splitPathExt(pathExtValue)
	if len(exts) == 0 {
		exts = []string{".COM", ".EXE", ".BAT", ".CMD"}
	}

	candidates := make([]string, 0, len(exts)+1)
	candidates = append(candidates, command)
	for _, ext := range exts {
		candidates = append(candidates, command+ext)
	}
	return candidates
}

func splitPathExt(value string) []string {
	parts := strings.Split(value, ";")
	exts := make([]string, 0, len(parts))
	seen := make(map[string]struct{}, len(parts))

	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed == "" {
			continue
		}
		if !strings.HasPrefix(trimmed, ".") {
			trimmed = "." + trimmed
		}
		upper := strings.ToUpper(trimmed)
		if _, ok := seen[upper]; ok {
			continue
		}
		seen[upper] = struct{}{}
		exts = append(exts, upper)
	}

	return exts
}

func resolveExecutablePath(candidatePath, pathExtValue string) (resolvedPath string, ok bool) {
	info, err := os.Stat(candidatePath)
	if err != nil || !info.Mode().IsRegular() {
		return "", false
	}

	if runtime.GOOS == "windows" {
		if !isWindowsExecutableExtension(candidatePath, pathExtValue) {
			return "", false
		}
	} else if info.Mode()&0o111 == 0 {
		return "", false
	}

	absPath, err := filepath.Abs(candidatePath)
	if err != nil {
		return "", false
	}
	return filepath.Clean(absPath), true
}

func isWindowsExecutableExtension(pathValue, pathExtValue string) bool {
	ext := strings.ToUpper(filepath.Ext(pathValue))
	if ext == "" {
		return false
	}

	allowed := splitPathExt(pathExtValue)
	if len(allowed) == 0 {
		allowed = []string{".COM", ".EXE", ".BAT", ".CMD"}
	}
	for _, candidate := range allowed {
		if ext == candidate {
			return true
		}
	}
	return false
}
