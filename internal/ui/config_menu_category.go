package ui

import (
	"fmt"
	"io"

	"github.com/susugadx/xelyon-cli/internal/config"
)

const configMenuCategoryPageSize = 10

type configCategoryMenuState struct {
	pageSize    int
	currentPage int
	totalPages  int
}

type configCategoryPage struct {
	start      int
	categories []config.ConfigCategory
	pageInfo   string
}

type configCategorySelectionAction int

const (
	configCategorySelectionIgnore configCategorySelectionAction = iota
	configCategorySelectionCancel
	configCategorySelectionNext
	configCategorySelectionPrev
	configCategorySelectionPick
)

type configCategorySelection struct {
	action        configCategorySelectionAction
	categoryIndex int
}

// Run はメインメニューを表示し、選択されたカテゴリを返す。
func (m *ConfigMenu) Run() (*config.ConfigCategory, error) {
	ctx := newConfigPromptContext(m.Runtime)
	promptIO := ctx.promptIO
	out := ctx.out

	state := newConfigCategoryMenuState(len(m.Categories))

	for {
		page := buildConfigCategoryPage(m.Categories, state)
		m.renderConfigCategoryPage(out, page, state)

		input := readConfigMenuInput(&promptIO)
		selection := resolveConfigCategorySelection(input, page, state, len(m.Categories))
		switch selection.action {
		case configCategorySelectionCancel:
			return nil, nil
		case configCategorySelectionNext:
			state.currentPage++
		case configCategorySelectionPrev:
			state.currentPage--
		case configCategorySelectionPick:
			return &m.Categories[selection.categoryIndex], nil
		}
	}
}

func newConfigCategoryMenuState(totalCategories int) configCategoryMenuState {
	totalPages := (totalCategories + configMenuCategoryPageSize - 1) / configMenuCategoryPageSize
	return configCategoryMenuState{
		pageSize:    configMenuCategoryPageSize,
		currentPage: 0,
		totalPages:  totalPages,
	}
}

func buildConfigCategoryPage(categories []config.ConfigCategory, state configCategoryMenuState) configCategoryPage {
	start := state.currentPage * state.pageSize
	end := start + state.pageSize
	if end > len(categories) {
		end = len(categories)
	}

	pageInfo := ""
	if state.totalPages > 1 {
		pageInfo = fmt.Sprintf(" (%d/%d)", state.currentPage+1, state.totalPages)
	}

	return configCategoryPage{
		start:      start,
		categories: categories[start:end],
		pageInfo:   pageInfo,
	}
}

func (m *ConfigMenu) renderConfigCategoryPage(out io.Writer, page configCategoryPage, state configCategoryMenuState) {
	_, _ = fmt.Fprintf(out, "\n%s── Configuration%s ──────────────────────%s\n\n", colorCyan, page.pageInfo, colorReset)

	for i, cat := range page.categories {
		num := i + 1
		if num == 10 {
			num = 0
		}
		_, _ = fmt.Fprintf(out, "  [%d] %s %s\n", num, cat.Icon, cat.DisplayName)
	}

	_, _ = fmt.Fprintln(out)
	if state.totalPages > 1 {
		if state.currentPage < state.totalPages-1 {
			_, _ = fmt.Fprintln(out, "  [n] Next page")
		}
		if state.currentPage > 0 {
			_, _ = fmt.Fprintln(out, "  [p] Previous page")
		}
	}
	_, _ = fmt.Fprintln(out, "  [q] Cancel")
	_, _ = fmt.Fprintf(out, "\n%sSelect category:%s ", colorCyan, colorReset)
}

func resolveConfigCategorySelection(input string, page configCategoryPage, state configCategoryMenuState, totalCategories int) configCategorySelection {
	switch input {
	case "q", "quit", "cancel":
		return configCategorySelection{action: configCategorySelectionCancel}
	case "n", "next":
		if state.currentPage < state.totalPages-1 {
			return configCategorySelection{action: configCategorySelectionNext}
		}
		return configCategorySelection{action: configCategorySelectionIgnore}
	case "p", "prev", "previous":
		if state.currentPage > 0 {
			return configCategorySelection{action: configCategorySelectionPrev}
		}
		return configCategorySelection{action: configCategorySelectionIgnore}
	default:
		localIdx, ok := parseMenuNumberWithZeroAsTen(input, len(page.categories))
		if !ok {
			return configCategorySelection{action: configCategorySelectionIgnore}
		}
		categoryIdx := page.start + localIdx
		if categoryIdx < 0 || categoryIdx >= totalCategories {
			return configCategorySelection{action: configCategorySelectionIgnore}
		}
		return configCategorySelection{
			action:        configCategorySelectionPick,
			categoryIndex: categoryIdx,
		}
	}
}
