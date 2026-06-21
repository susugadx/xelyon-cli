package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/tui/configscreen"
)

func TestConfigScreen_DiscardAndQuit_DoesNotApplyInflightSave(t *testing.T) {
	agent := &stubAgent{}
	agent.saveStatusLine = "new-status-from-save"
	m := newModelWithViewport(agent)
	cfg := config.DefaultConfig()
	m.screen = screenConfig
	m.configScreen = configscreen.New(cfg)
	setConfigModifiedForTest(t, &m)

	sMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("s")}
	updated, saveCmd := m.Update(sMsg)
	m = updated.(Model)
	cs := configTestScreen(t, m)

	if got := cs.Snapshot().SaveStatus; got != statusSaving {
		t.Fatalf("saveStatus = %d, want statusSaving(%d)", got, statusSaving)
	}
	if saveCmd == nil {
		t.Fatal("saveCmd should not be nil")
	}

	m = sendConfigKey(m, "q")
	cs = m.configScreen
	if !cs.Snapshot().ConfirmQuit {
		t.Fatal("confirmQuit should be true")
	}

	setConfigConfirmIndexForTest(t, &m, 1)
	m = sendConfigKey(m, "enter")
	cs = configTestScreen(t, m)

	if m.screen != screenConfig {
		t.Fatal("discard should be blocked while save is in-flight; screen should remain screenConfig")
	}
	if cs == nil {
		t.Fatal("configScreen should not be nil")
	}
	if !cs.Snapshot().ConfirmQuit {
		t.Fatal("confirmQuit should remain true since discard was blocked")
	}

	resultMsg := saveCmd()
	updated, _ = m.Update(resultMsg)
	m = updated.(Model)

	cs = configTestScreen(t, m)
	if cs == nil {
		t.Fatal("configScreen should not be nil after save completes")
	}
	if got := cs.Snapshot().SaveStatus; got != statusSaved {
		t.Fatalf("saveStatus = %d, want statusSaved(%d)", got, statusSaved)
	}
}

func TestConfigScreen_SaveAndQuit_StillAppliesInflightSave(t *testing.T) {
	agent := &stubAgent{}
	agent.saveStatusLine = "save-and-quit-status"
	m := newModelWithViewport(agent)
	cfg := config.DefaultConfig()
	m.screen = screenConfig
	m.configScreen = configscreen.New(cfg)
	setConfigModifiedForTest(t, &m)

	m = sendConfigKey(m, "q")
	cs := configTestScreen(t, m)
	if !cs.Snapshot().ConfirmQuit {
		t.Fatal("confirmQuit should be true")
	}

	setConfigConfirmIndexForTest(t, &m, 0)
	updated, saveCmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(Model)
	cs = configTestScreen(t, m)

	if cs == nil {
		t.Fatal("configScreen should not be nil yet")
	}
	if !cs.Snapshot().PendingClose {
		t.Fatal("pendingClose should be true")
	}
	if saveCmd == nil {
		t.Fatal("saveCmd should not be nil for save-and-quit")
	}

	resultMsg := saveCmd()
	updated, _ = m.Update(resultMsg)
	m = updated.(Model)

	if m.screen != screenChat {
		t.Fatalf("screen = %d, want screenChat after save-and-quit", m.screen)
	}
	if m.configScreen != nil {
		t.Fatal("configScreen should be nil after save-and-quit")
	}

	agent.mu.RLock()
	sl := agent.statusLine
	agent.mu.RUnlock()
	if sl != "save-and-quit-status" {
		t.Fatalf("statusLine = %q, want %q", sl, "save-and-quit-status")
	}
}

func TestConfigScreen_SaveWhileSaveAndQuitInFlight_KeepsPendingClose(t *testing.T) {
	agent := &stubAgent{}
	m := newModelWithViewport(agent)
	cfg := config.DefaultConfig()
	m.screen = screenConfig
	m.configScreen = configscreen.New(cfg)
	setConfigDirtyForTest(t, &m, true)

	m = sendConfigKey(m, "q")
	updated, saveAndQuitCmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(Model)

	if saveAndQuitCmd == nil {
		t.Fatal("saveAndQuitCmd should not be nil")
	}
	if !m.configScreen.Snapshot().PendingClose {
		t.Fatal("pendingClose should be true after save-and-quit")
	}

	updated, plainSaveCmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("s")})
	m = updated.(Model)

	if plainSaveCmd == nil {
		t.Fatal("plainSaveCmd should not be nil during in-flight save")
	}
	if !m.configScreen.Snapshot().PendingClose {
		t.Fatal("pendingClose should remain true after plain save during save-and-quit")
	}

	updated, _ = m.Update(plainSaveCmd())
	m = updated.(Model)

	if m.screen != screenChat {
		t.Fatalf("screen = %d, want screenChat after save result", m.screen)
	}
	if m.configScreen != nil {
		t.Fatal("configScreen should be nil after pending close completes")
	}
}
