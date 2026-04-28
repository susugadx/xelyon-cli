package configgen

import (
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/config"
)

func TestFilterInternalFields(t *testing.T) {
	input := []byte(`
loop_detection:
  enabled: true
lsp:
  enabled: true
  skip_install_prompt: false
  servers: {}
  internal_only: true
output:
  max_lines: 5
  hidden: true
provider_models:
  openai:
    default_model: gpt-5.4
thinking:
  enabled: true
`)

	filtered, err := FilterInternalFields(input)
	if err != nil {
		t.Fatalf("FilterInternalFields returned error: %v", err)
	}

	text := string(filtered)
	for _, unexpected := range []string{"loop_detection:", "thinking:", "internal_only:", "hidden:", "servers: {}"} {
		if strings.Contains(text, unexpected) {
			t.Fatalf("unexpected field remained after filtering: %q in %s", unexpected, text)
		}
	}
	for _, expected := range []string{"lsp:", "enabled: true", "provider_models:"} {
		if !strings.Contains(text, expected) {
			t.Fatalf("expected field missing after filtering: %q in %s", expected, text)
		}
	}
}

func TestAddComments(t *testing.T) {
	input := "general:\n    ui_language: auto\n"
	output := AddComments(input)

	for _, expected := range []string{
		"# 一般設定",
		"# 表示言語（auto, ja, en）",
		"general:",
		"    ui_language: auto",
	} {
		if !strings.Contains(output, expected) {
			t.Fatalf("expected output to contain %q, got %s", expected, output)
		}
	}
}

func TestGenerateExampleFile(t *testing.T) {
	output, err := GenerateExampleFile(config.DefaultConfig())
	if err != nil {
		t.Fatalf("GenerateExampleFile returned error: %v", err)
	}

	text := string(output)
	for _, expected := range []string{
		"# XELYON CLI 設定例",
		"provider: gemini",
		"lsp:",
	} {
		if !strings.Contains(text, expected) {
			t.Fatalf("expected generated example to contain %q", expected)
		}
	}
	for _, unexpected := range []string{"loop_detection:", "responses:", "thinking:", "servers: {}"} {
		if strings.Contains(text, unexpected) {
			t.Fatalf("unexpected internal section in generated example: %q", unexpected)
		}
	}
}
