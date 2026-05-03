package agent

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/susugadx/xelyon-cli/internal/config"
)

func TestAgent_LoadProjectInstructionBundleCached_UsesInvocationCWDForLoad(t *testing.T) {
	processCWD := t.TempDir()
	invocationCWD := t.TempDir()

	writeTestFile(t, filepath.Join(processCWD, "xelyon.yaml"), "context: \"process\"\n")
	writeTestFile(t, filepath.Join(invocationCWD, "xelyon.yaml"), "context: \"invocation\"\n")

	originalDir, _ := os.Getwd()
	defer func() { _ = os.Chdir(originalDir) }()
	_ = os.Chdir(processCWD)

	runtime := NewAgentRuntimeWithConfig(config.DefaultConfig())
	runtime.InvocationCWD = invocationCWD
	agent := &Agent{Runtime: runtime}

	bundle := agent.loadProjectInstructionBundleCached(true)
	if bundle == nil || bundle.ProjectConfig == nil {
		t.Fatal("expected non-nil project instruction bundle")
	}
	if bundle.ProjectConfig.Context != "invocation" {
		t.Fatalf("bundle project context = %q, want %q", bundle.ProjectConfig.Context, "invocation")
	}
}

func TestAgent_LoadProjectInstructionBundleCached_InvocationCWDCacheKeyMatchesLoadedBundle(t *testing.T) {
	processCWD := t.TempDir()
	invocationCWD := t.TempDir()
	invocationConfigPath := filepath.Join(invocationCWD, "xelyon.yaml")

	writeTestFile(t, filepath.Join(processCWD, "xelyon.yaml"), "context: \"process\"\n")
	writeTestFile(t, invocationConfigPath, "context: \"invocation-v1\"\n")

	originalDir, _ := os.Getwd()
	defer func() { _ = os.Chdir(originalDir) }()
	_ = os.Chdir(processCWD)

	runtime := NewAgentRuntimeWithConfig(config.DefaultConfig())
	runtime.InvocationCWD = invocationCWD
	agent := &Agent{Runtime: runtime}

	first := agent.loadProjectInstructionBundleCached(true)
	second := agent.loadProjectInstructionBundleCached(false)
	if first == nil || second == nil {
		t.Fatal("expected non-nil bundles")
	}
	if first != second {
		t.Fatal("expected cached bundle pointer before invocation cwd file change")
	}
	if second.ProjectConfig == nil || second.ProjectConfig.Context != "invocation-v1" {
		t.Fatalf("bundle project context = %q, want %q", second.ProjectConfig.Context, "invocation-v1")
	}

	writeTestFile(t, invocationConfigPath, "context: \"invocation-v2\"\n")
	touchTestFile(t, invocationConfigPath, time.Now().Add(2*time.Second))

	third := agent.loadProjectInstructionBundleCached(false)
	if third == nil || third.ProjectConfig == nil {
		t.Fatal("expected non-nil bundle after invocation cwd config change")
	}
	if third == second {
		t.Fatal("expected cache reload when invocation cwd project config changes")
	}
	if third.ProjectConfig.Context != "invocation-v2" {
		t.Fatalf("bundle project context = %q, want %q", third.ProjectConfig.Context, "invocation-v2")
	}
}
