package agent

import (
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/lsp"
	"github.com/susugadx/xelyon-cli/internal/ui"
)

func newLSPCommandTestAgent(t *testing.T, workdir string, configs map[string]lsp.ServerConfig) (*Agent, *lsp.Client, *bytes.Buffer) {
	t.Helper()

	var out bytes.Buffer
	runtime := NewAgentRuntimeWithConfig(newProjectMapDisabledConfig())
	runtime.UI = ui.NewRuntime(strings.NewReader(""), &out, &out)

	client := lsp.NewClient(workdir)
	client.SetConfigs(configs)
	client.SetOutput(&out)
	client.SetErrorOutput(&out)

	agent := &Agent{
		Runtime:   runtime,
		lspClient: client,
	}
	return agent, client, &out
}

func TestHandleLSPCommand_NoClient(t *testing.T) {
	disableColors(t)

	var out bytes.Buffer
	runtime := NewAgentRuntimeWithConfig(newProjectMapDisabledConfig())
	runtime.UI = ui.NewRuntime(strings.NewReader(""), &out, &out)
	agent := &Agent{Runtime: runtime}

	if !handleLSPCommand(agent, nil) {
		t.Fatal("handleLSPCommand() = false, want true")
	}
	if !strings.Contains(out.String(), "LSP is not enabled.") {
		t.Fatalf("output = %q, want disabled message", out.String())
	}
}

func TestHandleLSPCommand_UnknownSubcommand(t *testing.T) {
	disableColors(t)

	workdir := withTempWorkdir(t)
	agent, _, out := newLSPCommandTestAgent(t, workdir, map[string]lsp.ServerConfig{})

	if !handleLSPCommand(agent, []string{"mystery"}) {
		t.Fatal("handleLSPCommand() = false, want true")
	}
	if !strings.Contains(out.String(), "Unknown subcommand: mystery") {
		t.Fatalf("output = %q, want unknown subcommand", out.String())
	}
}

func TestHandleLSPStatus_NoConfiguredServers(t *testing.T) {
	disableColors(t)

	workdir := withTempWorkdir(t)
	agent, client, out := newLSPCommandTestAgent(t, workdir, map[string]lsp.ServerConfig{})

	if !handleLSPStatus(agent, client) {
		t.Fatal("handleLSPStatus() = false, want true")
	}
	if !strings.Contains(out.String(), "No LSP servers configured") {
		t.Fatalf("output = %q, want no configured servers", out.String())
	}
}

