package tui

import (
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/susugadx/xelyon-cli/internal/config"
)

func (m Model) startFieldEdit(field *config.ConfigField) (tea.Model, tea.Cmd) {
	cs := m.configScreen
	targetPane := m.configEditTargetPane()

	switch field.FieldType {
	case config.FieldTypeBool:
		// Enter でも bool をトグル
		current, _ := field.Current.(bool)
		if err := config.SetFieldValue(cs.cfg, field.Path, !current); err == nil {
			cs.dirty = true
			cs.saveStatus = statusModified
			cs.refreshCategories()
		}
		return m, nil

	case config.FieldTypeSelect:
		cs.editMode = editSelect
		cs.editSelect = 0
		current, _ := field.Current.(string)
		for i, opt := range field.Options {
			if opt == current {
				cs.editSelect = i
				break
			}
		}
		cs.activePane = targetPane
		return m, nil

	case config.FieldTypeString:
		cs.editMode = editInput
		current, _ := field.Current.(string)
		cs.editInput.SetValue(current)
		cs.editInput.Focus()
		cs.editInput.CursorEnd()
		cs.activePane = targetPane
		return m, nil

	case config.FieldTypeInt:
		cs.editMode = editInput
		current := 0
		switch v := field.Current.(type) {
		case int:
			current = v
		case int64:
			current = int(v)
		case float64:
			current = int(v)
		}
		cs.editInput.SetValue(strconv.Itoa(current))
		cs.editInput.Focus()
		cs.editInput.CursorEnd()
		cs.activePane = targetPane
		return m, nil

	case config.FieldTypeFloat:
		cs.editMode = editInput
		current := 0.0
		switch v := field.Current.(type) {
		case float64:
			current = v
		case float32:
			current = float64(v)
		case int:
			current = float64(v)
		}
		cs.editInput.SetValue(fmt.Sprintf("%g", current))
		cs.editInput.Focus()
		cs.editInput.CursorEnd()
		cs.activePane = targetPane
		return m, nil

	case config.FieldTypeStringSlice:
		cs.editMode = editSlice
		if slice, ok := field.Current.([]string); ok {
			cs.editSliceItems = make([]string, len(slice))
			copy(cs.editSliceItems, slice)
		} else {
			cs.editSliceItems = nil
		}
		cs.editSliceIndex = 0
		cs.editSliceAdding = false
		cs.editSliceEditing = false
		cs.activePane = targetPane
		return m, nil

	case config.FieldTypeStructMap:
		cs.editMode = editStructMap
		cs.editStructKeys = nil
		val, _ := config.GetFieldValue(cs.cfg, field.Path)
		switch v := val.(type) {
		case map[string]config.ProviderModelConfig:
			for k := range v {
				cs.editStructKeys = append(cs.editStructKeys, k)
			}
		case map[string]config.LSPServerConfig:
			for k := range v {
				cs.editStructKeys = append(cs.editStructKeys, k)
			}
		}
		sort.Strings(cs.editStructKeys)
		cs.editStructIndex = 0
		cs.editStructAdding = false
		cs.activePane = targetPane
		return m, nil
	}

	return m, nil
}

// handleConfigEditKey は編集モード中のキー処理。
func (m Model) handleConfigEditKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	cs := m.configScreen

	switch cs.editMode {
	case editSelect:
		return m.handleSelectEdit(msg)
	case editInput:
		return m.handleInputEdit(msg)
	case editSlice:
		return m.handleSliceEdit(msg)
	case editStructMap:
		return m.handleStructMapEdit(msg)
	}
	return m, nil
}

// handleConfigCtrlC は config screen 上の Ctrl+C を処理する。
// processing 中の Ctrl+C は confirmQuit 表示中でも request cancel を優先する。
func (m Model) handleConfigCtrlC() (tea.Model, tea.Cmd) {
	cs := m.configScreen
	if cs == nil {
		return m.handleCtrlC()
	}
	// processing 中は confirmQuit の有無に関係なく request cancel を優先する
	if m.agent.IsProcessing() {
		return m.handleCtrlC()
	}
	if cs.confirmQuit {
		return m, nil
	}
	if cs.dirty {
		return m.tryCloseConfig()
	}
	return m.handleCtrlC()
}

