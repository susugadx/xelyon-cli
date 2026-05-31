package probe

type scratchOnlyDirs struct {
	ScratchDir     string
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

func prepareScratchOnlyDirs(scratchDir string) (scratchOnlyDirs, error) {
	prepared, err := prepareIsolatedProbeRuntimeDirs(isolatedProbeRuntimeLayout{
		rootDir:         scratchDir,
		workDir:         scratchDir,
		runtimeBaseDir:  scratchDir,
		createErrorName: "scratch",
	})
	if err != nil {
		return scratchOnlyDirs{}, err
	}

	return scratchOnlyDirs{
		ScratchDir:     prepared.RootDir,
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
