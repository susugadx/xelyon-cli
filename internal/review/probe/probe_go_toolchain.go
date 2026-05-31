package probe

import (
	"os"
	"path/filepath"
	"strings"
)

const probeGoRootEnvKey = "GOROOT"

func probeHostGoRootDir(baseEnv []string, repoRoot string) (string, bool) {
	return validateProbeHostGoRootDir(envValue(baseEnv, probeGoRootEnvKey), repoRoot)
}

func probeHostGoRootEnv(baseEnv []string, repoRoot string) []string {
	goRoot, ok := probeHostGoRootDir(baseEnv, repoRoot)
	if !ok {
		return nil
	}
	return []string{probeGoRootEnvKey + "=" + goRoot}
}

func withProbeGoRootEnvForCommand(cmd probeExecCommand) probeExecCommand {
	if _, ok := probeHostGoRootDir(cmd.env, ""); ok {
		return cmd
	}
	goRoot, ok := probeGoRootFromCommandPath(cmd.command, cmd.commandPath)
	if !ok {
		return cmd
	}
	cmd.env = append(append([]string(nil), cmd.env...), probeGoRootEnvKey+"="+goRoot)
	return cmd
}

func probeGoRootFromCommandPath(command, commandPath string) (string, bool) {
	if !isProbeGoCommand(command, commandPath) {
		return "", false
	}

	evaluated, err := filepath.EvalSymlinks(filepath.Clean(commandPath))
	if err != nil {
		return "", false
	}
	goBinDir := filepath.Dir(evaluated)
	if filepath.Base(goBinDir) != "bin" {
		return "", false
	}
	return validateProbeHostGoRootDir(filepath.Dir(goBinDir), "")
}

func isProbeGoCommand(command, commandPath string) bool {
	command = strings.TrimSpace(command)
	if command == "go" || strings.EqualFold(command, "go.exe") {
		return true
	}
	base := filepath.Base(strings.TrimSpace(commandPath))
	return base == "go" || strings.EqualFold(base, "go.exe")
}

func validateProbeHostGoRootDir(candidate, repoRoot string) (string, bool) {
	goRoot, ok := validateProbeHostReadOnlyDir(candidate, repoRoot)
	if !ok {
		return "", false
	}
	if !probeGoRootHasDir(goRoot, "pkg", "tool") {
		return "", false
	}
	if !probeGoRootHasDir(goRoot, "src") {
		return "", false
	}
	return goRoot, true
}

func probeGoRootHasDir(goRoot string, parts ...string) bool {
	info, err := os.Stat(filepath.Join(append([]string{goRoot}, parts...)...))
	return err == nil && info.IsDir()
}

func probeGoRootReadOnlyBind(env []string) (probeProcessSandboxBind, bool) {
	goRoot, ok := validateProbeHostGoRootDir(envValue(env, probeGoRootEnvKey), "")
	if !ok {
		return probeProcessSandboxBind{}, false
	}
	return probeProcessSandboxBind{source: goRoot, target: goRoot}, true
}
