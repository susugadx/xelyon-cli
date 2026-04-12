package ui

import (
	"bytes"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/config"
)

func TestStructMapEditorProviderModels_AllowsClaudeSiblingWhenAnthropicExists(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.SetProviderModelsForEdit(map[string]config.ProviderModelConfig{
		"anthropic": {DefaultModel: "anthropic-custom"},
	})

	var out bytes.Buffer
	runtime := NewRuntime(strings.NewReader("a\nclaude\nclaude-custom\ns\n"), &out, &out)
	editor := NewStructMapEditorWithRuntime("provider_models", config.FieldTypeStructMap, runtime)

	changed, err := editor.Run(cfg)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if !changed {
		t.Fatal("Run() changed = false, want true")
	}

	providerModels := cfg.ProviderModelsForEdit()
	if _, ok := providerModels["anthropic"]; !ok {
		t.Fatal("provider_models.anthropic should remain configured")
	}
	if got := providerModels["claude"].DefaultModel; got != "claude-custom" {
		t.Fatalf("provider_models.claude.default_model = %q, want %q", got, "claude-custom")
	}
	if strings.Contains(out.String(), "Provider already configured") {
		t.Fatalf("unexpected duplicate warning: %q", out.String())
	}
}

func TestStructMapEditorProviderModels_AllowsAnthropicSiblingWhenClaudeExists(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.SetProviderModelsForEdit(map[string]config.ProviderModelConfig{
		"claude": {DefaultModel: "claude-custom"},
	})

	var out bytes.Buffer
	runtime := NewRuntime(strings.NewReader("a\nanthropic\nanthropic-custom\ns\n"), &out, &out)
	editor := NewStructMapEditorWithRuntime("provider_models", config.FieldTypeStructMap, runtime)

	changed, err := editor.Run(cfg)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if !changed {
		t.Fatal("Run() changed = false, want true")
	}

	providerModels := cfg.ProviderModelsForEdit()
	if _, ok := providerModels["claude"]; !ok {
		t.Fatal("provider_models.claude should remain configured")
	}
	if got := providerModels["anthropic"].DefaultModel; got != "anthropic-custom" {
		t.Fatalf("provider_models.anthropic.default_model = %q, want %q", got, "anthropic-custom")
	}
	if strings.Contains(out.String(), "Provider already configured") {
		t.Fatalf("unexpected duplicate warning: %q", out.String())
	}
}

func TestNewEditors_UseDefaultRuntime(t *testing.T) {
	if editor := NewStringSliceEditor("hooks.on_completion", nil); editor.Runtime == nil {
		t.Fatal("NewStringSliceEditor() Runtime should not be nil")
	}
	if editor := NewStringMapEditor("command_aliases", nil); editor.Runtime == nil {
		t.Fatal("NewStringMapEditor() Runtime should not be nil")
	}
	if editor := NewStructMapEditor("lsp.servers", config.FieldTypeStructMap); editor.Runtime == nil {
		t.Fatal("NewStructMapEditor() Runtime should not be nil")
	}
}

func TestStringSliceEditor_Run_DeleteAndCancel(t *testing.T) {
	t.Run("delete and save", func(t *testing.T) {
		runtime := NewRuntime(strings.NewReader("d\n1\ns\n"), &bytes.Buffer{}, &bytes.Buffer{})
		editor := NewStringSliceEditorWithRuntime("hooks.on_completion", []string{"one", "two"}, runtime)

		got, changed, err := editor.Run()
		if err != nil {
			t.Fatalf("Run() error = %v", err)
		}
		if !changed {
			t.Fatal("Run() changed = false, want true")
		}
		if len(got) != 1 || got[0] != "two" {
			t.Fatalf("Run() value = %#v, want [two]", got)
		}
	})

	t.Run("cancel", func(t *testing.T) {
		runtime := NewRuntime(strings.NewReader("c\n"), &bytes.Buffer{}, &bytes.Buffer{})
		editor := NewStringSliceEditorWithRuntime("hooks.on_completion", []string{"one"}, runtime)

		got, changed, err := editor.Run()
		if err != nil {
			t.Fatalf("Run() error = %v", err)
		}
		if got != nil || changed {
			t.Fatalf("Run() = %#v, %v; want nil, false", got, changed)
		}
	})
}

func TestStringMapEditor_Run_AddEditDeleteAndCancel(t *testing.T) {
	t.Run("add edit delete save", func(t *testing.T) {
		runtime := NewRuntime(strings.NewReader("a\nbeta\ntwo\ne\n1\nONE\nd\n2\ns\n"), &bytes.Buffer{}, &bytes.Buffer{})
		editor := NewStringMapEditorWithRuntime("command_aliases", map[string]string{"alpha": "one"}, runtime)

		got, changed, err := editor.Run()
		if err != nil {
			t.Fatalf("Run() error = %v", err)
		}
		if !changed {
			t.Fatal("Run() changed = false, want true")
		}
		if len(got) != 1 || got["alpha"] != "ONE" {
			t.Fatalf("Run() value = %#v, want alpha=ONE only", got)
		}
	})

	t.Run("cancel", func(t *testing.T) {
		runtime := NewRuntime(strings.NewReader("c\n"), &bytes.Buffer{}, &bytes.Buffer{})
		editor := NewStringMapEditorWithRuntime("command_aliases", map[string]string{"alpha": "one"}, runtime)

		got, changed, err := editor.Run()
		if err != nil {
			t.Fatalf("Run() error = %v", err)
		}
		if got != nil || changed {
			t.Fatalf("Run() = %#v, %v; want nil, false", got, changed)
		}
	})
}

