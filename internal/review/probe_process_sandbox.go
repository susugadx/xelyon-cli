package review

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

const probeProcessSandboxCommand = "bwrap"

type probeProcessSandbox struct {
	enabled        bool
	runnerPath     string
	readOnlyBinds  []probeProcessSandboxBind
	readWriteBinds []probeProcessSandboxBind
}

type probeProcessSandboxBind struct {
	source string
	target string
}

func newHostReadOnlyProcessSandbox(repoRoot, runtimeRoot string, extraReadOnlyBinds ...probeProcessSandboxBind) (probeProcessSandbox, error) {
	readOnlyBinds := []probeProcessSandboxBind{
		{source: repoRoot, target: repoRoot},
	}
	readOnlyBinds = append(readOnlyBinds, extraReadOnlyBinds...)
	return newBubblewrapProcessSandbox(
		readOnlyBinds,
		[]probeProcessSandboxBind{
			{source: runtimeRoot, target: runtimeRoot},
		},
	)
}

func newScratchOnlyProcessSandbox(repoRoot, scratchDir string, extraReadOnlyBinds ...probeProcessSandboxBind) (probeProcessSandbox, error) {
	readOnlyBinds := []probeProcessSandboxBind{
		{source: repoRoot, target: repoRoot},
	}
	readOnlyBinds = append(readOnlyBinds, extraReadOnlyBinds...)
	return newBubblewrapProcessSandbox(
		readOnlyBinds,
		[]probeProcessSandboxBind{
			{source: scratchDir, target: scratchDir},
		},
	)
}

func newRepoSandboxProcessSandbox(sandboxRoot string, extraReadOnlyBinds ...probeProcessSandboxBind) (probeProcessSandbox, error) {
	return newBubblewrapProcessSandbox(extraReadOnlyBinds, []probeProcessSandboxBind{
		{source: sandboxRoot, target: sandboxRoot},
	})
}

func newBubblewrapProcessSandbox(readOnlyBinds, readWriteBinds []probeProcessSandboxBind) (probeProcessSandbox, error) {
	if runtime.GOOS != "linux" {
		return probeProcessSandbox{}, newBlockedCommandErrorf("process sandbox requires Linux bubblewrap; current OS is %s", runtime.GOOS)
	}

	bwrapPath, ok := trustedBubblewrapPath()
	if !ok {
		return probeProcessSandbox{}, newBlockedCommandErrorf("process sandbox requires trusted %s at /usr/bin/bwrap or /bin/bwrap", probeProcessSandboxCommand)
	}

	return probeProcessSandbox{
		enabled:        true,
		runnerPath:     bwrapPath,
		readOnlyBinds:  cloneProbeProcessSandboxBinds(readOnlyBinds),
		readWriteBinds: cloneProbeProcessSandboxBinds(readWriteBinds),
	}, nil
}

func trustedBubblewrapPath() (string, bool) {
	for _, candidate := range []string{"/usr/bin/bwrap", "/bin/bwrap"} {
		info, err := os.Stat(candidate)
		if err != nil || info.IsDir() || info.Mode().Perm()&0o111 == 0 {
			continue
		}
		return candidate, true
	}
	return "", false
}

func cloneProbeProcessSandboxBinds(binds []probeProcessSandboxBind) []probeProcessSandboxBind {
	if len(binds) == 0 {
		return nil
	}
	cloned := make([]probeProcessSandboxBind, 0, len(binds))
	for _, bind := range binds {
		cloned = append(cloned, probeProcessSandboxBind{
			source: filepath.Clean(bind.source),
			target: filepath.Clean(bind.target),
		})
	}
	return cloned
}

func buildProbeProcessSandboxExec(cmd probeExecCommand) (probeExecCommand, error) {
	if !cmd.sandbox.enabled {
		return cmd, nil
	}

	args, err := cmd.sandbox.buildBubblewrapArgs(cmd)
	if err != nil {
		return probeExecCommand{}, err
	}

	wrapped := cmd
	wrapped.commandPath = cmd.sandbox.runnerPath
	wrapped.args = args
	return wrapped, nil
}

