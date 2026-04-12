package tui

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/susugadx/xelyon-cli/internal/config"
)

// newConfigScreen は設定データから configScreen を初期化する。
func newConfigScreen(cfg *config.Config) *configScreen {
	ti := textinput.New()
	ti.Prompt = ""
	ti.CharLimit = 256

	filterTI := textinput.New()
	filterTI.Prompt = "/"
	filterTI.CharLimit = 64

	sliceTI := textinput.New()
	sliceTI.Prompt = ""
	sliceTI.CharLimit = 256

	structTI := textinput.New()
	structTI.Prompt = ""
	structTI.CharLimit = 256

	return &configScreen{
		cfg:                    cfg,
		categories:             config.BuildConfigRegistry(cfg),
		initialDefaultProvider: cfg.DefaultProvider,
		editInput:              ti,
		filterInput:            filterTI,
		editSliceInput:         sliceTI,
		editStructInput:        structTI,
	}
}

// selectedCategory は現在選択中のカテゴリを返す。
func (cs *configScreen) selectedCategory() *config.ConfigCategory {
	if cs.catIndex >= 0 && cs.catIndex < len(cs.categories) {
		return &cs.categories[cs.catIndex]
	}
	return nil
}

// filteredFields はフィルタ適用済みのフィールドリストを返す。
func (cs *configScreen) filteredFields() []config.ConfigField {
	cat := cs.selectedCategory()
	if cat == nil {
		return nil
	}
	if cs.filterText == "" {
		return cat.Fields
	}

	lower := strings.ToLower(cs.filterText)
	var result []config.ConfigField
	for _, field := range cat.Fields {
		if strings.Contains(strings.ToLower(field.DisplayName), lower) ||
			strings.Contains(strings.ToLower(field.Path), lower) ||
			strings.Contains(strings.ToLower(field.Description), lower) {
			result = append(result, field)
		}
	}
	return result
}

// selectedField は現在選択中のフィールドを返す。
func (cs *configScreen) selectedField() *config.ConfigField {
	fields := cs.filteredFields()
	if cs.fieldIndex >= 0 && cs.fieldIndex < len(fields) {
		return &fields[cs.fieldIndex]
	}
	return nil
}

// refreshCategories はカテゴリを再構築する。
func (cs *configScreen) refreshCategories() {
	cs.categories = config.BuildConfigRegistry(cs.cfg)
}
