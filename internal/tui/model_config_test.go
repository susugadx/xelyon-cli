package tui

import (
	"fmt"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/susugadx/xelyon-cli/internal/config"
)

// --- helpers ---

func newConfigTestModel() Model {
	agent := &stubAgent{}
	m := newModelWithViewport(agent)
	cfg := config.DefaultConfig()
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

func setConfigFieldSelection(t *testing.T, cs *configScreen, categoryName, fieldPath string) {
	t.Helper()

	for i, cat := range cs.categories {
		if cat.Name == categoryName {
			cs.catIndex = i
			cs.activePane = paneField
			for j, f := range cs.filteredFields() {
				if f.Path == fieldPath {
					cs.fieldIndex = j
					return
				}
			}
			t.Fatalf("field %q not found in category %q", fieldPath, categoryName)
		}
	}

	t.Fatalf("category %q not found", categoryName)
}

func selectConfigOption(t *testing.T, m Model, categoryName, fieldPath, option string) Model {
	t.Helper()

	cs := m.configScreen
	setConfigFieldSelection(t, cs, categoryName, fieldPath)

	m = sendConfigKey(m, "enter")
	cs = m.configScreen
	if cs.editMode != editSelect {
		t.Fatalf("editMode = %d, want editSelect", cs.editMode)
	}

	field := cs.selectedField()
	if field == nil {
		t.Fatal("selectedField is nil")
	}

	found := false
	for i, candidate := range field.Options {
		if candidate == option {
			cs.editSelect = i
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("option %q not found in %s", option, fieldPath)
	}

	m = sendConfigKey(m, "enter")
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

// --- tests ---

func TestConfigScreen_OpenAndClose(t *testing.T) {
	agent := &stubAgent{}
	m := newModelWithViewport(agent)

	// /config で config screen に遷移
	updated, _ := m.openConfigScreen()
	m = updated.(Model)
	if m.screen != screenConfig {
		t.Fatalf("screen = %d, want screenConfig(%d)", m.screen, screenConfig)
	}
	if m.configScreen == nil {
		t.Fatal("configScreen is nil")
	}

	// q で閉じる
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

func TestConfigScreen_CategoryNavigation(t *testing.T) {
	m := newConfigTestModel()

	if m.configScreen.catIndex != 0 {
		t.Fatalf("initial catIndex = %d, want 0", m.configScreen.catIndex)
	}

	// j で下移動
	m = sendConfigKey(m, "j")
	if m.configScreen.catIndex != 1 {
		t.Fatalf("catIndex after j = %d, want 1", m.configScreen.catIndex)
	}

	// k で上移動
	m = sendConfigKey(m, "k")
	if m.configScreen.catIndex != 0 {
		t.Fatalf("catIndex after k = %d, want 0", m.configScreen.catIndex)
	}

	// 矢印キーでも同じ
	m = sendConfigKey(m, "down")
	if m.configScreen.catIndex != 1 {
		t.Fatalf("catIndex after down = %d, want 1", m.configScreen.catIndex)
	}
}

func TestConfigScreen_PaneNavigation(t *testing.T) {
	m := newConfigTestModel()

	// 初期はカテゴリペイン
	if m.configScreen.activePane != paneCategory {
		t.Fatalf("initial pane = %d, want paneCategory", m.configScreen.activePane)
	}

	// l でフィールドペインへ
	m = sendConfigKey(m, "l")
	if m.configScreen.activePane != paneField {
		t.Fatalf("pane after l = %d, want paneField", m.configScreen.activePane)
	}

	// l で詳細ペインへ
	m = sendConfigKey(m, "l")
	if m.configScreen.activePane != paneDetail {
		t.Fatalf("pane after l+l = %d, want paneDetail", m.configScreen.activePane)
	}

	// h でフィールドペインに戻る
	m = sendConfigKey(m, "h")
	if m.configScreen.activePane != paneField {
		t.Fatalf("pane after h = %d, want paneField", m.configScreen.activePane)
	}

	// h でカテゴリペインに戻る
	m = sendConfigKey(m, "h")
	if m.configScreen.activePane != paneCategory {
		t.Fatalf("pane after h+h = %d, want paneCategory", m.configScreen.activePane)
	}
}

func TestConfigScreen_EnterMovesToFieldPane(t *testing.T) {
	m := newConfigTestModel()

	// Enter でフィールドペインに遷移
	m = sendConfigKey(m, "enter")
	if m.configScreen.activePane != paneField {
		t.Fatalf("pane after Enter = %d, want paneField", m.configScreen.activePane)
	}
}

func TestConfigScreen_BoolToggle(t *testing.T) {
	m := newConfigTestModel()

	// "compression" カテゴリの "enabled" フィールドを見つける
	cs := m.configScreen
	for i, cat := range cs.categories {
		if cat.Name == "compression" {
			cs.catIndex = i
			break
		}
	}

	// フィールドペインに移動
	cs.activePane = paneField
	cs.fieldIndex = 0

	// compression.enabled を見つける
	fields := cs.filteredFields()
	for i, f := range fields {
		if f.Path == "compression.enabled" {
			cs.fieldIndex = i
			break
		}
	}

	// 現在値を取得
	field := cs.selectedField()
	if field == nil {
		t.Fatal("selectedField is nil")
	}
	current, _ := field.Current.(bool)

	// Space でトグル
	m = sendConfigKey(m, " ")

	// 値が変わったか確認
	cs = m.configScreen
	newVal, _ := config.GetFieldValue(cs.cfg, "compression.enabled")
	if newVal.(bool) == current {
		t.Fatalf("bool value did not toggle: still %v", current)
	}

	// dirty になったか
	if !cs.dirty {
		t.Fatal("dirty should be true after toggle")
	}
	if cs.saveStatus != statusModified {
		t.Fatalf("saveStatus = %d, want statusModified(%d)", cs.saveStatus, statusModified)
	}
}

func TestConfigScreen_SelectEdit(t *testing.T) {
	m := newConfigTestModel()
	cs := m.configScreen

	// "execution" カテゴリに移動
	for i, cat := range cs.categories {
		if cat.Name == "execution" {
			cs.catIndex = i
			break
		}
	}

	cs.activePane = paneField
	// execution.mode フィールドを探す
	fields := cs.filteredFields()
	for i, f := range fields {
		if f.Path == "execution.mode" {
			cs.fieldIndex = i
			break
		}
	}

	// Enter で select 編集開始
	m = sendConfigKey(m, "enter")
	cs = m.configScreen
	if cs.editMode != editSelect {
		t.Fatalf("editMode = %d, want editSelect(%d)", cs.editMode, editSelect)
	}

	// j で下移動、Enter で確定
	m = sendConfigKeys(m, "j", "enter")
	cs = m.configScreen
	if cs.editMode != editNone {
		t.Fatalf("editMode after select = %d, want editNone", cs.editMode)
	}
	if !cs.dirty {
		t.Fatal("dirty should be true after select edit")
	}
}

func TestConfigScreen_StringEdit(t *testing.T) {
	m := newConfigTestModel()
	cs := m.configScreen

	// provider カテゴリの default_model
	for i, cat := range cs.categories {
		if cat.Name == "provider" {
			cs.catIndex = i
			break
		}
	}
	cs.activePane = paneField
	fields := cs.filteredFields()
	for i, f := range fields {
		if f.Path == "default_model" {
			cs.fieldIndex = i
			break
		}
	}

	// Enter で入力開始
	m = sendConfigKey(m, "enter")
	cs = m.configScreen
	if cs.editMode != editInput {
		t.Fatalf("editMode = %d, want editInput(%d)", cs.editMode, editInput)
	}

	// Esc でキャンセル
	m = sendConfigKey(m, "esc")
	cs = m.configScreen
	if cs.editMode != editNone {
		t.Fatalf("editMode after esc = %d, want editNone", cs.editMode)
	}
}

func TestConfigScreen_NarrowWidth_StringEdit_RemainsVisible(t *testing.T) {
	m := newConfigTestModel()
	m.width = 72
	m.height = 20

	cs := m.configScreen
	setConfigFieldSelection(t, cs, "provider", "default_model")

	_, _, rightW := configPaneWidths(m.width)
	if rightW != 0 {
		t.Fatalf("rightW = %d, want 0 for narrow width", rightW)
	}

	m = sendConfigKey(m, "enter")
	cs = m.configScreen
	if cs.editMode != editInput {
		t.Fatalf("editMode = %d, want editInput", cs.editMode)
	}
	if cs.activePane != paneField {
		t.Fatalf("activePane = %d, want paneField when detail pane is hidden", cs.activePane)
	}

	view := stripANSI(m.View())
	if !strings.Contains(view, "Edit:") {
		t.Fatal("narrow width view should render the input editor")
	}
	if !strings.Contains(view, "deepseek-chat") {
		t.Fatal("narrow width view should include the current input value")
	}
}

func TestConfigScreen_NarrowWidth_SelectEdit_RemainsVisible(t *testing.T) {
	m := newConfigTestModel()
	m.width = 72
	m.height = 20

	cs := m.configScreen
	setConfigFieldSelection(t, cs, "execution", "execution.mode")

	_, _, rightW := configPaneWidths(m.width)
	if rightW != 0 {
		t.Fatalf("rightW = %d, want 0 for narrow width", rightW)
	}

	m = sendConfigKey(m, "enter")
	cs = m.configScreen
	if cs.editMode != editSelect {
		t.Fatalf("editMode = %d, want editSelect", cs.editMode)
	}
	if cs.activePane != paneField {
		t.Fatalf("activePane = %d, want paneField when detail pane is hidden", cs.activePane)
	}

	view := stripANSI(m.View())
	if !strings.Contains(view, "Select:") {
		t.Fatal("narrow width view should render the select editor")
	}
	if !strings.Contains(view, "balanced") {
		t.Fatal("narrow width view should render visible select options")
	}
}

func TestConfigScreen_VeryNarrowWidth_ConfigDoesNotEnterInvisiblePane(t *testing.T) {
	m := newConfigTestModel()
	m.width = 30
	m.height = 20

	cs := m.configScreen
	setConfigFieldSelection(t, cs, "lsp", "lsp.servers")

	leftW, midW, rightW := configPaneWidths(m.width)
	if leftW != 30 || midW != 0 || rightW != 0 {
		t.Fatalf("configPaneWidths(%d) = (%d, %d, %d), want (30, 0, 0)", m.width, leftW, midW, rightW)
	}

	m = sendConfigKey(m, "enter")
	cs = m.configScreen
	if cs.editMode != editStructMap {
		t.Fatalf("editMode = %d, want editStructMap", cs.editMode)
	}
	if cs.activePane == paneDetail {
		t.Fatalf("activePane = %d, should not enter invisible detail pane", cs.activePane)
	}

	view := stripANSI(m.View())
	if !strings.Contains(view, "Keys:") {
		t.Fatal("very narrow view should render a visible struct map editor")
	}
	if !strings.Contains(view, "go") {
		t.Fatal("very narrow view should render struct map entries")
	}
}

func TestConfigScreen_NormalWidth_BehaviorUnchanged(t *testing.T) {
	m := newConfigTestModel()
	m.width = 120
	m.height = 30

	cs := m.configScreen
	setConfigFieldSelection(t, cs, "provider", "default_model")

	_, _, rightW := configPaneWidths(m.width)
	if rightW <= 0 {
		t.Fatalf("rightW = %d, want visible detail pane on normal width", rightW)
	}

	m = sendConfigKey(m, "enter")
	cs = m.configScreen
	if cs.editMode != editInput {
		t.Fatalf("editMode = %d, want editInput", cs.editMode)
	}
	if cs.activePane != paneDetail {
		t.Fatalf("activePane = %d, want paneDetail on normal width", cs.activePane)
	}

	view := stripANSI(m.View())
	if !strings.Contains(view, "Edit:") {
		t.Fatal("normal width view should render the input editor")
	}
}

func TestConfigScreen_SliceEdit(t *testing.T) {
	m := newConfigTestModel()
	cs := m.configScreen

	// execution カテゴリの safe_shell_commands
	for i, cat := range cs.categories {
		if cat.Name == "execution" {
			cs.catIndex = i
			break
		}
	}
	cs.activePane = paneField
	fields := cs.filteredFields()
	for i, f := range fields {
		if f.Path == "execution.safe_shell_commands" {
			cs.fieldIndex = i
			break
		}
	}

	// Enter でスライス編集
	m = sendConfigKey(m, "enter")
	cs = m.configScreen
	if cs.editMode != editSlice {
		t.Fatalf("editMode = %d, want editSlice(%d)", cs.editMode, editSlice)
	}

	// a で追加モードに入る
	m = sendConfigKey(m, "a")
	cs = m.configScreen
	if !cs.editSliceAdding {
		t.Fatal("editSliceAdding should be true")
	}

	// Esc で追加キャンセル
	m = sendConfigKey(m, "esc")
	cs = m.configScreen
	if cs.editSliceAdding {
		t.Fatal("editSliceAdding should be false after esc")
	}

	// Esc でスライス編集終了
	m = sendConfigKey(m, "esc")
	cs = m.configScreen
	if cs.editMode != editNone {
		t.Fatalf("editMode after esc = %d, want editNone", cs.editMode)
	}
}

func TestConfigScreen_StructMapEdit(t *testing.T) {
	m := newConfigTestModel()
	cs := m.configScreen

	// lsp カテゴリの lsp.servers
	for i, cat := range cs.categories {
		if cat.Name == "lsp" {
			cs.catIndex = i
			break
		}
	}
	cs.activePane = paneField
	fields := cs.filteredFields()
	for i, f := range fields {
		if f.Path == "lsp.servers" {
			cs.fieldIndex = i
			break
		}
	}

	// Enter で structmap 編集
	m = sendConfigKey(m, "enter")
	cs = m.configScreen
	if cs.editMode != editStructMap {
		t.Fatalf("editMode = %d, want editStructMap(%d)", cs.editMode, editStructMap)
	}
	if len(cs.editStructKeys) == 0 {
		t.Fatal("editStructKeys should not be empty for lsp.servers")
	}

	// Esc で戻る
	m = sendConfigKey(m, "esc")
	cs = m.configScreen
	if cs.editMode != editNone {
		t.Fatalf("editMode after esc = %d, want editNone", cs.editMode)
	}
}

func TestConfigScreen_LSPServersEdit(t *testing.T) {
	m := newConfigTestModel()
	cs := m.configScreen

	// lsp カテゴリ
	for i, cat := range cs.categories {
		if cat.Name == "lsp" {
			cs.catIndex = i
			break
		}
	}
	cs.activePane = paneField
	fields := cs.filteredFields()
	found := false
	for i, f := range fields {
		if f.Path == "lsp.servers" {
			cs.fieldIndex = i
			found = true
			break
		}
	}
	if !found {
		t.Fatal("lsp.servers not found in lsp category fields")
	}

	// Enter で structmap 編集
	m = sendConfigKey(m, "enter")
	cs = m.configScreen
	if cs.editMode != editStructMap {
		t.Fatalf("editMode = %d, want editStructMap(%d)", cs.editMode, editStructMap)
	}
	// デフォルト設定では23言語分のサーバーがある
	if len(cs.editStructKeys) == 0 {
		t.Fatal("editStructKeys should not be empty for lsp.servers")
	}
}

func TestConfigScreen_DirtyState(t *testing.T) {
	m := newConfigTestModel()
	cs := m.configScreen

	// 初期状態
	if cs.dirty {
		t.Fatal("dirty should be false initially")
	}
	if cs.saveStatus != statusSaved {
		t.Fatalf("saveStatus = %d, want statusSaved(%d)", cs.saveStatus, statusSaved)
	}

	// compression.enabled をトグル
	for i, cat := range cs.categories {
		if cat.Name == "compression" {
			cs.catIndex = i
			break
		}
	}
	cs.activePane = paneField
	fields := cs.filteredFields()
	for i, f := range fields {
		if f.Path == "compression.enabled" {
			cs.fieldIndex = i
			break
		}
	}

	m = sendConfigKey(m, " ")
	cs = m.configScreen
	if !cs.dirty {
		t.Fatal("dirty should be true after edit")
	}

	// s で保存
	updated, saveCmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("s")})
	m = updated.(Model)
	cs = m.configScreen
	if cs.saveStatus != statusSaving {
		t.Fatalf("saveStatus after s = %d, want statusSaving(%d)", cs.saveStatus, statusSaving)
	}
	if saveCmd == nil {
		t.Fatal("saveCmd should not be nil")
	}

	// ConfigSavedMsg で保存完了
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

func TestConfigScreen_ConfirmQuitWithDirty_Discard(t *testing.T) {
	m := newConfigTestModel()
	cs := m.configScreen

	// dirty にする
	for i, cat := range cs.categories {
		if cat.Name == "compression" {
			cs.catIndex = i
			break
		}
	}
	cs.activePane = paneField
	fields := cs.filteredFields()
	for i, f := range fields {
		if f.Path == "compression.enabled" {
			cs.fieldIndex = i
			break
		}
	}
	m = sendConfigKey(m, " ")

	// q で終了確認表示
	m = sendConfigKey(m, "q")
	cs = m.configScreen
	if !cs.confirmQuit {
		t.Fatal("confirmQuit should be true")
	}

	// Discard and quit (index 1)
	m = sendConfigKeys(m, "j", "enter")
	if m.screen != screenChat {
		t.Fatalf("screen = %d after discard, want screenChat", m.screen)
	}
}

func TestConfigScreen_SaveAndQuit_Success(t *testing.T) {
	m := newConfigTestModel()
	cs := m.configScreen
	cs.dirty = true

	// q → 終了確認 → Save and quit (index 0)
	m = sendConfigKey(m, "q")
	updated, saveCmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter}) // index 0 = Save and quit
	m = updated.(Model)
	if saveCmd == nil {
		t.Fatal("saveCmd should not be nil")
	}

	// pendingClose フラグが立ち、saveStatus=statusSaving
	cs = m.configScreen
	if !cs.pendingClose {
		t.Fatal("pendingClose should be true")
	}
	if cs.saveStatus != statusSaving {
		t.Fatalf("saveStatus = %d, want statusSaving(%d)", cs.saveStatus, statusSaving)
	}
	// まだ画面は閉じていない
	if m.screen != screenConfig {
		t.Fatalf("screen = %d, want screenConfig (save in progress)", m.screen)
	}

	// ConfigSavedMsg(成功) → 閉じる
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

	cs := m.configScreen
	cs.dirty = true

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
	cs := m.configScreen
	cs.dirty = true

	// q → 終了確認 → Save and quit (index 0)
	m = sendConfigKey(m, "q")
	m = sendConfigKey(m, "enter")

	// saveConfigCmd が非同期に ConfigSavedMsg を返す — 手動で送る
	updated, _ := m.Update(ConfigSavedMsg{Error: fmt.Errorf("disk full")})
	m = updated.(Model)

	// 画面は閉じない
	if m.screen != screenConfig {
		t.Fatalf("screen = %d after failed save, want screenConfig", m.screen)
	}
	cs = m.configScreen
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

func TestConfigScreen_SaveKeepsDirtyWhenEditedAfterSaveStarts(t *testing.T) {
	agent := &stubAgent{}
	m := newModelWithViewport(agent)
	m.screen = screenConfig
	m.configScreen = newConfigScreen(config.DefaultConfig())

	cs := m.configScreen
	setConfigFieldSelection(t, cs, "compression", "compression.enabled")

	// Make the screen dirty with an initial edit.
	m = sendConfigKey(m, " ")
	cs = m.configScreen
	if !cs.dirty {
		t.Fatal("dirty should be true after initial edit")
	}

	// Start save and capture the snapshot used by the async command.
	updated, saveCmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("s")})
	m = updated.(Model)
	if saveCmd == nil {
		t.Fatal("saveCmd should not be nil")
	}

	// Apply another edit while the save is still in flight.
	m = sendConfigKey(m, " ")
	cs = m.configScreen
	if !cs.dirty {
		t.Fatal("dirty should remain true after late edit")
	}
	if cs.saveStatus != statusModified {
		t.Fatalf("saveStatus after late edit = %d, want statusModified(%d)", cs.saveStatus, statusModified)
	}

	// When the outdated snapshot save completes, the current cfg must stay unsaved.
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

	// Start save and quit.
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

	// Edit again while the save is still in flight.
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

func TestConfigScreen_CtrlC_CancelsWhileProcessing(t *testing.T) {
	agent := &stubAgent{}
	agent.setProcessing(true)

	m := newModelWithViewport(agent)
	updated, _ := m.openConfigScreen()
	m = updated.(Model)

	if m.screen != screenConfig {
		t.Fatalf("screen = %d, want screenConfig", m.screen)
	}

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	m = updated.(Model)

	if cmd != nil {
		t.Fatal("ctrl+c while processing should not quit")
	}
	agent.mu.RLock()
	cancelCalls := agent.cancelCalls
	agent.mu.RUnlock()
	if cancelCalls != 1 {
		t.Fatalf("cancelCalls = %d, want 1", cancelCalls)
	}
	if m.screen != screenConfig {
		t.Fatalf("screen = %d after ctrl+c, want screenConfig", m.screen)
	}
	if m.configScreen == nil {
		t.Fatal("configScreen should remain available after ctrl+c")
	}
}

func TestConfigScreen_CtrlC_NoUnexpectedEffectWhenIdle(t *testing.T) {
	agent := &stubAgent{}
	m := newModelWithViewport(agent)
	updated, _ := m.openConfigScreen()
	m = updated.(Model)

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	m = updated.(Model)

	if cmd != nil {
		t.Fatal("first ctrl+c while idle should not quit")
	}
	agent.mu.RLock()
	cancelCalls := agent.cancelCalls
	agent.mu.RUnlock()
	if cancelCalls != 0 {
		t.Fatalf("cancelCalls = %d, want 0", cancelCalls)
	}
	if m.lastInterrupt.IsZero() {
		t.Fatal("lastInterrupt should be recorded on idle ctrl+c")
	}
	if m.screen != screenConfig {
		t.Fatalf("screen = %d after idle ctrl+c, want screenConfig", m.screen)
	}
	if m.configScreen == nil {
		t.Fatal("configScreen should remain available after idle ctrl+c")
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
	cs := m.configScreen
	cs.dirty = true

	// q で確認表示
	m = sendConfigKey(m, "q")
	cs = m.configScreen
	if !cs.confirmQuit {
		t.Fatal("confirmQuit should be true")
	}

	// Cancel (index 2)
	m = sendConfigKeys(m, "j", "j", "enter")
	cs = m.configScreen
	if cs.confirmQuit {
		t.Fatal("confirmQuit should be false after cancel")
	}
	if m.screen != screenConfig {
		t.Fatalf("screen = %d after cancel, want screenConfig", m.screen)
	}
}

func TestConfigScreen_CloseWithoutDirty(t *testing.T) {
	m := newConfigTestModel()

	// dirty でないので即閉じ
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

func TestConfigScreen_ResetToDefault(t *testing.T) {
	m := newConfigTestModel()
	cs := m.configScreen

	// compression.trigger_percent を変更してからリセット
	for i, cat := range cs.categories {
		if cat.Name == "compression" {
			cs.catIndex = i
			break
		}
	}
	cs.activePane = paneField
	fields := cs.filteredFields()
	for i, f := range fields {
		if f.Path == "compression.trigger_percent" {
			cs.fieldIndex = i
			break
		}
	}

	// まず値を変更
	if err := config.SetFieldValue(cs.cfg, "compression.trigger_percent", 50); err != nil {
		t.Fatalf("SetFieldValue failed: %v", err)
	}
	cs.dirty = true
	cs.refreshCategories()

	// r でデフォルトに戻す
	m = sendConfigKey(m, "r")
	cs = m.configScreen

	val, _ := config.GetFieldValue(cs.cfg, "compression.trigger_percent")
	defCfg := config.DefaultConfig()
	defVal, _ := config.GetFieldValue(defCfg, "compression.trigger_percent")
	if val != defVal {
		t.Fatalf("value after reset = %v, want default %v", val, defVal)
	}
}

func TestConfigScreen_FilterFields(t *testing.T) {
	m := newConfigTestModel()
	cs := m.configScreen

	// execution カテゴリに移動（複数フィールドがある）
	for i, cat := range cs.categories {
		if cat.Name == "execution" {
			cs.catIndex = i
			break
		}
	}

	// / でフィルタモード
	m = sendConfigKey(m, "/")
	cs = m.configScreen
	if !cs.filterMode {
		t.Fatal("filterMode should be true")
	}

	// Esc でキャンセル
	m = sendConfigKey(m, "esc")
	cs = m.configScreen
	if cs.filterMode {
		t.Fatal("filterMode should be false after esc")
	}
}

func TestConfigScreen_ViewRenders(t *testing.T) {
	m := newConfigTestModel()
	m.width = 120
	m.height = 30

	view := m.View()
	if view == "" {
		t.Fatal("View returned empty string")
	}

	// ヘッダーに Configuration が含まれる
	if !strings.Contains(view, "Configuration") {
		t.Fatal("View should contain 'Configuration'")
	}

	// カテゴリ名が含まれる
	if !strings.Contains(view, "Provider") {
		t.Fatal("View should contain 'Provider'")
	}
}

func TestConfigScreen_AllCategoriesPresent(t *testing.T) {
	cfg := config.DefaultConfig()
	categories := config.BuildConfigRegistry(cfg)

	expectedCats := []string{
		"provider", "general", "execution", "compression",
		"paste", "project_map", "lsp", "output",
		"web_search", "sub_agent", "mcp", "hooks",
	}

	catNames := make(map[string]bool)
	for _, cat := range categories {
		catNames[cat.Name] = true
	}

	for _, name := range expectedCats {
		if !catNames[name] {
			t.Errorf("category %q missing from registry", name)
		}
	}
}

func TestConfigScreen_LSPServersInRegistry(t *testing.T) {
	cfg := config.DefaultConfig()
	categories := config.BuildConfigRegistry(cfg)

	found := false
	for _, cat := range categories {
		if cat.Name == "lsp" {
			for _, f := range cat.Fields {
				if f.Path == "lsp.servers" {
					found = true
					if f.FieldType != config.FieldTypeStructMap {
						t.Errorf("lsp.servers type = %v, want FieldTypeStructMap", f.FieldType)
					}
					break
				}
			}
			break
		}
	}
	if !found {
		t.Fatal("lsp.servers not found in generated registry")
	}
}

func TestConfigScreen_EscFromFieldPane(t *testing.T) {
	m := newConfigTestModel()
	cs := m.configScreen

	// フィールドペインへ移動
	cs.activePane = paneField
	cs.fieldIndex = 1

	// Esc でカテゴリペインに戻る
	m = sendConfigKey(m, "esc")
	cs = m.configScreen
	if cs.activePane != paneCategory {
		t.Fatalf("pane after esc from field = %d, want paneCategory", cs.activePane)
	}
	// fieldIndex はリセットされる
	if cs.fieldIndex != 0 {
		t.Fatalf("fieldIndex after esc = %d, want 0", cs.fieldIndex)
	}
}

func TestConfigScreen_FieldScroll_FollowsCursor(t *testing.T) {
	m := newConfigTestModel()
	// 高さを非常に小さくして、field pane が 3 行しか表示できないようにする
	m.height = 5 // header(1) + body(3) + status(1) → field pane 可視行 = 3
	cs := m.configScreen

	// hooks カテゴリ（4フィールド: max_retry, on_completion, on_step_complete, timeout）
	for i, cat := range cs.categories {
		if cat.Name == "hooks" {
			cs.catIndex = i
			break
		}
	}
	cs.activePane = paneField
	cs.fieldIndex = 0
	cs.fieldScroll = 0

	fields := cs.filteredFields()
	if len(fields) < 4 {
		t.Skipf("hooks category has %d fields, need >=4 for scroll test", len(fields))
	}

	// j で3回下移動 → fieldIndex=3 になり画面外へ → fieldScroll が追従するはず
	m = sendConfigKeys(m, "j", "j", "j")
	cs = m.configScreen
	if cs.fieldIndex != 3 {
		t.Fatalf("fieldIndex = %d, want 3", cs.fieldIndex)
	}
	// fieldScroll は fieldIndex が可視範囲内に入るよう調整される
	// 可視行=3, fieldIndex=3 → fieldScroll >= 1
	if cs.fieldScroll < 1 {
		t.Fatalf("fieldScroll = %d, want >= 1 (fieldIndex=3 with 3 visible rows)", cs.fieldScroll)
	}
	if cs.fieldIndex < cs.fieldScroll || cs.fieldIndex >= cs.fieldScroll+3 {
		t.Fatalf("fieldIndex=%d out of visible range [%d, %d)",
			cs.fieldIndex, cs.fieldScroll, cs.fieldScroll+3)
	}

	// k で上に戻ると fieldScroll も戻る
	m = sendConfigKeys(m, "k", "k", "k")
	cs = m.configScreen
	if cs.fieldIndex != 0 {
		t.Fatalf("fieldIndex = %d, want 0", cs.fieldIndex)
	}
	if cs.fieldScroll != 0 {
		t.Fatalf("fieldScroll = %d, want 0", cs.fieldScroll)
	}
}

func TestConfigScreen_FieldScroll_ResetOnCategoryChange(t *testing.T) {
	m := newConfigTestModel()
	m.height = 5
	cs := m.configScreen

	// hooks カテゴリで下にスクロール
	for i, cat := range cs.categories {
		if cat.Name == "hooks" {
			cs.catIndex = i
			break
		}
	}
	cs.activePane = paneField
	m = sendConfigKeys(m, "j", "j", "j")
	cs = m.configScreen
	if cs.fieldScroll == 0 {
		t.Skip("fieldScroll did not advance, not enough fields")
	}

	// Esc でカテゴリペインに戻る → fieldIndex, fieldScroll がリセット
	m = sendConfigKey(m, "esc")
	cs = m.configScreen
	if cs.fieldIndex != 0 {
		t.Fatalf("fieldIndex = %d after Esc, want 0", cs.fieldIndex)
	}
	if cs.fieldScroll != 0 {
		t.Fatalf("fieldScroll = %d after Esc, want 0", cs.fieldScroll)
	}
}

// --- structmap entry value editing tests ---

// enterStructMapEdit は指定の path の structmap 編集モードに入った Model を返す。
func enterStructMapEdit(t *testing.T, path string) Model {
	t.Helper()
	m := newConfigTestModel()
	cs := m.configScreen

	// カテゴリとフィールドを見つける
	parts := strings.SplitN(path, ".", 2)
	catName := parts[0]
	if path == "provider_models" {
		catName = "provider"
	}
	for i, cat := range cs.categories {
		if cat.Name == catName {
			cs.catIndex = i
			break
		}
	}
	cs.activePane = paneField
	fields := cs.filteredFields()
	for i, f := range fields {
		if f.Path == path {
			cs.fieldIndex = i
			break
		}
	}
	m = sendConfigKey(m, "enter")
	cs = m.configScreen
	if cs.editMode != editStructMap {
		t.Fatalf("editMode = %d, want editStructMap", cs.editMode)
	}
	return m
}

func TestConfigScreen_LSPServers_EntryEdit(t *testing.T) {
	m := enterStructMapEdit(t, "lsp.servers")
	cs := m.configScreen

	// "go" キーを見つける
	goIdx := -1
	for i, k := range cs.editStructKeys {
		if k == "go" {
			goIdx = i
			break
		}
	}
	if goIdx < 0 {
		t.Fatal("go key not found in lsp.servers")
	}
	cs.editStructIndex = goIdx

	// Enter で entry 編集
	m = sendConfigKey(m, "enter")
	cs = m.configScreen
	if !cs.editEntryActive {
		t.Fatal("editEntryActive should be true")
	}
	if cs.editEntryKey != "go" {
		t.Fatalf("editEntryKey = %q, want \"go\"", cs.editEntryKey)
	}

	// command フィールドを探す
	for i, ef := range cs.editEntryFields {
		if ef.Name == "command" {
			cs.editEntryIndex = i
			if ef.Type != "string" {
				t.Fatalf("command type = %q, want string", ef.Type)
			}
			break
		}
	}

	// args フィールドを探す
	argsIdx := -1
	for i, ef := range cs.editEntryFields {
		if ef.Name == "args" {
			argsIdx = i
			if ef.Type != "[]string" {
				t.Fatalf("args type = %q, want []string", ef.Type)
			}
			break
		}
	}
	if argsIdx < 0 {
		t.Fatal("args field not found")
	}

	// args フィールドで Enter → slice 編集
	cs.editEntryIndex = argsIdx
	m = sendConfigKey(m, "enter")
	cs = m.configScreen
	if cs.editEntryFieldEdit != "slice" {
		t.Fatalf("editEntryFieldEdit = %q, want \"slice\"", cs.editEntryFieldEdit)
	}

	// Esc で slice 編集を抜ける
	m = sendConfigKey(m, "esc")
	cs = m.configScreen
	if cs.editEntryFieldEdit != "" {
		t.Fatal("editEntryFieldEdit should be empty after esc from slice")
	}
}

func TestConfigScreen_StructMapAdd_ThenEdit(t *testing.T) {
	m := enterStructMapEdit(t, "lsp.servers")
	cs := m.configScreen

	initialLen := len(cs.editStructKeys)

	// a で新規キー追加、"testlang" を入力
	m = sendConfigKey(m, "a")
	cs = m.configScreen
	if !cs.editStructAdding {
		t.Fatal("editStructAdding should be true")
	}

	// テキスト入力（bubbletea のモデルに直接値をセット）
	cs.editStructInput.SetValue("testlang")
	m = sendConfigKey(m, "enter")
	cs = m.configScreen

	if len(cs.editStructKeys) != initialLen+1 {
		t.Fatalf("keys count = %d, want %d", len(cs.editStructKeys), initialLen+1)
	}

	// カーソルが新規キーに合っている
	if cs.editStructKeys[cs.editStructIndex] != "testlang" {
		t.Fatalf("cursor on %q, want \"testlang\"", cs.editStructKeys[cs.editStructIndex])
	}

	// Enter で entry 編集に入れる
	m = sendConfigKey(m, "enter")
	cs = m.configScreen
	if !cs.editEntryActive {
		t.Fatal("editEntryActive should be true for new key")
	}
	if cs.editEntryKey != "testlang" {
		t.Fatalf("editEntryKey = %q, want \"testlang\"", cs.editEntryKey)
	}
}

func TestConfigScreen_StructMapDelete_AfterEdit(t *testing.T) {
	m := enterStructMapEdit(t, "lsp.servers")
	cs := m.configScreen

	if len(cs.editStructKeys) < 2 {
		t.Skip("not enough keys to test delete after edit")
	}

	// Enter で entry 編集に入る
	firstKey := cs.editStructKeys[0]
	m = sendConfigKey(m, "enter")
	// Esc で戻る
	m = sendConfigKey(m, "esc")
	cs = m.configScreen
	if cs.editEntryActive {
		t.Fatal("should be back to key list")
	}

	// キー順が壊れていないこと
	for i := 1; i < len(cs.editStructKeys); i++ {
		if cs.editStructKeys[i] < cs.editStructKeys[i-1] {
			t.Fatalf("keys not sorted after edit+back: %q after %q",
				cs.editStructKeys[i], cs.editStructKeys[i-1])
		}
	}

	// d で削除
	cs.editStructIndex = 0
	m = sendConfigKey(m, "d")
	cs = m.configScreen

	for _, k := range cs.editStructKeys {
		if k == firstKey {
			t.Fatalf("key %q should have been deleted", firstKey)
		}
	}
}

func TestConfigScreen_LSPServers_NilMap_AddEntry_DoesNotPanic(t *testing.T) {
	m := newConfigTestModel()
	cs := m.configScreen
	cs.cfg.LSP.Servers = nil
	cs.refreshCategories()

	setConfigFieldSelection(t, cs, "lsp", "lsp.servers")
	m = sendConfigKey(m, "enter")
	cs = m.configScreen
	if cs.editMode != editStructMap {
		t.Fatalf("editMode = %d, want editStructMap", cs.editMode)
	}

	m = sendConfigKey(m, "a")
	cs = m.configScreen
	if !cs.editStructAdding {
		t.Fatal("editStructAdding should be true")
	}

	cs.editStructInput.SetValue("nil_server")
	m = sendConfigKey(m, "enter")
	cs = m.configScreen

	if cs.cfg.LSP.Servers == nil {
		t.Fatal("LSP.Servers should be initialized after add")
	}
	if _, ok := cs.cfg.LSP.Servers["nil_server"]; !ok {
		t.Fatal("LSP.Servers should contain the added key")
	}
	if !cs.dirty {
		t.Fatal("dirty should be true after add")
	}
	if cs.saveStatus != statusModified {
		t.Fatalf("saveStatus = %d, want statusModified", cs.saveStatus)
	}
}

func TestConfigScreen_StructMap_NonNil_AddEntry_BehaviorUnchanged(t *testing.T) {
	m := enterStructMapEdit(t, "lsp.servers")
	cs := m.configScreen

	initialLen := len(cs.editStructKeys)
	initialDirty := cs.dirty

	m = sendConfigKey(m, "a")
	cs = m.configScreen
	cs.editStructInput.SetValue("behavior_test_lsp")
	m = sendConfigKey(m, "enter")
	cs = m.configScreen

	if len(cs.editStructKeys) != initialLen+1 {
		t.Fatalf("keys count = %d, want %d", len(cs.editStructKeys), initialLen+1)
	}
	if _, ok := cs.cfg.LSP.Servers["behavior_test_lsp"]; !ok {
		t.Fatal("LSP.Servers should contain the added key")
	}
	if cs.dirty == initialDirty {
		t.Fatalf("dirty = %v, want changed from initial state", cs.dirty)
	}
	if cs.saveStatus != statusModified {
		t.Fatalf("saveStatus = %d, want statusModified", cs.saveStatus)
	}

	addedLen := len(cs.editStructKeys)
	m = sendConfigKey(m, "a")
	cs = m.configScreen
	cs.editStructInput.SetValue("behavior_test_lsp")
	m = sendConfigKey(m, "enter")
	cs = m.configScreen

	if len(cs.editStructKeys) != addedLen {
		t.Fatalf("duplicate add changed key count: got %d, want %d", len(cs.editStructKeys), addedLen)
	}
}

func TestConfigScreen_SpaceBoolOnly(t *testing.T) {
	m := newConfigTestModel()
	cs := m.configScreen

	// --- bool フィールド: Space でトグル ---
	for i, cat := range cs.categories {
		if cat.Name == "compression" {
			cs.catIndex = i
			break
		}
	}
	cs.activePane = paneField
	for i, f := range cs.filteredFields() {
		if f.Path == "compression.enabled" {
			cs.fieldIndex = i
			break
		}
	}
	before, _ := config.GetFieldValue(cs.cfg, "compression.enabled")
	m = sendConfigKey(m, " ")
	after, _ := config.GetFieldValue(m.configScreen.cfg, "compression.enabled")
	if before == after {
		t.Fatal("Space should toggle bool")
	}

	// --- select フィールド: Space は no-op ---
	cs = m.configScreen
	for i, cat := range cs.categories {
		if cat.Name == "execution" {
			cs.catIndex = i
			break
		}
	}
	cs.fieldIndex = 0
	cs.fieldScroll = 0
	for i, f := range cs.filteredFields() {
		if f.Path == "execution.mode" {
			cs.fieldIndex = i
			break
		}
	}
	m = sendConfigKey(m, " ")
	cs = m.configScreen
	if cs.editMode != editNone {
		t.Fatalf("Space on select should be no-op, but editMode = %d", cs.editMode)
	}

	// --- string フィールド: Space は no-op ---
	for i, cat := range cs.categories {
		if cat.Name == "provider" {
			cs.catIndex = i
			break
		}
	}
	cs.fieldIndex = 0
	cs.fieldScroll = 0
	for i, f := range cs.filteredFields() {
		if f.Path == "default_model" {
			cs.fieldIndex = i
			break
		}
	}
	m = sendConfigKey(m, " ")
	cs = m.configScreen
	if cs.editMode != editNone {
		t.Fatalf("Space on string should be no-op, but editMode = %d", cs.editMode)
	}

	// --- structmap フィールド: Space は no-op ---
	for i, cat := range cs.categories {
		if cat.Name == "lsp" {
			cs.catIndex = i
			break
		}
	}
	cs.fieldIndex = 0
	cs.fieldScroll = 0
	for i, f := range cs.filteredFields() {
		if f.Path == "lsp.servers" {
			cs.fieldIndex = i
			break
		}
	}
	m = sendConfigKey(m, " ")
	cs = m.configScreen
	if cs.editMode != editNone {
		t.Fatalf("Space on structmap should be no-op, but editMode = %d", cs.editMode)
	}

	// --- Enter は従来通り select 編集開始 ---
	for i, cat := range cs.categories {
		if cat.Name == "execution" {
			cs.catIndex = i
			break
		}
	}
	cs.fieldIndex = 0
	cs.fieldScroll = 0
	for i, f := range cs.filteredFields() {
		if f.Path == "execution.mode" {
			cs.fieldIndex = i
			break
		}
	}
	m = sendConfigKey(m, "enter")
	cs = m.configScreen
	if cs.editMode != editSelect {
		t.Fatalf("Enter on select should start edit, but editMode = %d", cs.editMode)
	}
}

func TestConfigScreen_StructMapEntryEdit_HintTransition(t *testing.T) {
	m := enterStructMapEdit(t, "lsp.servers")
	_ = m.configScreen // state 確認不要、View 経由でヒントを検証

	// key list 中のヒントを確認（View 経由で間接的に）
	m.width = 120
	m.height = 30
	view := m.View()
	if !strings.Contains(view, "Enter:edit entry") {
		t.Fatal("key list hint should contain 'Enter:edit entry'")
	}

	// Enter で entry 編集
	m = sendConfigKey(m, "enter")
	m.width = 120
	m.height = 30
	view = m.View()
	if !strings.Contains(view, "Esc:back") {
		t.Fatal("entry edit hint should contain 'Esc:back'")
	}

	// Esc で戻る
	m = sendConfigKey(m, "esc")
	if m.configScreen.editEntryActive {
		t.Fatal("should be back to key list")
	}
}

func TestConfigScreen_StructMapOrder_LSPServers(t *testing.T) {
	m := newConfigTestModel()
	cs := m.configScreen

	// lsp カテゴリの servers
	for i, cat := range cs.categories {
		if cat.Name == "lsp" {
			cs.catIndex = i
			break
		}
	}
	cs.activePane = paneField
	fields := cs.filteredFields()
	for i, f := range fields {
		if f.Path == "lsp.servers" {
			cs.fieldIndex = i
			break
		}
	}

	// Enter で structmap 編集
	m = sendConfigKey(m, "enter")
	cs = m.configScreen
	if cs.editMode != editStructMap {
		t.Fatalf("editMode = %d, want editStructMap", cs.editMode)
	}

	keys := cs.editStructKeys
	if len(keys) < 5 {
		t.Fatalf("expected many LSP server keys, got %d", len(keys))
	}

	// ソート済みか確認
	for i := 1; i < len(keys); i++ {
		if keys[i] < keys[i-1] {
			t.Fatalf("lsp.servers keys not sorted: %q comes after %q", keys[i], keys[i-1])
		}
	}

	// index と表示が一致するか: index=2 で d → keys[2] が消える
	target := keys[2]
	cs.editStructIndex = 2
	m = sendConfigKey(m, "d")
	cs = m.configScreen

	for _, k := range cs.editStructKeys {
		if k == target {
			t.Fatalf("key %q at index 2 should have been deleted", target)
		}
	}

	// 削除後もソート済み
	for i := 1; i < len(cs.editStructKeys); i++ {
		if cs.editStructKeys[i] < cs.editStructKeys[i-1] {
			t.Fatalf("keys not sorted after delete")
		}
	}
}

// --- 実値変更テスト ---

// enterStructMapEntryForKey は structmap の指定 key の entry 編集に入った Model を返す。
func enterStructMapEntryForKey(t *testing.T, path, key string) Model {
	t.Helper()
	m := enterStructMapEdit(t, path)
	cs := m.configScreen

	// 指定 key にカーソルを合わせる
	for i, k := range cs.editStructKeys {
		if k == key {
			cs.editStructIndex = i
			break
		}
	}
	if cs.editStructKeys[cs.editStructIndex] != key {
		t.Fatalf("key %q not found in editStructKeys", key)
	}

	// Enter で entry 編集に入る
	m = sendConfigKey(m, "enter")
	cs = m.configScreen
	if !cs.editEntryActive {
		t.Fatalf("editEntryActive should be true for key %q", key)
	}
	return m
}

// setEntryFieldIndex は entry 内の指定 field にカーソルを合わせる。
func setEntryFieldIndex(t *testing.T, cs *configScreen, name string) int {
	t.Helper()
	for i, ef := range cs.editEntryFields {
		if ef.Name == name {
			cs.editEntryIndex = i
			return i
		}
	}
	t.Fatalf("entry field %q not found", name)
	return -1
}

func makeConfigScreenDirty(t *testing.T, m Model) Model {
	t.Helper()

	cs := m.configScreen
	setConfigFieldSelection(t, cs, "compression", "compression.enabled")
	m = sendConfigKey(m, " ")
	if !m.configScreen.dirty {
		t.Fatal("dirty should be true after toggle")
	}
	return m
}

func TestConfigScreen_StructEntryInt_InvalidInput_DoesNotCloseOrDirty(t *testing.T) {
	m := enterStructMapEntryForKey(t, "provider_models", "openai")
	cs := m.configScreen

	setEntryFieldIndex(t, cs, "max_output_tokens")
	original := cs.cfg.ProviderModels["openai"].MaxOutputTokens

	m = sendConfigKey(m, "enter")
	cs = m.configScreen
	if cs.editEntryFieldEdit != "input" {
		t.Fatalf("editEntryFieldEdit = %q, want \"input\"", cs.editEntryFieldEdit)
	}

	cs.editInput.SetValue("not-a-number")
	m = sendConfigKey(m, "enter")
	cs = m.configScreen

	if cs.editEntryFieldEdit != "input" {
		t.Fatalf("editEntryFieldEdit after invalid input = %q, want \"input\"", cs.editEntryFieldEdit)
	}
	if cs.dirty {
		t.Fatal("dirty should remain false after invalid int input")
	}
	if cs.saveStatus != statusSaved {
		t.Fatalf("saveStatus = %d, want statusSaved(%d)", cs.saveStatus, statusSaved)
	}
	if got := cs.cfg.ProviderModels["openai"].MaxOutputTokens; got != original {
		t.Fatalf("ProviderModels[openai].MaxOutputTokens = %d, want %d", got, original)
	}
}

func TestConfigScreen_StructEntryInt_ValidInput_StillApplies(t *testing.T) {
	m := enterStructMapEntryForKey(t, "provider_models", "openai")
	cs := m.configScreen

	setEntryFieldIndex(t, cs, "max_output_tokens")

	m = sendConfigKey(m, "enter")
	cs = m.configScreen
	if cs.editEntryFieldEdit != "input" {
		t.Fatalf("editEntryFieldEdit = %q, want \"input\"", cs.editEntryFieldEdit)
	}

	cs.editInput.SetValue("4321")
	m = sendConfigKey(m, "enter")
	cs = m.configScreen

	if cs.editEntryFieldEdit != "" {
		t.Fatalf("editEntryFieldEdit after valid input = %q, want empty", cs.editEntryFieldEdit)
	}
	if got := cs.cfg.ProviderModels["openai"].MaxOutputTokens; got != 4321 {
		t.Fatalf("ProviderModels[openai].MaxOutputTokens = %d, want 4321", got)
	}
	if !cs.dirty {
		t.Fatal("dirty should be true after valid int input")
	}
	if cs.saveStatus != statusModified {
		t.Fatalf("saveStatus = %d, want statusModified(%d)", cs.saveStatus, statusModified)
	}
}

func TestConfigScreen_LSPServers_CommandChange_Persists(t *testing.T) {
	m := enterStructMapEntryForKey(t, "lsp.servers", "go")
	cs := m.configScreen

	// command フィールドにカーソル
	setEntryFieldIndex(t, cs, "command")

	// Enter → input 編集
	m = sendConfigKey(m, "enter")
	cs = m.configScreen
	if cs.editEntryFieldEdit != "input" {
		t.Fatalf("editEntryFieldEdit = %q, want \"input\"", cs.editEntryFieldEdit)
	}

	// 値を変更
	cs.editInput.SetValue("custom-gopls")

	// Enter で確定
	m = sendConfigKey(m, "enter")
	cs = m.configScreen

	// Config 実値を検証
	srv, ok := cs.cfg.LSP.Servers["go"]
	if !ok {
		t.Fatal("go not found in LSP.Servers")
	}
	if srv.Command != "custom-gopls" {
		t.Fatalf("LSP.Servers[go].Command = %q, want \"custom-gopls\"", srv.Command)
	}
}

func TestConfigScreen_LSPServers_ArgsChange_Persists(t *testing.T) {
	m := enterStructMapEntryForKey(t, "lsp.servers", "go")
	cs := m.configScreen

	// args にカーソル
	setEntryFieldIndex(t, cs, "args")

	// 元の args を記録
	origArgs := cs.cfg.LSP.Servers["go"].Args
	origLen := len(origArgs)

	// Enter → slice 編集
	m = sendConfigKey(m, "enter")
	cs = m.configScreen
	if cs.editEntryFieldEdit != "slice" {
		t.Fatalf("editEntryFieldEdit = %q, want \"slice\"", cs.editEntryFieldEdit)
	}

	// a で追加
	m = sendConfigKey(m, "a")
	cs = m.configScreen
	cs.editSliceInput.SetValue("--extra-flag")
	m = sendConfigKey(m, "enter") // 追加確定

	// Esc で slice editor を抜ける → entry field に args が書き戻される
	m = sendConfigKey(m, "esc")
	cs = m.configScreen

	// Config 実値を検証
	srv := cs.cfg.LSP.Servers["go"]
	if len(srv.Args) != origLen+1 {
		t.Fatalf("LSP.Servers[go].Args length = %d, want %d", len(srv.Args), origLen+1)
	}
	found := false
	for _, a := range srv.Args {
		if a == "--extra-flag" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("--extra-flag not found in LSP.Servers[go].Args: %v", srv.Args)
	}
}

func TestConfigScreen_LSPServers_DisabledToggle_Persists(t *testing.T) {
	m := enterStructMapEntryForKey(t, "lsp.servers", "go")
	cs := m.configScreen

	// disabled の初期値を記録
	before := cs.cfg.LSP.Servers["go"].Disabled

	// disabled フィールドにカーソル
	setEntryFieldIndex(t, cs, "disabled")

	// Space でトグル
	m = sendConfigKey(m, " ")
	cs = m.configScreen

	// Config 実値を検証
	after := cs.cfg.LSP.Servers["go"].Disabled
	if after == before {
		t.Fatalf("Disabled should have toggled: before=%v, after=%v", before, after)
	}

	// Enter でもトグルできること
	m = sendConfigKey(m, "enter")
	cs = m.configScreen
	after2 := cs.cfg.LSP.Servers["go"].Disabled
	if after2 != before {
		t.Fatalf("Disabled should have toggled back: expected=%v, got=%v", before, after2)
	}
}

func TestConfigScreen_SaveSnapshot_Isolation(t *testing.T) {
	agent := &stubAgent{}
	m := newModelWithViewport(agent)
	cfg := config.DefaultConfig()
	m.screen = screenConfig
	m.configScreen = newConfigScreen(cfg)
	cs := m.configScreen

	// default_model を変更
	cs.cfg.DefaultModel = "gpt-5.4"
	cs.dirty = true

	// s → saveCmd 実行 → ConfigSavedMsg
	sMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("s")}
	updated, saveCmd := m.Update(sMsg)
	m = updated.(Model)
	if saveCmd == nil {
		t.Fatal("saveCmd should not be nil")
	}
	resultMsg := saveCmd()
	updated, _ = m.Update(resultMsg)
	m = updated.(Model)

	// 保存時点の値を確認
	agent.mu.RLock()
	saved := agent.lastSavedConfig
	agent.mu.RUnlock()
	if saved == nil {
		t.Fatal("lastSavedConfig is nil")
	}
	if saved.DefaultModel != "gpt-5.4" {
		t.Fatalf("saved.DefaultModel = %q, want \"gpt-5.4\"", saved.DefaultModel)
	}

	// 元の cfg をさらに変更
	m.configScreen.cfg.DefaultModel = "gpt-5.4-mini"

	// lastSavedConfig は保存時点の値のまま変わっていないこと
	agent.mu.RLock()
	savedAfter := agent.lastSavedConfig
	agent.mu.RUnlock()
	if savedAfter.DefaultModel != "gpt-5.4" {
		t.Fatalf("savedAfter.DefaultModel = %q, want \"gpt-5.4\" (snapshot should be isolated)", savedAfter.DefaultModel)
	}

	// map 内部も独立していること — ProviderModels の変更が汚染しないか確認
	if pm, ok := m.configScreen.cfg.ProviderModels["openai"]; ok {
		pm.DefaultModel = "mutated"
		m.configScreen.cfg.ProviderModels["openai"] = pm
	}
	agent.mu.RLock()
	savedPM := agent.lastSavedConfig.ProviderModels["openai"]
	agent.mu.RUnlock()
	if savedPM.DefaultModel == "mutated" {
		t.Fatal("saved ProviderModels[openai].DefaultModel was mutated — snapshot is not isolated")
	}
}

func TestConfigScreen_ConfigAlias_OpensTUIScreen(t *testing.T) {
	agent := &stubAgent{}
	m := newModelWithViewport(agent)

	// stubAgent.ResolveAlias は "/c" → "/config" を返す

	// "/c" で config screen に入る
	m.textInput.SetValue("/c")
	m = sendConfigKey(m, "enter")
	if m.screen != screenConfig {
		t.Fatalf("screen = %d after /c, want screenConfig(%d)", m.screen, screenConfig)
	}

	// 閉じる
	m = sendConfigKey(m, "q")

	// "/c show" は config screen には入らず command handling に流れる
	m.textInput.SetValue("/c show")
	m = sendConfigKey(m, "enter")
	// HandleCommand returns false (stubAgent), so it falls through to sendChat
	// 重要なのは screenConfig に入らないこと
	if m.screen == screenConfig {
		t.Fatal("/c show should not open config screen")
	}
}

func TestConfigScreen_StructMap_DuplicateKey_NoUIAppend(t *testing.T) {
	m := enterStructMapEdit(t, "lsp.servers")
	cs := m.configScreen

	if len(cs.editStructKeys) == 0 {
		t.Fatal("no keys")
	}
	existingKey := cs.editStructKeys[0]
	initialLen := len(cs.editStructKeys)
	initialDirty := cs.dirty

	// a で追加モード → 既存 key を入力
	m = sendConfigKey(m, "a")
	cs = m.configScreen
	cs.editStructInput.SetValue(existingKey)
	m = sendConfigKey(m, "enter")
	cs = m.configScreen

	// UI キー一覧が増えていないこと
	if len(cs.editStructKeys) != initialLen {
		t.Fatalf("editStructKeys length = %d after duplicate add, want %d", len(cs.editStructKeys), initialLen)
	}

	// dirty が変わっていないこと
	if cs.dirty != initialDirty {
		t.Fatalf("dirty = %v after duplicate add, want %v", cs.dirty, initialDirty)
	}

	// 重複がないこと
	seen := make(map[string]bool)
	for _, k := range cs.editStructKeys {
		if seen[k] {
			t.Fatalf("duplicate key %q in editStructKeys", k)
		}
		seen[k] = true
	}
}

func TestConfigScreen_SaveCmd_SnapshotIsolation(t *testing.T) {
	// saveConfigCmd が cfg の snapshot を渡すため、
	// save 開始後に元 cfg を変更しても保存対象が変わらないことを検証する。
	agent := &stubAgent{}
	m := newModelWithViewport(agent)
	cfg := config.DefaultConfig()
	m.screen = screenConfig
	m.configScreen = newConfigScreen(cfg)
	m.configScreen.cfg.DefaultModel = "at-save-time"
	m.configScreen.dirty = true

	// s で save → Cmd を取得（まだ実行しない）
	sMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("s")}
	updated, saveCmd := m.Update(sMsg)
	m = updated.(Model)
	if saveCmd == nil {
		t.Fatal("saveCmd is nil")
	}

	// save 開始後に元 cfg を変更
	m.configScreen.cfg.DefaultModel = "mutated-after-save"

	// Cmd を実行 — snapshot を保存するので "at-save-time" が渡るはず
	resultMsg := saveCmd()
	updated, _ = m.Update(resultMsg)
	m = updated.(Model)

	agent.mu.RLock()
	saved := agent.lastSavedConfig
	agent.mu.RUnlock()
	if saved == nil {
		t.Fatal("lastSavedConfig is nil")
	}
	if saved.DefaultModel != "at-save-time" {
		t.Fatalf("saved.DefaultModel = %q, want \"at-save-time\" (snapshot should be isolated from post-save mutation)", saved.DefaultModel)
	}
}

// --- regression tests: save/quit semantics & Ctrl+C cancel semantics ---

func TestConfigScreen_DiscardAndQuit_DoesNotApplyInflightSave(t *testing.T) {
	// save 開始後、完了前に "Discard and quit" を選んでも、
	// その save snapshot が後から適用されないことを確認する。
	agent := &stubAgent{}
	agent.saveStatusLine = "new-status-from-save"
	m := newModelWithViewport(agent)
	cfg := config.DefaultConfig()
	m.screen = screenConfig
	m.configScreen = newConfigScreen(cfg)
	cs := m.configScreen

	// dirty にして save 開始
	cs.dirty = true
	cs.saveStatus = statusModified

	// s で save 開始
	sMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("s")}
	updated, saveCmd := m.Update(sMsg)
	m = updated.(Model)
	cs = m.configScreen

	if cs.saveStatus != statusSaving {
		t.Fatalf("saveStatus = %d, want statusSaving(%d)", cs.saveStatus, statusSaving)
	}
	if saveCmd == nil {
		t.Fatal("saveCmd should not be nil")
	}

	// save 完了前に close → confirmQuit ダイアログ表示
	m = sendConfigKey(m, "q")
	cs = m.configScreen
	if !cs.confirmQuit {
		t.Fatal("confirmQuit should be true")
	}

	// "Discard and quit" (confirmIdx=1) を選択
	cs.confirmIdx = 1
	m = sendConfigKey(m, "enter")
	cs = m.configScreen

	// save in-flight 中は discard が拒否される → config screen がまだ開いている
	if m.screen != screenConfig {
		t.Fatal("discard should be blocked while save is in-flight; screen should remain screenConfig")
	}
	if cs == nil {
		t.Fatal("configScreen should not be nil")
	}
	if !cs.confirmQuit {
		t.Fatal("confirmQuit should remain true since discard was blocked")
	}

	// save 完了後に ConfigSavedMsg を送る → 正常に保存成功として扱われる
	resultMsg := saveCmd()
	updated, _ = m.Update(resultMsg)
	m = updated.(Model)

	// save は成功して screen に反映される（discard は未実行なので通常の save 成功経路）
	cs = m.configScreen
	if cs == nil {
		t.Fatal("configScreen should not be nil after save completes")
	}
	if cs.saveStatus != statusSaved {
		t.Fatalf("saveStatus = %d, want statusSaved(%d)", cs.saveStatus, statusSaved)
	}
}

