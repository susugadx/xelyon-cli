package config

import "testing"

func TestRegistryFieldDisplayName(t *testing.T) {
	tests := []struct {
		path string
		want string
	}{
		{path: "default_provider", want: "default_provider"},
		{path: "compression.trigger_percent", want: "trigger_percent"},
		{path: "lsp.servers", want: "servers"},
	}

	for _, tt := range tests {
		if got := registryFieldDisplayName(tt.path); got != tt.want {
			t.Errorf("registryFieldDisplayName(%q) = %q, want %q", tt.path, got, tt.want)
		}
	}
}

func TestRegistryFieldType(t *testing.T) {
	if got := registryFieldType("default_provider"); got != FieldTypeSelect {
		t.Fatalf("registryFieldType(default_provider) = %v, want %v", got, FieldTypeSelect)
	}
	if got := registryFieldType("unknown.field.path"); got != FieldTypeString {
		t.Fatalf("registryFieldType(unknown.field.path) = %v, want %v", got, FieldTypeString)
	}
}

func TestRegistryFieldResolver_ProviderModelsDefaultUsesAdapterOverride(t *testing.T) {
	resolver := newRegistryFieldResolver(DefaultConfig())
	_, def := resolver.resolve("provider_models")

	defaultVal, ok := def.(map[string]ProviderModelConfig)
	if !ok {
		t.Fatalf("resolver.resolve(provider_models) default type = %T, want map[string]ProviderModelConfig", def)
	}
	if defaultVal != nil {
		t.Fatalf("resolver.resolve(provider_models) default = %#v, want nil", defaultVal)
	}
}

func TestBuildConfigRegistry(t *testing.T) {
	cfg := DefaultConfig()
	categories := BuildConfigRegistry(cfg)

	if len(categories) == 0 {
		t.Error("BuildConfigRegistry returned empty categories")
	}

	for _, cat := range categories {
		if cat.Name == "" {
			t.Error("Category has empty name")
		}
		if cat.DisplayName == "" {
			t.Errorf("Category %s has empty display name", cat.Name)
		}
		if cat.Icon == "" {
			t.Errorf("Category %s has empty icon", cat.Name)
		}
	}

	var providerCat *ConfigCategory
	for i := range categories {
		if categories[i].Name == "provider" {
			providerCat = &categories[i]
			break
		}
	}

	if providerCat == nil {
		t.Error("Provider category not found")
		return
	}

	fieldPaths := make(map[string]bool)
	for _, field := range providerCat.Fields {
		fieldPaths[field.Path] = true
	}

	expectedFields := []string{"default_provider", "default_model", "provider_models"}
	for _, expected := range expectedFields {
		if !fieldPaths[expected] {
			t.Errorf("Provider category missing field: %s", expected)
		}
	}

	var reviewCat *ConfigCategory
	for i := range categories {
		if categories[i].Name == "review" {
			reviewCat = &categories[i]
			break
		}
	}

	if reviewCat == nil {
		t.Error("Review category not found")
		return
	}

	reviewFields := make(map[string]bool)
	for _, field := range reviewCat.Fields {
		reviewFields[field.Path] = true
	}

	for _, expected := range []string{"review.provider", "review.model"} {
		if !reviewFields[expected] {
			t.Errorf("Review category missing field: %s", expected)
		}
	}
}
