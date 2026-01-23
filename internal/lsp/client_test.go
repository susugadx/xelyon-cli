package lsp

import (
	"testing"
)

func TestNewClient(t *testing.T) {
	client := NewClient("/home/user/project")
	if client == nil {
		t.Fatal("NewClient returned nil")
	}

	if client.servers == nil {
		t.Error("servers map is nil")
	}

	if client.configs == nil {
		t.Error("configs map is nil")
	}

	if client.rootURI != "file:///home/user/project" {
		t.Errorf("rootURI = %q, want %q", client.rootURI, "file:///home/user/project")
	}
}

func TestClientSetConfigs(t *testing.T) {
	client := NewClient("/tmp")
	configs := map[string]ServerConfig{
		"go": {
			Command: "gopls",
			Args:    []string{},
		},
		"python": {
			Command:  "pyright-langserver",
			Args:     []string{"--stdio"},
			Disabled: true,
		},
	}

	client.SetConfigs(configs)

	if len(client.configs) != 2 {
		t.Errorf("len(configs) = %d, want 2", len(client.configs))
	}

	if client.configs["go"].Command != "gopls" {
		t.Errorf("go command = %q, want gopls", client.configs["go"].Command)
	}

	if !client.configs["python"].Disabled {
		t.Error("python should be disabled")
	}
}

func TestClientStatus(t *testing.T) {
	client := NewClient("/tmp")
	configs := map[string]ServerConfig{
		"go": {
			Command:  "gopls",
			Disabled: false,
		},
		"python": {
			Command:  "pyright-langserver",
			Disabled: true,
		},
	}
	client.SetConfigs(configs)

	status := client.Status()

	if status["python"] != "disabled" {
		t.Errorf("python status = %q, want disabled", status["python"])
	}

	// go should be either "not started (lazy)" or "not installed (...)"
	goStatus := status["go"]
	if goStatus != "not started (lazy)" && goStatus[:13] != "not installed" {
		t.Errorf("go status = %q, want 'not started (lazy)' or 'not installed (...)'", goStatus)
	}
}

func TestClientClose(t *testing.T) {
	client := NewClient("/tmp")
	client.SetConfigs(map[string]ServerConfig{
		"go": {Command: "gopls"},
	})

	// Close should not panic even with no running servers
	client.Close()

	if len(client.servers) != 0 {
		t.Errorf("servers not cleared after Close")
	}
}
