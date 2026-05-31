package probe

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
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

type isolatedProbeEnvSpec struct {
	repoRootEnvKey   string
	repoRootEnvValue string
	modeRootEnvKey   string
	modeRootEnvValue string
}

func buildIsolatedProbeEnv(baseEnv []string, dirs isolatedProbeRuntimeDirs, spec isolatedProbeEnvSpec) []string {
	env := make([]string, 0, len(scratchOnlyInheritedEnvKeys)+24)

	env = append(env, collectIsolatedProbeInheritedEnv(baseEnv)...)
	env = append(env, probeHostGoRootEnv(baseEnv, spec.repoRootEnvValue)...)

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
		"GOENV=off",
		"GOTOOLCHAIN=local",
		"GOPROXY=off",
		"GOSUMDB=off",
		"GIT_CONFIG_NOSYSTEM=1",
		"GIT_CONFIG_GLOBAL="+isolatedProbeGitConfigGlobalFile(dirs),
		"GIT_CONFIG_SYSTEM="+isolatedProbeGitConfigSystemFile(dirs),
		"GIT_OPTIONAL_LOCKS=0",
		"GIT_TERMINAL_PROMPT=0",
		"NPM_CONFIG_USERCONFIG="+isolatedProbeNPMUserConfigFile(dirs),
		"NPM_CONFIG_GLOBALCONFIG="+isolatedProbeNPMGlobalConfigFile(dirs),
		"NPM_CONFIG_CACHE="+isolatedProbeNPMCacheDir(dirs),
		"NPM_CONFIG_PREFIX="+isolatedProbeNPMPrefixDir(dirs),
		"NPM_CONFIG_AUDIT=false",
		"NPM_CONFIG_FUND=false",
		"CARGO_HOME="+isolatedProbeCargoHomeDir(dirs),
		"CARGO_TARGET_DIR="+isolatedProbeCargoTargetDir(dirs),
		"PYTHONDONTWRITEBYTECODE=1",
		"PYTHONPYCACHEPREFIX="+dirs.PythonCacheDir,
		"PYTHONNOUSERSITE=1",
		"PIP_DISABLE_PIP_VERSION_CHECK=1",
		"PIP_NO_INDEX=1",
	)
	return env
}

func collectIsolatedProbeInheritedEnv(baseEnv []string) []string {
	base := collectEnvMap(baseEnv)
	inherited := make([]string, 0, len(scratchOnlyInheritedEnvKeys))
	seen := make(map[string]struct{}, len(scratchOnlyInheritedEnvKeys)+4)

	for _, key := range scratchOnlyInheritedEnvKeys {
		if value, ok := base[key]; ok {
			inherited = append(inherited, key+"="+value)
			seen[key] = struct{}{}
		}
	}

	for _, entry := range baseEnv {
		key, value, ok := splitEnvEntry(entry)
		if !ok || !strings.HasPrefix(key, "LC_") {
			continue
		}
		if _, exists := seen[key]; exists {
			continue
		}
		inherited = append(inherited, key+"="+value)
		seen[key] = struct{}{}
	}

	return inherited
}

func splitEnvEntry(entry string) (key, value string, ok bool) {
	idx := strings.IndexByte(entry, '=')
	if idx <= 0 {
		return "", "", false
	}
	return entry[:idx], entry[idx+1:], true
}

func isolatedProbeRuntimeBaseDir(dirs isolatedProbeRuntimeDirs) string {
	return filepath.Clean(filepath.Dir(dirs.HomeDir))
}

func isolatedProbeGitConfigGlobalFile(dirs isolatedProbeRuntimeDirs) string {
	return filepath.Join(dirs.HomeDir, ".gitconfig")
}

func isolatedProbeGitConfigSystemFile(dirs isolatedProbeRuntimeDirs) string {
	return filepath.Join(dirs.XDGConfigDir, "gitconfig-system")
}

func isolatedProbeNPMUserConfigFile(dirs isolatedProbeRuntimeDirs) string {
	return filepath.Join(dirs.HomeDir, ".npmrc")
}

func isolatedProbeNPMGlobalConfigFile(dirs isolatedProbeRuntimeDirs) string {
	return filepath.Join(dirs.XDGConfigDir, "npm-globalconfig")
}

func isolatedProbeNPMCacheDir(dirs isolatedProbeRuntimeDirs) string {
	return filepath.Join(dirs.CacheDir, "npm")
}

func isolatedProbeNPMPrefixDir(dirs isolatedProbeRuntimeDirs) string {
	return filepath.Join(isolatedProbeRuntimeBaseDir(dirs), "npm-prefix")
}

func isolatedProbeCargoHomeDir(dirs isolatedProbeRuntimeDirs) string {
	return filepath.Join(isolatedProbeRuntimeBaseDir(dirs), "cargo-home")
}

func isolatedProbeCargoTargetDir(dirs isolatedProbeRuntimeDirs) string {
	return filepath.Join(isolatedProbeRuntimeBaseDir(dirs), "cargo-target")
}

func isolatedProbeEmptyConfigFiles(dirs isolatedProbeRuntimeDirs) []string {
	return []string{
		isolatedProbeGitConfigGlobalFile(dirs),
		isolatedProbeGitConfigSystemFile(dirs),
		isolatedProbeNPMUserConfigFile(dirs),
		isolatedProbeNPMGlobalConfigFile(dirs),
	}
}
