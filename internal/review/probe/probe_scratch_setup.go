package probe

type scratchOnlyRuntime struct {
	request scratchOnlyRequest
	env     []string
	sandbox probeProcessSandbox
}

func (e *scratchOnlyExecutor) prepareRuntime(req ReviewProbeRequest, scratchDir string) (scratchOnlyRuntime, error) {
	if err := validateScratchDirOutsideRepo(e.repoRoot, scratchDir); err != nil {
		return scratchOnlyRuntime{}, err
	}

	dirs, err := prepareScratchOnlyDirs(scratchDir)
	if err != nil {
		return scratchOnlyRuntime{}, newBlockedCommandErrorf("failed to prepare scratch directories: %v", err)
	}
	env := buildScratchOnlyEnv(e.baseEnv, e.repoRoot, dirs)

	normalized, err := e.validateRequest(req, dirs.ScratchDir, env)
	if err != nil {
		return scratchOnlyRuntime{}, err
	}

	sandboxReadOnlyBinds := probeGoModuleCacheReadOnlyBinds(e.baseEnv, e.repoRoot, dirs.GoModCacheDir)
	sandbox, err := newScratchOnlyProcessSandbox(e.repoRoot, dirs.ScratchDir, sandboxReadOnlyBinds...)
	if err != nil {
		return scratchOnlyRuntime{}, err
	}

	return scratchOnlyRuntime{
		request: normalized,
		env:     env,
		sandbox: sandbox,
	}, nil
}
