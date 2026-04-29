package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/susugadx/xelyon-cli/internal/config"
)

func TestProjectScreen_ContextEditAndSave(t *testing.T) {
	agent := &stubAgent{
		statusLine: "ready",
		projectConfig: &config.ProjectConfig{
			Context: "old context",
			Rules:   []string{"old rule"},
		},
	}
	m := newProjectTestModel(agent)

	m = sendProjectKey(m, "enter")
	if m.projectScreen.editMode != projectEditContext {
		t.Fatalf("editMode = %d, want context edit", m.projectScreen.editMode)
	}
	m.projectScreen.contextArea.SetValue("new context\nsecond line")
	m = sendProjectCtrlS(m)

	if !m.projectScreen.dirty {
		t.Fatal("project screen should be dirty after context edit")
	}
	if got := m.projectScreen.pc.Context; got != "new context\nsecond line" {
		t.Fatalf("Context = %q", got)
	}

	m = saveProjectAndWait(t, m)
	if agent.lastSavedProject == nil {
		t.Fatal("lastSavedProject is nil")
	}
	if got := agent.lastSavedProject.Context; got != "new context\nsecond line" {
		t.Fatalf("saved Context = %q", got)
	}
	if m.projectScreen.dirty {
		t.Fatal("dirty should be false after save")
	}
}

func TestProjectScreen_SaveKeepsDirtyWhenEditedAfterSaveStarts(t *testing.T) {
	agent := &stubAgent{
		statusLine: "ready",
		projectConfig: &config.ProjectConfig{
			Context: "old context",
			Rules:   []string{"old rule"},
		},
	}
	m := newProjectTestModel(agent)

	m = editProjectContext(t, m, "saved snapshot")
	updated, saveCmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("s")})
	m = updated.(Model)
	if saveCmd == nil {
		t.Fatal("saveCmd should not be nil")
	}
	if m.projectScreen.saveStatus != projectStatusSaving {
		t.Fatalf("saveStatus = %d, want projectStatusSaving(%d)", m.projectScreen.saveStatus, projectStatusSaving)
	}

	m = editProjectContext(t, m, "late edit")
	updated, _ = m.Update(saveCmd())
	m = updated.(Model)

	if m.screen != screenProject {
		t.Fatalf("screen = %d, want screenProject", m.screen)
	}
	if got := m.projectScreen.pc.Context; got != "late edit" {
		t.Fatalf("Context after stale save = %q, want late edit", got)
	}
	if !m.projectScreen.dirty {
		t.Fatal("dirty should stay true when project config changed after save started")
	}
	if m.projectScreen.saveStatus != projectStatusModified {
		t.Fatalf("saveStatus after stale save = %d, want projectStatusModified(%d)", m.projectScreen.saveStatus, projectStatusModified)
	}
	if got := agent.lastSavedProject.Context; got != "saved snapshot" {
		t.Fatalf("saved Context = %q, want saved snapshot", got)
	}
}

func TestProjectScreen_SaveWhileInFlightQueuesLatestSnapshot(t *testing.T) {
	agent := &stubAgent{
		statusLine: "ready",
		projectConfig: &config.ProjectConfig{
			Context: "old context",
			Rules:   []string{"old rule"},
		},
	}
	m := newProjectTestModel(agent)

	m = editProjectContext(t, m, "first snapshot")
	updated, firstSaveCmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("s")})
	m = updated.(Model)
	if firstSaveCmd == nil {
		t.Fatal("firstSaveCmd should not be nil")
	}

	m = editProjectContext(t, m, "second snapshot")
	updated, immediateSecondSaveCmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("s")})
	m = updated.(Model)
	if immediateSecondSaveCmd != nil {
		t.Fatalf("second save while in flight returned cmd %v, want queued save", immediateSecondSaveCmd)
	}
	if !m.projectScreen.saveQueued {
		t.Fatal("saveQueued should be true after saving during in-flight save")
	}

	updated, queuedSaveCmd := m.Update(firstSaveCmd())
	m = updated.(Model)
	if queuedSaveCmd == nil {
		t.Fatal("queuedSaveCmd should start after first save completes")
	}
	if got := len(agent.savedProjects); got != 1 {
		t.Fatalf("saved project count after first save = %d, want 1", got)
	}
	if got := agent.savedProjects[0].Context; got != "first snapshot" {
		t.Fatalf("first saved Context = %q, want first snapshot", got)
	}

	updated, _ = m.Update(queuedSaveCmd())
	m = updated.(Model)
	if got := len(agent.savedProjects); got != 2 {
		t.Fatalf("saved project count after queued save = %d, want 2", got)
	}
	if got := agent.savedProjects[1].Context; got != "second snapshot" {
		t.Fatalf("queued saved Context = %q, want second snapshot", got)
	}
	if m.projectScreen.dirty {
		t.Fatal("dirty should be false after queued save")
	}
	if m.projectScreen.saveInFlight {
		t.Fatal("saveInFlight should be false after queued save")
	}
}

