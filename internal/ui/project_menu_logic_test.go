package ui

import (
	"testing"

	"github.com/susugadx/xelyon-cli/internal/config"
)

func TestResolveProjectMainMenuAction(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  projectMainMenuAction
	}{
		{name: "context", input: "1", want: projectMainMenuShowContext},
		{name: "rules", input: "2", want: projectMainMenuEditRules},
		{name: "final checks", input: "3", want: projectMainMenuEditFinalChecks},
		{name: "save alias", input: "save", want: projectMainMenuSave},
		{name: "cancel alias", input: "cancel", want: projectMainMenuCancel},
		{name: "unknown", input: "x", want: projectMainMenuUnknown},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := resolveProjectMainMenuAction(tt.input); got != tt.want {
				t.Fatalf("resolveProjectMainMenuAction(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestBuildProjectMainMenuSnapshot(t *testing.T) {
	snapshot := buildProjectMainMenuSnapshot(&config.ProjectConfig{
		Context: "  hello  ",
		Rules:   []string{"a", "b"},
		FinalChecks: &config.FinalChecksConfig{
			Commands: []string{"go test ./...", "go vet ./..."},
		},
	})

	if snapshot.contextPreview != "hello" {
		t.Fatalf("contextPreview = %q, want %q", snapshot.contextPreview, "hello")
	}
	if snapshot.rulesCount != 2 {
		t.Fatalf("rulesCount = %d, want 2", snapshot.rulesCount)
	}
	if snapshot.finalChecksPreview != "(2 cmds)" {
		t.Fatalf("finalChecksPreview = %q, want %q", snapshot.finalChecksPreview, "(2 cmds)")
	}
}

func TestResolveProjectFinalChecksAction(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  projectFinalChecksAction
	}{
		{name: "commands", input: "1", want: projectFinalChecksEditCommands},
		{name: "timeout", input: "2", want: projectFinalChecksEditTimeout},
		{name: "back", input: "b", want: projectFinalChecksBack},
		{name: "unknown", input: "x", want: projectFinalChecksUnknown},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := resolveProjectFinalChecksAction(tt.input); got != tt.want {
				t.Fatalf("resolveProjectFinalChecksAction(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestBuildProjectFinalChecksSnapshot(t *testing.T) {
	t.Run("with commands and timeout", func(t *testing.T) {
		snapshot := buildProjectFinalChecksSnapshot(&config.FinalChecksConfig{
			Commands: []string{"go test ./..."},
			Timeout:  120,
		})
		if snapshot.commandsInfo != "(1 cmds)" {
			t.Fatalf("commandsInfo = %q, want %q", snapshot.commandsInfo, "(1 cmds)")
		}
		if snapshot.timeout != 120 {
			t.Fatalf("timeout = %d, want 120", snapshot.timeout)
		}
	})

	t.Run("empty uses defaults", func(t *testing.T) {
		snapshot := buildProjectFinalChecksSnapshot(&config.FinalChecksConfig{})
		if snapshot.commandsInfo != "(empty)" {
			t.Fatalf("commandsInfo = %q, want %q", snapshot.commandsInfo, "(empty)")
		}
		if snapshot.timeout != 600 {
			t.Fatalf("timeout = %d, want 600", snapshot.timeout)
		}
	})
}

func TestParsePositiveIntInput(t *testing.T) {
	if got, ok := parsePositiveIntInput("120"); !ok || got != 120 {
		t.Fatalf("parsePositiveIntInput(valid) = (%d,%v), want (120,true)", got, ok)
	}
	if _, ok := parsePositiveIntInput("0"); ok {
		t.Fatal("parsePositiveIntInput(0) should be invalid")
	}
	if _, ok := parsePositiveIntInput("abc"); ok {
		t.Fatal("parsePositiveIntInput(abc) should be invalid")
	}
}

func TestClearEmptyFinalChecksConfig(t *testing.T) {
	menu := NewProjectMenuWithRuntime(&config.ProjectConfig{
		FinalChecks: &config.FinalChecksConfig{},
	}, nil)
	menu.clearEmptyFinalChecksConfig()
	if menu.PC.FinalChecks != nil {
		t.Fatalf("FinalChecks = %#v, want nil", menu.PC.FinalChecks)
	}
}
