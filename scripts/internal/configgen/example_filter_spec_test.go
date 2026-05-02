package configgen

import (
	"testing"

	"github.com/susugadx/xelyon-cli/internal/config"
)

func TestBuildExampleFilterSpec(t *testing.T) {
	spec := buildExampleFilterSpec(Sections)
	if len(spec.allowedSections) == 0 {
		t.Fatal("expected allowed sections in filter spec")
	}

	lsp, ok := spec.sectionFilters["lsp"]
	if !ok {
		t.Fatal("expected lsp filter spec")
	}
	if lsp.mode != ExampleFilterModeFields {
		t.Fatalf("unexpected lsp filter mode: %q", lsp.mode)
	}
	if !lsp.omittedFields["servers"] {
		t.Fatal("expected lsp.servers omitted in filter spec")
	}

	providerModels, ok := spec.sectionFilters["provider_models"]
	if !ok {
		t.Fatal("expected provider_models filter spec")
	}
	if providerModels.mode != ExampleFilterModeKeepAll {
		t.Fatalf("unexpected provider_models mode: %q", providerModels.mode)
	}
}

func TestSectionHasOnlyMapLikeFieldTypes(t *testing.T) {
	if !sectionHasOnlyMapLikeFieldTypes(map[string]string{"a": "map", "b": "structmap"}) {
		t.Fatal("expected map-like types to return true")
	}
	if sectionHasOnlyMapLikeFieldTypes(map[string]string{"a": "string"}) {
		t.Fatal("expected non-map-like types to return false")
	}
	if sectionHasOnlyMapLikeFieldTypes(nil) {
		t.Fatal("expected nil map to return false")
	}
}

func TestApplyExampleOverrides_UsesSectionMetadata(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.LSP.Servers = map[string]config.LSPServerConfig{
		"gopls": {Command: "gopls"},
	}
	cfg.WebSearch.Provider = "openai"

	applyExampleOverrides(cfg)

	if cfg.LSP.Servers != nil {
		t.Fatalf("LSP.Servers should be overridden to nil: %#v", cfg.LSP.Servers)
	}
	if cfg.WebSearch.Provider != "gemini" {
		t.Fatalf("WebSearch.Provider = %q, want gemini", cfg.WebSearch.Provider)
	}
}