func TestProjectScreen_SaveAndQuitDoesNotCloseIfEditedAfterSaveStarts(t *testing.T) {
	agent := &stubAgent{
		statusLine: "ready",
		projectConfig: &config.ProjectConfig{
			Context: "old context",
			Rules:   []string{"old rule"},
		},
	}
	m := newProjectTestModel(agent)

	m = editProjectContext(t, m, "saved snapshot")
	m = sendProjectKey(m, "q")
	if !m.projectScreen.confirmQuit {
		t.Fatal("confirmQuit should be true before save and quit")
	}
	updated, saveCmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(Model)
	if saveCmd == nil {
		t.Fatal("saveCmd should not be nil")
	}
	if !m.projectScreen.pendingClose {
		t.Fatal("pendingClose should be true while save and quit is in flight")
	}

	m = editProjectContext(t, m, "late edit")
	updated, _ = m.Update(saveCmd())
	m = updated.(Model)

	if m.screen != screenProject {
		t.Fatalf("screen = %d, want screenProject when late edits remain unsaved", m.screen)
	}
	if m.projectScreen.pendingClose {
		t.Fatal("pendingClose should be cleared after stale save result")
	}
	if got := m.projectScreen.pc.Context; got != "late edit" {
		t.Fatalf("Context after stale save-and-quit = %q, want late edit", got)
	}
	if !m.projectScreen.dirty {
		t.Fatal("dirty should remain true after stale save-and-quit")
	}
	if m.projectScreen.saveStatus != projectStatusModified {
		t.Fatalf("saveStatus after stale save-and-quit = %d, want projectStatusModified(%d)", m.projectScreen.saveStatus, projectStatusModified)
	}
}

func TestProjectScreen_DiscardQuitDisabledWhileSaveInFlight(t *testing.T) {
	agent := &stubAgent{
		statusLine: "ready",
		projectConfig: &config.ProjectConfig{
			Context: "old context",
			Rules:   []string{"old rule"},
		},
	}
	m := newProjectTestModel(agent)

	m = editProjectContext(t, m, "saved snapshot")
	updated, saveCmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("s")})
	m = updated.(Model)
	if saveCmd == nil {
		t.Fatal("saveCmd should not be nil")
	}

	m = editProjectContext(t, m, "late edit")
	m = sendProjectKey(m, "q")
	m = sendProjectKey(m, "down")
	m = sendProjectKey(m, "enter")

	if m.screen != screenProject {
		t.Fatalf("screen = %d, want screenProject while save is in flight", m.screen)
	}
	if !m.projectScreen.confirmQuit {
		t.Fatal("confirmQuit should remain true while discard is blocked")
	}

	updated, _ = m.Update(saveCmd())
	m = updated.(Model)
	if m.screen != screenProject {
		t.Fatalf("screen after save completion = %d, want screenProject", m.screen)
	}
}

