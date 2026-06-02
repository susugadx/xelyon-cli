package probe

import "path/filepath"

type repoSandboxDirs struct {
	SandboxRoot    string
	WorktreeDir    string
	HomeDir        string
	TempDir        string
	CacheDir       string
	GoCacheDir     string
	GoModCacheDir  string
	PythonCacheDir string
	XDGCacheDir    string
	GoTempDir      string
	XDGConfigDir   string
	XDGDataDir     string
}

func prepareRepoSandboxDirs(sandboxRoot string) (repoSandboxDirs, error) {
	resolvedSandboxRoot := filepath.Clean(sandboxRoot)
	prepared, err := prepareIsolatedProbeRuntimeDirs(isolatedProbeRuntimeLayout{
		rootDir:         resolvedSandboxRoot,
		workDir:         filepath.Join(resolvedSandboxRoot, "worktree"),
		runtimeBaseDir:  filepath.Join(resolvedSandboxRoot, "runtime"),
		createErrorName: "repo sandbox",
	})
	if err != nil {
		return repoSandboxDirs{}, err
	}

	return repoSandboxDirs{
		SandboxRoot:    prepared.RootDir,
		WorktreeDir:    prepared.WorkDir,
		HomeDir:        prepared.HomeDir,
		TempDir:        prepared.TempDir,
		CacheDir:       prepared.CacheDir,
		GoCacheDir:     prepared.GoCacheDir,
		GoModCacheDir:  prepared.GoModCacheDir,
		PythonCacheDir: prepared.PythonCacheDir,
		XDGCacheDir:    prepared.XDGCacheDir,
		GoTempDir:      prepared.GoTempDir,
		XDGConfigDir:   prepared.XDGConfigDir,
		XDGDataDir:     prepared.XDGDataDir,
	}, nil
}
