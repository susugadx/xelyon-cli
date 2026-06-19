package config

import (
	"os"
	"path/filepath"
	"strings"
)

type scopedGuidanceLoadPlanResolver func(path, repositoryScope string) guidanceLoadPlan

type projectInstructionSearchDir struct {
	RelPath string
}

func buildScopedProjectGuidanceLoadPlans(rootPath, cwd string, inputPaths []string, aiCfg AgentInstructionsConfig, resolver scopedGuidanceLoadPlanResolver) []guidanceLoadPlan {
	if resolver == nil {
		return nil
	}
	searchDirs := projectInstructionScopedSearchDirs(rootPath, cwd, inputPaths)
	if len(searchDirs) == 0 {
		searchDirs = []projectInstructionSearchDir{{RelPath: ""}}
	}

	var plans []guidanceLoadPlan
	seen := make(map[string]struct{})
	candidatePaths := filteredProjectGuidanceCandidatePaths(aiCfg.Project.Files, aiCfg.IncludeLocalFiles)
	for _, dir := range searchDirs {
		rootScope := isRootRepositoryInstructionScope(dir.RelPath)
		for _, path := range candidatePaths {
			switch {
			case isScopedProjectGuidanceBasename(path):
				candidatePath := joinRepositoryInstructionPath(dir.RelPath, path)
				appendScopedGuidancePlan(&plans, seen, candidatePath, normalizeRepositoryInstructionScope(dir.RelPath), resolver)
			case rootScope:
				appendScopedGuidancePlan(&plans, seen, path, ".", resolver)
			}
		}
	}
	return plans
}

func filteredProjectGuidanceCandidatePaths(paths []string, includeLocalFiles bool) []string {
	if len(paths) == 0 {
		return nil
	}
	candidates := make([]string, 0, len(paths))
	forEachGuidanceCandidatePath(paths, includeLocalFiles, func(path string) bool {
		candidates = append(candidates, path)
		return true
	})
	return candidates
}

func appendScopedGuidancePlan(plans *[]guidanceLoadPlan, seen map[string]struct{}, path, repositoryScope string, resolver scopedGuidanceLoadPlanResolver) {
	if plans == nil || resolver == nil {
		return
	}
	key := normalizedRepositoryInstructionCandidate(path)
	if key == "" {
		return
	}
	if _, ok := seen[key]; ok {
		return
	}
	seen[key] = struct{}{}
	*plans = append(*plans, resolver(key, repositoryScope))
}

func isRootRepositoryInstructionScope(scope string) bool {
	return normalizeRepositoryInstructionScope(scope) == "."
}

func projectInstructionScopedSearchDirs(rootPath, cwd string, inputPaths []string) []projectInstructionSearchDir {
	rootPath = strings.TrimSpace(rootPath)
	if rootPath == "" {
		return nil
	}

	seen := make(map[string]struct{})
	var dirs []projectInstructionSearchDir
	appendProjectInstructionSearchChain(&dirs, seen, rootPath, cwd)
	for _, inputPath := range inputPaths {
		inputDir, ok := resolveProjectInstructionInputDirectory(rootPath, inputPath)
		if !ok {
			continue
		}
		appendProjectInstructionSearchChain(&dirs, seen, rootPath, inputDir)
	}
	return dirs
}

func projectInstructionInputDirectoryScopes(rootPath string, inputPaths []string) []string {
	rootPath = strings.TrimSpace(rootPath)
	if rootPath == "" || len(inputPaths) == 0 {
		return nil
	}
	seen := make(map[string]struct{})
	scopes := make([]string, 0, len(inputPaths))
	for _, inputPath := range inputPaths {
		inputDir, ok := resolveProjectInstructionInputDirectory(rootPath, inputPath)
		if !ok {
			continue
		}
		rootAbs, targetAbs, ok := normalizedProjectInstructionRootAndTarget(rootPath, inputDir)
		if !ok {
			continue
		}
		relPath, err := filepath.Rel(rootAbs, targetAbs)
		if err != nil {
			continue
		}
		scope := normalizeRepositoryInstructionScope(filepath.ToSlash(relPath))
		if _, ok := seen[scope]; ok {
			continue
		}
		seen[scope] = struct{}{}
		scopes = append(scopes, scope)
	}
	return scopes
}

