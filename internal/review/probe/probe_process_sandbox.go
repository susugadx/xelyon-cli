package probe

import (
	"os"
	"path/filepath"
	"runtime"
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
