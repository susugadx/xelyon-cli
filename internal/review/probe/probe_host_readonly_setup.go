package probe

func (e *hostReadOnlyExecutor) prepareRuntime(req ReviewProbeRequest, runtimeRoot string) (hostReadOnlyRuntime, error) {
	if err := validateHostReadOnlyRuntimeRootOutsideRepo(e.repoRoot, runtimeRoot); err != nil {
		return hostReadOnlyRuntime{}, err
	}

	dirs, err := prepareHostReadOnlyRuntimeDirs(runtimeRoot)
	if err != nil {
		return hostReadOnlyRuntime{}, newBlockedCommandErrorf("failed to prepare host_readonly runtime directories: %v", err)
	}
	env := buildHostReadOnlyEnv(e.baseEnv, e.repoRoot, dirs)

	normalized, err := e.validateRequest(req, env, dirs.RootDir)
	if err != nil {
		return hostReadOnlyRuntime{}, err
	}

	sandboxReadOnlyBinds := probeGoModuleCacheReadOnlyBinds(e.baseEnv, e.repoRoot, dirs.GoModCacheDir)
	sandbox, err := newHostReadOnlyProcessSandbox(e.repoRoot, dirs.RootDir, sandboxReadOnlyBinds...)
	if err != nil {
		return hostReadOnlyRuntime{}, err
	}

	return hostReadOnlyRuntime{
		request: normalized,
		env:     env,
		sandbox: sandbox,
	}, nil
}

func (e *hostReadOnlyExecutor) cleanupHostReadOnlyRuntimeRoot(result *ReviewProbeResult, runtimeRoot string) {
	if err := e.removeAll(runtimeRoot); err != nil {
		appendHostReadOnlyCleanupError(result, runtimeRoot, err)
	}
}

func appendHostReadOnlyCleanupError(result *ReviewProbeResult, runtimeRoot string, err error) {
	appendIsolatedCleanupError(result, "host_readonly runtime root", runtimeRoot, err)
}

func validateHostReadOnlyRuntimeRootOutsideRepo(repoRoot, runtimeRoot string) error {
	return validateIsolatedRootOutsideRepo(repoRoot, runtimeRoot, "host_readonly runtime")
}
