package lsp

import (
	"bytes"
	"context"
	"io"
	"os"
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

	if client.out() != io.Discard {
		t.Fatalf("expected default output to be io.Discard")
	}
	if client.errorOutput() != io.Discard {
		t.Fatalf("expected default error output to be io.Discard")
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

func TestServerDebugOutputUsesInjectedWriter(t *testing.T) {
	t.Setenv("XELYON_DEBUG_LSP", "1")

	server := NewServer("test")
	var buf bytes.Buffer
	server.SetDebugOutput(&buf)
	server.debugf("[LSP %s] hello\n", server.name)

	if got := buf.String(); got != "[LSP test] hello\n" {
		t.Fatalf("debug output = %q", got)
	}

	server.SetDebugOutput(nil)
	if server.debugOut != io.Discard {
		t.Fatalf("expected nil debug output to normalize to io.Discard")
	}

	t.Setenv("XELYON_DEBUG_LSP", "")
	server.debugf("hidden")
	if buf.String() != "[LSP test] hello\n" {
		t.Fatalf("debug output should not change when debug is disabled")
	}
}

func TestClient_GetDiagnostics_WithHelperServer(t *testing.T) {
	t.Setenv("GO_WANT_XELYON_LSP_HELPER", "1")

	tmpDir := t.TempDir()
	filePath := tmpDir + "/main.go"
	if err := os.WriteFile(filePath, []byte("package main\n\nfunc main() {}\n"), 0644); err != nil {
		t.Fatalf("os.WriteFile() error = %v", err)
	}

	cmd, args := lspHelperCommand(t)
	client := NewClient(tmpDir)
	client.SetConfigs(map[string]ServerConfig{
		"go": {Command: cmd, Args: args},
	})

	var out bytes.Buffer
	client.SetOutput(&out)

	diags, err := client.GetDiagnostics(context.Background(), filePath)
	if err != nil {
		t.Fatalf("GetDiagnostics() error = %v", err)
	}
	if len(diags) != 2 {
		t.Fatalf("len(diags) = %d, want 2", len(diags))
	}
	if diags[0].Message != "undefined: helper" {
		t.Fatalf("diags[0].Message = %q, want %q", diags[0].Message, "undefined: helper")
	}
	if diags[1].Severity != DiagnosticSeverityWarning {
		t.Fatalf("diags[1].Severity = %d, want %d", diags[1].Severity, DiagnosticSeverityWarning)
	}
	if got := out.String(); got == "" || !bytes.Contains([]byte(got), []byte("LSP server")) {
		t.Fatalf("client output = %q, want server started message", got)
	}
	client.Close()
}

func TestClient_PositionMethods_WithHelperServer(t *testing.T) {
	t.Setenv("GO_WANT_XELYON_LSP_HELPER", "1")

	tmpDir := t.TempDir()
	filePath := tmpDir + "/main.go"
	if err := os.WriteFile(filePath, []byte("package main\n\nfunc helper() {}\n"), 0644); err != nil {
		t.Fatalf("os.WriteFile() error = %v", err)
	}

	cmd, args := lspHelperCommand(t)
	client := NewClient(tmpDir)
	client.SetConfigs(map[string]ServerConfig{
		"go": {Command: cmd, Args: args},
	})

	ctx := context.Background()
	refs, err := client.FindReferences(ctx, filePath, 3, 6, true)
	if err != nil {
		t.Fatalf("FindReferences() error = %v", err)
	}
	if len(refs) != 2 {
		t.Fatalf("len(refs) = %d, want 2", len(refs))
	}
	if refs[0].Range.Start.Line != 9 {
		t.Fatalf("refs[0].Range.Start.Line = %d, want 9", refs[0].Range.Start.Line)
	}

	defs, err := client.GotoDefinition(ctx, filePath, 3, 6)
	if err != nil {
		t.Fatalf("GotoDefinition() error = %v", err)
	}
	if len(defs) != 1 || defs[0].Range.Start.Line != 0 {
		t.Fatalf("GotoDefinition() = %+v, want line 0", defs)
	}

	impls, err := client.GotoImplementation(ctx, filePath, 3, 6)
	if err != nil {
		t.Fatalf("GotoImplementation() error = %v", err)
	}
	if len(impls) != 1 || impls[0].Range.Start.Line != 20 {
		t.Fatalf("GotoImplementation() = %+v, want line 20", impls)
	}
	client.Close()
}
