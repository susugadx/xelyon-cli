package tui

import (
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/config"
)

func TestConfigScreen_ViewRenders(t *testing.T) {
	m := newConfigTestModel()
	m.width = 120
	m.height = 30

	view := m.View()
	if view == "" {
		t.Fatal("View returned empty string")
	}
	if !strings.Contains(view, "Configuration") {
		t.Fatal("View should contain 'Configuration'")
	}
	if !strings.Contains(view, "Provider") {
		t.Fatal("View should contain 'Provider'")
	}
}

func TestConfigScreen_AllCategoriesPresent(t *testing.T) {
	cfg := config.DefaultConfig()
	categories := config.BuildConfigRegistry(cfg)

	expectedCats := []string{
		"provider", "general", "execution", "compression",
		"paste", "project_map", "lsp", "output",
		"web_search", "sub_agent", "mcp", "hooks",
	}

	catNames := make(map[string]bool)
	for _, cat := range categories {
		catNames[cat.Name] = true
	}

	for _, name := range expectedCats {
		if !catNames[name] {
			t.Errorf("category %q missing from registry", name)
		}
	}
}

func TestConfigScreen_LSPServersInRegistry(t *testing.T) {
	cfg := config.DefaultConfig()
	categories := config.BuildConfigRegistry(cfg)

	found := false
	for _, cat := range categories {
		if cat.Name == "lsp" {
			for _, f := range cat.Fields {
				if f.Path == "lsp.servers" {
					found = true
					if f.FieldType != config.FieldTypeStructMap {
						t.Errorf("lsp.servers type = %v, want FieldTypeStructMap", f.FieldType)
					}
					break
				}
			}
			break
		}
	}
	if !found {
		t.Fatal("lsp.servers not found in generated registry")
	}
}