func TestStructMapEditor_RunLSPServersAndUnsupportedPath(t *testing.T) {
	t.Run("add edit toggle save", func(t *testing.T) {
		cfg := config.DefaultConfig()
		cfg.LSP.Servers = map[string]config.LSPServerConfig{}

		runtime := NewRuntime(strings.NewReader("a\nzig\nzls\n1\n1\nzls-updated\n1\n2\ns\n"), &bytes.Buffer{}, &bytes.Buffer{})
		editor := NewStructMapEditorWithRuntime("lsp.servers", config.FieldTypeStructMap, runtime)

		changed, err := editor.Run(cfg)
		if err != nil {
			t.Fatalf("Run() error = %v", err)
		}
		if !changed {
			t.Fatal("Run() changed = false, want true")
		}
		server := cfg.LSP.Servers["zig"]
		if server.Command != "zls-updated" || !server.Disabled {
			t.Fatalf("cfg.LSP.Servers[zig] = %#v, want updated command and disabled=true", server)
		}
	})

	t.Run("delete", func(t *testing.T) {
		cfg := config.DefaultConfig()
		cfg.LSP.Servers = map[string]config.LSPServerConfig{
			"zig": {Command: "zls"},
		}

		runtime := NewRuntime(strings.NewReader("d\n1\ns\n"), &bytes.Buffer{}, &bytes.Buffer{})
		editor := NewStructMapEditorWithRuntime("lsp.servers", config.FieldTypeStructMap, runtime)

		changed, err := editor.Run(cfg)
		if err != nil {
			t.Fatalf("Run() error = %v", err)
		}
		if !changed {
			t.Fatal("Run() changed = false, want true")
		}
		if len(cfg.LSP.Servers) != 0 {
			t.Fatalf("cfg.LSP.Servers = %#v, want empty", cfg.LSP.Servers)
		}
	})

	t.Run("unsupported path", func(t *testing.T) {
		var out bytes.Buffer
		runtime := NewRuntime(strings.NewReader(""), &out, &out)
		editor := NewStructMapEditorWithRuntime("unsupported.path", config.FieldTypeStructMap, runtime)

		changed, err := editor.Run(config.DefaultConfig())
		if err != nil {
			t.Fatalf("Run() error = %v", err)
		}
		if changed {
			t.Fatal("Run() changed = true, want false")
		}
		if !strings.Contains(out.String(), "StructMap editing not supported") {
			t.Fatalf("output = %q, want unsupported message", out.String())
		}
	})
}

func TestReadLineWithIO_StripsBracketedPasteAndWhitespace(t *testing.T) {
	runtime := NewRuntime(strings.NewReader("\x1b[200~  hello world  \x1b[201~\n"), &bytes.Buffer{}, &bytes.Buffer{})
	promptIO := runtime.PromptIO()

	if got := readLineWithIO(&promptIO); got != "hello world" {
		t.Fatalf("readLineWithIO() = %q, want %q", got, "hello world")
	}
}

func TestStructMapEditorProviderModels_EditDeleteAndCancel(t *testing.T) {
	t.Run("edit then delete", func(t *testing.T) {
		cfg := config.DefaultConfig()
		cfg.SetProviderModelsForEdit(map[string]config.ProviderModelConfig{
			"openai": {DefaultModel: "gpt-4o"},
		})

		runtime := NewRuntime(strings.NewReader("1\ngpt-5\n"+"d\n1\n"+"s\n"), &bytes.Buffer{}, &bytes.Buffer{})
		editor := NewStructMapEditorWithRuntime("provider_models", config.FieldTypeStructMap, runtime)

		changed, err := editor.Run(cfg)
		if err != nil {
			t.Fatalf("Run() error = %v", err)
		}
		if !changed {
			t.Fatal("Run() changed = false, want true")
		}
		if len(cfg.ProviderModelsForEdit()) != 0 {
			t.Fatalf("provider models = %#v, want empty", cfg.ProviderModelsForEdit())
		}
	})

	t.Run("cancel keeps current state", func(t *testing.T) {
		cfg := config.DefaultConfig()
		cfg.SetProviderModelsForEdit(map[string]config.ProviderModelConfig{
			"openai": {DefaultModel: "gpt-4o"},
		})

		runtime := NewRuntime(strings.NewReader("c\n"), &bytes.Buffer{}, &bytes.Buffer{})
		editor := NewStructMapEditorWithRuntime("provider_models", config.FieldTypeStructMap, runtime)

		changed, err := editor.Run(cfg)
		if err != nil {
			t.Fatalf("Run() error = %v", err)
		}
		if changed {
			t.Fatal("Run() changed = true, want false")
		}
		if got := cfg.ProviderModelsForEdit()["openai"].DefaultModel; got != "gpt-4o" {
			t.Fatalf("provider_models.openai.default_model = %q, want %q", got, "gpt-4o")
		}
	})
}
