package ui

import (
	"bytes"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/config"
)

func TestEditFinalChecks_SubmenuShowsCommandsAndTimeoutOnly(t *testing.T) {
	pc := &config.ProjectConfig{
		FinalChecks: &config.FinalChecksConfig{
			Commands: []string{"go test ./..."},
		},
	}
	runtime := NewRuntime(strings.NewReader("3\nb\nc\n"), &bytes.Buffer{}, &bytes.Buffer{})
	out := runtime.Output().(*bytes.Buffer)

	menu := NewProjectMenuWithRuntime(pc, runtime)
	_, _ = menu.Run()

	output := out.String()
	if !strings.Contains(output, "commands") {
		t.Fatalf("expected final checks submenu to show commands, got:\n%s", output)
	}
	if strings.Contains(output, "max_retry") {
		t.Fatalf("expected final checks submenu to hide max_retry, got:\n%s", output)
	}
}

func TestEditFinalChecks_CommandsEdit(t *testing.T) {
	pc := &config.ProjectConfig{
		FinalChecks: &config.FinalChecksConfig{},
	}
	input := "3\n1\na\ngo test ./...\ns\nb\ns\n"
	runtime := NewRuntime(strings.NewReader(input), &bytes.Buffer{}, &bytes.Buffer{})

	menu := NewProjectMenuWithRuntime(pc, runtime)
	changed, err := menu.Run()
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if !changed {
		t.Fatal("Run() changed = false, want true")
	}
	if pc.FinalChecks == nil {
		t.Fatal("FinalChecks should not be nil after editing commands")
	}
	if len(pc.FinalChecks.Commands) != 1 || pc.FinalChecks.Commands[0] != "go test ./..." {
		t.Fatalf("Commands = %v, want [go test ./...]", pc.FinalChecks.Commands)
	}
}

func TestEditFinalChecks_NilifyWhenAllEmpty(t *testing.T) {
	pc := &config.ProjectConfig{
		FinalChecks: &config.FinalChecksConfig{},
	}
	runtime := NewRuntime(strings.NewReader("3\nb\ns\n"), &bytes.Buffer{}, &bytes.Buffer{})

	menu := NewProjectMenuWithRuntime(pc, runtime)
	_, _ = menu.Run()

	if pc.FinalChecks != nil {
		t.Fatalf("FinalChecks should be nil when all fields are empty, got %+v", pc.FinalChecks)
	}
}

func TestEditFinalChecks_MainMenuPreviewCountsCommands(t *testing.T) {
	pc := &config.ProjectConfig{
		FinalChecks: &config.FinalChecksConfig{
			Commands: []string{"cmd1", "cmd2"},
		},
	}
	runtime := NewRuntime(strings.NewReader("c\n"), &bytes.Buffer{}, &bytes.Buffer{})
	out := runtime.Output().(*bytes.Buffer)

	menu := NewProjectMenuWithRuntime(pc, runtime)
	_, _ = menu.Run()

	output := out.String()
	if !strings.Contains(output, "(2 cmds)") {
		t.Fatalf("expected main menu to show (2 cmds), got:\n%s", output)
	}
}

func TestEditFinalChecks_TimeoutIsOption2(t *testing.T) {
	pc := &config.ProjectConfig{
		FinalChecks: &config.FinalChecksConfig{},
	}
	runtime := NewRuntime(strings.NewReader("3\n2\n120\nb\ns\n"), &bytes.Buffer{}, &bytes.Buffer{})

	menu := NewProjectMenuWithRuntime(pc, runtime)
	changed, _ := menu.Run()
	if !changed {
		t.Fatal("Run() changed = false, want true")
	}
	if pc.FinalChecks == nil || pc.FinalChecks.Timeout != 120 {
		t.Fatalf("Timeout = %v, want 120", pc.FinalChecks)
	}
}
