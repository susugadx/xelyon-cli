package agent

import (
	"math"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/config"
)

func TestCalcProjectMapBudget_Auto_LargeContext(t *testing.T) {
	// gpt-5.4: 1M context × 2% = 20000
	agent := &Agent{CurrentModel: "gpt-5.4"}
	cfg := config.DefaultConfig()
	cfg.ProjectMap.ContextRatio = 0.02

	got := calcProjectMapBudget(agent, cfg, 50, 500)
	if got != 20000 {
		t.Errorf("calcProjectMapBudget() = %d, want 20000", got)
	}
}

func TestCalcProjectMapBudget_Auto_SmallContext(t *testing.T) {
	// unknown model → GetModelTokenLimit default 100K × 2% = 2000
	agent := &Agent{CurrentModel: "unknown-small-model"}
	cfg := config.DefaultConfig()
	cfg.ProjectMap.ContextRatio = 0.02

	got := calcProjectMapBudget(agent, cfg, 50, 500)
	if got != 2000 {
		t.Errorf("calcProjectMapBudget() = %d, want 2000 (100K default × 2%%)", got)
	}
}

func TestCalcProjectMapBudget_Auto_MediumContext(t *testing.T) {
	// claude-opus-4-6: 200K context × 2% = 4000
	agent := &Agent{CurrentModel: "claude-opus-4-6"}
	cfg := config.DefaultConfig()
	cfg.ProjectMap.ContextRatio = 0.02

	got := calcProjectMapBudget(agent, cfg, 50, 500)
	if got != 4000 {
		t.Errorf("calcProjectMapBudget() = %d, want 4000", got)
	}
}

func TestCalcProjectMapBudget_InvalidRatio(t *testing.T) {
	agent := &Agent{CurrentModel: "claude-opus-4-6"}
	cfg := config.DefaultConfig()

	tests := []struct {
		name  string
		ratio float64
	}{
		{"zero", 0},
		{"negative", -0.5},
		{"over_max", 0.25},
		{"nan", math.NaN()},
		{"inf", math.Inf(1)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg.ProjectMap.ContextRatio = tt.ratio
			got := calcProjectMapBudget(agent, cfg, 50, 500)
			// デフォルト 0.05 にフォールバック → 200K × 5% = 10000
			if got != 10000 {
				t.Errorf("calcProjectMapBudget() = %d, want 10000 (fallback to 0.05)", got)
			}
		})
	}
}

func TestCalcProjectMapBudget_RatioOverride(t *testing.T) {
	// claude-opus-4-6: 200K × 5% = 10000
	agent := &Agent{CurrentModel: "claude-opus-4-6"}
	cfg := config.DefaultConfig()
	cfg.ProjectMap.ContextRatio = 0.05

	got := calcProjectMapBudget(agent, cfg, 50, 500)
	if got != 10000 {
		t.Errorf("calcProjectMapBudget() = %d, want 10000", got)
	}
}

func TestCalcProjectMapBudget_SmallModelHasNoFixedFloor(t *testing.T) {
	// deepseek-chat: 128K context × 1% = 1280
	agent := &Agent{CurrentModel: "deepseek-chat"}
	cfg := config.DefaultConfig()
	cfg.ProjectMap.ContextRatio = 0.01

	got := calcProjectMapBudget(agent, cfg, 50, 500)
	if got != 1280 {
		t.Errorf("calcProjectMapBudget() = %d, want 1280", got)
	}
}

func TestEffectiveProjectMapContextRatio_AutoBoost(t *testing.T) {
	tests := []struct {
		name      string
		baseRatio float64
		fileCount int
		symbols   int
		wantRatio float64
	}{
		{name: "small repo keeps default", baseRatio: 0.02, fileCount: 80, symbols: 900, wantRatio: 0.02},
		{name: "medium repo boosts to three percent", baseRatio: 0.02, fileCount: 220, symbols: 1800, wantRatio: 0.03},
		{name: "large repo boosts to four percent", baseRatio: 0.02, fileCount: 420, symbols: 3500, wantRatio: 0.04},
		{name: "symbol-heavy repo boosts to four percent", baseRatio: 0.02, fileCount: 120, symbols: 4500, wantRatio: 0.04},
		{name: "user higher ratio is preserved", baseRatio: 0.05, fileCount: 420, symbols: 4500, wantRatio: 0.05},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := effectiveProjectMapContextRatio(tt.baseRatio, tt.fileCount, tt.symbols)
			if got != tt.wantRatio {
				t.Errorf("effectiveProjectMapContextRatio() = %v, want %v", got, tt.wantRatio)
			}
		})
	}
}

func TestCalcProjectMapBudget_AutoBoostForLargeRepo(t *testing.T) {
	// gpt-5.1: 400K context, large repo boosts 2% -> 4%
	agent := &Agent{CurrentModel: "gpt-5.1"}
	cfg := config.DefaultConfig()
	cfg.ProjectMap.ContextRatio = 0.02

	got := calcProjectMapBudget(agent, cfg, 430, 3200)
	if got != 16000 {
		t.Errorf("calcProjectMapBudget() = %d, want 16000", got)
	}
}