func TestConfigScreen_SaveAndQuit_StillAppliesInflightSave(t *testing.T) {
	// save-and-quit 経路は従来どおり成立し、discard と意味論が混ざらないことを確認する。
	agent := &stubAgent{}
	agent.saveStatusLine = "save-and-quit-status"
	m := newModelWithViewport(agent)
	cfg := config.DefaultConfig()
	m.screen = screenConfig
	m.configScreen = newConfigScreen(cfg)
	cs := m.configScreen

	// dirty にする
	cs.dirty = true
	cs.saveStatus = statusModified

	// q → confirmQuit
	m = sendConfigKey(m, "q")
	cs = m.configScreen
	if !cs.confirmQuit {
		t.Fatal("confirmQuit should be true")
	}

	// "Save and quit" (confirmIdx=0) → 非同期保存開始
	cs.confirmIdx = 0
	updated, saveCmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(Model)
	cs = m.configScreen

	if cs == nil {
		t.Fatal("configScreen should not be nil yet")
	}
	if !cs.pendingClose {
		t.Fatal("pendingClose should be true")
	}
	if saveCmd == nil {
		t.Fatal("saveCmd should not be nil for save-and-quit")
	}

	// ConfigSavedMsg で保存完了 → 画面が閉じる
	resultMsg := saveCmd()
	updated, _ = m.Update(resultMsg)
	m = updated.(Model)

	if m.screen != screenChat {
		t.Fatalf("screen = %d, want screenChat after save-and-quit", m.screen)
	}
	if m.configScreen != nil {
		t.Fatal("configScreen should be nil after save-and-quit")
	}

	// agent の statusLine が更新されていること
	agent.mu.RLock()
	sl := agent.statusLine
	agent.mu.RUnlock()
	if sl != "save-and-quit-status" {
		t.Fatalf("statusLine = %q, want %q", sl, "save-and-quit-status")
	}
}

