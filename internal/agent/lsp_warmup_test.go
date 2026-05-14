package agent

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/lsp"
)

func TestLSPWarmupTargets_UsesDetectedInstalledProjectLanguages(t *testing.T) {
	workdir := withTempWorkdir(t)
	if err := os.WriteFile(filepath.Join(workdir, "app.ts"), []byte("export const value = 1;\n"), 0600); err != nil {
		t.Fatalf("WriteFile(app.ts) error = %v", err)
	}

	withLSPCommandAvailability(t, map[string]bool{
		"gopls": true,
		"vtsls": true,
	})

	targets := lspWarmupTargets(workdir, map[string]lsp.ServerConfig{
		"go":         {Command: "gopls"},
		"typescript": {Command: "vtsls"},
	})

	want := []string{"typescript"}
	if !slices.Equal(targets, want) {
		t.Fatalf("lspWarmupTargets() = %#v, want %#v", targets, want)
	}
}

func TestLSPWarmupTargets_SkipsUnavailableAndDisabledServers(t *testing.T) {
	workdir := withTempWorkdir(t)
	for name, content := range map[string]string{
		"main.go": "package main\n",
		"app.ts":  "export const value = 1;\n",
		"app.py":  "print('hi')\n",
	} {
		if err := os.WriteFile(filepath.Join(workdir, name), []byte(content), 0600); err != nil {
			t.Fatalf("WriteFile(%s) error = %v", name, err)
		}
	}

	withLSPCommandAvailability(t, map[string]bool{
		"pyright-langserver": true,
	})

	targets := lspWarmupTargets(workdir, map[string]lsp.ServerConfig{
		"go":         {Command: "gopls"},
		"python":     {Command: "pyright-langserver", Disabled: true},
		"typescript": {Command: ""},
	})

	if len(targets) != 0 {
		t.Fatalf("lspWarmupTargets() = %#v, want no targets", targets)
	}
}
