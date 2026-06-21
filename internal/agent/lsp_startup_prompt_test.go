package agent

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/commandcatalog"
	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/lsp"
	"github.com/susugadx/xelyon-cli/internal/uiruntime"
)

func newLSPStartupPromptTestAgent(t *testing.T, workdir string, cfgServers map[string]lsp.ServerConfig) (*Agent, *bytes.Buffer) {
	t.Helper()
	return newLSPStartupPromptTestAgentWithProjectMarker(t, workdir, cfgServers, true)
}

func newLSPStartupPromptTestAgentWithoutProjectRoot(t *testing.T, workdir string, cfgServers map[string]lsp.ServerConfig) (*Agent, *bytes.Buffer) {
	t.Helper()
	return newLSPStartupPromptTestAgentWithProjectMarker(t, workdir, cfgServers, false)
}

func newLSPStartupPromptTestAgentWithProjectMarker(t *testing.T, workdir string, cfgServers map[string]lsp.ServerConfig, markProjectRoot bool) (*Agent, *bytes.Buffer) {
	t.Helper()
	if markProjectRoot {
		if err := os.WriteFile(filepath.Join(workdir, "xelyon.yaml"), []byte("context: test\n"), 0600); err != nil {
			t.Fatalf("WriteFile(xelyon.yaml) error = %v", err)
		}
	}

	var out bytes.Buffer
	cfg := newProjectMapDisabledConfig()
	cfg.LSP.Servers = make(map[string]config.LSPServerConfig, len(cfgServers))
	for lang, server := range cfgServers {
		cfg.LSP.Servers[lang] = config.LSPServerConfig{
			Command:  server.Command,
			Args:     server.Args,
			Disabled: server.Disabled,
		}
	}
	runtime := NewAgentRuntimeWithConfig(cfg)
	runtime.UI = uiruntime.NewRuntime(strings.NewReader(""), &out, io.Discard)

	client := lsp.NewClient(workdir)
	client.SetConfigs(cfgServers)
	client.SetOutput(&out)
	client.SetErrorOutput(&out)

	return &Agent{
		Runtime:   runtime,
		lspClient: client,
	}, &out
}