func TestProjectScreen_StaleSaveDoesNotCancelNewerSaveAndQuit(t *testing.T) {
	agent := &stubAgent{
		statusLine: "ready",
		projectConfig: &config.ProjectConfig{
			Context: "old context",
			Rules:   []string{"old rule"},
		},
	}
	m := newProjectTestModel(agent)

	m = editProjectContext(t, m, "first snapshot")
	updated, firstSaveCmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("s")})
	m = updated.(Model)
	if firstSaveCmd == nil {
		t.Fatal("firstSaveCmd should not be nil")
	}

	m = editProjectContext(t, m, "second snapshot")
	m = sendProjectKey(m, "q")
	updated, queuedImmediately := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(Model)
	if queuedImmediately != nil {
		t.Fatalf("save-and-quit during in-flight save returned cmd %v, want nil queued save", queuedImmediately)
	}
	if !m.projectScreen.pendingClose {
		t.Fatal("pendingClose should be true for newer save-and-quit")
	}
	if !m.projectScreen.saveQueued {
		t.Fatal("saveQueued should be true for newer save-and-quit")
	}

	updated, queuedSaveAndQuitCmd := m.Update(firstSaveCmd())
	m = updated.(Model)
	if m.screen != screenProject {
		t.Fatalf("screen after first save = %d, want screenProject", m.screen)
	}
	if queuedSaveAndQuitCmd == nil {
		t.Fatal("queued save-and-quit command should start after first save completes")
	}
	if m.projectScreen.saveStatus != projectStatusSaving {
		t.Fatalf("saveStatus after starting queued save = %d, want projectStatusSaving(%d)", m.projectScreen.saveStatus, projectStatusSaving)
	}

	updated, _ = m.Update(queuedSaveAndQuitCmd())
	m = updated.(Model)
	if m.screen != screenChat {
		t.Fatalf("screen after newer save-and-quit = %d, want screenChat", m.screen)
	}
	if m.projectScreen != nil {
		t.Fatal("projectScreen should be nil after newer save-and-quit completes")
	}
	if got := len(agent.savedProjects); got != 2 {
		t.Fatalf("saved project count = %d, want 2", got)
	}
	if got := agent.savedProjects[0].Context; got != "first snapshot" {
		t.Fatalf("first saved Context = %q, want first snapshot", got)
	}
	if got := agent.savedProjects[1].Context; got != "second snapshot" {
		t.Fatalf("second saved Context = %q, want second snapshot", got)
	}
}

func TestProjectScreen_SaveAndQuitCanBeCanceledWhileSaveInFlight(t *testing.T) {
	agent := &stubAgent{
		statusLine: "ready",
		projectConfig: &config.ProjectConfig{
			Context: "old context",
			Rules:   []string{"old rule"},
		},
	}
	m := newProjectTestModel(agent)

	m = editProjectContext(t, m, "saved snapshot")
	m = sendProjectKey(m, "q")
	updated, saveCmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(Model)
	if saveCmd == nil {
		t.Fatal("saveCmd should not be nil")
	}
	if !m.projectScreen.pendingClose {
		t.Fatal("pendingClose should be true after save-and-quit")
	}

	m = sendProjectKey(m, "q")
	if !m.projectScreen.confirmQuit {
		t.Fatal("confirmQuit should be true before canceling pending close")
	}
	m = sendProjectKey(m, "esc")
	if m.projectScreen.pendingClose {
		t.Fatal("pendingClose should be false after canceling close intent")
	}

	updated, _ = m.Update(saveCmd())
	m = updated.(Model)
	if m.screen != screenProject {
		t.Fatalf("screen after canceled save-and-quit = %d, want screenProject", m.screen)
	}
	if m.projectScreen.dirty {
		t.Fatal("dirty should be false after save completes")
	}
}

func TestProjectScreen_SaveAndQuitDoesNotCloseOverActiveEditDraft(t *testing.T) {
	agent := &stubAgent{
		statusLine: "ready",
		projectConfig: &config.ProjectConfig{
			Context: "old context",
		},
	}
	m := newProjectTestModel(agent)

	m = editProjectContext(t, m, "saved snapshot")
	m = sendProjectKey(m, "q")
	updated, saveCmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(Model)
	if saveCmd == nil {
		t.Fatal("saveCmd should not be nil")
	}

	m = sendProjectKey(m, "enter")
	if m.projectScreen.editMode != projectEditContext {
		t.Fatalf("editMode = %d, want projectEditContext(%d)", m.projectScreen.editMode, projectEditContext)
	}
	m.projectScreen.contextArea.SetValue("draft after save")

	updated, _ = m.Update(saveCmd())
	m = updated.(Model)
	if m.screen != screenProject {
		t.Fatalf("screen after save with active draft = %d, want screenProject", m.screen)
	}
	if m.projectScreen.editMode != projectEditContext {
		t.Fatalf("editMode after save = %d, want projectEditContext(%d)", m.projectScreen.editMode, projectEditContext)
	}
	if m.projectScreen.pendingClose {
		t.Fatal("pendingClose should be false after starting an edit")
	}

	m = sendProjectCtrlS(m)
	if got := m.projectScreen.pc.Context; got != "draft after save" {
		t.Fatalf("Context after confirming draft = %q, want draft after save", got)
	}
	if !m.projectScreen.dirty {
		t.Fatal("confirming the draft after save should mark the screen dirty")
	}
}
