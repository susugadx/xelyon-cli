package tui

import (
	"fmt"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/susugadx/xelyon-cli/internal/config"
)

func TestConfigScreen_DirtyState(t *testing.T) {
	m := newConfigTestModel()
	cs := m.configScreen

	if cs.dirty {
		t.Fatal("dirty should be false initially")
	}
	if cs.saveStatus != statusSaved {
		t.Fatalf("saveStatus = %d, want statusSaved(%d)", cs.saveStatus, statusSaved)
	}

	selectConfigField(t, &m, "compression", "compression.enabled")

	m = sendConfigKey(m, " ")
	cs = configTestScreen(t, m)
	if !cs.dirty {
		t.Fatal("dirty should be true after edit")
	}

	updated, saveCmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("s")})
	m = updated.(Model)
	cs = m.configScreen
	if cs.saveStatus != statusSaving {
		t.Fatalf("saveStatus after s = %d, want statusSaving(%d)", cs.saveStatus, statusSaving)
	}
	if saveCmd == nil {
		t.Fatal("saveCmd should not be nil")
	}

	updated, _ = m.Update(saveCmd())
	m = updated.(Model)
	cs = m.configScreen
	if cs.dirty {
		t.Fatal("dirty should be false after save")
	}
	if cs.saveStatus != statusSaved {
		t.Fatalf("saveStatus = %d, want statusSaved(%d)", cs.saveStatus, statusSaved)
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
	if !cs.pendingClose {
		t.Fatal("pendingClose should be true")
	}
	if cs.saveStatus != statusSaving {
		t.Fatalf("saveStatus = %d, want statusSaving(%d)", cs.saveStatus, statusSaving)
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
	m.configScreen = newConfigScreen(config.DefaultConfig())
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
	m.configScreen = newConfigScreen(config.DefaultConfig())
	m.statusLine = agent.GetStatusLine()

	m = saveConfigAndWait(t, m)

	if m.screen != screenConfig {
		t.Fatalf("screen = %d, want screenConfig after normal save", m.screen)
	}
	if got := m.statusLine; got != "provider: openai model: gpt-5.4" {
		t.Fatalf("statusLine after save = %q, want updated runtime status", got)
	}
	if m.configScreen.saveStatus != statusSaved {
		t.Fatalf("saveStatus = %d, want statusSaved", m.configScreen.saveStatus)
	}
}

func TestConfigScreen_SaveAndQuit_Failure(t *testing.T) {
	agent := &stubAgent{saveErr: fmt.Errorf("disk full")}
	m := newModelWithViewport(agent)
	cfg := config.DefaultConfig()
	m.screen = screenConfig
	m.configScreen = newConfigScreen(cfg)
	setConfigDirtyForTest(t, &m, true)

	m = sendConfigKey(m, "q")
	m = sendConfigKey(m, "enter")

	updated, _ := m.Update(ConfigSavedMsg{Error: fmt.Errorf("disk full")})
	m = updated.(Model)

	if m.screen != screenConfig {
		t.Fatalf("screen = %d after failed save, want screenConfig", m.screen)
	}
	cs := configTestScreen(t, m)
	if cs.saveStatus != statusFailed {
		t.Fatalf("saveStatus = %d, want statusFailed(%d)", cs.saveStatus, statusFailed)
	}
	if cs.saveError != "disk full" {
		t.Fatalf("saveError = %q, want %q", cs.saveError, "disk full")
	}
	if cs.pendingClose {
		t.Fatal("pendingClose should be reset to false after failure")
	}
	if cs.confirmQuit {
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
	m.configScreen = newConfigScreen(config.DefaultConfig())
	m.statusLine = agent.GetStatusLine()

	m = saveConfigAndWait(t, m)

	if got := m.statusLine; got != "provider: deepseek model: deepseek-chat" {
		t.Fatalf("statusLine after failed save = %q, want unchanged pre-save status", got)
	}
	if m.configScreen.saveStatus != statusFailed {
		t.Fatalf("saveStatus = %d, want statusFailed", m.configScreen.saveStatus)
	}
	if m.configScreen.saveError != "disk full" {
		t.Fatalf("saveError = %q, want %q", m.configScreen.saveError, "disk full")
	}
}