// handleSelectEdit は select 型の編集キー処理。
func (m Model) handleSelectEdit(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	cs := m.configScreen
	field := cs.selectedField()
	if field == nil {
		cs.editMode = editNone
		return m, nil
	}

	s := msg.String()
	switch {
	case msg.Type == tea.KeyEsc:
		cs.editMode = editNone
		return m, nil

	case msg.Type == tea.KeyUp || s == "k":
		if cs.editSelect > 0 {
			cs.editSelect--
		}
		return m, nil

	case msg.Type == tea.KeyDown || s == "j":
		if cs.editSelect < len(field.Options)-1 {
			cs.editSelect++
		}
		return m, nil

	case isEnterKey(msg):
		if cs.editSelect >= 0 && cs.editSelect < len(field.Options) {
			newVal := field.Options[cs.editSelect]
			if err := config.SetFieldValue(cs.cfg, field.Path, newVal); err == nil {
				cs.dirty = true
				cs.saveStatus = statusModified
				cs.refreshCategories()
			}
		}
		cs.editMode = editNone
		return m, nil
	}
	return m, nil
}

// handleInputEdit は string/int/float 型の入力キー処理。
func (m Model) handleInputEdit(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	cs := m.configScreen
	field := cs.selectedField()
	if field == nil {
		cs.editMode = editNone
		return m, nil
	}

	switch {
	case msg.Type == tea.KeyEsc:
		cs.editMode = editNone
		cs.editInput.Blur()
		return m, nil

	case isEnterKey(msg):
		raw := cs.editInput.Value()
		var newVal interface{}
		var err error

		switch field.FieldType {
		case config.FieldTypeString:
			newVal = raw
		case config.FieldTypeInt:
			v, e := strconv.Atoi(raw)
			if e != nil {
				return m, nil // 不正入力は無視
			}
			newVal = v
		case config.FieldTypeFloat:
			v, e := strconv.ParseFloat(raw, 64)
			if e != nil || math.IsNaN(v) || math.IsInf(v, 0) {
				return m, nil
			}
			if field.Path == "project_map.context_ratio" &&
				(v < config.ProjectMapContextRatioMin || v > config.ProjectMapContextRatioMax) {
				return m, nil
			}
			newVal = v
		}

		if newVal != nil {
			err = config.SetFieldValue(cs.cfg, field.Path, newVal)
			if err == nil {
				// default_model 変更時は編集中 config の default provider override も同期する。
				if field.Path == "default_model" {
					m.syncEditedProviderDefaultModel()
				}
				cs.dirty = true
				cs.saveStatus = statusModified
				cs.refreshCategories()
			}
		}
		cs.editMode = editNone
		cs.editInput.Blur()
		return m, nil

	default:
		var cmd tea.Cmd
		cs.editInput, cmd = cs.editInput.Update(msg)
		return m, cmd
	}
}

// handleSliceEdit は []string 型のサブビューキー処理。
func (m Model) handleSliceEdit(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	cs := m.configScreen

	// 追加入力中
	if cs.editSliceAdding || cs.editSliceEditing {
		return m.handleSliceInputKey(msg)
	}

	field := cs.selectedField()
	if field == nil {
		cs.editMode = editNone
		return m, nil
	}

	s := msg.String()
	switch {
	case msg.Type == tea.KeyEsc:
		// []string の変更を確定して戻る
		if err := config.SetFieldValue(cs.cfg, field.Path, cs.editSliceItems); err == nil {
			if !sliceEqual(cs.editSliceItems, field.Current) {
				cs.dirty = true
				cs.saveStatus = statusModified
				cs.refreshCategories()
			}
		}
		cs.editMode = editNone
		return m, nil

	case msg.Type == tea.KeyUp || s == "k":
		if cs.editSliceIndex > 0 {
			cs.editSliceIndex--
		}
		return m, nil

	case msg.Type == tea.KeyDown || s == "j":
		if cs.editSliceIndex < len(cs.editSliceItems)-1 {
			cs.editSliceIndex++
		}
		return m, nil

	case s == "a":
		cs.editSliceAdding = true
		cs.editSliceInput.SetValue("")
		cs.editSliceInput.Focus()
		return m, nil

	case s == "d":
		if cs.editSliceIndex >= 0 && cs.editSliceIndex < len(cs.editSliceItems) {
			cs.editSliceItems = append(cs.editSliceItems[:cs.editSliceIndex], cs.editSliceItems[cs.editSliceIndex+1:]...)
			if cs.editSliceIndex >= len(cs.editSliceItems) && cs.editSliceIndex > 0 {
				cs.editSliceIndex--
			}
		}
		return m, nil

	case isEnterKey(msg):
		if cs.editSliceIndex >= 0 && cs.editSliceIndex < len(cs.editSliceItems) {
			cs.editSliceEditing = true
			cs.editSliceInput.SetValue(cs.editSliceItems[cs.editSliceIndex])
			cs.editSliceInput.Focus()
			cs.editSliceInput.CursorEnd()
		}
		return m, nil
	}
	return m, nil
}

