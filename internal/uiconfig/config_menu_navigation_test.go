package uiconfig

import (
	"testing"

	"github.com/susugadx/xelyon-cli/internal/config"
)

func TestParseMenuNumberWithZeroAsTen(t *testing.T) {
	tests := []struct {
		name  string
		input string
		max   int
		want  int
		ok    bool
	}{
		{name: "one based number", input: "1", max: 3, want: 0, ok: true},
		{name: "zero maps to tenth", input: "0", max: 10, want: 9, ok: true},
		{name: "ten accepted", input: "10", max: 10, want: 9, ok: true},
		{name: "out of range", input: "11", max: 10, want: 0, ok: false},
		{name: "non number", input: "x", max: 10, want: 0, ok: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := parseMenuNumberWithZeroAsTen(tt.input, tt.max)
			if ok != tt.ok {
				t.Fatalf("parseMenuNumberWithZeroAsTen() ok = %v, want %v", ok, tt.ok)
			}
			if got != tt.want {
				t.Fatalf("parseMenuNumberWithZeroAsTen() idx = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestResolveConfigCategorySelectionCommands(t *testing.T) {
	state := configCategoryMenuState{pageSize: 10, currentPage: 1, totalPages: 3}
	page := configCategoryPage{
		start:      10,
		categories: make([]config.ConfigCategory, 2),
	}

	tests := []struct {
		name   string
		input  string
		action configCategorySelectionAction
		index  int
	}{
		{name: "cancel", input: "q", action: configCategorySelectionCancel},
		{name: "next", input: "n", action: configCategorySelectionNext},
		{name: "prev", input: "p", action: configCategorySelectionPrev},
		{name: "pick", input: "2", action: configCategorySelectionPick, index: 11},
		{name: "invalid", input: "x", action: configCategorySelectionIgnore},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			selection := resolveConfigCategorySelection(tt.input, page, state, 20)
			if selection.action != tt.action {
				t.Fatalf("action = %v, want %v", selection.action, tt.action)
			}
			if tt.action == configCategorySelectionPick && selection.categoryIndex != tt.index {
				t.Fatalf("categoryIndex = %d, want %d", selection.categoryIndex, tt.index)
			}
		})
	}
}

func TestResolveConfigCategorySelection_IgnoresOutOfRangeNavigation(t *testing.T) {
	firstPageState := configCategoryMenuState{pageSize: 10, currentPage: 0, totalPages: 2}
	lastPageState := configCategoryMenuState{pageSize: 10, currentPage: 1, totalPages: 2}
	page := configCategoryPage{start: 0, categories: make([]config.ConfigCategory, 2)}

	if got := resolveConfigCategorySelection("p", page, firstPageState, 12); got.action != configCategorySelectionIgnore {
		t.Fatalf("previous on first page action = %v, want ignore", got.action)
	}
	if got := resolveConfigCategorySelection("n", page, lastPageState, 12); got.action != configCategorySelectionIgnore {
		t.Fatalf("next on last page action = %v, want ignore", got.action)
	}
}

func TestResolveConfigCategorySelection_ZeroSelectsTenthEntry(t *testing.T) {
	state := configCategoryMenuState{pageSize: 10, currentPage: 0, totalPages: 2}
	page := configCategoryPage{start: 0, categories: make([]config.ConfigCategory, 10)}

	got := resolveConfigCategorySelection("0", page, state, 20)
	if got.action != configCategorySelectionPick {
		t.Fatalf("action = %v, want pick", got.action)
	}
	if got.categoryIndex != 9 {
		t.Fatalf("categoryIndex = %d, want 9", got.categoryIndex)
	}
}

func TestResolveConfigFieldSelection(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		fieldSize int
		wantBack  bool
		wantPick  bool
		wantIndex int
	}{
		{name: "back", input: "b", fieldSize: 5, wantBack: true, wantPick: false},
		{name: "pick number", input: "3", fieldSize: 5, wantBack: false, wantPick: true, wantIndex: 2},
		{name: "pick zero as tenth", input: "0", fieldSize: 10, wantBack: false, wantPick: true, wantIndex: 9},
		{name: "invalid", input: "x", fieldSize: 5, wantBack: false, wantPick: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := resolveConfigFieldSelection(tt.input, tt.fieldSize)
			if got.back != tt.wantBack {
				t.Fatalf("back = %v, want %v", got.back, tt.wantBack)
			}
			if got.hasSelect != tt.wantPick {
				t.Fatalf("hasSelect = %v, want %v", got.hasSelect, tt.wantPick)
			}
			if got.fieldIndex != tt.wantIndex {
				t.Fatalf("fieldIndex = %d, want %d", got.fieldIndex, tt.wantIndex)
			}
		})
	}
}
