package review

type repoSandboxRuntime struct {
	request repoSandboxRequest
	env     []string
	sandbox probeProcessSandbox
}

func (e *repoSandboxExecutor) prepareRuntime(req ReviewProbeRequest, sandboxRoot string) (repoSandboxRuntime, error) {
	if err := validateRepoSandboxRootOutsideRepo(e.repoRoot, sandboxRoot); err != nil {
		return repoSandboxRuntime{}, err
	}

	dirs, err := prepareRepoSandboxDirs(sandboxRoot)
	if err != nil {
		return repoSandboxRuntime{}, newBlockedCommandErrorf("failed to prepare repo_sandbox directories: %v", err)
	}
	if _, err := copyRepoToSandboxWorktree(e.repoRoot, dirs.WorktreeDir, defaultRepoSandboxCopyLimits()); err != nil {
		return repoSandboxRuntime{}, err
	}

	env := buildRepoSandboxEnv(e.baseEnv, dirs)
	normalized, err := e.validateRequest(req, dirs, env)
	if err != nil {
		return repoSandboxRuntime{}, err
	}

	sandboxReadOnlyBinds := probeGoModuleCacheReadOnlyBinds(e.baseEnv, e.repoRoot, dirs.GoModCacheDir)
	sandbox, err := newRepoSandboxProcessSandbox(dirs.SandboxRoot, sandboxReadOnlyBinds...)
	if err != nil {
		return repoSandboxRuntime{}, err
	}

	return repoSandboxRuntime{
		request: normalized,
		env:     env,
		sandbox: sandbox,
	}, nil
}

func validateRepoSandboxRootOutsideRepo(repoRoot, sandboxRoot string) error {
	return validateIsolatedRootOutsideRepo(repoRoot, sandboxRoot, "repo_sandbox")
}
