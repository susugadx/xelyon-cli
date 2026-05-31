package probe

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWithProbeGoRootEnvForCommandInfersFromResolvedGoPath(t *testing.T) {
	goRoot := createProbeTestGoRoot(t, filepath.Join(t.TempDir(), "go"))
	cmd := withProbeGoRootEnvForCommand(probeExecCommand{
		command:     "go",
		commandPath: filepath.Join(goRoot, "bin", "go"),
		env:         []string{"PATH=" + filepath.Join(goRoot, "bin")},
	})

	if got := envValue(cmd.env, probeGoRootEnvKey); got != goRoot {
		t.Fatalf("GOROOT = %q, want %q", got, goRoot)
	}
}

func TestWithProbeGoRootEnvForCommandKeepsExistingGoRoot(t *testing.T) {
	existingGoRoot := createProbeTestGoRoot(t, filepath.Join(t.TempDir(), "existing-go"))
	commandGoRoot := createProbeTestGoRoot(t, filepath.Join(t.TempDir(), "command-go"))
	cmd := withProbeGoRootEnvForCommand(probeExecCommand{
		command:     "go",
		commandPath: filepath.Join(commandGoRoot, "bin", "go"),
		env:         []string{probeGoRootEnvKey + "=" + existingGoRoot},
	})

	if got := envValue(cmd.env, probeGoRootEnvKey); got != existingGoRoot {
		t.Fatalf("GOROOT = %q, want existing %q", got, existingGoRoot)
	}
}

func createProbeTestGoRoot(t *testing.T, goRoot string) string {
	t.Helper()

	goRoot = filepath.Clean(goRoot)
	for _, dir := range []string{
		filepath.Join(goRoot, "bin"),
		filepath.Join(goRoot, "pkg", "tool"),
		filepath.Join(goRoot, "src"),
	} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("MkdirAll(%q) error = %v", dir, err)
		}
	}
	goPath := filepath.Join(goRoot, "bin", "go")
	if err := os.WriteFile(goPath, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatalf("WriteFile(%q) error = %v", goPath, err)
	}
	return goRoot
}
