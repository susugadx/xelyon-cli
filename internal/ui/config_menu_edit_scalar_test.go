package ui

import (
	"bytes"
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