func TestConfigScreen_ConfirmQuit_CtrlC_StillCancelsProcessing(t *testing.T) {
	// dirty=true, confirmQuit=true, processing=true の状態で Ctrl+C →
	// cancel が呼ばれ、confirmQuit state が不正に壊れないことを確認する。
	agent := &stubAgent{}
	agent.setProcessing(true)
	m := newModelWithViewport(agent)
	cfg := config.DefaultConfig()
	m.screen = screenConfig
	m.configScreen = newConfigScreen(cfg)
	cs := m.configScreen

	cs.dirty = true
	cs.saveStatus = statusModified
	cs.confirmQuit = true
	cs.confirmIdx = 1

	// Ctrl+C を送る
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

	// config screen はまだ開いている
	if m.screen != screenConfig {
		t.Fatalf("screen = %d, want screenConfig", m.screen)
	}
	if m.configScreen == nil {
		t.Fatal("configScreen should remain available")
	}
}

func TestConfigScreen_ConfirmQuit_CtrlC_WhenIdle_DoesNotBreakQuitDialog(t *testing.T) {
	// dirty=true, confirmQuit=true, processing=false で Ctrl+C →
	// panic や不正 state にならないことを確認する。
	agent := &stubAgent{}
	m := newModelWithViewport(agent)
	cfg := config.DefaultConfig()
	m.screen = screenConfig
	m.configScreen = newConfigScreen(cfg)
	cs := m.configScreen

	cs.dirty = true
	cs.saveStatus = statusModified
	cs.confirmQuit = true
	cs.confirmIdx = 0

	// Ctrl+C を送る（idle 時は quit dialog に吸われてよい）
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

	// config screen はまだ開いている（quit dialog が吸うが panic しない）
	if m.screen != screenConfig {
		t.Fatalf("screen = %d, want screenConfig", m.screen)
	}
	if m.configScreen == nil {
		t.Fatal("configScreen should not be nil")
	}
}

func TestConfigScreen_MessagesNotLostDuringConfigScreen(t *testing.T) {
	// config screen 表示中でも StreamTextMsg / AppendMessageMsg / AgentDoneMsg が
	// chat 側の状態に反映されること。
	m := newConfigTestModel()

	// config screen 表示中に AppendMessageMsg を送る
	msgsBefore := len(m.messages)
	updated, _ := m.Update(AppendMessageMsg{
		Message: ChatMessage{Role: "assistant", Content: "background output"},
	})
	m = updated.(Model)

	// messages が増えていること（chat 側に蓄積される）
	if len(m.messages) <= msgsBefore {
		t.Fatal("AppendMessageMsg should be buffered during config screen")
	}

	// screen は config のまま
	if m.screen != screenConfig {
		t.Fatalf("screen = %d, want screenConfig", m.screen)
	}

	// AgentDoneMsg も通ること
	updated, _ = m.Update(AgentDoneMsg{})
	m = updated.(Model)
	if m.screen != screenConfig {
		t.Fatalf("screen = %d after AgentDoneMsg, want screenConfig", m.screen)
	}
}
