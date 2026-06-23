package agent

import (
	"strings"

	"github.com/susugadx/xelyon-cli/internal/config"
)

type projectInstructionBundleCacheDecision struct {
	reuse bool
}

type projectInstructionBundleCache struct {
	agent *Agent
}

func newProjectInstructionBundleCache(agent *Agent) projectInstructionBundleCache {
	return projectInstructionBundleCache{agent: agent}
}

func (c projectInstructionBundleCache) decision(forceReload bool, input string) projectInstructionBundleCacheDecision {
	a := c.agent
	if a == nil || forceReload || !a.projectInstructionBundleLoaded {
		return projectInstructionBundleCacheDecision{}
	}
	cacheKey := c.currentKey(input)
	return projectInstructionBundleCacheDecision{
		reuse: cacheKey == a.projectInstructionBundleKey,
	}
}

func (c projectInstructionBundleCache) currentKey(input string) string {
	a := c.agent
	if a == nil {
		return ""
	}
	cwd := strings.TrimSpace(a.invocationCWD())
	if cwd == "" {
		return ""
	}
	inputPaths := projectInstructionInputPathsForAgent(a, input)
	return config.ComputeProjectInstructionBundleFingerprintForDirWithInputPaths(a.cfg(), cwd, inputPaths, a.projectInstructionBundle)
}

func (c projectInstructionBundleCache) invalidate() {
	a := c.agent
	if a == nil {
		return
	}
	a.projectInstructionBundle = nil
	a.projectInstructionBundleLoaded = false
	a.projectInstructionBundleKey = ""
}
