package config

import (
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestYamlMarshalIncludesAllSections(t *testing.T) {
	cfg := DefaultConfig()
	data, err := yaml.Marshal(cfg)
	if err != nil {
		t.Fatal(err)
	}
	out := string(data)

	sections := []string{
		"repomap:", "thinking:", "tool_confirm:", "streaming:",
		"web_search:", "diff:", "output:", "general:", "compression:",
		"backup:", "loop_detection:", "api_retry:", "prompt_cache:",
		"paste:", "bash:", "code_health:", "git_stage:", "plan_mode:",
		"lsp:", "openai:",
	}

	for _, s := range sections {
		if !strings.Contains(out, s) {
			t.Errorf("section %q is missing from yaml output", s)
		}
	}
}

func TestYamlMarshalRepoMapEnabledFalse(t *testing.T) {
	cfg := DefaultConfig()
	// DefaultConfig now has Enabled: false
	if cfg.RepoMap.Enabled {
		t.Error("DefaultConfig().RepoMap.Enabled should be false")
	}

	data, err := yaml.Marshal(cfg)
	if err != nil {
		t.Fatal(err)
	}
	out := string(data)

	// repomap セクション内に enabled: false があること
	lines := strings.Split(out, "\n")
	inRepomap := false
	found := false
	for _, line := range lines {
		if strings.HasPrefix(line, "repomap:") {
			inRepomap = true
			continue
		}
		if inRepomap {
			trimmed := strings.TrimSpace(line)
			if trimmed == "enabled: false" {
				found = true
				break
			}
			// repomap セクション外に出たら終了
			if !strings.HasPrefix(line, " ") && !strings.HasPrefix(line, "\t") && line != "" {
				break
			}
		}
	}
	if !found {
		t.Error("repomap.enabled: false not found in yaml output")
	}
}

func TestYamlRoundTripPreservesFalse(t *testing.T) {
	// false に設定して Save → Load しても false のままであること
	cfg := DefaultConfig()
	cfg.RepoMap.Enabled = false
	cfg.Thinking.Enabled = false

	data, err := yaml.Marshal(cfg)
	if err != nil {
		t.Fatal(err)
	}

	var loaded Config
	if err := yaml.Unmarshal(data, &loaded); err != nil {
		t.Fatal(err)
	}

	if loaded.RepoMap.Enabled {
		t.Error("RepoMap.Enabled should be false after round-trip")
	}
	if loaded.Thinking.Enabled {
		t.Error("Thinking.Enabled should be false after round-trip")
	}
}
