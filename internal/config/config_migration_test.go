package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfig_LegacyHooksMigratesToFinalChecks(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	configDir := filepath.Join(tmpDir, ".xelyon")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("Failed to create config dir: %v", err)
	}

	configPath := filepath.Join(configDir, "config.yaml")
	configYAML := `hooks:
  on_completion:
    - go test ./...
  timeout: 120
`
	if err := os.WriteFile(configPath, []byte(configYAML), 0600); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	if len(cfg.FinalChecks.Commands) != 1 || cfg.FinalChecks.Commands[0] != "go test ./..." {
		t.Fatalf("FinalChecks.Commands = %v, want [go test ./...]", cfg.FinalChecks.Commands)
	}
	if cfg.FinalChecks.Timeout != 120 {
		t.Fatalf("FinalChecks.Timeout = %d, want 120", cfg.FinalChecks.Timeout)
	}
}

func TestLoadConfig_FinalChecksWinsOverLegacyVerificationAndHooks(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	configDir := filepath.Join(tmpDir, ".xelyon")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("Failed to create config dir: %v", err)
	}

	configPath := filepath.Join(configDir, "config.yaml")
	configYAML := `final_checks:
  commands:
    - make test
  timeout: 300
verification:
  commands:
    - legacy verify
  timeout: 200
hooks:
  on_completion:
    - go test ./...
  timeout: 120
`
	if err := os.WriteFile(configPath, []byte(configYAML), 0600); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	if len(cfg.FinalChecks.Commands) != 1 || cfg.FinalChecks.Commands[0] != "make test" {
		t.Fatalf("FinalChecks.Commands = %v, want [make test]", cfg.FinalChecks.Commands)
	}
	if cfg.FinalChecks.Timeout != 300 {
		t.Fatalf("FinalChecks.Timeout = %d, want 300", cfg.FinalChecks.Timeout)
	}
}
