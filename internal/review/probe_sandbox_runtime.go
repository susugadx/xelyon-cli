package review

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
	}
	for _, dir := range requiredDirs {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return isolatedProbeRuntimeDirs{}, fmt.Errorf("failed to create %s directory %q: %w", layout.createErrorName, dir, err)
		}
	}

	return dirs, nil
}

type isolatedProbeEnvSpec struct {
	repoRootEnvKey   string
	repoRootEnvValue string
	modeRootEnvKey   string
	modeRootEnvValue string
}

func buildIsolatedProbeEnv(baseEnv []string, dirs isolatedProbeRuntimeDirs, spec isolatedProbeEnvSpec) []string {
	base := collectEnvMap(baseEnv)
	env := make([]string, 0, len(scratchOnlyInheritedEnvKeys)+24)

	for _, key := range scratchOnlyInheritedEnvKeys {
		if value, ok := base[key]; ok {
			env = append(env, key+"="+value)
		}
	}

	if spec.repoRootEnvKey != "" {
		env = append(env, spec.repoRootEnvKey+"="+spec.repoRootEnvValue)
	}
	if spec.modeRootEnvKey != "" {
		env = append(env, spec.modeRootEnvKey+"="+spec.modeRootEnvValue)
	}

	env = append(env,
		"HOME="+dirs.HomeDir,
		"USERPROFILE="+dirs.HomeDir,
		"TMPDIR="+dirs.TempDir,
		"TEMP="+dirs.TempDir,
		"TMP="+dirs.TempDir,
		"XDG_CACHE_HOME="+dirs.XDGCacheDir,
		"XDG_CONFIG_HOME="+dirs.XDGConfigDir,
		"XDG_DATA_HOME="+dirs.XDGDataDir,
		"GOCACHE="+dirs.GoCacheDir,
		"GOMODCACHE="+dirs.GoModCacheDir,
		"GOTMPDIR="+dirs.GoTempDir,
		"GOTOOLCHAIN=local",
		"GOPROXY=off",
		"GOSUMDB=off",
		"PYTHONDONTWRITEBYTECODE=1",
		"PYTHONPYCACHEPREFIX="+dirs.PythonCacheDir,
		"PYTHONNOUSERSITE=1",
		"PIP_DISABLE_PIP_VERSION_CHECK=1",
		"PIP_NO_INDEX=1",
	)
	return env
}
