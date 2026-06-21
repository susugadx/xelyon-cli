package probe

import (
	"fmt"
	"os"
	"path/filepath"
)

type isolatedProbeRuntimeDirs struct {
	RootDir        string
	WorkDir        string
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

type isolatedProbeRuntimeLayout struct {
	rootDir         string
	workDir         string
	runtimeBaseDir  string
	createErrorName string
}

func prepareIsolatedProbeRuntimeDirs(layout isolatedProbeRuntimeLayout) (isolatedProbeRuntimeDirs, error) {
	resolvedRoot := filepath.Clean(layout.rootDir)
	resolvedWorkDir := filepath.Clean(layout.workDir)
	resolvedRuntimeBase := filepath.Clean(layout.runtimeBaseDir)

	dirs := isolatedProbeRuntimeDirs{
		RootDir:        resolvedRoot,
		WorkDir:        resolvedWorkDir,
		HomeDir:        filepath.Join(resolvedRuntimeBase, "home"),
		TempDir:        filepath.Join(resolvedRuntimeBase, "tmp"),
		CacheDir:       filepath.Join(resolvedRuntimeBase, "cache"),
		GoCacheDir:     filepath.Join(resolvedRuntimeBase, "cache", "go-build"),
		GoModCacheDir:  filepath.Join(resolvedRuntimeBase, "cache", "go-mod"),
		PythonCacheDir: filepath.Join(resolvedRuntimeBase, "cache", "pycache"),
		XDGCacheDir:    filepath.Join(resolvedRuntimeBase, "cache", "xdg"),
		GoTempDir:      filepath.Join(resolvedRuntimeBase, "tmp", "go"),
		XDGConfigDir:   filepath.Join(resolvedRuntimeBase, "home", ".config"),
		XDGDataDir:     filepath.Join(resolvedRuntimeBase, "home", ".local", "share"),
	}

	requiredDirs := []string{
		dirs.WorkDir,
		dirs.HomeDir,
		dirs.TempDir,
		dirs.CacheDir,
		dirs.GoCacheDir,
		dirs.GoModCacheDir,
		dirs.PythonCacheDir,
		dirs.XDGCacheDir,
		dirs.GoTempDir,
		dirs.XDGConfigDir,
		dirs.XDGDataDir,
		isolatedProbeNPMCacheDir(dirs),
		isolatedProbeNPMPrefixDir(dirs),
		isolatedProbeCargoHomeDir(dirs),
		isolatedProbeCargoTargetDir(dirs),
	}
	for _, dir := range requiredDirs {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return isolatedProbeRuntimeDirs{}, fmt.Errorf("failed to create %s directory %q: %w", layout.createErrorName, dir, err)
		}
	}

	for _, file := range isolatedProbeEmptyConfigFiles(dirs) {
		if err := os.WriteFile(file, nil, 0o644); err != nil {
			return isolatedProbeRuntimeDirs{}, fmt.Errorf("failed to create %s config file %q: %w", layout.createErrorName, file, err)
		}
	}

	return dirs, nil
}