func TestCheckLSPInstallPrompt_ShowsOnlyDetectedMissingServers(t *testing.T) {
	disableColors(t)

	workdir := withTempWorkdir(t)
	if err := os.WriteFile(filepath.Join(workdir, "main.go"), []byte("package main\n"), 0600); err != nil {
		t.Fatalf("WriteFile(main.go) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(workdir, "app.ts"), []byte("export const value = 1;\n"), 0600); err != nil {
		t.Fatalf("WriteFile(app.ts) error = %v", err)
	}

	agent, out := newLSPStartupPromptTestAgent(t, workdir, map[string]lsp.ServerConfig{
		"go":         {Command: "xelyon-missing-gopls"},
		"typescript": {Command: "xelyon-missing-vtsls"},
		"python":     {Command: "xelyon-missing-pyright"},
	})

	checkLSPInstallPrompt(agent, commandcatalog.CommandSurfaceClassic)

	got := out.String()
	for _, fragment := range []string{
		"LSP servers missing for detected project languages",
		"go: go install golang.org/x/tools/gopls@latest",
		"typescript: npm i -g @vtsls/language-server typescript",
		"Install the listed command(s) in your shell",
		"lsp.skip_install_prompt: true",
	} {
		if !strings.Contains(got, fragment) {
			t.Fatalf("output missing %q:\n%s", fragment, got)
		}
	}
	for _, fragment := range []string{"python:", "/lsp install"} {
		if strings.Contains(got, fragment) {
			t.Fatalf("output should not contain %q:\n%s", fragment, got)
		}
	}
}

func TestCheckLSPInstallPrompt_SkipsWhenConfigured(t *testing.T) {
	disableColors(t)

	workdir := withTempWorkdir(t)
	if err := os.WriteFile(filepath.Join(workdir, "main.go"), []byte("package main\n"), 0600); err != nil {
		t.Fatalf("WriteFile(main.go) error = %v", err)
	}

	agent, out := newLSPStartupPromptTestAgent(t, workdir, map[string]lsp.ServerConfig{
		"go": {Command: "xelyon-missing-gopls"},
	})
	agent.cfg().LSP.SkipInstallPrompt = true

	checkLSPInstallPrompt(agent, commandcatalog.CommandSurfaceClassic)

	if out.Len() != 0 {
		t.Fatalf("expected no output when skip_install_prompt is true, got:\n%s", out.String())
	}
}

func TestCheckLSPInstallPrompt_SkipsWhenLSPDisabled(t *testing.T) {
	disableColors(t)

	workdir := withTempWorkdir(t)
	if err := os.WriteFile(filepath.Join(workdir, "main.go"), []byte("package main\n"), 0600); err != nil {
		t.Fatalf("WriteFile(main.go) error = %v", err)
	}

	agent, out := newLSPStartupPromptTestAgent(t, workdir, map[string]lsp.ServerConfig{
		"go": {Command: "xelyon-missing-gopls"},
	})
	agent.cfg().LSP.Enabled = false

	checkLSPInstallPrompt(agent, commandcatalog.CommandSurfaceClassic)

	if out.Len() != 0 {
		t.Fatalf("expected no output when lsp.enabled is false, got:\n%s", out.String())
	}
}

func TestCheckLSPInstallPrompt_SkipsWithoutProjectRoot(t *testing.T) {
	disableColors(t)

	workdir := withTempWorkdir(t)
	if err := os.WriteFile(filepath.Join(workdir, "main.go"), []byte("package main\n"), 0600); err != nil {
		t.Fatalf("WriteFile(main.go) error = %v", err)
	}

	agent, out := newLSPStartupPromptTestAgentWithoutProjectRoot(t, workdir, map[string]lsp.ServerConfig{
		"go": {Command: "xelyon-missing-gopls"},
	})

	checkLSPInstallPrompt(agent, commandcatalog.CommandSurfaceClassic)

	if out.Len() != 0 {
		t.Fatalf("expected no output without project root, got:\n%s", out.String())
	}
}

func TestCheckLSPInstallPrompt_DoesNotRequireLSPClientStatus(t *testing.T) {
	disableColors(t)

	workdir := withTempWorkdir(t)
	if err := os.WriteFile(filepath.Join(workdir, "main.go"), []byte("package main\n"), 0600); err != nil {
		t.Fatalf("WriteFile(main.go) error = %v", err)
	}

	agent, out := newLSPStartupPromptTestAgent(t, workdir, map[string]lsp.ServerConfig{
		"go": {Command: "xelyon-missing-gopls"},
	})
	agent.lspClient = nil

	checkLSPInstallPrompt(agent, commandcatalog.CommandSurfaceClassic)

	if got := out.String(); !strings.Contains(got, "go: go install golang.org/x/tools/gopls@latest") {
		t.Fatalf("expected prompt to use config availability without LSP client status, got:\n%s", got)
	}
}

func TestMissingDetectedLSPServers_UsesConfigAndInstallInfo(t *testing.T) {
	withLSPCommandAvailability(t, map[string]bool{
		"xelyon-installed-vtsls": true,
	})

	items := missingDetectedLSPServers(
		map[string]config.LSPServerConfig{
			"go":         {Command: "xelyon-missing-gopls"},
			"typescript": {Command: "xelyon-installed-vtsls"},
			"swift":      {Command: "xelyon-missing-sourcekit-lsp"},
			"rust":       {Command: "xelyon-missing-rust-analyzer", Disabled: true},
		},
		[]lsp.LanguageInfo{
			{ServerKey: "go"},
			{ServerKey: "typescript"},
			{ServerKey: "swift"},
			{ServerKey: "rust"},
			{ServerKey: "python"},
		},
	)

	if len(items) != 1 {
		t.Fatalf("len(items) = %d, want 1: %#v", len(items), items)
	}
	if items[0].serverKey != "go" || !strings.Contains(items[0].command, "gopls") {
		t.Fatalf("items[0] = %#v, want go gopls install command", items[0])
	}
}

func TestCheckLSPInstallPrompt_TUISkipsClassicOnlySlashCommands(t *testing.T) {
	disableColors(t)

	workdir := withTempWorkdir(t)
	if err := os.WriteFile(filepath.Join(workdir, "main.go"), []byte("package main\n"), 0600); err != nil {
		t.Fatalf("WriteFile(main.go) error = %v", err)
	}

	agent, out := newLSPStartupPromptTestAgent(t, workdir, map[string]lsp.ServerConfig{
		"go": {Command: "xelyon-missing-gopls"},
	})

	checkLSPInstallPrompt(agent, commandcatalog.CommandSurfaceTUI)

	got := out.String()
	for _, fragment := range []string{
		"LSP servers missing for detected project languages",
		"go: go install golang.org/x/tools/gopls@latest",
		"Install in your shell",
		"Config > LSP Servers",
	} {
		if !strings.Contains(got, fragment) {
			t.Fatalf("output missing %q:\n%s", fragment, got)
		}
	}
	if strings.Contains(got, "/lsp install") {
		t.Fatalf("TUI startup prompt should not suggest removed /lsp install:\n%s", got)
	}
	if strings.Contains(got, "/config") {
		t.Fatalf("TUI startup prompt should not suggest an install-related slash command:\n%s", got)
	}
}