// handleSliceInputKey は []string の入力モードキー処理。
func (m Model) handleSliceInputKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	cs := m.configScreen
	switch {
	case msg.Type == tea.KeyEsc:
		cs.editSliceAdding = false
		cs.editSliceEditing = false
		cs.editSliceInput.Blur()
		return m, nil

	case isEnterKey(msg):
		val := strings.TrimSpace(cs.editSliceInput.Value())
		if val != "" {
			if cs.editSliceAdding {
				cs.editSliceItems = append(cs.editSliceItems, val)
				cs.editSliceIndex = len(cs.editSliceItems) - 1
			} else if cs.editSliceEditing && cs.editSliceIndex < len(cs.editSliceItems) {
				cs.editSliceItems[cs.editSliceIndex] = val
			}
		}
		cs.editSliceAdding = false
		cs.editSliceEditing = false
		cs.editSliceInput.Blur()
		return m, nil

	default:
		var cmd tea.Cmd
		cs.editSliceInput, cmd = cs.editSliceInput.Update(msg)
		return m, cmd
	}
}

// handleStructMapEdit は structmap 型のサブビューキー処理。
func (m Model) handleStructMapEdit(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	cs := m.configScreen

	// entry value 編集中
	if cs.editEntryActive {
		return m.handleStructEntryEdit(msg)
	}

	// 追加入力中
	if cs.editStructAdding {
		return m.handleStructMapAddKey(msg)
	}

	s := msg.String()
	switch {
	case msg.Type == tea.KeyEsc:
		cs.editMode = editNone
		return m, nil

	case msg.Type == tea.KeyUp || s == "k":
		if cs.editStructIndex > 0 {
			cs.editStructIndex--
		}
		return m, nil

	case msg.Type == tea.KeyDown || s == "j":
		if cs.editStructIndex < len(cs.editStructKeys)-1 {
			cs.editStructIndex++
		}
		return m, nil

	case s == "d":
		// structmap のキー削除
		if cs.editStructIndex >= 0 && cs.editStructIndex < len(cs.editStructKeys) {
			field := cs.selectedField()
			if field != nil {
				key := cs.editStructKeys[cs.editStructIndex]
				m.deleteStructMapKey(field.Path, key)
				cs.editStructKeys = append(cs.editStructKeys[:cs.editStructIndex], cs.editStructKeys[cs.editStructIndex+1:]...)
				if cs.editStructIndex >= len(cs.editStructKeys) && cs.editStructIndex > 0 {
					cs.editStructIndex--
				}
				cs.dirty = true
				cs.saveStatus = statusModified
				cs.refreshCategories()
			}
		}
		return m, nil

	case s == "a":
		cs.editStructAdding = true
		cs.editStructInput.SetValue("")
		cs.editStructInput.Focus()
		return m, nil

	case isEnterKey(msg):
		// 選択中 key の entry 編集に入る
		if cs.editStructIndex >= 0 && cs.editStructIndex < len(cs.editStructKeys) {
			field := cs.selectedField()
			if field != nil {
				key := cs.editStructKeys[cs.editStructIndex]
				fields := cs.loadEntryFields(field.Path, key)
				if len(fields) > 0 {
					cs.editEntryActive = true
					cs.editEntryKey = key
					cs.editEntryFields = fields
					cs.editEntryIndex = 0
					cs.editEntryFieldEdit = ""
				}
			}
		}
		return m, nil
	}
	return m, nil
}

