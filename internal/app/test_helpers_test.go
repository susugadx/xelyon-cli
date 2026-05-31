package app

import (
	"context"
	"os"
	"testing"

	"github.com/fatih/color"
	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/config"
)

func disableColors(t *testing.T) {
	t.Helper()
	prev := color.NoColor
	color.NoColor = true
	t.Cleanup(func() {
		color.NoColor = prev
	})
}

func newProjectMapDisabledConfig() *config.Config {
	cfg := config.DefaultConfig()
	cfg.ProjectMap.Enabled = false
	cfg.MCP.Enabled = false
	cfg.LSP.Enabled = false
	cfg.Compression.Enabled = false
	return cfg
}

func withTempWorkdir(t *testing.T) string {
	t.Helper()
	tmpDir := t.TempDir()
	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd() error = %v", err)
	}
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("Chdir() error = %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(oldWd)
	})
	return tmpDir
}

type mockProvider struct {
	name string
}

func (m *mockProvider) Name() string {
	return m.name
}

func (m *mockProvider) SupportsImages() bool {
	return false
}

func (m *mockProvider) IsFunctionCallingEnabled() bool {
	return true
}

func (m *mockProvider) ChatWithTools(context.Context, string, []api.Message, string) (string, error) {
	return "mock response", nil
}

func (m *mockProvider) ChatWithImage(context.Context, string, []api.Message, string, *api.ImageData, string) (string, error) {
	return "mock image response", nil
}
