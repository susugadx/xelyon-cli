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
