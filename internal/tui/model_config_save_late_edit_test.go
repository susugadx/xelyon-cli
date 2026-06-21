package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/tui/configscreen"
)

func TestConfigScreen_SaveKeepsDirtyWhenEditedAfterSaveStarts(t *testing.T) {
	agent := &stubAgent{}
	m := newModelWithViewport(agent)
	m.screen = screenConfig
	m.configScreen = configscreen.New(config.DefaultConfig())

	selectConfigField(t, &m, "compression", "compression.enabled")

	m = sendConfigKey(m, " ")
	cs := configTestScreen(t, m)
	if !cs.Snapshot().Dirty {
		t.Fatal("dirty should be true after initial edit")
	}

	updated, saveCmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("s")})
	m = updated.(Model)
	if saveCmd == nil {
		t.Fatal("saveCmd should not be nil")
	}

	m = sendConfigKey(m, " ")
	cs = configTestScreen(t, m)
	snapshot := cs.Snapshot()
	if !snapshot.Dirty {
		t.Fatal("dirty should remain true after late edit")
	}
	if snapshot.SaveStatus != statusModified {
		t.Fatalf("saveStatus after late edit = %d, want statusModified(%d)", snapshot.SaveStatus, statusModified)
	}

	resultMsg := saveCmd()
	updated, _ = m.Update(resultMsg)
	m = updated.(Model)
	cs = configTestScreen(t, m)
	snapshot = cs.Snapshot()
	if !snapshot.Dirty {
		t.Fatal("dirty should stay true when cfg changed after save started")
	}
	if snapshot.SaveStatus != statusModified {
		t.Fatalf("saveStatus after save completion = %d, want statusModified(%d)", snapshot.SaveStatus, statusModified)
	}
}

func TestConfigScreen_SaveAndQuitDoesNotCloseIfEditedAfterSaveStarts(t *testing.T) {
	agent := &stubAgent{}
	m := newModelWithViewport(agent)
	m.screen = screenConfig
	m.configScreen = configscreen.New(config.DefaultConfig())

	selectConfigField(t, &m, "compression", "compression.enabled")
	m = sendConfigKey(m, " ")

	m = sendConfigKey(m, "q")
	updated, saveCmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(Model)
	if saveCmd == nil {
		t.Fatal("saveCmd should not be nil")
	}

	cs := configTestScreen(t, m)
	snapshot := cs.Snapshot()
	if !snapshot.PendingClose {
		t.Fatal("pendingClose should be true while save and quit is in flight")
	}
	if snapshot.SaveStatus != statusSaving {
		t.Fatalf("saveStatus = %d, want statusSaving(%d)", snapshot.SaveStatus, statusSaving)
	}

	m = sendConfigKey(m, " ")
	cs = configTestScreen(t, m)
	if !cs.Snapshot().Dirty {
		t.Fatal("dirty should stay true after late edit")
	}

	resultMsg := saveCmd()
	updated, _ = m.Update(resultMsg)
	m = updated.(Model)

	if m.screen != screenConfig {
		t.Fatalf("screen = %d, want screenConfig when late edits remain unsaved", m.screen)
	}
	cs = configTestScreen(t, m)
	snapshot = cs.Snapshot()
	if snapshot.PendingClose {
		t.Fatal("pendingClose should be cleared after successful save of an outdated snapshot")
	}
	if !snapshot.Dirty {
		t.Fatal("dirty should remain true after outdated snapshot save completes")
	}
	if snapshot.SaveStatus != statusModified {
		t.Fatalf("saveStatus after save completion = %d, want statusModified(%d)", snapshot.SaveStatus, statusModified)
	}
}
