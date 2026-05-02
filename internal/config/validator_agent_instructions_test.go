package config

import "testing"

func TestValidateConfig_AgentInstructionsMode(t *testing.T) {
	cfg := DefaultConfig()
	cfg.AgentInstructions.Project.Mode = "invalid"

	result := ValidateConfig(cfg)

	found := false
	for _, issue := range result.Issues {
		if issue.Field != "agent_instructions.project.mode" {
			continue
		}
		found = true
		if issue.Severity != ValidationSeverityWarning {
			t.Fatalf("Severity = %s, want warning", issue.Severity)
		}
	}
	if !found {
		t.Fatal("expected issue for agent_instructions.project.mode")
	}
}

func TestValidateConfig_AgentInstructionsByteLimits(t *testing.T) {
	cfg := DefaultConfig()
	cfg.AgentInstructions.MaxFileBytes = 0
	cfg.AgentInstructions.MaxTotalBytes = -1

	result := ValidateConfig(cfg)

	var foundFile bool
	var foundTotal bool
	for _, issue := range result.Issues {
		switch issue.Field {
		case "agent_instructions.max_file_bytes":
			foundFile = true
		case "agent_instructions.max_total_bytes":
			foundTotal = true
		}
	}
	if !foundFile {
		t.Fatal("expected issue for agent_instructions.max_file_bytes")
	}
	if !foundTotal {
		t.Fatal("expected issue for agent_instructions.max_total_bytes")
	}
}

func TestApplyDefaults_AgentInstructionsFallbacks(t *testing.T) {
	cfg := DefaultConfig()
	cfg.AgentInstructions.Project.Mode = ""
	cfg.AgentInstructions.Project.Files = nil
	cfg.AgentInstructions.Global.Files = nil
	cfg.AgentInstructions.MaxFileBytes = 0
	cfg.AgentInstructions.MaxTotalBytes = -100

	applyDefaults(cfg)

	if cfg.AgentInstructions.Project.Mode != "fallback" {
		t.Fatalf("Mode = %q, want fallback", cfg.AgentInstructions.Project.Mode)
	}
	if len(cfg.AgentInstructions.Project.Files) == 0 {
		t.Fatal("Project.Files should be defaulted")
	}
	if len(cfg.AgentInstructions.Global.Files) == 0 {
		t.Fatal("Global.Files should be defaulted")
	}
	if cfg.AgentInstructions.MaxFileBytes != 20000 {
		t.Fatalf("MaxFileBytes = %d, want 20000", cfg.AgentInstructions.MaxFileBytes)
	}
	if cfg.AgentInstructions.MaxTotalBytes != 60000 {
		t.Fatalf("MaxTotalBytes = %d, want 60000", cfg.AgentInstructions.MaxTotalBytes)
	}
}
