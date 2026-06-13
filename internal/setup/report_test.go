package setup

import (
	"errors"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/lsp"
)

func withReportHooks(t *testing.T) {
	t.Helper()

	originalLookPath := lookPath
	originalGetwd := getwd
	originalDetectProjectLanguages := detectProjectLanguages
	originalLoadProjectConfig := loadProjectConfig
	originalResolveProjectRoot := resolveProjectRoot

	t.Cleanup(func() {
		lookPath = originalLookPath
		getwd = originalGetwd
		detectProjectLanguages = originalDetectProjectLanguages
		loadProjectConfig = originalLoadProjectConfig
		resolveProjectRoot = originalResolveProjectRoot
	})
}

func TestBuildReport_MissingProviderCredentialUsesDescriptorInstructions(t *testing.T) {
	withReportHooks(t)
	t.Setenv("OPENAI_API_KEY", "")

	loadProjectConfig = func(cwd string) (*config.ProjectConfig, error) {
		return nil, nil
	}
	resolveProjectRoot = func(cfg *config.Config, cwd string) (string, bool) {
		return "", false
	}
	lookPath = func(file string) (string, error) {
		return "/usr/bin/" + file, nil
	}

	report := BuildReport(Options{
		Config:   config.DefaultConfig(),
		CWD:      "/repo",
		Provider: "openai",
		Model:    "gpt-5.4",
	})

	if report.Provider.Ready {
		t.Fatal("provider should require credential setup")
	}
	if !strings.Contains(report.Provider.Detail, "OPENAI_API_KEY") {
		t.Fatalf("provider detail = %q, want OPENAI_API_KEY", report.Provider.Detail)
	}

	rendered := RenderString(report)
	for _, fragment := range []string{
		"Provider credential: missing OPENAI_API_KEY",
		"export OPENAI_API_KEY=your-api-key",
		"XELYON does not store API keys",
	} {
		if !strings.Contains(rendered, fragment) {
			t.Fatalf("rendered report missing %q:\n%s", fragment, rendered)
		}
	}
}

func TestBuildReport_ConfiguredProviderCredentialHidesSetupInstructions(t *testing.T) {
	withReportHooks(t)
	t.Setenv("OPENAI_API_KEY", "sk-test")

	loadProjectConfig = func(cwd string) (*config.ProjectConfig, error) {
		return nil, nil
	}
	resolveProjectRoot = func(cfg *config.Config, cwd string) (string, bool) {
		return "", false
	}
	lookPath = func(file string) (string, error) {
		return "/usr/bin/" + file, nil
	}

	report := BuildReport(Options{
		Config:   config.DefaultConfig(),
		CWD:      "/repo",
		Provider: "openai",
		Model:    "gpt-5.4",
	})

	if !report.Provider.Ready {
		t.Fatal("provider should be ready when OPENAI_API_KEY is set")
	}
	providerItem := findReportItem(report.Global, "provider")
	if providerItem.Status != "ok" || providerItem.Detail != "credential configured" {
		t.Fatalf("provider item = %+v, want ok credential configured", providerItem)
	}
	if providerItem.Instruction != "" {
		t.Fatalf("provider instruction = %q, want empty when credential is configured", providerItem.Instruction)
	}

	rendered := RenderString(report)
	if !strings.Contains(rendered, "Provider credential: credential configured") {
		t.Fatalf("rendered report missing configured credential status:\n%s", rendered)
	}
	if strings.Contains(rendered, "export OPENAI_API_KEY=your-api-key") {
		t.Fatalf("rendered report includes setup instruction for configured provider:\n%s", rendered)
	}
}

