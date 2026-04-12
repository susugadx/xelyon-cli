package ui

import (
	"bytes"
	"strconv"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/config"
)

func TestConfigMenuEditFloat_ProjectMapContextRatio(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		wantValue   float64
		wantChanged bool
	}{
		{name: "valid", input: "0.05\n", wantValue: 0.05, wantChanged: true},
		{name: "out of range", input: "0.5\n", wantValue: config.ProjectMapContextRatioDefault, wantChanged: false},
		{name: "nan", input: "NaN\n", wantValue: config.ProjectMapContextRatioDefault, wantChanged: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runtime := NewRuntime(strings.NewReader(tt.input), &bytes.Buffer{}, &bytes.Buffer{})
			menu := NewConfigMenuWithRuntime(config.DefaultConfig(), nil, runtime)

			got, changed, err := menu.editFloat(runtime.PromptIO(), &config.ConfigField{
				Path:      "project_map.context_ratio",
				FieldType: config.FieldTypeFloat,
				Current:   config.ProjectMapContextRatioDefault,
			})
			if err != nil {
				t.Fatalf("editFloat() error = %v", err)
			}

			gotValue, ok := got.(float64)
			if !ok {
				t.Fatalf("editFloat() returned %T, want float64", got)
			}
			if gotValue != tt.wantValue {
				t.Errorf("editFloat() value = %v, want %v", gotValue, tt.wantValue)
			}
			if changed != tt.wantChanged {
				t.Errorf("editFloat() changed = %v, want %v", changed, tt.wantChanged)
			}
		})
	}
}

func TestNewConfigMenu_UsesDefaultRuntime(t *testing.T) {
	menu := NewConfigMenu(config.DefaultConfig(), nil)
	if menu.Runtime == nil {
		t.Fatal("Runtime should not be nil")
	}
	if menu.Config == nil {
		t.Fatal("Config should not be nil")
	}
}

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

func TestConfigMenu_EditField_ScalarTypes(t *testing.T) {
	tests := []struct {
		name        string
		field       config.ConfigField
		input       string
		wantValue   interface{}
		wantChanged bool
	}{
		{
			name:        "bool",
			field:       config.ConfigField{Path: "thinking.enabled", FieldType: config.FieldTypeBool, Current: false},
			input:       "y\n",
			wantValue:   true,
			wantChanged: true,
		},
		{
			name:        "int",
			field:       config.ConfigField{Path: "api_retry.count", FieldType: config.FieldTypeInt, Current: 3},
			input:       "7\n",
			wantValue:   7,
			wantChanged: true,
		},
		{
			name:        "string",
			field:       config.ConfigField{Path: "default_model", FieldType: config.FieldTypeString, Current: "gpt-5"},
			input:       "gpt-5.4\n",
			wantValue:   "gpt-5.4",
			wantChanged: true,
		},
		{
			name: "select",
			field: config.ConfigField{
				Path:      "general.ui_language",
				FieldType: config.FieldTypeSelect,
				Current:   "ja",
				Options:   []string{"auto", "ja", "en"},
			},
			input:       "3\n",
			wantValue:   "en",
			wantChanged: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runtime := NewRuntime(strings.NewReader(tt.input), &bytes.Buffer{}, &bytes.Buffer{})
			menu := NewConfigMenuWithRuntime(config.DefaultConfig(), nil, runtime)

			got, changed, err := menu.EditField(&tt.field)
			if err != nil {
				t.Fatalf("EditField() error = %v", err)
			}
			if got != tt.wantValue {
				t.Fatalf("EditField() value = %#v, want %#v", got, tt.wantValue)
			}
			if changed != tt.wantChanged {
				t.Fatalf("EditField() changed = %v, want %v", changed, tt.wantChanged)
			}
		})
	}
}

