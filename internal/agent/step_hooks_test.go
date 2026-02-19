package agent

import (
	"testing"

	"github.com/susugadx/xelyon-cli/internal/config"
)

func TestExpandStepTemplate(t *testing.T) {
	tests := []struct {
		name            string
		cmd             string
		stepID          int
		stepDescription string
		stepStatus      string
		want            string
	}{
		{
			name:            "all_variables",
			cmd:             "git commit -m 'Step {{step_id}}: {{step_description}} ({{step_status}})'",
			stepID:          3,
			stepDescription: "フックの実装",
			stepStatus:      "completed",
			want:            "git commit -m 'Step 3: フックの実装 (completed)'",
		},
		{
			name:            "step_id_only",
			cmd:             "echo step {{step_id}} done",
			stepID:          1,
			stepDescription: "anything",
			stepStatus:      "completed",
			want:            "echo step 1 done",
		},
		{
			name:            "no_variables",
			cmd:             "make test",
			stepID:          2,
			stepDescription: "test step",
			stepStatus:      "completed",
			want:            "make test",
		},
		{
			name:            "multiple_occurrences",
			cmd:             "echo {{step_id}} && echo {{step_id}}",
			stepID:          5,
			stepDescription: "desc",
			stepStatus:      "completed",
			want:            "echo 5 && echo 5",
		},
		{
			name:            "description_with_spaces",
			cmd:             "git add -A && git commit -m '{{step_description}}'",
			stepID:          2,
			stepDescription: "Add new feature",
			stepStatus:      "completed",
			want:            "git add -A && git commit -m 'Add new feature'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := expandStepTemplate(tt.cmd, tt.stepID, tt.stepDescription, tt.stepStatus)
			if got != tt.want {
				t.Errorf("expandStepTemplate() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestRunStepCompleteHooks_NoHooks(t *testing.T) {
	a := &Agent{}
	cfg := config.GetGlobalConfig()
	cfg.Hooks.OnStepComplete = nil
	config.SetGlobalConfig(cfg)

	needsContinue, feedback := a.runStepCompleteHooks(1, "test step", "completed")
	if needsContinue {
		t.Error("expected needsContinue=false when no hooks configured")
	}
	if feedback != "" {
		t.Errorf("expected empty feedback, got %q", feedback)
	}
}

func TestRunStepCompleteHooks_Success(t *testing.T) {
	a := &Agent{}
	cfg := config.GetGlobalConfig()
	cfg.Hooks.OnStepComplete = []string{"echo 'step {{step_id}} done'"}
	cfg.Hooks.Timeout = 10
	config.SetGlobalConfig(cfg)
	t.Cleanup(func() {
		cfg2 := config.GetGlobalConfig()
		cfg2.Hooks.OnStepComplete = nil
		config.SetGlobalConfig(cfg2)
	})

	needsContinue, feedback := a.runStepCompleteHooks(2, "テストステップ", "completed")
	if needsContinue {
		t.Errorf("expected needsContinue=false, got true; feedback=%q", feedback)
	}
	if feedback != "" {
		t.Errorf("expected empty feedback, got %q", feedback)
	}
}

func TestRunStepCompleteHooks_Failure(t *testing.T) {
	a := &Agent{}
	cfg := config.GetGlobalConfig()
	cfg.Hooks.OnStepComplete = []string{"exit 1"}
	cfg.Hooks.Timeout = 10
	config.SetGlobalConfig(cfg)
	t.Cleanup(func() {
		cfg2 := config.GetGlobalConfig()
		cfg2.Hooks.OnStepComplete = nil
		config.SetGlobalConfig(cfg2)
	})

	needsContinue, feedback := a.runStepCompleteHooks(3, "failing step", "completed")
	if !needsContinue {
		t.Error("expected needsContinue=true when hook fails")
	}
	if feedback == "" {
		t.Error("expected non-empty feedback when hook fails")
	}
}

func TestRunStepCompleteHooks_TemplateExpanded(t *testing.T) {
	// テンプレート変数がコマンドに正しく展開されることを確認
	a := &Agent{}
	cfg := config.GetGlobalConfig()
	// step_id が展開されていれば "exit 0" (成功) になるコマンド
	cfg.Hooks.OnStepComplete = []string{"test '{{step_id}}' = '7'"}
	cfg.Hooks.Timeout = 10
	config.SetGlobalConfig(cfg)
	t.Cleanup(func() {
		cfg2 := config.GetGlobalConfig()
		cfg2.Hooks.OnStepComplete = nil
		config.SetGlobalConfig(cfg2)
	})

	needsContinue, _ := a.runStepCompleteHooks(7, "desc", "completed")
	if needsContinue {
		t.Error("expected needsContinue=false: template should have been expanded to step_id=7")
	}
}

func TestRunStepCompleteHooksWithRetry_NoHooks(t *testing.T) {
	a := &Agent{}
	cfg := config.GetGlobalConfig()
	cfg.Hooks.OnStepComplete = nil
	config.SetGlobalConfig(cfg)

	// hooks が未設定なら即 true を返す
	result := a.runStepCompleteHooksWithRetry(t.Context(), 1, "step", "completed")
	if !result {
		t.Error("expected true when no hooks configured")
	}
}
