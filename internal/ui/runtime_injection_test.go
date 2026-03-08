package ui

import (
	"bytes"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/config"
)

func TestConfigMenu_RunWithRuntimeUsesInjectedWriter(t *testing.T) {
	cfg := config.DefaultConfig()
	runtime := NewRuntime(strings.NewReader("q\n"), &bytes.Buffer{}, &bytes.Buffer{})
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

func TestProjectMenu_RunWithRuntimeUsesInjectedWriter(t *testing.T) {
	runtime := NewRuntime(strings.NewReader("c\n"), &bytes.Buffer{}, &bytes.Buffer{})
	out := runtime.Output().(*bytes.Buffer)

	menu := NewProjectMenuWithRuntime(&config.ProjectConfig{}, runtime)
	changed, err := menu.Run()
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if changed {
		t.Fatal("Run() changed = true, want false")
	}

	output := out.String()
	if !strings.Contains(output, "Project Settings") {
		t.Fatalf("expected injected output to contain project header, got %q", output)
	}
	if !strings.Contains(output, "Select:") {
		t.Fatalf("expected injected output to contain prompt, got %q", output)
	}
}

func TestPager_DisplayWithRuntimeUsesInjectedWriter(t *testing.T) {
	runtime := NewRuntime(strings.NewReader("q\n"), &bytes.Buffer{}, &bytes.Buffer{})
	out := runtime.Output().(*bytes.Buffer)

	pager := NewPagerWithRuntime(runtime)
	pager.pageSize = 2
	pager.Display("line1\nline2\nline3")

	output := out.String()
	if !strings.Contains(output, "line1") || !strings.Contains(output, "line2") {
		t.Fatalf("expected injected output to contain first page, got %q", output)
	}
	if !strings.Contains(output, "--more--") {
		t.Fatalf("expected injected output to contain pager prompt, got %q", output)
	}
	if strings.Contains(output, "line3") {
		t.Fatalf("expected quit input to stop before second page, got %q", output)
	}
}