func TestConfigMenu_EditField_InvalidInputKeepsCurrentValue(t *testing.T) {
	tests := []struct {
		name      string
		field     config.ConfigField
		input     string
		wantValue interface{}
	}{
		{
			name:      "bool invalid",
			field:     config.ConfigField{Path: "thinking.enabled", FieldType: config.FieldTypeBool, Current: true},
			input:     "maybe\n",
			wantValue: true,
		},
		{
			name:      "int invalid",
			field:     config.ConfigField{Path: "api_retry.count", FieldType: config.FieldTypeInt, Current: 3},
			input:     "abc\n",
			wantValue: 3,
		},
		{
			name: "select invalid",
			field: config.ConfigField{
				Path:      "general.ui_language",
				FieldType: config.FieldTypeSelect,
				Current:   "ja",
				Options:   []string{"auto", "ja", "en"},
			},
			input:     "9\n",
			wantValue: "ja",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runtime := NewRuntime(strings.NewReader(tt.input), &bytes.Buffer{}, &bytes.Buffer{})
			menu := NewConfigMenuWithRuntime(config.DefaultConfig(), nil, runtime)

			got, changed, err := menu.EditField(&tt.field)
			if err != nil {
				t.Fatalf("EditField() error = %v", err)
			}
			if got != tt.wantValue {
				t.Fatalf("EditField() value = %#v, want %#v", got, tt.wantValue)
			}
			if changed {
				t.Fatal("EditField() changed = true, want false")
			}
		})
	}
}

func TestConfigMenu_EditField_DelegatesCompositeEditors(t *testing.T) {
	t.Run("string slice", func(t *testing.T) {
		runtime := NewRuntime(strings.NewReader("a\nnew-item\ns\n"), &bytes.Buffer{}, &bytes.Buffer{})
		menu := NewConfigMenuWithRuntime(config.DefaultConfig(), nil, runtime)

		got, changed, err := menu.EditField(&config.ConfigField{
			Path:      "hooks.on_completion",
			FieldType: config.FieldTypeStringSlice,
			Current:   []string{"existing"},
		})
		if err != nil {
			t.Fatalf("EditField() error = %v", err)
		}
		slice, ok := got.([]string)
		if !ok {
			t.Fatalf("EditField() returned %T, want []string", got)
		}
		if len(slice) != 2 || slice[1] != "new-item" {
			t.Fatalf("EditField() value = %#v, want appended item", slice)
		}
		if !changed {
			t.Fatal("EditField() changed = false, want true")
		}
	})

	t.Run("string map", func(t *testing.T) {
		runtime := NewRuntime(strings.NewReader("a\nFOO\nBAR\ns\n"), &bytes.Buffer{}, &bytes.Buffer{})
		menu := NewConfigMenuWithRuntime(config.DefaultConfig(), nil, runtime)

		got, changed, err := menu.EditField(&config.ConfigField{
			Path:      "command_aliases",
			FieldType: config.FieldTypeStringMap,
			Current:   map[string]string{"A": "1"},
		})
		if err != nil {
			t.Fatalf("EditField() error = %v", err)
		}
		mp, ok := got.(map[string]string)
		if !ok {
			t.Fatalf("EditField() returned %T, want map[string]string", got)
		}
		if mp["FOO"] != "BAR" {
			t.Fatalf("EditField() value = %#v, want FOO=BAR", mp)
		}
		if !changed {
			t.Fatal("EditField() changed = false, want true")
		}
	})

	t.Run("struct map", func(t *testing.T) {
		cfg := config.DefaultConfig()
		cfg.LSP.Servers = map[string]config.LSPServerConfig{}
		runtime := NewRuntime(strings.NewReader("a\nzig\nzls\ns\n"), &bytes.Buffer{}, &bytes.Buffer{})
		menu := NewConfigMenuWithRuntime(cfg, nil, runtime)

		got, changed, err := menu.EditField(&config.ConfigField{
			Path:      "lsp.servers",
			FieldType: config.FieldTypeStructMap,
		})
		if err != nil {
			t.Fatalf("EditField() error = %v", err)
		}
		if got != nil {
			t.Fatalf("EditField() value = %#v, want nil for struct map", got)
		}
		if !changed {
			t.Fatal("EditField() changed = false, want true")
		}
		if cfg.LSP.Servers["zig"].Command != "zls" {
			t.Fatalf("cfg.LSP.Servers = %#v, want zig=zls", cfg.LSP.Servers)
		}
	})
}

func TestFormatValue(t *testing.T) {
	tests := []struct {
		name string
		in   interface{}
		want string
	}{
		{name: "nil", in: nil, want: "null"},
		{name: "bool true", in: true, want: "true"},
		{name: "empty string", in: "", want: "(empty)"},
		{name: "string slice", in: []string{"a", "b"}, want: "[2 items]"},
		{name: "string map", in: map[string]string{"A": "B"}, want: "{1 entries}"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := formatValue(tt.in); got != tt.want {
				t.Fatalf("formatValue(%#v) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}
