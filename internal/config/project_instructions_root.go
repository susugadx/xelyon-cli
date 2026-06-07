package config

import (
	"bytes"
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

type projectInstructionRootResolution struct {
	RootPath string
	Source   ProjectInstructionRootSource
}

func loadProjectConfigForDir(cwd string) (*ProjectConfig, error) {
	return LoadProjectConfigForDirWithError(cwd)
}

// ResolveProjectInstructionProjectRootForDir は project-level scan に使ってよい root path を返す。
func ResolveProjectInstructionProjectRootForDir(cfg *Config, cwd string) (string, bool) {
	if strings.TrimSpace(cwd) == "" {
		return "", false
	}
	cfgForLoad := cfg
	if cfgForLoad == nil {
		cfgForLoad = DefaultConfig()
	}
	projectCfg, err := loadProjectConfigForDir(cwd)
	if err != nil {
		return "", false
	}
	gitRoot := findGitRoot(cwd)
	root := resolveBundleRoot(cwd, projectCfg, gitRoot, cfgForLoad.AgentInstructions)
	if !root.Source.hasProjectRoot() || strings.TrimSpace(root.RootPath) == "" {
		return "", false
	}
	return root.RootPath, true
}

func resolveBundleRoot(cwd string, projectCfg *ProjectConfig, gitRoot string, aiCfg AgentInstructionsConfig) projectInstructionRootResolution {
	if projectCfg != nil && strings.TrimSpace(projectCfg.FilePath) != "" {
		return projectInstructionRootResolution{
			RootPath: filepath.Dir(projectCfg.FilePath),
			Source:   ProjectInstructionRootSourceProjectConfig,
		}
	}
	if gitRoot != "" {
		return projectInstructionRootResolution{
			RootPath: gitRoot,
			Source:   ProjectInstructionRootSourceGit,
		}
	}
	if guidanceRoot := findGuidanceRootUpward(cwd, aiCfg.Project.Files, aiCfg.IncludeLocalFiles); guidanceRoot != "" {
		return projectInstructionRootResolution{
			RootPath: guidanceRoot,
			Source:   ProjectInstructionRootSourceGuidance,
		}
	}
	return projectInstructionRootResolution{
		RootPath: cwd,
		Source:   ProjectInstructionRootSourceFallbackCWD,
	}
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