func appendProjectInstructionSearchChain(dirs *[]projectInstructionSearchDir, seen map[string]struct{}, rootPath, targetDir string) {
	if dirs == nil {
		return
	}
	rootAbs, targetAbs, ok := normalizedProjectInstructionRootAndTarget(rootPath, targetDir)
	if !ok {
		return
	}
	relPath, err := filepath.Rel(rootAbs, targetAbs)
	if err != nil {
		return
	}
	relPath = filepath.Clean(relPath)
	if relPath == "." {
		appendProjectInstructionSearchDir(dirs, seen, "")
		return
	}
	if relPath == ".." || strings.HasPrefix(relPath, ".."+string(filepath.Separator)) {
		return
	}

	appendProjectInstructionSearchDir(dirs, seen, "")
	var current string
	for _, part := range strings.Split(relPath, string(filepath.Separator)) {
		if part == "" || part == "." {
			continue
		}
		current = filepath.Join(current, part)
		appendProjectInstructionSearchDir(dirs, seen, filepath.ToSlash(current))
	}
}

func appendProjectInstructionSearchDir(dirs *[]projectInstructionSearchDir, seen map[string]struct{}, relPath string) {
	if dirs == nil {
		return
	}
	relPath = normalizeRepositoryInstructionScope(relPath)
	key := relPath
	if key == "." {
		key = ""
	}
	if _, ok := seen[key]; ok {
		return
	}
	seen[key] = struct{}{}
	*dirs = append(*dirs, projectInstructionSearchDir{RelPath: key})
}

func resolveProjectInstructionInputDirectory(rootPath, inputPath string) (string, bool) {
	rootPath = strings.TrimSpace(rootPath)
	inputPath = strings.TrimSpace(inputPath)
	if rootPath == "" || inputPath == "" {
		return "", false
	}
	normalized := filepath.FromSlash(inputPath)
	if filepath.IsAbs(normalized) {
		return "", false
	}
	cleaned := filepath.Clean(normalized)
	if cleaned == "." || cleaned == ".." || strings.HasPrefix(cleaned, ".."+string(filepath.Separator)) {
		return "", false
	}
	fullPath := filepath.Join(rootPath, cleaned)
	if !isPathWithinRoot(rootPath, fullPath) {
		return "", false
	}
	info, err := os.Stat(fullPath)
	if err != nil {
		return "", false
	}
	var targetDir string
	if info.IsDir() {
		targetDir = fullPath
	} else {
		targetDir = filepath.Dir(fullPath)
	}
	if !isSafeInstructionPathWithinRoot(rootPath, "", targetDir) {
		return "", false
	}
	return targetDir, true
}

func normalizedProjectInstructionRootAndTarget(rootPath, targetPath string) (string, string, bool) {
	rootPath = strings.TrimSpace(rootPath)
	targetPath = strings.TrimSpace(targetPath)
	if rootPath == "" || targetPath == "" {
		return "", "", false
	}
	rootAbs, err := filepath.Abs(rootPath)
	if err != nil {
		return "", "", false
	}
	targetAbs, err := filepath.Abs(targetPath)
	if err != nil {
		return "", "", false
	}
	rootAbs = filepath.Clean(rootAbs)
	targetAbs = filepath.Clean(targetAbs)
	if !isPathWithinBase(rootAbs, targetAbs) {
		return "", "", false
	}
	return rootAbs, targetAbs, true
}

func isScopedProjectGuidanceBasename(path string) bool {
	path = strings.TrimSpace(path)
	if path == "" {
		return false
	}
	if strings.Contains(path, "/") || strings.Contains(path, "\\") {
		return false
	}
	return !filepath.IsAbs(path)
}

func joinRepositoryInstructionPath(relDir, baseName string) string {
	relDir = strings.TrimSpace(relDir)
	baseName = strings.TrimSpace(baseName)
	if relDir == "" || relDir == "." {
		return baseName
	}
	return filepath.ToSlash(filepath.Join(filepath.FromSlash(relDir), baseName))
}

func normalizedRepositoryInstructionCandidate(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return ""
	}
	cleaned := filepath.ToSlash(filepath.Clean(filepath.FromSlash(path)))
	if cleaned == "." {
		return ""
	}
	return cleaned
}

func normalizeRepositoryInstructionScope(scope string) string {
	scope = strings.TrimSpace(scope)
	if scope == "" || scope == "." {
		return "."
	}
	cleaned := filepath.ToSlash(filepath.Clean(filepath.FromSlash(scope)))
	if cleaned == "." {
		return "."
	}
	return cleaned
}
