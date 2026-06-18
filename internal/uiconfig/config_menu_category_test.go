package uiconfig

import (
	"bytes"
	"strconv"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/config"
)

func TestConfigMenu_Run_PaginationSelectsNextPageItem(t *testing.T) {
	categories := make([]config.ConfigCategory, 0, 11)
	for i := 1; i <= 11; i++ {
		categories = append(categories, config.ConfigCategory{
			Name:        "cat",
			DisplayName: "Category",
			Icon:        "C",
			Fields: []config.ConfigField{
				{Path: "field"},
			},
		})
		categories[i-1].DisplayName = categories[i-1].DisplayName + " " + strconv.Itoa(i)
	}

	runtime := NewRuntime(strings.NewReader("n\n1\n"), &bytes.Buffer{}, &bytes.Buffer{})
	menu := NewConfigMenuWithRuntime(config.DefaultConfig(), categories, runtime)

	selected, err := menu.Run()
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if selected == nil || selected.DisplayName != "Category 11" {
		t.Fatalf("Run() selected = %+v, want Category 11", selected)
	}
}

func TestConfigMenu_ShowFieldList_SelectAndBack(t *testing.T) {
	category := &config.ConfigCategory{
		DisplayName: "General",
		Icon:        "G",
		Fields: []config.ConfigField{
			{Path: "general.ui_language", DisplayName: "ui_language", Current: "ja"},
			{Path: "default_model", DisplayName: "default_model", Current: "gpt-5"},
		},
	}

	t.Run("select field", func(t *testing.T) {
		runtime := NewRuntime(strings.NewReader("2\n"), &bytes.Buffer{}, &bytes.Buffer{})
		menu := NewConfigMenuWithRuntime(config.DefaultConfig(), nil, runtime)

		field, err := menu.ShowFieldList(category)
		if err != nil {
			t.Fatalf("ShowFieldList() error = %v", err)
		}
		if field == nil || field.Path != "default_model" {
			t.Fatalf("ShowFieldList() = %+v, want default_model", field)
		}
	})

	t.Run("back", func(t *testing.T) {
		runtime := NewRuntime(strings.NewReader("b\n"), &bytes.Buffer{}, &bytes.Buffer{})
		menu := NewConfigMenuWithRuntime(config.DefaultConfig(), nil, runtime)

		field, err := menu.ShowFieldList(category)
		if err == nil || err.Error() != "back" {
			t.Fatalf("ShowFieldList() error = %v, want back", err)
		}
		if field != nil {
			t.Fatalf("ShowFieldList() field = %+v, want nil", field)
		}
	})
}
