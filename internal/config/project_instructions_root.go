package config

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
)

type gitRootResolver struct {
	cache sync.Map
}

var defaultGitRootResolver gitRootResolver

func loadProjectConfigForDir(cwd string) (*ProjectConfig, error) {
	if path := findFileUpward(cwd, "xelyon.yaml"); path != "" {
		pc, err := loadProjectConfigFromYAML(path)
		if err != nil {
			return nil, fmt.Errorf("failed to load %s: %w", path, err)
		}
		return pc, nil
	}
	return nil, nil
}

func resolveBundleRootPath(cwd string, projectCfg *ProjectConfig, gitRoot string, aiCfg AgentInstructionsConfig) string {
	if projectCfg != nil && strings.TrimSpace(projectCfg.FilePath) != "" {
		return filepath.Dir(projectCfg.FilePath)
	}
	if gitRoot != "" {
		return gitRoot
	}
	if guidanceRoot := findGuidanceRootUpward(cwd, aiCfg.Project.Files, aiCfg.IncludeLocalFiles); guidanceRoot != "" {
		return guidanceRoot
	}
	return cwd
}

func findGuidanceRootUpward(cwd string, files []string, includeLocal bool) string {
	for dir := cwd; ; dir = filepath.Dir(dir) {
		for _, candidate := range files {
			candidate = strings.TrimSpace(candidate)
			if candidate == "" {
				continue
			}
			if !includeLocal && isLocalGuidanceFile(candidate) {
				continue
			}
			fullPath := filepath.Join(dir, filepath.FromSlash(candidate))
			if _, err := os.Stat(fullPath); err == nil {
				return dir
			}
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
	}
	return ""
}

func findGitRoot(cwd string) string {
	return defaultGitRootResolver.find(cwd)
}

func (r *gitRootResolver) find(cwd string) string {
	cacheKey := normalizeGitRootCacheKey(cwd)
	if cacheKey == "" {
		return ""
	}
	if root, ok := r.loadCachedRoot(cacheKey); ok {
		return root
	}

	root, ok := resolveGitRoot(cwd)
	if !ok {
		return ""
	}
	r.cache.Store(cacheKey, root)
	return root
}

func (r *gitRootResolver) loadCachedRoot(cacheKey string) (string, bool) {
	cached, ok := r.cache.Load(cacheKey)
	if !ok {
		return "", false
	}
	root, ok := cached.(string)
	if !ok || strings.TrimSpace(root) == "" {
		r.cache.Delete(cacheKey)
		return "", false
	}
	return root, true
}

func resolveGitRoot(cwd string) (string, bool) {
	if strings.TrimSpace(cwd) == "" {
		return "", false
	}
	cmd := exec.Command("git", "-C", cwd, "rev-parse", "--show-toplevel")
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return "", false
	}
	root := strings.TrimSpace(out.String())
	if root == "" {
		return "", false
	}
	return root, true
}

func normalizeGitRootCacheKey(cwd string) string {
	cwd = strings.TrimSpace(cwd)
	if cwd == "" {
		return ""
	}
	abs, err := filepath.Abs(cwd)
	if err != nil {
		return filepath.Clean(cwd)
	}
	return filepath.Clean(abs)
}
