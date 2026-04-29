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

	m.projectScreen.sectionIndex = int(projectSectionFinalCommands)
	m = sendProjectKey(m, "a")
	m.projectScreen.editInput.SetValue("go test ./...")
	m = sendProjectKey(m, "enter")

	m.projectScreen.sectionIndex = int(projectSectionFinalTimeout)
	m = sendProjectKey(m, "enter")
	m.projectScreen.editInput.SetValue("120")
	m = sendProjectKey(m, "enter")

	if m.projectScreen.pc.FinalChecks == nil {
		t.Fatal("FinalChecks is nil")
	}
	if got := m.projectScreen.pc.FinalChecks.Commands; len(got) != 1 || got[0] != "go test ./..." {
		t.Fatalf("FinalChecks.Commands = %#v", got)
	}
	if got := m.projectScreen.pc.FinalChecks.Timeout; got != 120 {
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

	m.projectScreen.sectionIndex = int(projectSectionFinalTimeout)
	m = sendProjectKey(m, "enter")

	if m.projectScreen.editMode != projectEditNone {
		t.Fatalf("editMode = %d, want projectEditNone(%d)", m.projectScreen.editMode, projectEditNone)
	}
	if m.projectScreen.pc.FinalChecks != nil {
		t.Fatalf("FinalChecks = %#v, want nil", m.projectScreen.pc.FinalChecks)
	}
	if m.projectScreen.dirty {
		t.Fatal("dirty should stay false when timeout edit is refused")
	}
	if !strings.Contains(m.projectScreen.message, "add a final check command") {
		t.Fatalf("message = %q, want add-command guidance", m.projectScreen.message)
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

	if m.projectScreen.pc.FinalChecks == nil {
		t.Fatal("FinalChecks should be preserved after screen initialization")
	}
	if got := m.projectScreen.pc.FinalChecks.Timeout; got != 120 {
		t.Fatalf("FinalChecks.Timeout = %d, want 120", got)
	}

	m = editProjectContext(t, m, "updated ctx")
	m = saveProjectAndWait(t, m)

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

	if m.projectScreen.pc.FinalChecks == nil {
		t.Fatal("FinalChecks should be preserved after screen initialization")
	}

	m = editProjectContext(t, m, "updated ctx")
	m = saveProjectAndWait(t, m)

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

	m.projectScreen.sectionIndex = int(projectSectionFinalCommands)
	m.projectScreen.activePane = projectPaneItem
	m = sendProjectKey(m, "d")

	if m.projectScreen.pc.FinalChecks == nil {
		t.Fatal("FinalChecks should preserve existing override after deleting the last command")
	}
	if got := m.projectScreen.pc.FinalChecks.Commands; len(got) != 0 {
		t.Fatalf("FinalChecks.Commands = %#v, want empty", got)
	}
	if got := m.projectScreen.pc.FinalChecks.Timeout; got != 120 {
		t.Fatalf("FinalChecks.Timeout = %d, want 120", got)
	}
	if !m.projectScreen.dirty {
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

	m.projectScreen.sectionIndex = int(projectSectionFinalCommands)
	m = sendProjectKey(m, "a")
	m.projectScreen.editInput.SetValue("go test ./...")
	m = sendProjectKey(m, "enter")
	if m.projectScreen.pc.FinalChecks == nil {
		t.Fatal("FinalChecks should be created after adding a command")
	}

	m.projectScreen.activePane = projectPaneItem
	m = sendProjectKey(m, "d")

	if m.projectScreen.pc.FinalChecks != nil {
		t.Fatalf("FinalChecks = %#v, want nil after deleting the last TUI-created command", m.projectScreen.pc.FinalChecks)
	}
	if !m.projectScreen.dirty {
		t.Fatal("dirty should be true after deleting final check command")
	}
}
