package ui

import (
	"bytes"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/config"
)

// TestEditHooks_OnStepCompleteDisplayed は Hooks サブメニューに on_step_complete が表示されることを確認する
func TestEditHooks_OnStepCompleteDisplayed(t *testing.T) {
	pc := &config.ProjectConfig{
		Hooks: &config.HooksConfig{
			OnCompletion:   []string{"go test ./..."},
			OnStepComplete: []string{"echo step done"},
		},
	}
	// Hooks メニュー表示 → "b" で戻る → "c" でキャンセル
	runtime := NewRuntime(strings.NewReader("3\nb\nc\n"), &bytes.Buffer{}, &bytes.Buffer{})
	out := runtime.Output().(*bytes.Buffer)

	menu := NewProjectMenuWithRuntime(pc, runtime)
	_, _ = menu.Run()

	output := out.String()
	if !strings.Contains(output, "on_step_complete") {
		t.Fatalf("expected hooks submenu to show on_step_complete, got:\n%s", output)
	}
	if !strings.Contains(output, "(1 cmds)") {
		t.Fatalf("expected on_step_complete to show (1 cmds), got:\n%s", output)
	}
}

// TestEditHooks_OnStepCompleteEdit は on_step_complete の編集が反映されることを確認する
func TestEditHooks_OnStepCompleteEdit(t *testing.T) {
	pc := &config.ProjectConfig{
		Hooks: &config.HooksConfig{},
	}
	// メインメニュー "3" → Hooks サブ "2"(on_step_complete) → エディタで "a" → コマンド入力 → "s" → "b" → "s"
	input := "3\n2\na\necho step\ns\nb\ns\n"
	runtime := NewRuntime(strings.NewReader(input), &bytes.Buffer{}, &bytes.Buffer{})

	menu := NewProjectMenuWithRuntime(pc, runtime)
	changed, err := menu.Run()
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if !changed {
		t.Fatal("Run() changed = false, want true")
	}
	if pc.Hooks == nil {
		t.Fatal("Hooks should not be nil after editing on_step_complete")
	}
	if len(pc.Hooks.OnStepComplete) != 1 || pc.Hooks.OnStepComplete[0] != "echo step" {
		t.Fatalf("OnStepComplete = %v, want [echo step]", pc.Hooks.OnStepComplete)
	}
}

// TestEditHooks_NilifyWhenAllEmpty は全て空のとき Hooks が nil に戻ることを確認する
func TestEditHooks_NilifyWhenAllEmpty(t *testing.T) {
	pc := &config.ProjectConfig{
		Hooks: &config.HooksConfig{},
	}
	// Hooks メニュー表示 → "b" で戻る → "s" で保存
	runtime := NewRuntime(strings.NewReader("3\nb\ns\n"), &bytes.Buffer{}, &bytes.Buffer{})

	menu := NewProjectMenuWithRuntime(pc, runtime)
	_, _ = menu.Run()

	if pc.Hooks != nil {
		t.Fatalf("Hooks should be nil when all fields are empty, got %+v", pc.Hooks)
	}
}

// TestEditHooks_NilifyPreservedWithOnStepComplete は on_step_complete が残っていれば nil にならないことを確認する
func TestEditHooks_NilifyPreservedWithOnStepComplete(t *testing.T) {
	pc := &config.ProjectConfig{
		Hooks: &config.HooksConfig{
			OnStepComplete: []string{"echo step"},
		},
	}
	// Hooks メニュー表示 → "b" で戻る → "s" で保存
	runtime := NewRuntime(strings.NewReader("3\nb\ns\n"), &bytes.Buffer{}, &bytes.Buffer{})

	menu := NewProjectMenuWithRuntime(pc, runtime)
	_, _ = menu.Run()

	if pc.Hooks == nil {
		t.Fatal("Hooks should NOT be nil when on_step_complete has entries")
	}
	if len(pc.Hooks.OnStepComplete) != 1 {
		t.Fatalf("OnStepComplete = %v, want [echo step]", pc.Hooks.OnStepComplete)
	}
}

// TestEditHooks_MainMenuPreviewIncludesStepComplete はメインメニューの Hooks プレビューに on_step_complete も含まれることを確認する
func TestEditHooks_MainMenuPreviewIncludesStepComplete(t *testing.T) {
	pc := &config.ProjectConfig{
		Hooks: &config.HooksConfig{
			OnCompletion:   []string{"cmd1"},
			OnStepComplete: []string{"cmd2", "cmd3"},
		},
	}
	runtime := NewRuntime(strings.NewReader("c\n"), &bytes.Buffer{}, &bytes.Buffer{})
	out := runtime.Output().(*bytes.Buffer)

	menu := NewProjectMenuWithRuntime(pc, runtime)
	_, _ = menu.Run()

	output := out.String()
	// 1 (on_completion) + 2 (on_step_complete) = 3 cmds
	if !strings.Contains(output, "(3 cmds)") {
		t.Fatalf("expected main menu to show (3 cmds), got:\n%s", output)
	}
}

// TestEditHooks_TimeoutIsOption3 はメニュー番号変更後も timeout が [3] で動作することを確認する
func TestEditHooks_TimeoutIsOption3(t *testing.T) {
	pc := &config.ProjectConfig{
		Hooks: &config.HooksConfig{},
	}
	// Hooks メニュー "3"(timeout) → "120" 入力 → "b" → "s"
	runtime := NewRuntime(strings.NewReader("3\n3\n120\nb\ns\n"), &bytes.Buffer{}, &bytes.Buffer{})

	menu := NewProjectMenuWithRuntime(pc, runtime)
	changed, _ := menu.Run()
	if !changed {
		t.Fatal("Run() changed = false, want true")
	}
	if pc.Hooks == nil || pc.Hooks.Timeout != 120 {
		t.Fatalf("Timeout = %v, want 120", pc.Hooks)
	}
}

// TestEditHooks_MaxRetryIsOption4 はメニュー番号変更後も max_retry が [4] で動作することを確認する
func TestEditHooks_MaxRetryIsOption4(t *testing.T) {
	pc := &config.ProjectConfig{
		Hooks: &config.HooksConfig{},
	}
	// Hooks メニュー "4"(max_retry) → "5" 入力 → "b" → "s"
	runtime := NewRuntime(strings.NewReader("3\n4\n5\nb\ns\n"), &bytes.Buffer{}, &bytes.Buffer{})

	menu := NewProjectMenuWithRuntime(pc, runtime)
	changed, _ := menu.Run()
	if !changed {
		t.Fatal("Run() changed = false, want true")
	}
	if pc.Hooks == nil || pc.Hooks.MaxRetry != 5 {
		t.Fatalf("MaxRetry = %v, want 5", pc.Hooks)
	}
}
