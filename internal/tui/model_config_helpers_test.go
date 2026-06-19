package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/tui/configscreen"
)

type configPane = configscreen.Pane
type configEditMode = configscreen.EditMode
type configSaveStatus = configscreen.SaveStatus

const (
	paneCategory   = configscreen.PaneCategory
	paneField      = configscreen.PaneField
	paneDetail     = configscreen.PaneDetail
	editNone       = configscreen.EditNone
	editInput      = configscreen.EditInput
	editSelect     = configscreen.EditSelect
	editSlice      = configscreen.EditSlice
	editStructMap  = configscreen.EditStructMap
	statusSaved    = configscreen.StatusSaved
	statusModified = configscreen.StatusModified
	statusSaving   = configscreen.StatusSaving
	statusFailed   = configscreen.StatusFailed
)

func newConfigTestModel() Model {
	return newConfigTestModelWithConfig(config.DefaultConfig())
}

func newConfigTestModelWithConfig(cfg *config.Config) Model {
	agent := &stubAgent{}
	m := newModelWithViewport(agent)
	m.screen = screenConfig
	m.configScreen = configscreen.New(cfg)
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

func sendConfigKeyMsg(m Model, msg tea.KeyMsg) Model {
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

func configTestScreen(t *testing.T, m Model) *configscreen.Screen {
	t.Helper()
	if m.configScreen == nil {
		t.Fatal("configScreen is nil")
	}
	return m.configScreen
}

type configStateTestSnapshot struct {
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

func configSnapshot(t *testing.T, m Model) configStateTestSnapshot {
	t.Helper()
	cs := configTestScreen(t, m)
	snapshot := cs.Snapshot()
	return configStateTestSnapshot{
		categoryIndex: snapshot.CategoryIndex,
		fieldIndex:    snapshot.FieldIndex,
		fieldScroll:   snapshot.FieldScroll,
		activePane:    snapshot.ActivePane,
		editMode:      snapshot.EditMode,
		dirty:         snapshot.Dirty,
		saveStatus:    snapshot.SaveStatus,
		saveError:     snapshot.SaveError,
		confirmQuit:   snapshot.ConfirmQuit,
		pendingClose:  snapshot.PendingClose,
	}
}

func setConfigDirtyForTest(t *testing.T, m *Model, dirty bool) {
	t.Helper()
	if dirty {
		*m = makeConfigScreenDirty(t, *m)
		return
	}
	t.Fatal("setConfigDirtyForTest(false) no longer mutates screen internals")
}

func setConfigModifiedForTest(t *testing.T, m *Model) {
	t.Helper()
	*m = makeConfigScreenDirty(t, *m)
}

func setConfigConfirmQuitForTest(t *testing.T, m *Model, confirmIdx int) {
	t.Helper()
	*m = makeConfigScreenDirty(t, *m)
	*m = sendConfigKey(*m, "q")
	for configTestScreen(t, *m).Snapshot().ConfirmIndex < confirmIdx {
		*m = sendConfigKey(*m, "down")
	}
}

func setConfigConfirmIndexForTest(t *testing.T, m *Model, confirmIdx int) {
	t.Helper()
	for configTestScreen(t, *m).Snapshot().ConfirmIndex < confirmIdx {
		*m = sendConfigKey(*m, "down")
	}
	for configTestScreen(t, *m).Snapshot().ConfirmIndex > confirmIdx {
		*m = sendConfigKey(*m, "up")
	}
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
