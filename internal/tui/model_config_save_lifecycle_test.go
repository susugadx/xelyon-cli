package tui

import (
	"fmt"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/tui/configscreen"
)

func TestConfigScreen_DirtyState(t *testing.T) {
	m := newConfigTestModel()
	cs := m.configScreen

	snapshot := cs.Snapshot()
	if snapshot.Dirty {
		t.Fatal("dirty should be false initially")
	}
	if snapshot.SaveStatus != statusSaved {
		t.Fatalf("saveStatus = %d, want statusSaved(%d)", snapshot.SaveStatus, statusSaved)
	}

	selectConfigField(t, &m, "compression", "compression.enabled")

	m = sendConfigKey(m, " ")
	cs = configTestScreen(t, m)
	if !cs.Snapshot().Dirty {
		t.Fatal("dirty should be true after edit")
	}

	updated, saveCmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("s")})
	m = updated.(Model)
	cs = m.configScreen
	if got := cs.Snapshot().SaveStatus; got != statusSaving {
		t.Fatalf("saveStatus after s = %d, want statusSaving(%d)", got, statusSaving)
	}
	if saveCmd == nil {
		t.Fatal("saveCmd should not be nil")
	}

	updated, _ = m.Update(saveCmd())
	m = updated.(Model)
	cs = m.configScreen
	snapshot = cs.Snapshot()
	if snapshot.Dirty {
		t.Fatal("dirty should be false after save")
	}
	if snapshot.SaveStatus != statusSaved {
		t.Fatalf("saveStatus = %d, want statusSaved(%d)", snapshot.SaveStatus, statusSaved)
	}
}

func TestConfigScreen_SaveAndQuit_Success(t *testing.T) {
	m := newConfigTestModel()
	setConfigDirtyForTest(t, &m, true)

	m = sendConfigKey(m, "q")
	updated, saveCmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(Model)
	if saveCmd == nil {
		t.Fatal("saveCmd should not be nil")
	}

	cs := configTestScreen(t, m)
	snapshot := cs.Snapshot()
	if !snapshot.PendingClose {
		t.Fatal("pendingClose should be true")
	}
	if snapshot.SaveStatus != statusSaving {
		t.Fatalf("saveStatus = %d, want statusSaving(%d)", snapshot.SaveStatus, statusSaving)
	}
	if m.screen != screenConfig {
		t.Fatalf("screen = %d, want screenConfig (save in progress)", m.screen)
	}

	updated, _ = m.Update(saveCmd())
	m = updated.(Model)
	if m.screen != screenChat {
		t.Fatalf("screen = %d after successful save, want screenChat", m.screen)
	}
}

func TestConfigScreen_SaveAndClose_RefreshesStatusLine(t *testing.T) {
	agent := &stubAgent{
		statusLine:     "provider: deepseek model: deepseek-chat",
		saveStatusLine: "provider: openai model: gpt-5.4",
	}
	m := newModelWithViewport(agent)
	m.screen = screenConfig
	m.configScreen = configscreen.New(config.DefaultConfig())
	m.statusLine = agent.GetStatusLine()

	setConfigDirtyForTest(t, &m, true)

	m = sendConfigKey(m, "q")
	updated, saveCmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(Model)
	if saveCmd == nil {
		t.Fatal("saveCmd should not be nil")
	}

	updated, _ = m.Update(saveCmd())
	m = updated.(Model)

	if m.screen != screenChat {
		t.Fatalf("screen = %d after successful save, want screenChat", m.screen)
	}
	if got := m.statusLine; got != "provider: openai model: gpt-5.4" {
		t.Fatalf("statusLine after save-and-close = %q, want updated runtime status", got)
	}
}

func TestConfigScreen_SaveWithoutClose_RefreshesStatusLineIfNeeded(t *testing.T) {
	agent := &stubAgent{
		statusLine:     "provider: deepseek model: deepseek-chat",
		saveStatusLine: "provider: openai model: gpt-5.4",
	}
	m := newModelWithViewport(agent)
	m.screen = screenConfig
	m.configScreen = configscreen.New(config.DefaultConfig())
	m.statusLine = agent.GetStatusLine()

	m = saveConfigAndWait(t, m)

	if m.screen != screenConfig {
		t.Fatalf("screen = %d, want screenConfig after normal save", m.screen)
	}
	if got := m.statusLine; got != "provider: openai model: gpt-5.4" {
		t.Fatalf("statusLine after save = %q, want updated runtime status", got)
	}
	if got := m.configScreen.Snapshot().SaveStatus; got != statusSaved {
		t.Fatalf("saveStatus = %d, want statusSaved", got)
	}
}

func TestConfigScreen_SaveAndQuit_Failure(t *testing.T) {
	agent := &stubAgent{saveErr: fmt.Errorf("disk full")}
	m := newModelWithViewport(agent)
	cfg := config.DefaultConfig()
	m.screen = screenConfig
	m.configScreen = configscreen.New(cfg)
	setConfigDirtyForTest(t, &m, true)

	m = sendConfigKey(m, "q")
	m = sendConfigKey(m, "enter")

	updated, _ := m.Update(ConfigSavedMsg{Error: fmt.Errorf("disk full")})
	m = updated.(Model)

	if m.screen != screenConfig {
		t.Fatalf("screen = %d after failed save, want screenConfig", m.screen)
	}
	cs := configTestScreen(t, m)
	snapshot := cs.Snapshot()
	if snapshot.SaveStatus != statusFailed {
		t.Fatalf("saveStatus = %d, want statusFailed(%d)", snapshot.SaveStatus, statusFailed)
	}
	if snapshot.SaveError != "disk full" {
		t.Fatalf("saveError = %q, want %q", snapshot.SaveError, "disk full")
	}
	if snapshot.PendingClose {
		t.Fatal("pendingClose should be reset to false after failure")
	}
	if snapshot.ConfirmQuit {
		t.Fatal("confirmQuit should be false (dialog dismissed)")
	}
}

func TestConfigScreen_SaveFailure_DoesNotReportUpdatedFooter(t *testing.T) {
	agent := &stubAgent{
		statusLine:     "provider: deepseek model: deepseek-chat",
		saveStatusLine: "provider: openai model: gpt-5.4",
		saveErr:        fmt.Errorf("disk full"),
	}
	m := newModelWithViewport(agent)
	m.screen = screenConfig
	m.configScreen = configscreen.New(config.DefaultConfig())
	m.statusLine = agent.GetStatusLine()

	m = saveConfigAndWait(t, m)

	if got := m.statusLine; got != "provider: deepseek model: deepseek-chat" {
		t.Fatalf("statusLine after failed save = %q, want unchanged pre-save status", got)
	}
	snapshot := m.configScreen.Snapshot()
	if snapshot.SaveStatus != statusFailed {
		t.Fatalf("saveStatus = %d, want statusFailed", snapshot.SaveStatus)
	}
	if snapshot.SaveError != "disk full" {
		t.Fatalf("saveError = %q, want %q", snapshot.SaveError, "disk full")
	}
}
