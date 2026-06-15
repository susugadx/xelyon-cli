package tui

import (
	"errors"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/susugadx/xelyon-cli/internal/config"
)

func TestProjectCommand_OpensProjectScreen(t *testing.T) {
	agent := &stubAgent{
		statusLine: "ready",
		projectConfig: &config.ProjectConfig{
			Context: "project context",
			Rules:   []string{"keep tests green"},
		},
	}
	m := newModelWithViewport(agent)

	m.textInput.SetValue("/project")
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(Model)

	if cmd != nil {
		t.Fatalf("/project should not start chat, got cmd %v", cmd)
	}
	if m.screen != screenProject {
		t.Fatalf("screen = %d, want screenProject(%d)", m.screen, screenProject)
	}
	if m.projectScreen == nil {
		t.Fatal("projectScreen is nil")
	}
	if m.projectScreen.missing {
		t.Fatal("projectScreen should not be missing")
	}
	if m.projectScreen.pc.Context != "project context" {
		t.Fatalf("project context = %q", m.projectScreen.pc.Context)
	}
}

func TestProjectCommand_LoadErrorStaysInChat(t *testing.T) {
	agent := &stubAgent{
		statusLine:     "ready",
		projectLoadErr: errors.New("invalid xelyon.yaml"),
	}
	m := newModelWithViewport(agent)

	m.textInput.SetValue("/project")
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(Model)

	if cmd != nil {
		t.Fatalf("/project load error should not start chat, got cmd %v", cmd)
	}
	if m.screen != screenChat {
		t.Fatalf("screen = %d, want screenChat", m.screen)
	}
	if m.projectScreen != nil {
		t.Fatal("projectScreen should be nil after load error")
	}
	if len(m.messages) == 0 || !strings.Contains(m.messages[len(m.messages)-1].Content, "invalid xelyon.yaml") {
		t.Fatalf("last message should contain load error: %#v", m.messages)
	}
}

func TestProjectScreen_MissingConfigCreatesTemplate(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := newProjectTestModel(agent)

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(Model)
	if cmd == nil {
		t.Fatal("template creation command should not be nil")
	}

	updated, _ = m.Update(cmd())
	m = updated.(Model)

	if m.projectScreen == nil {
		t.Fatal("projectScreen is nil")
	}
	if m.projectScreen.missing {
		t.Fatal("projectScreen should no longer be missing")
	}
	if m.projectScreen.pc.Context != "" {
		t.Fatalf("template context = %q, want empty", m.projectScreen.pc.Context)
	}
	if view := m.View(); !strings.Contains(view, "template created") {
		t.Fatalf("View() missing template-created status: %q", view)
	}
}

func TestProjectScreen_MissingConfigIgnoresDuplicateCreateWhileSaving(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := newProjectTestModel(agent)

	updated, firstCmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(Model)
	if firstCmd == nil {
		t.Fatal("first template creation command should not be nil")
	}
	if m.projectScreen.saveStatus != projectStatusSaving {
		t.Fatalf("saveStatus = %d, want projectStatusSaving(%d)", m.projectScreen.saveStatus, projectStatusSaving)
	}

	updated, secondCmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(Model)
	if secondCmd != nil {
		t.Fatalf("second template creation command while saving = %v, want nil", secondCmd)
	}

	updated, _ = m.Update(firstCmd())
	m = updated.(Model)
	if m.projectScreen == nil || m.projectScreen.missing {
		t.Fatal("projectScreen should show created config after first command completes")
	}
	if view := m.View(); strings.Contains(view, "already exists") {
		t.Fatalf("View() should not show duplicate create error:\n%s", view)
	}
}

func TestProjectScreen_StaleTemplateCreateDoesNotReplaceReopenedScreen(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := newProjectTestModel(agent)

	updated, createCmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(Model)
	if createCmd == nil {
		t.Fatal("template creation command should not be nil")
	}
	oldScreenID := m.projectScreen.screenID

	m = sendProjectKey(m, "esc")
	if m.screen != screenChat {
		t.Fatalf("screen after closing missing project = %d, want screenChat", m.screen)
	}

	agent.projectConfig = &config.ProjectConfig{Context: "existing context"}
	updated, _ = m.openProjectScreen()
	m = updated.(Model)
	if m.projectScreen == nil {
		t.Fatal("projectScreen is nil after reopening")
	}
	if m.projectScreen.screenID == oldScreenID {
		t.Fatal("reopened project screen should have a new screenID")
	}
	m = editProjectContext(t, m, "reopened draft")

	updated, _ = m.Update(createCmd())
	m = updated.(Model)

	if got := m.projectScreen.pc.Context; got != "reopened draft" {
		t.Fatalf("stale template result replaced reopened screen context = %q, want reopened draft", got)
	}
	if !m.projectScreen.dirty {
		t.Fatal("reopened draft should remain dirty after stale template result")
	}
}
