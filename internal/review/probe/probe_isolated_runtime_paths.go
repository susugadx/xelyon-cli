package probe

import "path/filepath"

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
