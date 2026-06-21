package uiconfig

import (
	"bytes"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/config"
)

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
