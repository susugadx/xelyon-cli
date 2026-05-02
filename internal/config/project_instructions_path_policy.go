package config

import (
	"path/filepath"
	"strings"
)

type instructionPathBoundary struct {
	RootPath         string
	ResolvedRootPath string
}

func newInstructionPathBoundary(rootPath, resolvedRootPath string) instructionPathBoundary {
	return instructionPathBoundary{
		RootPath:         strings.TrimSpace(rootPath),
		ResolvedRootPath: strings.TrimSpace(resolvedRootPath),
	}
}

func (b instructionPathBoundary) Enabled() bool {
	return strings.TrimSpace(b.RootPath) != ""
}

func (b instructionPathBoundary) ContainsPath(fullPath string) bool {
	if !b.Enabled() {
		return true
	}
	return isSafeInstructionPathWithinRoot(b.RootPath, b.ResolvedRootPath, fullPath)
}

func resolveProjectGuidancePath(rootPath, candidatePath string) (string, bool) {
	rootPath = strings.TrimSpace(rootPath)
	candidatePath = strings.TrimSpace(candidatePath)
	if rootPath == "" || candidatePath == "" {
		return "", false
	}

	normalizedPath := filepath.FromSlash(candidatePath)
	if filepath.IsAbs(normalizedPath) {
		return "", false
	}

	cleanedPath := filepath.Clean(normalizedPath)
	if cleanedPath == "." || cleanedPath == ".." || strings.HasPrefix(cleanedPath, ".."+string(filepath.Separator)) {
		return "", false
	}

	fullPath := filepath.Join(rootPath, cleanedPath)
	if !isPathWithinRoot(rootPath, fullPath) {
		return "", false
	}
	return fullPath, true
}

func isSafeInstructionPathWithinRoot(rootPath, resolvedRootPath, fullPath string) bool {
	boundary := newInstructionPathBoundary(rootPath, resolvedRootPath)
	if !isPathWithinRoot(boundary.RootPath, fullPath) {
		return false
	}

	if strings.TrimSpace(boundary.ResolvedRootPath) == "" {
		var err error
		boundary.ResolvedRootPath, err = resolvePathForBoundaryComparison(boundary.RootPath)
		if err != nil {
			return false
		}
	}
	resolvedPath, err := resolvePathForBoundaryComparison(fullPath)
	if err != nil {
		return false
	}
	return isPathWithinBase(boundary.ResolvedRootPath, resolvedPath)
}

func resolvePathForBoundaryComparison(path string) (string, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	return filepath.EvalSymlinks(absPath)
}

func isPathWithinRoot(rootPath, fullPath string) bool {
	rootPath = strings.TrimSpace(rootPath)
	fullPath = strings.TrimSpace(fullPath)
	if rootPath == "" || fullPath == "" {
		return false
	}

	rootAbs, err := filepath.Abs(rootPath)
	if err != nil {
		return false
	}
	fullAbs, err := filepath.Abs(fullPath)
	if err != nil {
		return false
	}
	return isPathWithinBase(rootAbs, fullAbs)
}

func isPathWithinBase(basePath, targetPath string) bool {
	relPath, err := filepath.Rel(basePath, targetPath)
	if err != nil {
		return false
	}
	relPath = filepath.Clean(relPath)
	if relPath == "." {
		return true
	}
	return relPath != ".." && !strings.HasPrefix(relPath, ".."+string(filepath.Separator))
}