func (s probeProcessSandbox) buildBubblewrapArgs(cmd probeExecCommand) ([]string, error) {
	binds, err := s.bubblewrapBindArgs(cmd)
	if err != nil {
		return nil, err
	}

	args := []string{
		"--unshare-all",
		// 一部 CI では network namespace 内の loopback 設定が拒否される。
		// probe の安全境界は filesystem sandbox なので、network は host と共有する。
		"--share-net",
		"--die-with-parent",
		"--new-session",
		"--proc", "/proc",
		"--dev", "/dev",
		"--tmpfs", "/tmp",
		"--dir", "/var",
		"--tmpfs", "/var/tmp",
	}
	args = append(args, binds...)
	args = append(args, "--clearenv")
	args = append(args, bubblewrapEnvArgs(cmd.env)...)
	args = append(args, "--chdir", cmd.workDir)
	args = append(args, cmd.commandPath)
	args = append(args, cmd.args...)
	return args, nil
}

func (s probeProcessSandbox) bubblewrapBindArgs(cmd probeExecCommand) ([]string, error) {
	binds := make([]probeProcessSandboxBind, 0, 16)

	for _, root := range defaultProbeProcessSandboxSystemRoots() {
		binds = append(binds, probeProcessSandboxBind{source: root, target: root})
	}
	if goRootBind, ok := probeGoRootReadOnlyBind(cmd.env); ok {
		binds = append(binds, goRootBind)
	}
	commandPath := cmd.commandPath
	if commandRoot := probeProcessSandboxCommandRoot(commandPath); commandRoot != "" {
		binds = append(binds, probeProcessSandboxBind{source: commandRoot, target: commandRoot})
	}
	binds = append(binds, s.readOnlyBinds...)

	args := make([]string, 0, len(binds)*3+len(s.readWriteBinds)*3)
	for _, bind := range s.readWriteBinds {
		cleaned, ok := cleanProbeProcessSandboxBind(bind)
		if !ok {
			continue
		}
		args = append(args, "--bind", cleaned.source, cleaned.target)
	}

	seen := map[string]struct{}{}
	for _, bind := range binds {
		cleaned, ok := cleanProbeProcessSandboxBind(bind)
		if !ok {
			continue
		}
		if alreadyCoveredByProbeProcessBind(cleaned, seen) {
			continue
		}
		seen[cleaned.target] = struct{}{}
		args = append(args, "--ro-bind", cleaned.source, cleaned.target)
	}
	return args, nil
}

func defaultProbeProcessSandboxSystemRoots() []string {
	candidates := []string{"/usr", "/bin", "/lib", "/lib64", "/sbin"}
	roots := make([]string, 0, len(candidates))
	for _, root := range candidates {
		if _, err := os.Lstat(root); err == nil {
			roots = append(roots, root)
		}
	}
	return roots
}

func probeProcessSandboxCommandRoot(commandPath string) string {
	cleaned := filepath.Clean(commandPath)
	if cleaned == "" || !filepath.IsAbs(cleaned) {
		return ""
	}
	for _, root := range defaultProbeProcessSandboxSystemRoots() {
		if inside, err := isPathWithinRepoRoot(root, cleaned); err == nil && inside {
			return ""
		}
	}

	if nvmRoot := probeProcessSandboxNVMNodeRoot(cleaned); nvmRoot != "" {
		return nvmRoot
	}
	return filepath.Dir(cleaned)
}

func probeProcessSandboxNVMNodeRoot(path string) string {
	parts := strings.Split(filepath.ToSlash(filepath.Clean(path)), "/")
	for i := 0; i+3 < len(parts); i++ {
		if parts[i] == ".nvm" && parts[i+1] == "versions" && parts[i+2] == "node" {
			return filepath.FromSlash(strings.Join(parts[:i+4], "/"))
		}
	}
	return ""
}

func cleanProbeProcessSandboxBind(bind probeProcessSandboxBind) (probeProcessSandboxBind, bool) {
	source := strings.TrimSpace(bind.source)
	target := strings.TrimSpace(bind.target)
	if source == "" || target == "" {
		return probeProcessSandboxBind{}, false
	}
	source = filepath.Clean(source)
	target = filepath.Clean(target)
	return probeProcessSandboxBind{source: source, target: target}, true
}

func alreadyCoveredByProbeProcessBind(bind probeProcessSandboxBind, seen map[string]struct{}) bool {
	for target := range seen {
		inside, err := isPathWithinRepoRoot(target, bind.target)
		if err == nil && inside {
			return true
		}
	}
	return false
}

func bubblewrapEnvArgs(env []string) []string {
	args := make([]string, 0, len(env)*3)
	for _, entry := range env {
		key, value, ok := splitEnvEntry(entry)
		if !ok {
			continue
		}
		args = append(args, "--setenv", key, value)
	}
	return args
}
