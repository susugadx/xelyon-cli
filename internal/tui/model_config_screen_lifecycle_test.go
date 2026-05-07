package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestConfigScreen_OpenAndClose(t *testing.T) {
	agent := &stubAgent{}
	m := newModelWithViewport(agent)

	updated, _ := m.openConfigScreen()
	m = updated.(Model)
	if m.screen != screenConfig {
		t.Fatalf("screen = %d, want screenConfig(%d)", m.screen, screenConfig)
	}
	if m.configScreen == nil {
		t.Fatal("configScreen is nil")
	}

	m = sendConfigKey(m, "q")
	if m.screen != screenChat {
		t.Fatalf("screen = %d after q, want screenChat(%d)", m.screen, screenChat)
	}
}

func TestTUIConfig_Bare_OpensScreen(t *testing.T) {
	agent := &stubAgent{}
	m := newModelWithViewport(agent)

	m.textInput.SetValue("/config")
	m = sendConfigKey(m, "enter")

	if m.screen != screenConfig {
		t.Fatalf("screen = %d, want screenConfig(%d)", m.screen, screenConfig)
	}
	if m.configScreen == nil {
		t.Fatal("configScreen is nil")
	}
}

func TestConfigScreen_CloseWithoutDirty(t *testing.T) {
	m := newConfigTestModel()
	m = sendConfigKey(m, "q")
	if m.screen != screenChat {
		t.Fatalf("screen = %d, want screenChat", m.screen)
	}
}

func TestConfigScreen_CloseAfterResize_SyncsChatViewport(t *testing.T) {
	m := newModelWithViewport(&stubAgent{})
	setModelRawLines(&m, 20)

	updated, _ := m.openConfigScreen()
	m = updated.(Model)

	updated, _ = m.Update(tea.WindowSizeMsg{Width: 40, Height: 20})
	m = updated.(Model)

	if m.screen != screenConfig {
		t.Fatalf("screen after resize = %d, want screenConfig", m.screen)
	}

	updated, _ = m.closeConfigScreen()
	m = updated.(Model)

	if m.screen != screenChat {
		t.Fatalf("screen after close = %d, want screenChat", m.screen)
	}
	if m.vp.width != 40 {
		t.Fatalf("vp.width = %d, want 40", m.vp.width)
	}
	wantVPHeight := 20 - m.footerHeight()
	if m.vp.height != wantVPHeight {
		t.Fatalf("vp.height = %d, want %d", m.vp.height, wantVPHeight)
	}
	if m.layout == nil {
		t.Fatal("layout should be rebuilt after close")
	}
	if got := len(m.getVisualRowContents()); got == 0 {
		t.Fatalf("visual row contents length = %d, want > 0", got)
	}
	if !m.chromeDirty {
		t.Fatal("chromeDirty should be true after close to rebuild chat chrome")
	}
}

func TestConfigScreen_ConfigCommandWithArgsDoesNotOpenTUIScreen(t *testing.T) {
	agent := &stubAgent{}
	m := newModelWithViewport(agent)

	m.textInput.SetValue("/config show")
	m = sendConfigKey(m, "enter")
	if m.screen == screenConfig {
		t.Fatal("/config show should not open config screen")
	}
}