// handleStructEntryEdit は structmap entry の value 編集キー処理。
func (m Model) handleStructEntryEdit(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	cs := m.configScreen

	// 個別フィールド編集中
	if cs.editEntryFieldEdit != "" {
		return m.handleStructEntryFieldEdit(msg)
	}

	s := msg.String()
	switch {
	case msg.Type == tea.KeyEsc:
		// entry 編集を抜けて key list に戻る
		cs.editEntryActive = false
		cs.editEntryFields = nil
		return m, nil

	case msg.Type == tea.KeyUp || s == "k":
		if cs.editEntryIndex > 0 {
			cs.editEntryIndex--
		}
		return m, nil

	case msg.Type == tea.KeyDown || s == "j":
		if cs.editEntryIndex < len(cs.editEntryFields)-1 {
			cs.editEntryIndex++
		}
		return m, nil

	case s == " ":
		// bool フィールドの即トグル
		if cs.editEntryIndex < len(cs.editEntryFields) {
			ef := &cs.editEntryFields[cs.editEntryIndex]
			if ef.Type == "bool" {
				cur, _ := ef.Value.(bool)
				ef.Value = !cur
				m.applyEntryFieldAndMark(ef)
			}
		}
		return m, nil

	case isEnterKey(msg):
		if cs.editEntryIndex >= 0 && cs.editEntryIndex < len(cs.editEntryFields) {
			ef := &cs.editEntryFields[cs.editEntryIndex]
			switch ef.Type {
			case "bool":
				cur, _ := ef.Value.(bool)
				ef.Value = !cur
				m.applyEntryFieldAndMark(ef)
			case "string":
				cs.editEntryFieldEdit = "input"
				val, _ := ef.Value.(string)
				cs.editInput.SetValue(val)
				cs.editInput.Focus()
				cs.editInput.CursorEnd()
			case "int":
				cs.editEntryFieldEdit = "input"
				v := 0
				switch n := ef.Value.(type) {
				case int:
					v = n
				}
				cs.editInput.SetValue(strconv.Itoa(v))
				cs.editInput.Focus()
				cs.editInput.CursorEnd()
			case "[]string":
				cs.editEntryFieldEdit = "slice"
				if s, ok := ef.Value.([]string); ok {
					cs.editSliceItems = make([]string, len(s))
					copy(cs.editSliceItems, s)
				} else {
					cs.editSliceItems = nil
				}
				cs.editSliceIndex = 0
				cs.editSliceAdding = false
				cs.editSliceEditing = false
			}
		}
		return m, nil
	}
	return m, nil
}

// handleStructEntryFieldEdit は entry 内の個別フィールド編集キー処理。
func (m Model) handleStructEntryFieldEdit(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	cs := m.configScreen

	switch cs.editEntryFieldEdit {
	case "input":
		return m.handleEntryInputEdit(msg)
	case "slice":
		return m.handleEntrySliceEdit(msg)
	}
	return m, nil
}

// handleEntryInputEdit は entry 内の string/int フィールド入力処理。
func (m Model) handleEntryInputEdit(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	cs := m.configScreen
	switch {
	case msg.Type == tea.KeyEsc:
		cs.editEntryFieldEdit = ""
		cs.editInput.Blur()
		return m, nil

	case isEnterKey(msg):
		ef := &cs.editEntryFields[cs.editEntryIndex]
		raw := cs.editInput.Value()
		switch ef.Type {
		case "string":
			ef.Value = raw
		case "int":
			v, err := strconv.Atoi(raw)
			if err != nil {
				return m, nil
			}
			ef.Value = v
		}
		m.applyEntryFieldAndMark(ef)
		cs.editEntryFieldEdit = ""
		cs.editInput.Blur()
		return m, nil

	default:
		var cmd tea.Cmd
		cs.editInput, cmd = cs.editInput.Update(msg)
		return m, cmd
	}
}

// handleEntrySliceEdit は entry 内の []string フィールド編集処理。
// 既存の slice editor state を再利用する。
func (m Model) handleEntrySliceEdit(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	cs := m.configScreen

	// 追加/編集入力中
	if cs.editSliceAdding || cs.editSliceEditing {
		return m.handleSliceInputKey(msg)
	}

	s := msg.String()
	switch {
	case msg.Type == tea.KeyEsc:
		// slice 変更を entry field に書き戻す
		ef := &cs.editEntryFields[cs.editEntryIndex]
		items := make([]string, len(cs.editSliceItems))
		copy(items, cs.editSliceItems)
		ef.Value = items
		m.applyEntryFieldAndMark(ef)
		cs.editEntryFieldEdit = ""
		return m, nil

	case msg.Type == tea.KeyUp || s == "k":
		if cs.editSliceIndex > 0 {
			cs.editSliceIndex--
		}
		return m, nil

	case msg.Type == tea.KeyDown || s == "j":
		if cs.editSliceIndex < len(cs.editSliceItems)-1 {
			cs.editSliceIndex++
		}
		return m, nil

	case s == "a":
		cs.editSliceAdding = true
		cs.editSliceInput.SetValue("")
		cs.editSliceInput.Focus()
		return m, nil

	case s == "d":
		if cs.editSliceIndex >= 0 && cs.editSliceIndex < len(cs.editSliceItems) {
			cs.editSliceItems = append(cs.editSliceItems[:cs.editSliceIndex], cs.editSliceItems[cs.editSliceIndex+1:]...)
			if cs.editSliceIndex >= len(cs.editSliceItems) && cs.editSliceIndex > 0 {
				cs.editSliceIndex--
			}
		}
		return m, nil

	case isEnterKey(msg):
		if cs.editSliceIndex >= 0 && cs.editSliceIndex < len(cs.editSliceItems) {
			cs.editSliceEditing = true
			cs.editSliceInput.SetValue(cs.editSliceItems[cs.editSliceIndex])
			cs.editSliceInput.Focus()
			cs.editSliceInput.CursorEnd()
		}
		return m, nil
	}
	return m, nil
}