func TestHandleLSPStatus_ShowsDetectedLanguagesAndMissingServers(t *testing.T) {
	disableColors(t)

	workdir := withTempWorkdir(t)
	if err := os.WriteFile(filepath.Join(workdir, "main.go"), []byte("package main\n"), 0600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	agent, client, out := newLSPCommandTestAgent(t, workdir, map[string]lsp.ServerConfig{
		"go": {Command: "xelyon-missing-gopls"},
	})

	if !handleLSPStatus(agent, client) {
		t.Fatal("handleLSPStatus() = false, want true")
	}

	got := out.String()
	for _, fragment := range []string{
		"Detected languages in project: go",
		"Missing LSP servers:",
		"go: go install golang.org/x/tools/gopls@latest",
	} {
		if !strings.Contains(got, fragment) {
			t.Fatalf("output missing %q:\n%s", fragment, got)
		}
	}
}

func TestHandleLSPDetect_NotConfigured(t *testing.T) {
	disableColors(t)

	workdir := withTempWorkdir(t)
	if err := os.WriteFile(filepath.Join(workdir, "main.go"), []byte("package main\n"), 0600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	agent, client, out := newLSPCommandTestAgent(t, workdir, map[string]lsp.ServerConfig{})

	if !handleLSPDetect(agent, client) {
		t.Fatal("handleLSPDetect() = false, want true")
	}
	if !strings.Contains(out.String(), "❓ go: not configured") {
		t.Fatalf("output = %q, want not configured status", out.String())
	}
}

func TestHandleLSPDetect_NotInstalledSuggestion(t *testing.T) {
	disableColors(t)

	workdir := withTempWorkdir(t)
	if err := os.WriteFile(filepath.Join(workdir, "main.go"), []byte("package main\n"), 0600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	agent, client, out := newLSPCommandTestAgent(t, workdir, map[string]lsp.ServerConfig{
		"go": {Command: "xelyon-missing-gopls"},
	})

	if !handleLSPDetect(agent, client) {
		t.Fatal("handleLSPDetect() = false, want true")
	}
	got := out.String()
	if !strings.Contains(got, "❌ go: not installed") || !strings.Contains(got, "Run '/lsp install go'") {
		t.Fatalf("output = %q, want install suggestion", got)
	}
}

func TestHandleLSPInstall_NoArgsAndUnknownLanguage(t *testing.T) {
	disableColors(t)

	workdir := withTempWorkdir(t)
	agent, client, out := newLSPCommandTestAgent(t, workdir, map[string]lsp.ServerConfig{})

	if !handleLSPInstall(agent, client, nil) {
		t.Fatal("handleLSPInstall(nil) = false, want true")
	}
	if !strings.Contains(out.String(), "Usage: /lsp install <language|all>") {
		t.Fatalf("output = %q, want usage", out.String())
	}

	out.Reset()
	if !handleLSPInstall(agent, client, []string{"mystery"}) {
		t.Fatal("handleLSPInstall(unknown) = false, want true")
	}
	if !strings.Contains(out.String(), "Unknown language: mystery") {
		t.Fatalf("output = %q, want unknown language", out.String())
	}
}

func TestHandleLSPInstall_AlreadyInstalled(t *testing.T) {
	disableColors(t)

	workdir := withTempWorkdir(t)
	agent, client, out := newLSPCommandTestAgent(t, workdir, map[string]lsp.ServerConfig{
		"go": {Command: "sh"},
	})

	if !handleLSPInstall(agent, client, []string{"go"}) {
		t.Fatal("handleLSPInstall(go) = false, want true")
	}
	if !strings.Contains(out.String(), "already installed") {
		t.Fatalf("output = %q, want already installed", out.String())
	}
}

func TestHandleLSPInstall_FailureAndSuccess(t *testing.T) {
	disableColors(t)

	oldRunInstall := lspCommandRunInstallWithIO
	t.Cleanup(func() { lspCommandRunInstallWithIO = oldRunInstall })

	workdir := withTempWorkdir(t)
	agent, client, out := newLSPCommandTestAgent(t, workdir, map[string]lsp.ServerConfig{
		"go": {Command: "xelyon-missing-gopls"},
	})

	lspCommandRunInstallWithIO = func(serverKey string, in io.Reader, outWriter, errOut io.Writer) error {
		return errors.New("boom")
	}
	if !handleLSPInstall(agent, client, []string{"go"}) {
		t.Fatal("handleLSPInstall(go failure) = false, want true")
	}
	if !strings.Contains(out.String(), "Installation failed: boom") || !strings.Contains(out.String(), "Try installing manually:") {
		t.Fatalf("failure output = %q, want manual install guidance", out.String())
	}

	out.Reset()
	lspCommandRunInstallWithIO = func(serverKey string, in io.Reader, outWriter, errOut io.Writer) error {
		return nil
	}
	if !handleLSPInstall(agent, client, []string{"go"}) {
		t.Fatal("handleLSPInstall(go success) = false, want true")
	}
	if !strings.Contains(out.String(), "installed successfully") {
		t.Fatalf("success output = %q, want success message", out.String())
	}
}

func TestHandleLSPInstallAll(t *testing.T) {
	disableColors(t)

	oldRunInstall := lspCommandRunInstallWithIO
	t.Cleanup(func() { lspCommandRunInstallWithIO = oldRunInstall })

	workdir := withTempWorkdir(t)
	agent, client, out := newLSPCommandTestAgent(t, workdir, map[string]lsp.ServerConfig{
		"go": {Command: "sh"},
	})

	if !handleLSPInstallAll(agent, client) {
		t.Fatal("handleLSPInstallAll(no missing) = false, want true")
	}
	if !strings.Contains(out.String(), "All configured LSP servers are already installed") {
		t.Fatalf("output = %q, want all installed", out.String())
	}

	out.Reset()
	client.SetConfigs(map[string]lsp.ServerConfig{
		"go":     {Command: "xelyon-missing-gopls"},
		"python": {Command: "xelyon-missing-pyright"},
	})
	lspCommandRunInstallWithIO = func(serverKey string, in io.Reader, outWriter, errOut io.Writer) error {
		if serverKey == "go" {
			return nil
		}
		return errors.New("boom")
	}

	if !handleLSPInstallAll(agent, client) {
		t.Fatal("handleLSPInstallAll(mixed) = false, want true")
	}
	if !strings.Contains(out.String(), "1 of 2 servers installed successfully.") {
		t.Fatalf("output = %q, want partial success summary", out.String())
	}
}
