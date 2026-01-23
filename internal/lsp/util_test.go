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
		{
			name:    "path with spaces",
			path:    "/home/user/my file.go",
			wantPfx: "file:///",
		},
		{
			name:    "nested directory path",
			path:    "/home/user/project/src/main.go",
			wantPfx: "file:///",
		},
		{
			name:    "empty path",
			path:    "",
			wantPfx: "file:///",
		},
		{
			name:    "dot path",
			path:    ".",
			wantPfx: "file:///",
		},
		{
			name:    "double dot path",
			path:    "..",
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
		{
			name: "Windows-style path in URI",
			uri:  "file:///C:/Users/test/file.go",
			want: "C:/Users/test/file.go",
		},
		{
			name: "file URI with encoded spaces",
			uri:  "file:///home/user/my%20file.go",
			want: "/home/user/my file.go",
		},
		{
			name: "file URI with encoded special chars",
			uri:  "file:///home/user/%E6%97%A5%E6%9C%AC%E8%AA%9E.go",
			want: "/home/user/日本語.go",
		},
		{
			name: "file URI with query params",
			uri:  "file:///home/user/file.go?query=1",
			want: "/home/user/file.go",
		},
		{
			name: "file URI with fragment",
			uri:  "file:///home/user/file.go#line=10",
			want: "/home/user/file.go",
		},
		{
			name: "file URI root path",
			uri:  "file:///",
			want: "/",
		},
		{
			name: "file URI with double slashes",
			uri:  "file:///home//user//file.go",
			want: "/home//user//file.go",
		},
		{
			name: "Windows drive D",
			uri:  "file:///D:/Projects/code.rs",
			want: "D:/Projects/code.rs",
		},
		{
			name: "empty string passthrough",
			uri:  "",
			want: "",
		},
		{
			name: "ftp URI passthrough",
			uri:  "ftp://server/file",
			want: "ftp://server/file",
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
		// Existing (4 languages)
		{"main.go", "go"},
		{"app.ts", "typescript"},
		{"App.tsx", "typescriptreact"},
		{"index.js", "javascript"},
		{"Component.jsx", "javascriptreact"},
		{"script.py", "python"},
		{"lib.rs", "rust"},

		// Tier 1: Backend languages (11 languages)
		{"Main.java", "java"},
		{"module.c", "c"},
		{"header.h", "c"},
		{"class.cpp", "cpp"},
		{"template.hpp", "cpp"},
		{"source.cc", "cpp"},
		{"impl.cxx", "cpp"},
		{"header.hxx", "cpp"},
		{"Program.cs", "csharp"},
		{"app.rb", "ruby"},
		{"index.php", "php"},
		{"Main.kt", "kotlin"},
		{"script.kts", "kotlin"},
		{"App.swift", "swift"},
		{"Main.scala", "scala"},
		{"module.ex", "elixir"},
		{"script.exs", "elixir"},
		{"game.lua", "lua"},

		// Tier 2: Frontend languages (4 languages)
		{"style.css", "css"},
		{"style.scss", "css"},
		{"style.sass", "css"},
		{"style.less", "css"},
		{"index.html", "html"},
		{"page.htm", "html"},
		{"App.vue", "vue"},
		{"Component.svelte", "svelte"},

		// Tier 3: Config/Script languages (5 languages)
		{"config.yaml", "yaml"},
		{"config.yml", "yaml"},
		{"Cargo.toml", "toml"},
		{"query.sql", "sql"},
		{"script.sh", "bash"},
		{"script.bash", "bash"},
		{"script.zsh", "bash"},
		{"README.md", "markdown"},
		{"doc.markdown", "markdown"},

		// Unknown extensions
		{"unknown.xyz", ""},
		{"no_extension", ""},

		// Case insensitivity
		{"FILE.GO", "go"},
		{"File.PY", "python"},
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
		// TypeScript/JavaScript group - all map to "typescript"
		{"typescript", "typescript"},
		{"typescriptreact", "typescript"},
		{"javascript", "typescript"},
		{"javascriptreact", "typescript"},

		// C and C++ - return as-is (separate configs)
		{"c", "c"},
		{"cpp", "cpp"},

		// Other languages - return as-is
		{"go", "go"},
		{"python", "python"},
		{"rust", "rust"},
		{"java", "java"},
		{"ruby", "ruby"},
		{"php", "php"},
		{"kotlin", "kotlin"},
		{"swift", "swift"},
		{"scala", "scala"},
		{"elixir", "elixir"},
		{"lua", "lua"},
		{"css", "css"},
		{"html", "html"},
		{"vue", "vue"},
		{"svelte", "svelte"},
		{"yaml", "yaml"},
		{"toml", "toml"},
		{"sql", "sql"},
		{"bash", "bash"},
		{"markdown", "markdown"},
		{"csharp", "csharp"},

		// Unknown/empty - return as-is
		{"unknown", "unknown"},
		{"", ""},
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
