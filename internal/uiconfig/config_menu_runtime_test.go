package uiconfig

import (
	"bytes"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/uiruntime"
)

func TestConfigMenu_RunWithRuntimeUsesInjectedWriter(t *testing.T) {
	cfg := config.DefaultConfig()
	runtime := uiruntime.NewRuntime(strings.NewReader("q\n"), &bytes.Buffer{}, &bytes.Buffer{})
	out := runtime.Output().(*bytes.Buffer)

	menu := NewConfigMenuWithRuntime(cfg, config.BuildConfigRegistry(cfg), runtime)
	selected, err := menu.Run()
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if selected != nil {
		t.Fatalf("Run() selected = %v, want nil", selected)
	}

	output := out.String()
	if !strings.Contains(output, "Configuration") {
		t.Fatalf("expected injected output to contain menu header, got %q", output)
	}
	if !strings.Contains(output, "Select category:") {
		t.Fatalf("expected injected output to contain prompt, got %q", output)
	}
}
