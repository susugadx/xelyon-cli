package tui

import (
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/config"
)

func TestProjectScreen_FinalChecksEdits(t *testing.T) {
	agent := &stubAgent{
		statusLine: "ready",
		projectConfig: &config.ProjectConfig{
			Context: "ctx",
		},
	}
	m := newProjectTestModel(agent)

	m = moveProjectToSection(t, m, "final_commands")
	m = sendProjectKey(m, "a")
	m = sendProjectText(m, "go test ./...")
	m = sendProjectKey(m, "enter")

	m = moveProjectToSection(t, m, "final_timeout")
	m = sendProjectKey(m, "enter")
	m = sendProjectText(m, "120")
	m = sendProjectKey(m, "enter")

	snapshot := projectSnapshot(t, m)
	if snapshot.Config.FinalChecks == nil {
		t.Fatal("FinalChecks is nil")
	}
	if got := snapshot.Config.FinalChecks.Commands; len(got) != 1 || got[0] != "go test ./..." {
		t.Fatalf("FinalChecks.Commands = %#v", got)
	}
	if got := snapshot.Config.FinalChecks.Timeout; got != 120 {
		t.Fatalf("FinalChecks.Timeout = %d, want 120", got)
	}
}

func TestProjectScreen_FinalChecksTimeoutRequiresCommands(t *testing.T) {
	agent := &stubAgent{
		statusLine: "ready",
		projectConfig: &config.ProjectConfig{
			Context: "ctx",
		},
	}
	m := newProjectTestModel(agent)

	m = moveProjectToSection(t, m, "final_timeout")
	m = sendProjectKey(m, "enter")

	snapshot := projectSnapshot(t, m)
	if snapshot.EditMode != "none" {
		t.Fatalf("editMode = %s, want none", snapshot.EditMode)
	}
	if snapshot.Config.FinalChecks != nil {
		t.Fatalf("FinalChecks = %#v, want nil", snapshot.Config.FinalChecks)
	}
	if snapshot.Dirty {
		t.Fatal("dirty should stay false when timeout edit is refused")
	}
	if !strings.Contains(snapshot.Message, "add a final check command") {
		t.Fatalf("message = %q, want add-command guidance", snapshot.Message)
	}
}

func TestProjectScreen_ExistingTimeoutOnlyFinalChecksOverrideIsPreserved(t *testing.T) {
	agent := &stubAgent{
		statusLine: "ready",
		projectConfig: &config.ProjectConfig{
			Context: "ctx",
			FinalChecks: &config.FinalChecksConfig{
				Timeout: 120,
			},
		},
	}
	m := newProjectTestModel(agent)

	if projectSnapshot(t, m).Config.FinalChecks == nil {
		t.Fatal("FinalChecks should be preserved after screen initialization")
	}
	if got := projectSnapshot(t, m).Config.FinalChecks.Timeout; got != 120 {
		t.Fatalf("FinalChecks.Timeout = %d, want 120", got)
	}

	m = editProjectContext(t, m, "updated ctx")
	saveProjectAndWait(t, m)

	if agent.lastSavedProject == nil {
		t.Fatal("lastSavedProject is nil")
	}
	if agent.lastSavedProject.FinalChecks == nil {
		t.Fatal("saved FinalChecks should preserve existing timeout-only override")
	}
	if got := agent.lastSavedProject.FinalChecks.Timeout; got != 120 {
		t.Fatalf("saved FinalChecks.Timeout = %d, want 120", got)
	}
}

func TestProjectScreen_ExistingEmptyFinalChecksOverrideIsPreserved(t *testing.T) {
	agent := &stubAgent{
		statusLine: "ready",
		projectConfig: &config.ProjectConfig{
			Context:     "ctx",
			FinalChecks: &config.FinalChecksConfig{},
		},
	}
	m := newProjectTestModel(agent)

	if projectSnapshot(t, m).Config.FinalChecks == nil {
		t.Fatal("FinalChecks should be preserved after screen initialization")
	}

	m = editProjectContext(t, m, "updated ctx")
	saveProjectAndWait(t, m)

	if agent.lastSavedProject == nil {
		t.Fatal("lastSavedProject is nil")
	}
	if agent.lastSavedProject.FinalChecks == nil {
		t.Fatal("saved FinalChecks should preserve existing empty override")
	}
	if len(agent.lastSavedProject.FinalChecks.Commands) != 0 {
		t.Fatalf("saved FinalChecks.Commands = %#v, want empty", agent.lastSavedProject.FinalChecks.Commands)
	}
}

func TestProjectScreen_DeleteLastExistingFinalCheckCommandPreservesOverride(t *testing.T) {
	agent := &stubAgent{
		statusLine: "ready",
		projectConfig: &config.ProjectConfig{
			Context: "ctx",
			FinalChecks: &config.FinalChecksConfig{
				Commands: []string{"go test ./..."},
				Timeout:  120,
			},
		},
	}
	m := newProjectTestModel(agent)

	m = moveProjectToSection(t, m, "final_commands")
	m = moveProjectToItemPane(t, m)
	m = sendProjectKey(m, "d")

	snapshot := projectSnapshot(t, m)
	if snapshot.Config.FinalChecks == nil {
		t.Fatal("FinalChecks should preserve existing override after deleting the last command")
	}
	if got := snapshot.Config.FinalChecks.Commands; len(got) != 0 {
		t.Fatalf("FinalChecks.Commands = %#v, want empty", got)
	}
	if got := snapshot.Config.FinalChecks.Timeout; got != 120 {
		t.Fatalf("FinalChecks.Timeout = %d, want 120", got)
	}
	if !snapshot.Dirty {
		t.Fatal("dirty should be true after deleting final check command")
	}
}

func TestProjectScreen_DeleteLastTUICreatedFinalCheckCommandClearsFinalChecks(t *testing.T) {
	agent := &stubAgent{
		statusLine: "ready",
		projectConfig: &config.ProjectConfig{
			Context: "ctx",
		},
	}
	m := newProjectTestModel(agent)

	m = moveProjectToSection(t, m, "final_commands")
	m = sendProjectKey(m, "a")
	m = sendProjectText(m, "go test ./...")
	m = sendProjectKey(m, "enter")
	if projectSnapshot(t, m).Config.FinalChecks == nil {
		t.Fatal("FinalChecks should be created after adding a command")
	}

	m = moveProjectToItemPane(t, m)
	m = sendProjectKey(m, "d")

	snapshot := projectSnapshot(t, m)
	if snapshot.Config.FinalChecks != nil {
		t.Fatalf("FinalChecks = %#v, want nil after deleting the last TUI-created command", snapshot.Config.FinalChecks)
	}
	if !snapshot.Dirty {
		t.Fatal("dirty should be true after deleting final check command")
	}
}
