package ledger

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var windowsAbsPathRe = regexp.MustCompile(`^[A-Za-z]:[\\/]`)

type ledgerPathBase int

const (
	ledgerPathBaseInvocationCWD ledgerPathBase = iota
	ledgerPathBaseRepoRoot
)

func normalizeRepoRoot(root string) string {
	root = strings.TrimSpace(root)
	if root == "" {
		return ""
	}
	root = filepath.Clean(root)
	if !filepath.IsAbs(root) && !isWindowsAbsPath(root) {
		if abs, err := filepath.Abs(root); err == nil {
			root = abs
		}
	}
	return filepath.Clean(root)
}

func defaultInvocationCWD() string {
	cwd, err := os.Getwd()
	if err != nil {
		return ""
	}
	return cwd
}

func repoRootForInvocationCWD(cwd string) string {
	cwd = normalizeRepoRoot(cwd)
	if cwd == "" {
		return ""
	}
	if root := findGitRoot(cwd); root != "" {
		return root
	}
	return cwd
}

func findGitRoot(start string) string {
	dir := normalizeRepoRoot(start)
	for dir != "" {
		if _, err := os.Stat(filepath.Join(dir, ".git")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return ""
}

func normalizeLedgerPath(repoRoot, invocationCWD, candidate string) (string, bool) {
	return normalizeLedgerPathWithBaseOrder(repoRoot, invocationCWD, candidate, ledgerPathBaseInvocationCWD, ledgerPathBaseRepoRoot)
}

func normalizeRepoRelativeLedgerPath(repoRoot, invocationCWD, candidate string) (string, bool) {
	return normalizeLedgerPathWithBaseOrder(repoRoot, invocationCWD, candidate, ledgerPathBaseRepoRoot, ledgerPathBaseInvocationCWD)
}

func normalizeLedgerPathWithBaseOrder(repoRoot, invocationCWD, candidate string, bases ...ledgerPathBase) (string, bool) {
	candidate = cleanPathCandidate(candidate)
	if !isLedgerPathCandidateSafe(candidate) {
		return "", false
	}

	root := normalizeRepoRoot(repoRoot)
	if filepath.IsAbs(candidate) || isWindowsAbsPath(candidate) {
		if root == "" {
			return "", false
		}
		return normalizeAbsoluteLedgerPath(root, candidate)
	}

	cwd := normalizeRepoRoot(invocationCWD)
	if root != "" && cwd != "" {
		return normalizeRelativeLedgerPath(root, cwd, candidate, bases...)
	}

	repoRelative, repoRelativeOK := cleanLedgerRelativePath(candidate)
	return repoRelative, repoRelativeOK
}

func normalizeAbsoluteLedgerPath(repoRoot, candidate string) (string, bool) {
	cleaned := filepath.Clean(candidate)
	if relative, ok := relativeLedgerPath(repoRoot, cleaned); ok {
		return relative, true
	}

	evaluatedRoot, rootOK := evalLedgerSymlinkPath(repoRoot)
	evaluatedCandidate, candidateOK := evalLedgerPathBestEffort(cleaned)
	if !rootOK || !candidateOK {
		return "", false
	}
	return relativeLedgerPath(evaluatedRoot, evaluatedCandidate)
}

func relativeLedgerPath(repoRoot, candidate string) (string, bool) {
	rel, err := filepath.Rel(repoRoot, candidate)
	if err != nil {
		return "", false
	}
	return cleanLedgerRelativePath(rel)
}

func evalLedgerSymlinkPath(path string) (string, bool) {
	evaluated, err := filepath.EvalSymlinks(path)
	if err != nil {
		return "", false
	}
	return filepath.Clean(evaluated), true
}

func evalLedgerPathBestEffort(path string) (string, bool) {
	cleaned := filepath.Clean(path)
	if evaluated, ok := evalLedgerSymlinkPath(cleaned); ok {
		return evaluated, true
	}
	return evalLedgerExistingPrefixPath(cleaned)
}

func evalLedgerExistingPrefixPath(path string) (string, bool) {
	missingParts := make([]string, 0, 1)
	for current := filepath.Clean(path); ; current = filepath.Dir(current) {
		parent := filepath.Dir(current)
		if parent == current {
			return "", false
		}
		missingParts = append([]string{filepath.Base(current)}, missingParts...)
		if evaluatedParent, ok := evalLedgerSymlinkPath(parent); ok {
			parts := append([]string{evaluatedParent}, missingParts...)
			return filepath.Clean(filepath.Join(parts...)), true
		}
	}
}

func normalizeRelativeLedgerPath(repoRoot, invocationCWD, candidate string, bases ...ledgerPathBase) (string, bool) {
	if len(bases) == 0 {
		bases = []ledgerPathBase{ledgerPathBaseInvocationCWD, ledgerPathBaseRepoRoot}
	}

	candidates := make([]string, 0, len(bases))
	for _, base := range bases {
		relative, ok := ledgerRelativePathFromBase(repoRoot, invocationCWD, candidate, base)
		if !ok || containsString(candidates, relative) {
			continue
		}
		candidates = append(candidates, relative)
	}
	for _, relative := range candidates {
		if ledgerPathExists(repoRoot, relative) {
			return relative, true
		}
	}
	if len(candidates) > 0 {
		return candidates[0], true
	}
	return "", false
}

func ledgerRelativePathFromBase(repoRoot, invocationCWD, candidate string, base ledgerPathBase) (string, bool) {
	switch base {
	case ledgerPathBaseRepoRoot:
		return cleanLedgerRelativePath(candidate)
	default:
		abs := filepath.Clean(filepath.Join(invocationCWD, filepath.FromSlash(candidate)))
		rel, err := filepath.Rel(repoRoot, abs)
		if err != nil {
			return "", false
		}
		return cleanLedgerRelativePath(rel)
	}
}

func cleanPathCandidate(candidate string) string {
	candidate = strings.TrimSpace(candidate)
	candidate = strings.Trim(candidate, "`\"'")
	candidate = strings.TrimRight(candidate, ",;")
	return candidate
}

func isLedgerPathCandidateSafe(candidate string) bool {
	if candidate == "" || strings.Contains(candidate, "\x00") {
		return false
	}
	if strings.Contains(candidate, "\n") || strings.Contains(candidate, "\r") {
		return false
	}
	if strings.Contains(candidate, "://") {
		return false
	}
	if strings.HasPrefix(candidate, "locator:") || strings.HasPrefix(candidate, "L") && isDigits(candidate[1:]) {
		return false
	}
	if strings.ContainsAny(candidate, "*?") {
		return false
	}
	return true
}

func cleanLedgerRelativePath(candidate string) (string, bool) {
	cleaned := filepath.Clean(filepath.FromSlash(candidate))
	if cleaned == "." || cleaned == "" {
		return "", false
	}
	if filepath.IsAbs(cleaned) || isWindowsAbsPath(cleaned) {
		return "", false
	}
	if cleaned == ".." || strings.HasPrefix(cleaned, ".."+string(filepath.Separator)) {
		return "", false
	}
	return filepath.ToSlash(cleaned), true
}

func ledgerPathExists(repoRoot, relativePath string) bool {
	if repoRoot == "" || relativePath == "" {
		return false
	}
	_, err := os.Stat(filepath.Join(repoRoot, filepath.FromSlash(relativePath)))
	return err == nil
}

func isWindowsAbsPath(path string) bool {
	return windowsAbsPathRe.MatchString(path)
}

func isDigits(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

func containsString(items []string, item string) bool {
	for _, existing := range items {
		if existing == item {
			return true
		}
	}
	return false
}
