package agent

import (
	"bytes"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/tools/common"
	"github.com/susugadx/xelyon-cli/internal/ui"
)

func TestCheckRipgrepAvailability_NoRg(t *testing.T) {
	common.ResetRipgrepAvailabilityForTest()
	t.Cleanup(common.ResetRipgrepAvailabilityForTest)

	t.Setenv("PATH", t.TempDir())

	var out bytes.Buffer
	runtime := NewAgentRuntimeWithConfig(config.DefaultConfig())
	runtime.UI = ui.NewRuntime(strings.NewReader(""), &out, io.Discard)
	agent := &Agent{Runtime: runtime}

	checkRipgrepAvailability(agent)

	if !strings.Contains(out.String(), "ripgrep (rg) not found") {
		t.Fatalf("expected ripgrep warning, got: %s", out.String())
	}
}

func TestCheckRipgrepAvailability_WithRg(t *testing.T) {
	common.ResetRipgrepAvailabilityForTest()
	t.Cleanup(common.ResetRipgrepAvailabilityForTest)

	rgPath, err := exec.LookPath("rg")
	if err != nil {
		t.Skip("ripgrep (rg) not available")
	}
	t.Setenv("PATH", filepath.Dir(rgPath))

	var out bytes.Buffer
	runtime := NewAgentRuntimeWithConfig(config.DefaultConfig())
	runtime.UI = ui.NewRuntime(strings.NewReader(""), &out, io.Discard)
	agent := &Agent{Runtime: runtime}

	checkRipgrepAvailability(agent)

	if out.Len() != 0 {
		t.Fatalf("expected no output when rg exists, got: %s", out.String())
	}
}

func TestInjectProjectMap_AppendsProjectMap(t *testing.T) {
	common.ResetRipgrepAvailabilityForTest()
	t.Cleanup(common.ResetRipgrepAvailabilityForTest)

	if _, err := exec.LookPath("rg"); err != nil {
		t.Skip("ripgrep (rg) not available")
	}

	root := t.TempDir()
	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(root); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(oldwd) })

	if err := os.WriteFile(filepath.Join(root, "main.go"), []byte("package main\n\nfunc Build() {}\n"), 0644); err != nil {
		t.Fatal(err)
	}

	var out bytes.Buffer
	cfg := config.DefaultConfig()
	runtime := NewAgentRuntimeWithConfig(cfg)
	runtime.UI = ui.NewRuntime(strings.NewReader(""), &out, io.Discard)
	agent := &Agent{
		Runtime:      runtime,
		SystemPrompt: "base prompt",
	}

	injectProjectMap(agent)

	if !strings.Contains(agent.SystemPrompt, "## Project Map") {
		t.Fatalf("expected Project Map in system prompt, got: %s", agent.SystemPrompt)
	}
	if agent.projectMapFileCount == 0 {
		t.Fatal("expected projectMapFileCount to be populated")
	}
	if agent.projectMapSymbolCount == 0 {
		t.Fatal("expected projectMapSymbolCount to be populated")
	}
	if !strings.Contains(out.String(), "Project map loaded") {
		t.Fatalf("expected load message, got: %s", out.String())
	}
}

func TestExtractProjectMapSection_WithTrailingSection(t *testing.T) {
	prompt := "base\n\n## Project Map\n📂 src/\n└── 📄 main.go (10 lines)\n\n## Project Context:\nSome context"
	section := extractProjectMapSection(prompt)
	if strings.Contains(section, "Project Context") {
		t.Fatalf("section should not include trailing content:\n%s", section)
	}
	if !strings.Contains(section, "main.go") {
		t.Fatalf("section should include Project Map content:\n%s", section)
	}
}

func TestExtractProjectMapSection_AtEnd(t *testing.T) {
	prompt := "base\n\n## Project Map\n📂 src/\n└── 📄 main.go (10 lines)"
	section := extractProjectMapSection(prompt)
	if !strings.Contains(section, "main.go") {
		t.Fatalf("section should include Project Map content:\n%s", section)
	}
}

func TestExtractProjectMapSection_NotPresent(t *testing.T) {
	prompt := "base prompt without project map"
	section := extractProjectMapSection(prompt)
	if section != "" {
		t.Fatalf("expected empty string, got: %s", section)
	}
}
