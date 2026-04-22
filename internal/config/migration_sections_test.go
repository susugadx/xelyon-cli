package config

import "testing"

func TestParseYAMLRootMap_InvalidYAML(t *testing.T) {
	if raw := parseYAMLRootMap([]byte("general: [")); raw != nil {
		t.Fatalf("parseYAMLRootMap(invalid) = %v, want nil", raw)
	}
}

func TestApplyExecutionModeMigration_ExecutionSectionWins(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Execution.Mode = string(ExecutionBalanced)

	raw := map[string]interface{}{
		"execution": map[string]interface{}{
			"mode": string(ExecutionBalanced),
		},
		"tool_confirm": map[string]interface{}{
			"auto_approve_medium": true,
		},
	}
	applyExecutionModeMigration(raw, cfg)

	if got := cfg.Execution.Mode; got != string(ExecutionBalanced) {
		t.Fatalf("Execution.Mode = %q, want %q", got, string(ExecutionBalanced))
	}
}

func TestApplyCompressionKeyMigration_ThresholdMustBePositive(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Compression.TriggerPercent = 80

	raw := map[string]interface{}{
		"compression": map[string]interface{}{
			"threshold_percent": 0,
		},
	}
	applyCompressionKeyMigration(raw, cfg)

	if got := cfg.Compression.TriggerPercent; got != 80 {
		t.Fatalf("Compression.TriggerPercent = %d, want %d", got, 80)
	}
}
