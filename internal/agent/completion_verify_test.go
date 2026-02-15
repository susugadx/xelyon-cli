package agent

import (
	"context"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/tools"
)

func TestContainsCompletionDeclaration(t *testing.T) {
	tests := []struct {
		name     string
		response string
		want     bool
	}{
		// 日本語パターン
		{"japanese_kanryo", "すべての変更が完了しました。", true},
		{"japanese_shuhsei_shimashita", "ファイルを修正しました。", true},
		{"japanese_ijou_desu", "変更は以上です。", true},
		{"japanese_jisso_shimashita", "新機能を実装しました。", true},
		{"japanese_taiou_shimashita", "バグに対応しました。", true},
		{"japanese_shuhsei_kanryo", "修正完了です。", true},
		{"japanese_sagyo_ha_ijou", "作業は以上になります。", true},
		{"japanese_henkou_ha_ijou", "変更は以上となります。", true},

		// 英語パターン
		{"english_done", "I'm done with the changes.", true},
		{"english_completed", "The task is completed.", true},
		{"english_finished", "I've finished implementing the feature.", true},
		{"english_all_done", "That's all done now.", true},
		{"english_task_complete", "The task complete.", true},
		{"english_all_set", "Everything is all set.", true},
		{"english_thats_it", "That's it for the changes.", true},
		{"english_changes_complete", "The changes are complete.", true},
		{"english_impl_complete", "The implementation is complete.", true},

		// Case insensitive
		{"english_DONE", "DONE", true},
		{"english_Completed", "Completed successfully.", true},
		{"english_FINISHED", "FINISHED!", true},

		// マッチしないケース
		{"no_match_question", "Would you like me to make more changes?", false},
		{"no_match_explanation", "Here's how the function works...", false},
		{"no_match_tool_discussion", "I need to read the file first.", false},
		{"no_match_continuing", "I'll continue working on the remaining changes.", false},
		{"empty_string", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := containsCompletionDeclaration(tt.response)
			if got != tt.want {
				t.Errorf("containsCompletionDeclaration(%q) = %v, want %v", tt.response, got, tt.want)
			}
		})
	}
}

