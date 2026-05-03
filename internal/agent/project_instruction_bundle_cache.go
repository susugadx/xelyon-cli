package agent

import (
	"strings"

	"github.com/susugadx/xelyon-cli/internal/config"
)

type projectInstructionBundleCacheDecision struct {
	reuse    bool
	cacheKey string
}

type projectInstructionBundleCache struct {
	agent *Agent
}

func newProjectInstructionBundleCache(agent *Agent) projectInstructionBundleCache {
	return projectInstructionBundleCache{agent: agent}
}

func (c projectInstructionBundleCache) decision(forceReload bool) projectInstructionBundleCacheDecision {
	a := c.agent
	if a == nil || forceReload || !a.projectInstructionBundleLoaded {
		return projectInstructionBundleCacheDecision{}
	}
	cacheKey := c.currentKey()
	return projectInstructionBundleCacheDecision{
		reuse:    cacheKey == a.projectInstructionBundleKey,
		cacheKey: cacheKey,
	}
}

func (c projectInstructionBundleCache) currentKey() string {
	a := c.agent
	if a == nil {
		return ""
	}
	cwd := strings.TrimSpace(a.invocationCWD())
	if cwd == "" {
		return ""
	}
	return config.ComputeProjectInstructionBundleFingerprintForDir(a.cfg(), cwd, a.projectInstructionBundle)
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
