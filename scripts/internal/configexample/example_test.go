package configexample

import (
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/config"
	"gopkg.in/yaml.v3"
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
agent_instructions:
  include_local_files: false
  project:
    mode: fallback
    files:
      - AGENTS.md
    include_gitignored: false
    hidden: true
  global:
    enabled: false
    files:
      - ~/.xelyon/AGENTS.md
    hidden: true
provider_models:
  openai:
    default_model: gpt-5.4
thinking:
  enabled: true
`)

	filtered, err := filterInternalFields(input)
	if err != nil {
		t.Fatalf("filterInternalFields returned error: %v", err)
	}

	text := string(filtered)
	for _, unexpected := range []string{"internal_only:", "hidden:", "servers: {}"} {
		if strings.Contains(text, unexpected) {
			t.Fatalf("unexpected field remained after filtering: %q in %s", unexpected, text)
		}
	}
	for _, unexpectedTopLevel := range []string{"loop_detection", "thinking"} {
		if yamlHasTopLevelKey(t, filtered, unexpectedTopLevel) {
			t.Fatalf("unexpected top-level section remained after filtering: %q in %s", unexpectedTopLevel, text)
		}
	}

	var out map[string]interface{}
	if err := yaml.Unmarshal(filtered, &out); err != nil {
		t.Fatalf("unmarshal filtered yaml: %v", err)
	}
	if _, exists := out["loop_detection"]; exists {
		t.Fatal("loop_detection should be filtered out")
	}
	if _, exists := out["thinking"]; exists {
		t.Fatal("thinking should be filtered out")
	}

	lsp, ok := out["lsp"].(map[string]interface{})
	if !ok {
		t.Fatalf("lsp section missing or invalid: %#v", out["lsp"])
	}
	if _, exists := lsp["servers"]; exists {
		t.Fatal("lsp.servers should be omitted by example policy")
	}

	agentInstructions, ok := out["agent_instructions"].(map[string]interface{})
	if !ok {
		t.Fatalf("agent_instructions section missing or invalid: %#v", out["agent_instructions"])
	}
	project, ok := agentInstructions["project"].(map[string]interface{})
	if !ok {
		t.Fatalf("agent_instructions.project missing or invalid: %#v", agentInstructions["project"])
	}
	if _, exists := project["hidden"]; exists {
		t.Fatal("agent_instructions.project.hidden should be filtered")
	}
}

func TestFilterInternalFieldsGolden(t *testing.T) {
	input := []byte(`
general:
  ui_language: auto
  tool_loop_limit: 1
lsp:
  enabled: true
  skip_install_prompt: false
  servers:
    gopls:
      command: gopls
provider_models:
  openai:
    default_model: gpt-5.4
`)
	filtered, err := filterInternalFields(input)
	if err != nil {
		t.Fatalf("filterInternalFields returned error: %v", err)
	}

	expected := `general:
    ui_language: auto
lsp:
    enabled: true
    skip_install_prompt: false
provider_models:
    openai:
        default_model: gpt-5.4
`
	if string(filtered) != expected {
		t.Fatalf("unexpected filtered yaml\n--- got ---\n%s--- want ---\n%s", string(filtered), expected)
	}
}

func TestAddComments(t *testing.T) {
	input := "general:\n    ui_language: auto\nagent_instructions:\n    project:\n        mode: always\n"
	output := addComments(input)

	for _, expected := range []string{
		"# 一般設定",
		"# 表示言語（auto, ja, en）",
		"general:",
		"    ui_language: auto",
		"# Agent Instructions 設定",
		"    project:",
		"        # project-local guidance の読み込みモード（通常は always / off。fallback は legacy 互換）",
		"        mode: always",
	} {
		if !strings.Contains(output, expected) {
			t.Fatalf("expected output to contain %q, got %s", expected, output)
		}
	}
}

func TestAddCommentsSeparatesTopLevelSections(t *testing.T) {
	output := addComments("general:\n    ui_language: auto\ncompression:\n    enabled: true\n")

	if !strings.Contains(output, "ui_language: auto\n\n# ============================================================") {
		t.Fatalf("expected blank line before next section, got %s", output)
	}
	if strings.Contains(output, "# 一般設定\n\n# ============================================================") {
		t.Fatalf("unexpected blank line inside section header, got %s", output)
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
		"# デフォルトで使用するモデル",
		"# プロバイダーごとのモデル設定",
		"# raw output artifact store root path（env: XELYON_RAW_OUTPUT_ARTIFACT_ROOT）",
		"# root: /absolute/path/to/rawoutputs",
		"provider: gemini",
		"lsp:",
		"agent_instructions:",
		"project:",
		"# project-local guidance ファイル候補（basename は root→cwd / root→入力参照 path の scoped chain、/ を含む path は root 相対 explicit file。既定は AGENTS.md）",
		"mode: always",
	} {
		if !strings.Contains(text, expected) {
			t.Fatalf("expected generated example to contain %q", expected)
		}
	}
	for _, unexpected := range []string{"servers: {}"} {
		if strings.Contains(text, unexpected) {
			t.Fatalf("unexpected internal section in generated example: %q", unexpected)
		}
	}
	for _, unexpectedTopLevel := range []string{"loop_detection", "responses", "thinking"} {
		if yamlHasTopLevelKey(t, output, unexpectedTopLevel) {
			t.Fatalf("unexpected internal top-level section in generated example: %q", unexpectedTopLevel)
		}
	}
	if strings.Contains(text, "\n        root: /absolute/path/to/rawoutputs") {
		t.Fatal("raw_output_artifacts.root should be a commented example, not an active config value")
	}
}

func yamlHasTopLevelKey(t *testing.T, data []byte, key string) bool {
	t.Helper()

	var out map[string]interface{}
	if err := yaml.Unmarshal(data, &out); err != nil {
		t.Fatalf("unmarshal yaml: %v", err)
	}
	_, exists := out[key]
	return exists
}

func TestGenerateExampleFileKeepsOpenAISubscriptionMaxOutputZero(t *testing.T) {
	output, err := GenerateExampleFile(config.DefaultConfig())
	if err != nil {
		t.Fatalf("GenerateExampleFile returned error: %v", err)
	}

	text := string(output)
	expected := "    openai_subscription:\n        default_model: gpt-5.5\n        max_output_tokens: 0\n"
	if !strings.Contains(text, expected) {
		t.Fatalf("expected generated example to contain openai_subscription explicit max_output_tokens zero, got:\n%s", text)
	}
}