func TestGetTaskChangedFiles(t *testing.T) {
	tests := []struct {
		name      string
		stack     []tools.FileChange
		offset    int
		wantCount int
		wantFiles []string
	}{
		{
			name:      "no_changes",
			stack:     nil,
			offset:    0,
			wantCount: 0,
		},
		{
			name: "single_file",
			stack: []tools.FileChange{
				{FilePath: "/src/main.go"},
			},
			offset:    0,
			wantCount: 1,
			wantFiles: []string{"/src/main.go"},
		},
		{
			name: "duplicate_files",
			stack: []tools.FileChange{
				{FilePath: "/src/main.go"},
				{FilePath: "/src/main.go"},
			},
			offset:    0,
			wantCount: 1,
			wantFiles: []string{"/src/main.go"},
		},
		{
			name: "multiple_files",
			stack: []tools.FileChange{
				{FilePath: "/src/a.go"},
				{FilePath: "/src/b.go"},
				{FilePath: "/src/a.go"},
			},
			offset:    0,
			wantCount: 2,
			wantFiles: []string{"/src/a.go", "/src/b.go"},
		},
		{
			name: "respects_task_offset",
			stack: []tools.FileChange{
				{FilePath: "/old/prev1.go"},
				{FilePath: "/old/prev2.go"},
				{FilePath: "/old/prev3.go"},
				{FilePath: "/src/current1.go"},
				{FilePath: "/src/current2.go"},
			},
			offset:    3,
			wantCount: 2,
			wantFiles: []string{"/src/current1.go", "/src/current2.go"},
		},
		{
			name: "empty_filepath_skipped",
			stack: []tools.FileChange{
				{FilePath: "/src/a.go"},
				{FilePath: ""},
				{FilePath: "/src/b.go"},
			},
			offset:    0,
			wantCount: 2,
			wantFiles: []string{"/src/a.go", "/src/b.go"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := &Agent{
				changeStack:      tt.stack,
				taskChangeOffset: tt.offset,
			}

			got := a.getTaskChangedFiles()

			if len(got) != tt.wantCount {
				t.Fatalf("getTaskChangedFiles() returned %d files, want %d: %v", len(got), tt.wantCount, got)
			}

			for _, want := range tt.wantFiles {
				found := false
				for _, g := range got {
					if g == want {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("getTaskChangedFiles() missing expected file %q, got %v", want, got)
				}
			}
		})
	}
}

func TestVerifyCompletionWithDiagnostics_NoLSP(t *testing.T) {
	// LSP未起動時（LSPClient == nil）は HasErrors=false を返すため、
	// verifyCompletionWithDiagnostics は needsContinue=false を返す

	t.Run("no_lsp_with_completion_and_changes", func(t *testing.T) {
		a := &Agent{
			changeStack: []tools.FileChange{
				{FilePath: "/src/main.go"},
			},
			taskChangeOffset: 0,
		}

		needsContinue, feedback := a.verifyCompletionWithDiagnostics("変更が完了しました。")
		if needsContinue {
			t.Error("expected needsContinue=false when LSP is not available")
		}
		if feedback != "" {
			t.Errorf("expected empty feedback, got %q", feedback)
		}
	})

	t.Run("no_completion_declaration", func(t *testing.T) {
		a := &Agent{
			changeStack: []tools.FileChange{
				{FilePath: "/src/main.go"},
			},
			taskChangeOffset: 0,
		}

		needsContinue, feedback := a.verifyCompletionWithDiagnostics("Let me read the file next.")
		if needsContinue {
			t.Error("expected needsContinue=false when no completion declaration")
		}
		if feedback != "" {
			t.Errorf("expected empty feedback, got %q", feedback)
		}
	})

	t.Run("no_changed_files", func(t *testing.T) {
		a := &Agent{
			changeStack:      nil,
			taskChangeOffset: 0,
		}

		needsContinue, feedback := a.verifyCompletionWithDiagnostics("完了しました。")
		if needsContinue {
			t.Error("expected needsContinue=false when no changed files")
		}
		if feedback != "" {
			t.Errorf("expected empty feedback, got %q", feedback)
		}
	})
}

func TestRunCompletionHooks_NoHooks(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Hooks.OnCompletion = nil
	config.SetGlobalConfig(cfg)
	defer config.SetGlobalConfig(nil)

	a := &Agent{}
	needsContinue, feedback := a.runCompletionHooks([]string{"/src/main.go"})
	if needsContinue {
		t.Error("expected needsContinue=false when no hooks configured")
	}
	if feedback != "" {
		t.Errorf("expected empty feedback, got %q", feedback)
	}
}

func TestRunCompletionHooks_SuccessfulCommand(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Hooks.OnCompletion = []string{"echo 'test passed'"}
	cfg.Hooks.Timeout = 10
	config.SetGlobalConfig(cfg)
	defer config.SetGlobalConfig(nil)

	a := &Agent{}
	needsContinue, feedback := a.runCompletionHooks([]string{"/src/main.go"})
	if needsContinue {
		t.Error("expected needsContinue=false for successful command")
	}
	if feedback != "" {
		t.Errorf("expected empty feedback, got %q", feedback)
	}
}

func TestRunCompletionHooks_FailedCommand(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Hooks.OnCompletion = []string{"exit 1"}
	cfg.Hooks.Timeout = 10
	config.SetGlobalConfig(cfg)
	defer config.SetGlobalConfig(nil)

	a := &Agent{}
	needsContinue, feedback := a.runCompletionHooks([]string{"/src/main.go"})
	if !needsContinue {
		t.Error("expected needsContinue=true for failed command")
	}
	if !strings.Contains(feedback, "[SYSTEM] Completion verification failed") {
		t.Errorf("expected system feedback, got %q", feedback)
	}
	if !strings.Contains(feedback, "exit code 1") {
		t.Errorf("expected exit code in feedback, got %q", feedback)
	}
}

func TestRunCompletionHooks_ChangedFilesEnv(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Hooks.OnCompletion = []string{
		`test -n "$XELYON_CHANGED_FILES"`,
	}
	cfg.Hooks.Timeout = 10
	config.SetGlobalConfig(cfg)
	defer config.SetGlobalConfig(nil)

	a := &Agent{}
	needsContinue, _ := a.runCompletionHooks([]string{"/src/main.go", "/src/util.go"})
	if needsContinue {
		t.Error("expected needsContinue=false, XELYON_CHANGED_FILES should be set")
	}
}

func TestRunCompletionHooks_MultipleCommands_StopsOnFirstFailure(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Hooks.OnCompletion = []string{
		"echo 'first ok'",
		"exit 42",
		"echo 'should not run'",
	}
	cfg.Hooks.Timeout = 10
	config.SetGlobalConfig(cfg)
	defer config.SetGlobalConfig(nil)

	a := &Agent{}
	needsContinue, feedback := a.runCompletionHooks([]string{"/src/main.go"})
	if !needsContinue {
		t.Error("expected needsContinue=true when second command fails")
	}
	if !strings.Contains(feedback, "exit code 42") {
		t.Errorf("expected exit code 42 in feedback, got %q", feedback)
	}
}

func TestRunCompletionHooks_Timeout(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Hooks.OnCompletion = []string{"sleep 30"}
	cfg.Hooks.Timeout = 1
	config.SetGlobalConfig(cfg)
	defer config.SetGlobalConfig(nil)

	a := &Agent{}
	needsContinue, feedback := a.runCompletionHooks([]string{"/src/main.go"})
	if !needsContinue {
		t.Error("expected needsContinue=true for timed-out command")
	}
	if feedback == "" {
		t.Error("expected non-empty feedback for timed-out command")
	}
}

func TestHooksConfig_MaxRetryDefault(t *testing.T) {
	cfg := config.DefaultConfig()
	if cfg.Hooks.MaxRetry != 3 {
		t.Errorf("default MaxRetry = %d, want 3", cfg.Hooks.MaxRetry)
	}
}

func TestRunCompletionHooksWithRetry_NoHooks(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Hooks.OnCompletion = nil
	config.SetGlobalConfig(cfg)
	defer config.SetGlobalConfig(nil)

	a := &Agent{
		changeStack:      []tools.FileChange{{FilePath: "/src/main.go"}},
		taskChangeOffset: 0,
	}

	result := a.runCompletionHooksWithRetry(context.Background())
	if !result {
		t.Error("expected true when no hooks configured")
	}
}

func TestRunCompletionHooksWithRetry_NoChangedFiles(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Hooks.OnCompletion = []string{"exit 1"}
	config.SetGlobalConfig(cfg)
	defer config.SetGlobalConfig(nil)

	a := &Agent{
		changeStack:      nil,
		taskChangeOffset: 0,
	}

	result := a.runCompletionHooksWithRetry(context.Background())
	if !result {
		t.Error("expected true when no changed files")
	}
}

func TestRunCompletionHooksWithRetry_AllPass(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Hooks.OnCompletion = []string{"echo 'ok'"}
	cfg.Hooks.Timeout = 10
	config.SetGlobalConfig(cfg)
	defer config.SetGlobalConfig(nil)

	a := &Agent{
		changeStack:      []tools.FileChange{{FilePath: "/src/main.go"}},
		taskChangeOffset: 0,
	}

	result := a.runCompletionHooksWithRetry(context.Background())
	if !result {
		t.Error("expected true when all hooks pass")
	}
}

func TestRunCompletionHooksWithRetry_MaxRetryExhausted(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Hooks.OnCompletion = []string{"exit 1"}
	cfg.Hooks.Timeout = 1
	cfg.Hooks.MaxRetry = 2
	config.SetGlobalConfig(cfg)
	defer config.SetGlobalConfig(nil)

	// mockProvider を使って AI 修正試行をシミュレート
	provider := &mockProvider{name: "test"}
	a := &Agent{
		CurrentProvider:  provider,
		CurrentModel:     "test-model",
		History:          []api.Message{},
		changeStack:      []tools.FileChange{{FilePath: "/src/main.go"}},
		taskChangeOffset: 0,
	}

	result := a.runCompletionHooksWithRetry(context.Background())
	if result {
		t.Error("expected false when max_retry exhausted")
	}
}

func TestRunCompletionHooksWithRetry_MaxRetryZeroFallback(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Hooks.OnCompletion = []string{"exit 1"}
	cfg.Hooks.Timeout = 1
	cfg.Hooks.MaxRetry = 0 // 0 はフォールバックで 3 になる
	config.SetGlobalConfig(cfg)
	defer config.SetGlobalConfig(nil)

	provider := &mockProvider{name: "test"}
	a := &Agent{
		CurrentProvider:  provider,
		CurrentModel:     "test-model",
		History:          []api.Message{},
		changeStack:      []tools.FileChange{{FilePath: "/src/main.go"}},
		taskChangeOffset: 0,
	}

	result := a.runCompletionHooksWithRetry(context.Background())
	if result {
		t.Error("expected false when hooks always fail (MaxRetry=0 should fallback to 3)")
	}
	// History にフィードバックが追加されているか確認（2回のリトライ = 2 * 2 messages）
	// MaxRetry=3 (fallback): 2回の AI 呼び出し（attempt 1, 2）+ 最終回は AI を呼ばない
	if len(a.History) < 4 {
		t.Errorf("expected at least 4 history entries from retries, got %d", len(a.History))
	}
}