// applyEntryFieldAndMark は entry field の変更を Config に書き戻し dirty をマークする。
func (m *Model) applyEntryFieldAndMark(ef *structEntryField) {
	cs := m.configScreen
	field := cs.selectedField()
	if field == nil {
		return
	}
	cs.applyEntryField(field.Path, cs.editEntryKey, *ef)
	cs.dirty = true
	cs.saveStatus = statusModified
	cs.refreshCategories()
}

// handleStructMapAddKey は structmap の新規キー追加入力処理。
func (m Model) handleStructMapAddKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	cs := m.configScreen
	switch {
	case msg.Type == tea.KeyEsc:
		cs.editStructAdding = false
		cs.editStructInput.Blur()
		return m, nil

	case isEnterKey(msg):
		key := strings.TrimSpace(cs.editStructInput.Value())
		if key != "" {
			field := cs.selectedField()
			if field != nil && m.addStructMapKey(field.Path, key) {
				cs.editStructKeys = append(cs.editStructKeys, key)
				sort.Strings(cs.editStructKeys)
				cs.editStructIndex = sort.SearchStrings(cs.editStructKeys, key)
				cs.dirty = true
				cs.saveStatus = statusModified
				cs.refreshCategories()
			}
			// 既存 key の場合は UI / dirty を変更しない
		}
		cs.editStructAdding = false
		cs.editStructInput.Blur()
		return m, nil

	default:
		var cmd tea.Cmd
		cs.editStructInput, cmd = cs.editStructInput.Update(msg)
		return m, cmd
	}
}

// deleteStructMapKey は structmap のキーを削除する。
func (m *Model) deleteStructMapKey(path, key string) {
	cs := m.configScreen
	val, _ := config.GetFieldValue(cs.cfg, path)
	switch v := val.(type) {
	case map[string]config.ProviderModelConfig:
		cs.cfg.DeleteProviderModelConfig(key)
	case map[string]config.LSPServerConfig:
		delete(v, key)
	}
}

// addStructMapKey は structmap に空のキーを追加する。
// 既存キーの場合は false を返し、新規追加時は true を返す。
func (m *Model) addStructMapKey(path, key string) bool {
	cs := m.configScreen
	m.ensureStructMapInitialized(path)
	if path == "provider_models" {
		providerModels := cs.cfg.ProviderModelsForEdit()
		if providerModels == nil {
			providerModels = map[string]config.ProviderModelConfig{}
		}
		if _, ok := providerModels[key]; ok {
			return false
		}
		providerModels[key] = config.ProviderModelConfig{}
		cs.cfg.SetProviderModelsForEdit(providerModels)
		return true
	}
	val, _ := config.GetFieldValue(cs.cfg, path)
	switch v := val.(type) {
	case map[string]config.LSPServerConfig:
		if _, ok := v[key]; ok {
			return false
		}
		v[key] = config.LSPServerConfig{}
		return true
	}
	return false
}

// ensureStructMapInitialized は TUI が直接 mutate する struct map を必要に応じて初期化する。
func (m *Model) ensureStructMapInitialized(path string) {
	cs := m.configScreen
	if cs == nil {
		return
	}

	switch path {
	case "lsp.servers":
		if cs.cfg.LSP.Servers == nil {
			cs.cfg.LSP.Servers = make(map[string]config.LSPServerConfig)
		}
	}
}

// handleConfigFilterKey はフィルタモードのキー処理。
