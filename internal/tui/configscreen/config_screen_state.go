package configscreen

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/susugadx/xelyon-cli/internal/config"
)

// New は設定データから Screen を初期化する。
func New(cfg *config.Config) *Screen {
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

	return &Screen{
		cfg:                    cfg,
		categories:             config.BuildConfigRegistry(cfg),
		initialDefaultProvider: cfg.DefaultProvider,
		editInput:              ti,
		filterInput:            filterTI,
		editSliceInput:         sliceTI,
		editStructInput:        structTI,
	}
}

// ConfigSnapshot は現在編集中の config を保存用に clone して返す。
func (cs *Screen) ConfigSnapshot() *config.Config {
	if cs == nil || cs.cfg == nil {
		return nil
	}
	return config.CloneConfig(cs.cfg)
}

// Snapshot は /config screen の内部状態を読み取り専用に写す。
func (cs *Screen) Snapshot() Snapshot {
	if cs == nil {
		return Snapshot{}
	}
	var selected *config.ConfigField
	if field := cs.selectedField(); field != nil {
		copy := *field
		selected = &copy
	}
	entryFields := make([]StructEntryField, 0, len(cs.editEntryFields))
	for _, field := range cs.editEntryFields {
		entryFields = append(entryFields, StructEntryField(field))
	}
	guidanceChoices := make([]GuidanceFileChoice, 0, len(cs.editGuidanceChoices))
	for _, choice := range cs.editGuidanceChoices {
		guidanceChoices = append(guidanceChoices, GuidanceFileChoice(choice))
	}
	categoryNames := make([]string, 0, len(cs.categories))
	for _, category := range cs.categories {
		categoryNames = append(categoryNames, category.Name)
	}
	return Snapshot{
		CategoryIndex:       cs.catIndex,
		FieldIndex:          cs.fieldIndex,
		FieldScroll:         cs.fieldScroll,
		ActivePane:          cs.activePane,
		EditMode:            cs.editMode,
		Dirty:               cs.dirty,
		SaveStatus:          cs.saveStatus,
		SaveError:           cs.saveError,
		ConfirmQuit:         cs.confirmQuit,
		ConfirmIndex:        cs.confirmIdx,
		PendingClose:        cs.pendingClose,
		SelectedField:       selected,
		EditStructKeys:      append([]string(nil), cs.editStructKeys...),
		EditStructIndex:     cs.editStructIndex,
		EditStructAdding:    cs.editStructAdding,
		EditEntryActive:     cs.editEntryActive,
		EditEntryKey:        cs.editEntryKey,
		EditEntryFields:     entryFields,
		EditEntryIndex:      cs.editEntryIndex,
		EditEntryFieldEdit:  cs.editEntryFieldEdit,
		EditSliceItems:      append([]string(nil), cs.editSliceItems...),
		EditSliceIndex:      cs.editSliceIndex,
		EditSliceAdding:     cs.editSliceAdding,
		EditGuidanceChoices: guidanceChoices,
		EditSelect:          cs.editSelect,
		FilterMode:          cs.filterMode,
		CategoryNames:       categoryNames,
		FilteredFields:      append([]config.ConfigField(nil), cs.filteredFields()...),
	}
}

// selectedCategory は現在選択中のカテゴリを返す。
func (cs *Screen) selectedCategory() *config.ConfigCategory {
	if cs.catIndex >= 0 && cs.catIndex < len(cs.categories) {
		return &cs.categories[cs.catIndex]
	}
	return nil
}

// filteredFields はフィルタ適用済みのフィールドリストを返す。
func (cs *Screen) filteredFields() []config.ConfigField {
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
func (cs *Screen) selectedField() *config.ConfigField {
	fields := cs.filteredFields()
	if cs.fieldIndex >= 0 && cs.fieldIndex < len(fields) {
		return &fields[cs.fieldIndex]
	}
	return nil
}

// refreshCategories はカテゴリを再構築する。
func (cs *Screen) refreshCategories() {
	cs.categories = config.BuildConfigRegistry(cs.cfg)
}
