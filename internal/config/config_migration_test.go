package config

import (
	"testing"
)

func TestMigratePlanModeConfig_MaxParallelSteps(t *testing.T) {
	cfg := &PlanModeConfig{
		MaxParallelSteps: 5,
		MaxWorkers:       0, // 未設定
	}

	MigratePlanModeConfig(cfg)

	if cfg.MaxWorkers != 5 {
		t.Errorf("expected MaxWorkers = 5 (from MaxParallelSteps), got %d", cfg.MaxWorkers)
	}
}

func TestMigratePlanModeConfig_AutoRetry(t *testing.T) {
	cfg := &PlanModeConfig{
		AutoRetry: 15,
		MaxRetry:  0, // 未設定
	}

	MigratePlanModeConfig(cfg)

	if cfg.MaxRetry != 15 {
		t.Errorf("expected MaxRetry = 15 (from AutoRetry), got %d", cfg.MaxRetry)
	}
}

func TestMigratePlanModeConfig_NoOverwrite(t *testing.T) {
	// 新設定が既に設定されている場合は上書きしない
	cfg := &PlanModeConfig{
		MaxParallelSteps: 5,
		MaxWorkers:       10, // 既に設定済み
		AutoRetry:        15,
		MaxRetry:         20, // 既に設定済み
	}

	MigratePlanModeConfig(cfg)

	if cfg.MaxWorkers != 10 {
		t.Errorf("MaxWorkers should not be overwritten, expected 10, got %d", cfg.MaxWorkers)
	}
	if cfg.MaxRetry != 20 {
		t.Errorf("MaxRetry should not be overwritten, expected 20, got %d", cfg.MaxRetry)
	}
}

func TestMigratePlanModeConfig_DefaultValues(t *testing.T) {
	cfg := &PlanModeConfig{} // 全て未設定

	MigratePlanModeConfig(cfg)

	// デフォルト値の確認
	if cfg.MaxWorkers != 3 {
		t.Errorf("expected default MaxWorkers = 3, got %d", cfg.MaxWorkers)
	}
	if cfg.MaxRetry != 10 {
		t.Errorf("expected default MaxRetry = 10, got %d", cfg.MaxRetry)
	}
	if cfg.StepTimeout != 600 {
		t.Errorf("expected default StepTimeout = 600, got %d", cfg.StepTimeout)
	}
	if cfg.ConfirmLevel != "dangerous" {
		t.Errorf("expected default ConfirmLevel = 'dangerous', got '%s'", cfg.ConfirmLevel)
	}
}

func TestMigratePlanModeConfig_PartialDefaults(t *testing.T) {
	// 一部のみ設定されている場合
	cfg := &PlanModeConfig{
		MaxWorkers:  5,   // 設定済み
		MaxRetry:    0,   // 未設定 → デフォルト値
		StepTimeout: 300, // 設定済み
	}

	MigratePlanModeConfig(cfg)

	if cfg.MaxWorkers != 5 {
		t.Errorf("MaxWorkers should remain 5, got %d", cfg.MaxWorkers)
	}
	if cfg.MaxRetry != 10 {
		t.Errorf("expected default MaxRetry = 10, got %d", cfg.MaxRetry)
	}
	if cfg.StepTimeout != 300 {
		t.Errorf("StepTimeout should remain 300, got %d", cfg.StepTimeout)
	}
	if cfg.ConfirmLevel != "dangerous" {
		t.Errorf("expected default ConfirmLevel = 'dangerous', got '%s'", cfg.ConfirmLevel)
	}
}

func TestMigratePlanModeConfig_ConfirmLevelPreserved(t *testing.T) {
	tests := []struct {
		name     string
		level    string
		expected string
	}{
		{"all is preserved", "all", "all"},
		{"dangerous is preserved", "dangerous", "dangerous"},
		{"none is preserved", "none", "none"},
		{"empty gets default", "", "dangerous"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &PlanModeConfig{
				ConfirmLevel: tt.level,
			}
			MigratePlanModeConfig(cfg)
			if cfg.ConfirmLevel != tt.expected {
				t.Errorf("expected ConfirmLevel = '%s', got '%s'", tt.expected, cfg.ConfirmLevel)
			}
		})
	}
}

func TestMigratePlanModeConfig_MigrationPriority(t *testing.T) {
	// マイグレーション後にデフォルト値が適用されることを確認
	cfg := &PlanModeConfig{
		MaxParallelSteps: 7,
		AutoRetry:        12,
	}

	MigratePlanModeConfig(cfg)

	// マイグレーションが優先される
	if cfg.MaxWorkers != 7 {
		t.Errorf("expected MaxWorkers = 7 (from migration), got %d", cfg.MaxWorkers)
	}
	if cfg.MaxRetry != 12 {
		t.Errorf("expected MaxRetry = 12 (from migration), got %d", cfg.MaxRetry)
	}
}

func TestMigratePlanModeConfig_ParallelField(t *testing.T) {
	// Parallel フィールドはデフォルト false、設定されている場合は変更しない
	cfg := &PlanModeConfig{
		Parallel: true,
	}

	MigratePlanModeConfig(cfg)

	if !cfg.Parallel {
		t.Error("Parallel should remain true")
	}
}

func TestMigratePlanModeConfig_ModelFields(t *testing.T) {
	// モデルフィールドはデフォルト空文字（メインモデルを使用）
	cfg := &PlanModeConfig{
		SupervisorModel: "claude-3-opus",
		WorkerModel:     "",
	}

	MigratePlanModeConfig(cfg)

	// モデルフィールドは変更されない
	if cfg.SupervisorModel != "claude-3-opus" {
		t.Errorf("SupervisorModel should remain 'claude-3-opus', got '%s'", cfg.SupervisorModel)
	}
	if cfg.WorkerModel != "" {
		t.Errorf("WorkerModel should remain empty, got '%s'", cfg.WorkerModel)
	}
}
