package probe

import (
	"os"
	"path/filepath"
	"strings"
)

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
