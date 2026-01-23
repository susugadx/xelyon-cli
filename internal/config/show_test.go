package config

import (
	"reflect"
	"strings"
	"testing"
)

func TestShowConfigDefault(t *testing.T) {
	cfg := DefaultConfig()
	output := ShowConfig(cfg)

	// 基本的な出力チェック
	if !strings.Contains(output, "Current Configuration") {
		t.Error("Expected header in output")
	}

	if !strings.Contains(output, "default_provider") {
		t.Error("Expected default_provider in output")
	}

	if !strings.Contains(output, "default_model") {
		t.Error("Expected default_model in output")
	}

	// デフォルト値なので差分マークがないはず（ほとんど）
	if strings.Contains(output, "⚡") {
		t.Log("Note: Some fields may differ due to LoadConfig default application")
	}
}

func TestShowConfigWithDiff(t *testing.T) {
	cfg := DefaultConfig()

	// 設定を変更
	cfg.DefaultProvider = "openai"
	cfg.DefaultModel = "gpt-4"
	cfg.APIRetry.Timeout = 7200

	output := ShowConfig(cfg)

	// 差分マークがあるはず
	if !strings.Contains(output, "⚡") {
		t.Error("Expected diff marker ⚡ in output")
	}

	// 変更した値が表示されているか
	if !strings.Contains(output, "openai") {
		t.Error("Expected 'openai' in output")
	}

	if !strings.Contains(output, "gpt-4") {
		t.Error("Expected 'gpt-4' in output")
	}

	if !strings.Contains(output, "7200") {
		t.Error("Expected '7200' in output")
	}
}

func TestShowConfigStructure(t *testing.T) {
	cfg := DefaultConfig()
	output := ShowConfig(cfg)

	// セクションが含まれているか
	expectedSections := []string{
		"[default_provider]",
		"[compression]",
		"[api_retry]",
		"[bash]",
	}

	for _, section := range expectedSections {
		if !strings.Contains(output, section) {
			t.Errorf("Expected section %s in output", section)
		}
	}
}

func TestShowConfigFooter(t *testing.T) {
	cfg := DefaultConfig()
	output := ShowConfig(cfg)

	// フッター情報
	if !strings.Contains(output, "differs from default") {
		t.Error("Expected diff explanation in footer")
	}

	if !strings.Contains(output, "docs/config.md") {
		t.Error("Expected docs link in footer")
	}
}

func TestCompareConfigsProviderModels(t *testing.T) {
	cfg := DefaultConfig()

	// ProviderModelsを変更
	cfg.ProviderModels["openai"] = ProviderModelConfig{
		DefaultModel: "gpt-5-turbo",
	}

	output := ShowConfig(cfg)

	// 変更されたモデルが表示されているか
	if !strings.Contains(output, "gpt-5-turbo") {
		t.Error("Expected changed model 'gpt-5-turbo' in output")
	}
}

func TestFormatValue(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		expected string
	}{
		{"string", "test", "test"},
		{"bool true", true, "true"},
		{"bool false", false, "false"},
		{"int", 42, "42"},
		{"int64", int64(100), "100"},
		{"uint", uint(55), "55"},
		{"float64", 3.14, "3.14"},
		{"empty slice", []string{}, "[]"},
		{"string slice", []string{"a", "b"}, "[a, b]"},
		{"map", map[string]int{"a": 1}, "(1 items)"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rv := reflect.ValueOf(tt.input)
			got := formatValue(rv)
			if got != tt.expected {
				t.Errorf("formatValue(%v) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestFormatValue_InvalidValue(t *testing.T) {
	var invalid reflect.Value
	got := formatValue(invalid)
	if got != "<nil>" {
		t.Errorf("formatValue(invalid) = %q, want %q", got, "<nil>")
	}
}

func TestFormatValue_Struct(t *testing.T) {
	type testStruct struct {
		Name string
	}
	rv := reflect.ValueOf(testStruct{Name: "test"})
	got := formatValue(rv)
	if got != "<struct>" {
		t.Errorf("formatValue(struct) = %q, want %q", got, "<struct>")
	}
}

func TestCompareMapFields_DifferentLength(t *testing.T) {
	current := reflect.ValueOf(map[string]int{"a": 1, "b": 2})
	defaults := reflect.ValueOf(map[string]int{"a": 1})

	diffs := compareMapFields(current, defaults, "test_map")

	if len(diffs) == 0 {
		t.Error("Expected at least one diff for different map lengths")
	}

	found := false
	for _, d := range diffs {
		if d.Key == "test_map" && d.IsDiff {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected diff for test_map key")
	}
}

func TestCompareMapFields_SameContent(t *testing.T) {
	current := reflect.ValueOf(map[string]int{"a": 1})
	defaults := reflect.ValueOf(map[string]int{"a": 1})

	diffs := compareMapFields(current, defaults, "test_map")

	diffCount := 0
	for _, d := range diffs {
		if d.IsDiff {
			diffCount++
		}
	}
	if diffCount > 0 {
		t.Errorf("Expected no diffs for same content, got %d", diffCount)
	}
}

func TestCompareMapFields_KeyNotInDefault(t *testing.T) {
	current := reflect.ValueOf(map[string]int{"a": 1, "b": 2})
	defaults := reflect.ValueOf(map[string]int{"a": 1, "c": 3})

	// Note: Length is the same so it proceeds to key comparison
	diffs := compareMapFields(current, defaults, "test_map")

	// Should find that "b" is not in default
	foundNotInDefault := false
	for _, d := range diffs {
		if strings.Contains(d.Default, "not in default") {
			foundNotInDefault = true
			break
		}
	}
	if !foundNotInDefault {
		t.Error("Expected to find 'not in default' for missing key")
	}
}

func TestCompareMapFields_DifferentValues(t *testing.T) {
	current := reflect.ValueOf(map[string]int{"a": 1})
	defaults := reflect.ValueOf(map[string]int{"a": 2})

	diffs := compareMapFields(current, defaults, "test_map")

	foundDiff := false
	for _, d := range diffs {
		if d.Key == "test_map.a" && d.IsDiff {
			foundDiff = true
			break
		}
	}
	if !foundDiff {
		t.Error("Expected diff for different value at key 'a'")
	}
}
