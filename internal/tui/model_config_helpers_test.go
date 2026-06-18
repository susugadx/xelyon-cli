package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/susugadx/xelyon-cli/internal/config"
)

func newConfigTestModel() Model {
	return newConfigTestModelWithConfig(config.DefaultConfig())
}

func newConfigTestModelWithConfig(cfg *config.Config) Model {
	agent := &stubAgent{}
	m := newModelWithViewport(agent)
	m.screen = screenConfig
	m.configScreen = newConfigScreen(cfg)
	return m
}

func sendConfigKey(m Model, s string) Model {
	var msg tea.KeyMsg
	switch s {
	case "enter":
		msg = tea.KeyMsg{Type: tea.KeyEnter}
	case "esc":
		msg = tea.KeyMsg{Type: tea.KeyEsc}
	case "up":
		msg = tea.KeyMsg{Type: tea.KeyUp}
	case "down":
		msg = tea.KeyMsg{Type: tea.KeyDown}
	case "left":
		msg = tea.KeyMsg{Type: tea.KeyLeft}
	case "right":
		msg = tea.KeyMsg{Type: tea.KeyRight}
	default:
		if len(s) == 1 {
			msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)}
		}
	}
	updated, _ := m.Update(msg)
	return updated.(Model)
}

func sendConfigKeys(m Model, keys ...string) Model {
	for _, k := range keys {
		m = sendConfigKey(m, k)
	}
	return m
}

func saveConfigAndWait(t *testing.T, m Model) Model {
	t.Helper()

	updated, saveCmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("s")})
	m = updated.(Model)
	if saveCmd == nil {
		t.Fatal("saveCmd should not be nil")
	}

	resultMsg := saveCmd()
	updated, _ = m.Update(resultMsg)
	return updated.(Model)
}

func configTestScreen(t *testing.T, m Model) *configScreen {
	t.Helper()
	if m.configScreen == nil {
		t.Fatal("configScreen is nil")
	}
	return m.configScreen
}

type configScreenTestSnapshot struct {
	categoryIndex int
	fieldIndex    int
	fieldScroll   int
	activePane    configPane
	editMode      configEditMode
	dirty         bool
	saveStatus    configSaveStatus
	saveError     string
	confirmQuit   bool
	pendingClose  bool
}

func configSnapshot(t *testing.T, m Model) configScreenTestSnapshot {
	t.Helper()
	cs := configTestScreen(t, m)
	return configScreenTestSnapshot{
		categoryIndex: cs.catIndex,
		fieldIndex:    cs.fieldIndex,
		fieldScroll:   cs.fieldScroll,
		activePane:    cs.activePane,
		editMode:      cs.editMode,
		dirty:         cs.dirty,
		saveStatus:    cs.saveStatus,
		saveError:     cs.saveError,
		confirmQuit:   cs.confirmQuit,
		pendingClose:  cs.pendingClose,
	}
}

func setConfigDirtyForTest(t *testing.T, m *Model, dirty bool) {
	t.Helper()
	cs := configTestScreen(t, *m)
	cs.dirty = dirty
}

func setConfigModifiedForTest(t *testing.T, m *Model) {
	t.Helper()
	cs := configTestScreen(t, *m)
	cs.dirty = true
	cs.saveStatus = statusModified
}

func setConfigConfirmQuitForTest(t *testing.T, m *Model, confirmIdx int) {
	t.Helper()
	cs := configTestScreen(t, *m)
	cs.dirty = true
	cs.saveStatus = statusModified
	cs.confirmQuit = true
	cs.confirmIdx = confirmIdx
}

func setConfigConfirmIndexForTest(t *testing.T, m *Model, confirmIdx int) {
	t.Helper()
	configTestScreen(t, *m).confirmIdx = confirmIdx
}

func makeConfigScreenDirty(t *testing.T, m Model) Model {
	t.Helper()

	selectConfigField(t, &m, "compression", "compression.enabled")
	m = sendConfigKey(m, " ")
	if !configSnapshot(t, m).dirty {
		t.Fatal("dirty should be true after toggle")
	}
	return m
}