func TestBuildReport_ProjectRecommendationsShowDetectedMissingLSP(t *testing.T) {
	withReportHooks(t)
	root := t.TempDir()
	projectPath := filepath.Join(root, "xelyon.yaml")

	cfg := config.DefaultConfig()
	cfg.FinalChecks.Commands = []string{"make ci-check"}
	loadProjectConfig = func(cwd string) (*config.ProjectConfig, error) {
		if cwd != root {
			t.Fatalf("loadProjectConfig cwd = %q, want %q", cwd, root)
		}
		return &config.ProjectConfig{FilePath: projectPath}, nil
	}
	resolveProjectRoot = func(cfg *config.Config, cwd string) (string, bool) {
		return root, true
	}
	detectProjectLanguages = func(gotRoot string) ([]lsp.LanguageInfo, error) {
		if gotRoot != root {
			t.Fatalf("detectProjectLanguages root = %q, want %q", gotRoot, root)
		}
		return []lsp.LanguageInfo{{ServerKey: "go", FileCount: 2}}, nil
	}
	lookPath = func(file string) (string, error) {
		if file == "gopls" {
			return "", exec.ErrNotFound
		}
		return "/usr/bin/" + file, nil
	}

	report := BuildReport(Options{
		Config:   cfg,
		CWD:      root,
		Provider: "ollama",
		Model:    "qwen2.5-coder:7b",
	})

	xelyonItem := findReportItem(report.Project, "xelyon_yaml")
	if xelyonItem.Status != "ok" || xelyonItem.Detail != projectPath {
		t.Fatalf("xelyon_yaml item = %+v, want ok %s", xelyonItem, projectPath)
	}
	lspItem := findReportItem(report.Project, "lsp.go")
	if lspItem.Status != "todo" {
		t.Fatalf("lsp.go status = %q, want todo", lspItem.Status)
	}
	if !strings.Contains(lspItem.Detail, "2 file(s) detected") {
		t.Fatalf("lsp.go detail = %q, want detected file count", lspItem.Detail)
	}
	if !strings.Contains(lspItem.Instruction, "gopls") {
		t.Fatalf("lsp.go instruction = %q, want gopls install command", lspItem.Instruction)
	}
	finalChecks := findReportItem(report.Project, "final_checks")
	if finalChecks.Status != "ok" || finalChecks.Detail != "make ci-check" {
		t.Fatalf("final_checks item = %+v, want make ci-check", finalChecks)
	}
}

func TestBuildReport_LSPSkipInstallPromptSuppressesProjectRecommendations(t *testing.T) {
	withReportHooks(t)
	root := t.TempDir()

	cfg := config.DefaultConfig()
	cfg.LSP.SkipInstallPrompt = true
	loadProjectConfig = func(cwd string) (*config.ProjectConfig, error) {
		return &config.ProjectConfig{FilePath: filepath.Join(root, "xelyon.yaml")}, nil
	}
	resolveProjectRoot = func(cfg *config.Config, cwd string) (string, bool) {
		return root, true
	}
	detectProjectLanguages = func(root string) ([]lsp.LanguageInfo, error) {
		return nil, errors.New("language detection should be skipped")
	}
	lookPath = func(file string) (string, error) {
		return "/usr/bin/" + file, nil
	}

	report := BuildReport(Options{
		Config:   cfg,
		CWD:      root,
		Provider: "ollama",
	})

	lspItem := findReportItem(report.Project, "lsp")
	if lspItem.Status != "skip" {
		t.Fatalf("lsp status = %q, want skip", lspItem.Status)
	}
	if !strings.Contains(lspItem.Detail, "lsp.skip_install_prompt is true") {
		t.Fatalf("lsp detail = %q, want skip_install_prompt detail", lspItem.Detail)
	}
}

func TestBuildReport_LSPRecommendationsUseProjectRootWithoutXelyonYAML(t *testing.T) {
	withReportHooks(t)
	root := t.TempDir()
	cwd := filepath.Join(root, "nested")

	cfg := config.DefaultConfig()
	loadProjectConfig = func(cwd string) (*config.ProjectConfig, error) {
		return nil, nil
	}
	resolveProjectRoot = func(cfg *config.Config, gotCWD string) (string, bool) {
		if gotCWD != cwd {
			t.Fatalf("resolveProjectRoot cwd = %q, want %q", gotCWD, cwd)
		}
		return root, true
	}
	detectProjectLanguages = func(gotRoot string) ([]lsp.LanguageInfo, error) {
		if gotRoot != root {
			t.Fatalf("detectProjectLanguages root = %q, want %q", gotRoot, root)
		}
		return []lsp.LanguageInfo{{ServerKey: "go", FileCount: 1}}, nil
	}
	lookPath = func(file string) (string, error) {
		if file == "gopls" {
			return "", exec.ErrNotFound
		}
		return "/usr/bin/" + file, nil
	}

	report := BuildReport(Options{
		Config:   cfg,
		CWD:      cwd,
		Provider: "ollama",
	})

	if xelyonItem := findReportItem(report.Project, "xelyon_yaml"); xelyonItem.Status != "todo" {
		t.Fatalf("xelyon_yaml item = %+v, want todo", xelyonItem)
	}
	lspItem := findReportItem(report.Project, "lsp.go")
	if lspItem.Status != "todo" {
		t.Fatalf("lsp.go item = %+v, want todo recommendation", lspItem)
	}
}

func findReportItem(items []Item, key string) Item {
	for _, item := range items {
		if item.Key == key {
			return item
		}
	}
	return Item{}
}
