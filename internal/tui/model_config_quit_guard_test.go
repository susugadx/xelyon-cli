package tui

import (
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/susugadx/xelyon-cli/internal/config"
)

func TestConfigScreen_ConfirmQuitWithDirty_Discard(t *testing.T) {
	m := newConfigTestModel()
	selectConfigField(t, &m, "compression", "compression.enabled")
	m = sendConfigKey(m, " ")

	m = sendConfigKey(m, "q")
	cs := configTestScreen(t, m)
	if !cs.confirmQuit {
		t.Fatal("confirmQuit should be true")
	}

	m = sendConfigKeys(m, "j", "enter")
	if m.screen != screenChat {
		t.Fatalf("screen = %d after discard, want screenChat", m.screen)
	}
}

func TestConfigScreen_CtrlC_RespectsDirtyGuard(t *testing.T) {
	agent := &stubAgent{}
	m := newModelWithViewport(agent)
	updated, _ := m.openConfigScreen()
	m = updated.(Model)
	m = makeConfigScreenDirty(t, m)

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	m = updated.(Model)

	if cmd != nil {
		t.Fatal("ctrl+c with dirty config should not quit immediately")
	}
	if m.screen != screenConfig {
		t.Fatalf("screen = %d after ctrl+c, want screenConfig", m.screen)
	}
	if !m.configScreen.confirmQuit {
		t.Fatal("confirmQuit should be true after ctrl+c with dirty config")
	}
	if m.quitting {
		t.Fatal("quitting should remain false while dirty guard is open")
	}

	agent.mu.RLock()
	cancelCalls := agent.cancelCalls
	agent.mu.RUnlock()
	if cancelCalls != 0 {
		t.Fatalf("cancelCalls = %d, want 0", cancelCalls)
	}

	updated, cmd = m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	m = updated.(Model)
	if cmd != nil {
		t.Fatal("ctrl+c on confirm dialog should not quit")
	}
	if !m.configScreen.confirmQuit {
		t.Fatal("confirmQuit should remain true after second ctrl+c")
	}
	if m.screen != screenConfig {
		t.Fatalf("screen = %d after second ctrl+c, want screenConfig", m.screen)
	}
}

func TestConfigScreen_CtrlC_DoesNotBypassDirtyOnDoublePress(t *testing.T) {
	agent := &stubAgent{}
	m := newModelWithViewport(agent)
	updated, _ := m.openConfigScreen()
	m = updated.(Model)
	m = makeConfigScreenDirty(t, m)

	m.lastInterrupt = time.Now()

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	m = updated.(Model)
	if cmd != nil {
		t.Fatal("ctrl+c with dirty config should not quit even if interrupt window is active")
	}
	if !m.configScreen.confirmQuit {
		t.Fatal("confirmQuit should be true after first ctrl+c")
	}
	if m.quitting {
		t.Fatal("quitting should remain false after first ctrl+c")
	}

	updated, cmd = m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	m = updated.(Model)
	if cmd != nil {
		t.Fatal("second ctrl+c with dirty config should still not quit")
	}
	if m.quitting {
		t.Fatal("quitting should remain false after second ctrl+c")
	}
	if m.screen != screenConfig {
		t.Fatalf("screen = %d after double ctrl+c, want screenConfig", m.screen)
	}
	if !m.configScreen.dirty {
		t.Fatal("dirty should remain true after double ctrl+c")
	}
	if !m.configScreen.confirmQuit {
		t.Fatal("confirmQuit should remain true after double ctrl+c")
	}
}

func TestConfigScreen_ConfirmQuitCancel(t *testing.T) {
	m := newConfigTestModel()
	setConfigDirtyForTest(t, &m, true)

	m = sendConfigKey(m, "q")
	cs := configTestScreen(t, m)
	if !cs.confirmQuit {
		t.Fatal("confirmQuit should be true")
	}

	m = sendConfigKeys(m, "j", "j", "enter")
	cs = configTestScreen(t, m)
	if cs.confirmQuit {
		t.Fatal("confirmQuit should be false after cancel")
	}
	if m.screen != screenConfig {
		t.Fatalf("screen = %d after cancel, want screenConfig", m.screen)
	}
}

func TestConfigScreen_ConfirmQuit_CtrlC_StillCancelsProcessing(t *testing.T) {
	agent := &stubAgent{}
	agent.setProcessing(true)
	m := newModelWithViewport(agent)
	cfg := config.DefaultConfig()
	m.screen = screenConfig
	m.configScreen = newConfigScreen(cfg)

	setConfigConfirmQuitForTest(t, &m, 1)

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	m = updated.(Model)

	if cmd != nil {
		t.Fatal("ctrl+c while processing should not return tea.Quit")
	}
	agent.mu.RLock()
	cancelCalls := agent.cancelCalls
	agent.mu.RUnlock()
	if cancelCalls != 1 {
		t.Fatalf("cancelCalls = %d, want 1", cancelCalls)
	}
	if m.screen != screenConfig {
		t.Fatalf("screen = %d, want screenConfig", m.screen)
	}
	if m.configScreen == nil {
		t.Fatal("configScreen should remain available")
	}
}

func TestConfigScreen_ConfirmQuit_CtrlC_WhenIdle_DoesNotBreakQuitDialog(t *testing.T) {
	agent := &stubAgent{}
	m := newModelWithViewport(agent)
	cfg := config.DefaultConfig()
	m.screen = screenConfig
	m.configScreen = newConfigScreen(cfg)

	setConfigConfirmQuitForTest(t, &m, 0)

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	m = updated.(Model)

	if cmd != nil {
		t.Fatal("ctrl+c while idle+confirmQuit should not quit")
	}
	agent.mu.RLock()
	cancelCalls := agent.cancelCalls
	agent.mu.RUnlock()
	if cancelCalls != 0 {
		t.Fatalf("cancelCalls = %d, want 0 when idle", cancelCalls)
	}
	if m.screen != screenConfig {
		t.Fatalf("screen = %d, want screenConfig", m.screen)
	}
	if m.configScreen == nil {
		t.Fatal("configScreen should not be nil")
	}
}
