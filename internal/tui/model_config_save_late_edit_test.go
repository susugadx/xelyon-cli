package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/susugadx/xelyon-cli/internal/config"
)

func TestConfigScreen_SaveKeepsDirtyWhenEditedAfterSaveStarts(t *testing.T) {
	agent := &stubAgent{}
	m := newModelWithViewport(agent)
	m.screen = screenConfig
	m.configScreen = newConfigScreen(config.DefaultConfig())

	cs := m.configScreen
	setConfigFieldSelection(t, cs, "compression", "compression.enabled")

	m = sendConfigKey(m, " ")
	cs = m.configScreen
	if !cs.dirty {
		t.Fatal("dirty should be true after initial edit")
	}

	updated, saveCmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("s")})
	m = updated.(Model)
	if saveCmd == nil {
		t.Fatal("saveCmd should not be nil")
	}

	m = sendConfigKey(m, " ")
	cs = m.configScreen
	if !cs.dirty {
		t.Fatal("dirty should remain true after late edit")
	}
	if cs.saveStatus != statusModified {
		t.Fatalf("saveStatus after late edit = %d, want statusModified(%d)", cs.saveStatus, statusModified)
	}

	resultMsg := saveCmd()
	updated, _ = m.Update(resultMsg)
	m = updated.(Model)
	cs = m.configScreen
	if !cs.dirty {
		t.Fatal("dirty should stay true when cfg changed after save started")
	}
	if cs.saveStatus != statusModified {
		t.Fatalf("saveStatus after save completion = %d, want statusModified(%d)", cs.saveStatus, statusModified)
	}
}

func TestConfigScreen_SaveAndQuitDoesNotCloseIfEditedAfterSaveStarts(t *testing.T) {
	agent := &stubAgent{}
	m := newModelWithViewport(agent)
	m.screen = screenConfig
	m.configScreen = newConfigScreen(config.DefaultConfig())

	cs := m.configScreen
	setConfigFieldSelection(t, cs, "compression", "compression.enabled")
	m = sendConfigKey(m, " ")

	m = sendConfigKey(m, "q")
	updated, saveCmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(Model)
	if saveCmd == nil {
		t.Fatal("saveCmd should not be nil")
	}

	cs = m.configScreen
	if !cs.pendingClose {
		t.Fatal("pendingClose should be true while save and quit is in flight")
	}
	if cs.saveStatus != statusSaving {
		t.Fatalf("saveStatus = %d, want statusSaving(%d)", cs.saveStatus, statusSaving)
	}

	m = sendConfigKey(m, " ")
	cs = m.configScreen
	if !cs.dirty {
		t.Fatal("dirty should stay true after late edit")
	}

	resultMsg := saveCmd()
	updated, _ = m.Update(resultMsg)
	m = updated.(Model)

	if m.screen != screenConfig {
		t.Fatalf("screen = %d, want screenConfig when late edits remain unsaved", m.screen)
	}
	cs = m.configScreen
	if cs.pendingClose {
		t.Fatal("pendingClose should be cleared after successful save of an outdated snapshot")
	}
	if !cs.dirty {
		t.Fatal("dirty should remain true after outdated snapshot save completes")
	}
	if cs.saveStatus != statusModified {
		t.Fatalf("saveStatus after save completion = %d, want statusModified(%d)", cs.saveStatus, statusModified)
	}
}
