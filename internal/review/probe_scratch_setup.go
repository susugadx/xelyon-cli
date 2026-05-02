package review

type scratchOnlyRuntime struct {
	request scratchOnlyRequest
	dirs    scratchOnlyDirs
	env     []string
}

func (e *scratchOnlyExecutor) prepareRuntime(req ReviewProbeRequest, scratchDir string) (scratchOnlyRuntime, error) {
	if err := validateScratchDirOutsideRepo(e.repoRoot, scratchDir); err != nil {
		return scratchOnlyRuntime{}, err
	}

	dirs, err := prepareScratchOnlyDirs(scratchDir)
	if err != nil {
		return scratchOnlyRuntime{}, newBlockedCommandErrorf("failed to prepare scratch directories: %v", err)
	}

	normalized, err := e.validateRequest(req, dirs.ScratchDir)
	if err != nil {
		return scratchOnlyRuntime{}, err
	}

	return scratchOnlyRuntime{
		request: normalized,
		dirs:    dirs,
		env:     buildScratchOnlyEnv(e.baseEnv, e.repoRoot, dirs),
	}, nil
}
