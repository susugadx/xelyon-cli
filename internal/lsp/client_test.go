package lsp

import (
	"testing"
)

func TestFileToURI(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		wantPfx string
	}{
		{
			name:    "absolute path",
			path:    "/home/user/file.go",
			wantPfx: "file:///",
		},
		{
			name:    "relative path converts to absolute",
			path:    "file.go",
			wantPfx: "file:///",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FileToURI(tt.path)
			if len(got) < len(tt.wantPfx) || got[:len(tt.wantPfx)] != tt.wantPfx {
				t.Errorf("FileToURI(%q) = %q, want prefix %q", tt.path, got, tt.wantPfx)
			}
		})
	}
}

func TestURIToFile(t *testing.T) {
	tests := []struct {
		name string
		uri  string
		want string
	}{
		{
			name: "simple file URI",
			uri:  "file:///home/user/file.go",
			want: "/home/user/file.go",
		},
		{
			name: "non-file URI passthrough",
			uri:  "https://example.com",
			want: "https://example.com",
		},
		{
			name: "file URI without scheme",
			uri:  "/home/user/file.go",
			want: "/home/user/file.go",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := URIToFile(tt.uri)
			if got != tt.want {
				t.Errorf("URIToFile(%q) = %q, want %q", tt.uri, got, tt.want)
			}
		})
	}
}

func TestDetectLanguage(t *testing.T) {
	tests := []struct {
		path string
		want string
	}{
		{"main.go", "go"},
		{"app.ts", "typescript"},
		{"App.tsx", "typescriptreact"},
		{"index.js", "javascript"},
		{"Component.jsx", "javascriptreact"},
		{"script.py", "python"},
		{"lib.rs", "rust"},
		{"Main.java", "java"},
		{"module.c", "c"},
		{"header.h", "c"},
		{"class.cpp", "cpp"},
		{"Program.cs", "csharp"},
		{"app.rb", "ruby"},
		{"index.php", "php"},
		{"Main.kt", "kotlin"},
		{"App.swift", "swift"},
		{"Main.scala", "scala"},
		{"unknown.xyz", ""},
		{"no_extension", ""},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			got := DetectLanguage(tt.path)
			if got != tt.want {
				t.Errorf("DetectLanguage(%q) = %q, want %q", tt.path, got, tt.want)
			}
		})
	}
}

func TestLanguageServerKey(t *testing.T) {
	tests := []struct {
		language string
		want     string
	}{
		{"go", "go"},
		{"python", "python"},
		{"rust", "rust"},
		{"typescript", "typescript"},
		{"typescriptreact", "typescript"},
		{"javascript", "typescript"},
		{"javascriptreact", "typescript"},
	}

	for _, tt := range tests {
		t.Run(tt.language, func(t *testing.T) {
			got := LanguageServerKey(tt.language)
			if got != tt.want {
				t.Errorf("LanguageServerKey(%q) = %q, want %q", tt.language, got, tt.want)
			}
		})
	}
}

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
