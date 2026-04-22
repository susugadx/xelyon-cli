package ui

import (
	"bytes"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/config"
)

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
