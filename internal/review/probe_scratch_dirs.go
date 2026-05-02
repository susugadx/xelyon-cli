package review

import (
	"fmt"
	"os"
	"path/filepath"
)

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
	resolvedScratchDir := filepath.Clean(scratchDir)
	dirs := scratchOnlyDirs{
		ScratchDir:     resolvedScratchDir,
		HomeDir:        filepath.Join(resolvedScratchDir, "home"),
		TempDir:        filepath.Join(resolvedScratchDir, "tmp"),
		CacheDir:       filepath.Join(resolvedScratchDir, "cache"),
		GoCacheDir:     filepath.Join(resolvedScratchDir, "cache", "go-build"),
		GoModCacheDir:  filepath.Join(resolvedScratchDir, "cache", "go-mod"),
		PythonCacheDir: filepath.Join(resolvedScratchDir, "cache", "pycache"),
		XDGCacheDir:    filepath.Join(resolvedScratchDir, "cache", "xdg"),
		GoTempDir:      filepath.Join(resolvedScratchDir, "tmp", "go"),
		XDGConfigDir:   filepath.Join(resolvedScratchDir, "home", ".config"),
		XDGDataDir:     filepath.Join(resolvedScratchDir, "home", ".local", "share"),
	}

	requiredDirs := []string{
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
	}
	for _, dir := range requiredDirs {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return scratchOnlyDirs{}, fmt.Errorf("failed to create scratch directory %q: %w", dir, err)
		}
	}

	return dirs, nil
}
