package config

import "testing"

func TestLSPNilServers_RoundTrip_PreservesSiblingFields(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	cfg := DefaultConfig()
	cfg.LSP.Enabled = false
	cfg.LSP.SkipInstallPrompt = true
	cfg.LSP.Servers = nil

	if err := SaveConfig(cfg); err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}

	loaded, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	if loaded.LSP.Enabled {
		t.Fatal("LSP.Enabled = true, want false")
	}
	if !loaded.LSP.SkipInstallPrompt {
		t.Fatal("LSP.SkipInstallPrompt = false, want true")
	}
	if loaded.LSP.Servers != nil {
		t.Fatalf("LSP.Servers = %#v, want nil", loaded.LSP.Servers)
	}
}

func TestLSPEmptyServers_RoundTrip_PreservesSiblingFields(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	cfg := DefaultConfig()
	cfg.LSP.Enabled = false
	cfg.LSP.SkipInstallPrompt = true
	cfg.LSP.Servers = map[string]LSPServerConfig{}

	if err := SaveConfig(cfg); err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}

	loaded, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	if loaded.LSP.Enabled {
		t.Fatal("LSP.Enabled = true, want false")
	}
	if !loaded.LSP.SkipInstallPrompt {
		t.Fatal("LSP.SkipInstallPrompt = false, want true")
	}
	if loaded.LSP.Servers == nil {
		t.Fatal("LSP.Servers = nil, want empty map")
	}
	if got := len(loaded.LSP.Servers); got != 0 {
		t.Fatalf("len(LSP.Servers) = %d, want 0", got)
	}
	if _, ok := loaded.LSP.Servers["go"]; ok {
		t.Fatal("default LSP servers should not be restored for explicit empty map")
	}
}

func TestLSPNonEmptyServers_RoundTrip_Unchanged(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	cfg := DefaultConfig()
	cfg.LSP.Servers = map[string]LSPServerConfig{
		"go": {
			Command:  "custom-gopls",
			Args:     []string{"serve", "--stdio"},
			Disabled: true,
		},
	}

	if err := SaveConfig(cfg); err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}

	loaded, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	if got := len(loaded.LSP.Servers); got != 1 {
		t.Fatalf("len(LSP.Servers) = %d, want 1", got)
	}

	server, ok := loaded.LSP.Servers["go"]
	if !ok {
		t.Fatal("LSP.Servers[\"go\"] not found")
	}
	if server.Command != "custom-gopls" {
		t.Fatalf("LSP.Servers[\"go\"].Command = %q, want %q", server.Command, "custom-gopls")
	}
	if len(server.Args) != 2 || server.Args[0] != "serve" || server.Args[1] != "--stdio" {
		t.Fatalf("LSP.Servers[\"go\"].Args = %#v, want %#v", server.Args, []string{"serve", "--stdio"})
	}
	if !server.Disabled {
		t.Fatal("LSP.Servers[\"go\"].Disabled = false, want true")
	}
}
