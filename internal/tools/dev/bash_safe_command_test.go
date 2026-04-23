package dev

import (
	"testing"

	"github.com/susugadx/xelyon-cli/internal/config"
)

func TestIsSafeCommand_CustomSafeCommands(t *testing.T) {
	cfg := config.BashConfig{
		SafeCommands: []string{"npm run", "cargo build"},
	}

	if !IsSafeCommand("npm run build", cfg) {
		t.Error("IsSafeCommand() should recognize custom safe command 'npm run'")
	}

	if !IsSafeCommand("cargo build --release", cfg) {
		t.Error("IsSafeCommand() should recognize custom safe command 'cargo build'")
	}

	if IsSafeCommand("mystery_tool build", cfg) {
		t.Error("IsSafeCommand() should not recognize 'mystery_tool' as safe (not in list)")
	}
}

func TestIsSafeCommand_DefaultSafeCommands(t *testing.T) {
	cfg := config.BashConfig{}

	tests := []struct {
		name     string
		command  string
		expected bool
	}{
		{"ls (discovery)", "ls -la", false},
		{"cat (discovery not safe)", "cat file.txt", false},
		{"echo", "echo hello", true},
		{"git status", "git status", true},
		{"git log", "git log --oneline", true},
		{"git diff", "git diff HEAD", true},
		{"go version", "go version", true},
		{"go mod tidy", "go mod tidy", true},
		{"pwd", "pwd", true},
		{"head (discovery)", "head -10 file.txt", false},
		{"tail (discovery)", "tail -f file.txt", false},
		{"grep (discovery)", "grep pattern file.txt", false},
		{"sed -n (discovery)", "sed -n '1,5p' file.txt", false},
		{"diff", "diff file1.txt file2.txt", true},
		{"file", "file test.txt", true},
		{"du", "du -sh .", true},
		{"stat", "stat test.txt", true},
		{"md5sum", "md5sum file.txt", true},
		{"sha256sum", "sha256sum file.txt", true},
		{"unknown", "unknown_command", false},
		{"npm run", "npm run build", true},
		{"go test", "go test ./...", true},
		{"make", "make build", true},
		{"cat (discovery)", "cat foo.txt", false},
		{"cat prefix boundary negative", "catalog-destroyer", false},
		{"echo prefix boundary positive", "echo hello", true},
		{"echo prefix boundary negative", "echomalware", false},
		{"git status exact word boundary", "git status", true},
		{"git status boundary negative", "git statusx", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsSafeCommand(tt.command, cfg)
			if result != tt.expected {
				t.Errorf("IsSafeCommand(%q) = %v, want %v", tt.command, result, tt.expected)
			}
		})
	}
}

func TestIsSafeCommand_ChainedCommands(t *testing.T) {
	cfg := config.BashConfig{}

	tests := []struct {
		name     string
		command  string
		expected bool
	}{
		{"single safe", "git status", true},
		{"all safe &&", "git status && git log", true},
		{"ls ; discovery cat", "ls -la; cat file.txt", false},
		{"safe && unsafe", "git status && git push", false},
		{"safe && dangerous", "git status && rm -rf /tmp", false},
		{"discovery ls ; rm", "ls; rm file", false},
		{"pipe discovery grep", "grep foo | head", false},
		{"discovery cat && echo", "cat file && echo done", false},
		{"three safe", "git status && git diff && git log --oneline", true},
		{"discovery ls && unknown", "ls && unknown_cmd && echo done", false},
		{"safe || safe", "git status || git log", true},
		{"safe || unsafe", "git status || git push", false},
		{"mixed operators safe", "git status && git log || echo fail", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsSafeCommand(tt.command, cfg)
			if result != tt.expected {
				t.Errorf("IsSafeCommand(%q) = %v, want %v", tt.command, result, tt.expected)
			}
		})
	}
}

func TestSplitChainCommand(t *testing.T) {
	tests := []struct {
		command  string
		expected []string
	}{
		{"git status", []string{"git status"}},
		{"git status && git log", []string{"git status", "git log"}},
		{"ls; cat file", []string{"ls", "cat file"}},
		{"a || b && c", []string{"a", "b", "c"}},
		{"grep foo | head", []string{"grep foo | head"}},
		{"  ls  &&  pwd  ", []string{"ls", "pwd"}},
		{`echo "hello && world"`, []string{`echo "hello && world"`}},
		{"echo 'a;b' && ls", []string{"echo 'a;b'", "ls"}},
	}

	for _, tt := range tests {
		t.Run(tt.command, func(t *testing.T) {
			result := splitChainCommand(tt.command)
			if len(result) != len(tt.expected) {
				t.Fatalf("splitChainCommand(%q) = %v, want %v", tt.command, result, tt.expected)
			}
			for i, part := range result {
				if part != tt.expected[i] {
					t.Errorf("splitChainCommand(%q)[%d] = %q, want %q", tt.command, i, part, tt.expected[i])
				}
			}
		})
	}
}
