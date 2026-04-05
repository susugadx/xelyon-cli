package agent

import (
	"bytes"
	"context"
	"os"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/tools"
	"github.com/susugadx/xelyon-cli/internal/ui"
)

func newCompletionTestAgent(cfg *config.Config) *Agent {
	return &Agent{Runtime: NewAgentRuntimeWithConfig(cfg)}
}

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
			name: "multi_file_details",
			stack: []tools.FileChange{
				{
					Details: []tools.FileChangeDetail{
						{FilePath: "/src/a.go"},
						{FilePath: "/src/b.go"},
						{FilePath: "/src/a.go"},
					},
				},
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
				agentWorkspaceState: agentWorkspaceState{
					changeStack:      tt.stack,
					taskChangeOffset: tt.offset,
				},
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
			agentWorkspaceState: agentWorkspaceState{
				changeStack: []tools.FileChange{
					{FilePath: "/src/main.go"},
				},
				taskChangeOffset: 0,
			},
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
			agentWorkspaceState: agentWorkspaceState{
				changeStack: []tools.FileChange{
					{FilePath: "/src/main.go"},
				},
				taskChangeOffset: 0,
			},
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
			agentWorkspaceState: agentWorkspaceState{
				changeStack:      nil,
				taskChangeOffset: 0,
			},
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

	a := newCompletionTestAgent(cfg)
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

	a := newCompletionTestAgent(cfg)
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

	a := newCompletionTestAgent(cfg)
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

	a := newCompletionTestAgent(cfg)
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

	a := newCompletionTestAgent(cfg)
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

	a := newCompletionTestAgent(cfg)
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

	a := newCompletionTestAgent(cfg)
	a.changeStack = []tools.FileChange{{FilePath: "/src/main.go"}}
	a.taskChangeOffset = 0

	result := a.runCompletionHooksWithRetry(context.Background())
	if !result {
		t.Error("expected true when no hooks configured")
	}
}

func TestRunCompletionHooksWithRetry_NoChangedFiles(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Hooks.OnCompletion = []string{"exit 1"}

	a := newCompletionTestAgent(cfg)
	a.changeStack = nil
	a.taskChangeOffset = 0

	result := a.runCompletionHooksWithRetry(context.Background())
	if !result {
		t.Error("expected true when no changed files")
	}
}

func TestRunCompletionHooksWithRetry_AllPass(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Hooks.OnCompletion = []string{"echo 'ok'"}
	cfg.Hooks.Timeout = 10

	a := newCompletionTestAgent(cfg)
	a.changeStack = []tools.FileChange{{FilePath: "/src/main.go"}}
	a.taskChangeOffset = 0

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

	// mockProvider を使って AI 修正試行をシミュレート
	provider := &mockProvider{name: "test"}
	a := newCompletionTestAgent(cfg)
	a.CurrentProvider = provider
	a.CurrentModel = "test-model"
	a.History = []api.Message{}
	a.changeStack = []tools.FileChange{{FilePath: "/src/main.go"}}
	a.taskChangeOffset = 0

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

	provider := &mockProvider{name: "test"}
	a := newCompletionTestAgent(cfg)
	a.CurrentProvider = provider
	a.CurrentModel = "test-model"
	a.History = []api.Message{}
	a.changeStack = []tools.FileChange{{FilePath: "/src/main.go"}}
	a.taskChangeOffset = 0

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

func TestRunCompletionHooksWithRetry_UsesRuntimeOutput(t *testing.T) {
	runtimeA := NewAgentRuntime()
	runtimeB := NewAgentRuntime()
	var outA bytes.Buffer
	var outB bytes.Buffer
	runtimeA.UI = ui.NewRuntime(strings.NewReader(""), &outA, &outA)
	runtimeB.UI = ui.NewRuntime(strings.NewReader(""), &outB, &outB)
	runtimeA.Config.Hooks.OnCompletion = []string{"exit 1"}
	runtimeA.Config.Hooks.Timeout = 1
	runtimeA.Config.Hooks.MaxRetry = 2
	runtimeB.Config.Hooks.OnCompletion = []string{"exit 1"}
	runtimeB.Config.Hooks.Timeout = 1
	runtimeB.Config.Hooks.MaxRetry = 3

	provider := &mockProvider{name: "test"}
	agentA := &Agent{
		Runtime:         runtimeA,
		CurrentProvider: provider,
		CurrentModel:    "test-model-a",
		agentWorkspaceState: agentWorkspaceState{
			changeStack: []tools.FileChange{{FilePath: "/src/a.go"}},
		},
	}
	agentB := &Agent{
		Runtime:         runtimeB,
		CurrentProvider: provider,
		CurrentModel:    "test-model-b",
		agentWorkspaceState: agentWorkspaceState{
			changeStack: []tools.FileChange{{FilePath: "/src/b.go"}},
		},
	}

	if got := agentA.runCompletionHooksWithRetry(context.Background()); got {
		t.Fatal("agent A should exhaust retries and return false")
	}
	if got := agentB.runCompletionHooksWithRetry(context.Background()); got {
		t.Fatal("agent B should exhaust retries and return false")
	}

	if !strings.Contains(outA.String(), "Completion hook failed (1/2)") {
		t.Fatalf("agent A output should contain runtime-specific retry message, got %q", outA.String())
	}
	if strings.Contains(outA.String(), "(1/3)") {
		t.Fatalf("agent A output should not contain agent B retry count, got %q", outA.String())
	}
	if !strings.Contains(outB.String(), "Completion hook failed (1/3)") {
		t.Fatalf("agent B output should contain runtime-specific retry message, got %q", outB.String())
	}
	if strings.Contains(outB.String(), "(1/2)") {
		t.Fatalf("agent B output should not contain agent A retry count, got %q", outB.String())
	}
}

func TestAddPendingLSPFile(t *testing.T) {
	a := &Agent{}

	// 空パスは無視される
	a.addPendingLSPFile("")
	if len(a.pendingLSPFiles) != 0 {
		t.Errorf("expected 0 files after adding empty path, got %d", len(a.pendingLSPFiles))
	}

	// 単一ファイル追加
	a.addPendingLSPFile("/src/main.go")
	if len(a.pendingLSPFiles) != 1 || a.pendingLSPFiles[0] != "/src/main.go" {
		t.Errorf("expected [/src/main.go], got %v", a.pendingLSPFiles)
	}

	// 重複追加は無視される
	a.addPendingLSPFile("/src/main.go")
	if len(a.pendingLSPFiles) != 1 {
		t.Errorf("expected 1 file after duplicate add, got %d", len(a.pendingLSPFiles))
	}

	// 別ファイルは追加される
	a.addPendingLSPFile("/src/util.go")
	if len(a.pendingLSPFiles) != 2 {
		t.Errorf("expected 2 files, got %d", len(a.pendingLSPFiles))
	}
}

func TestAddPendingLSPFilesFromChange(t *testing.T) {
	t.Run("nil change is safe", func(t *testing.T) {
		a := &Agent{}
		a.addPendingLSPFilesFromChange(nil)
		if len(a.pendingLSPFiles) != 0 {
			t.Errorf("expected 0 files, got %d", len(a.pendingLSPFiles))
		}
	})

	t.Run("empty details is safe", func(t *testing.T) {
		a := &Agent{}
		a.addPendingLSPFilesFromChange(&tools.FileChange{})
		if len(a.pendingLSPFiles) != 0 {
			t.Errorf("expected 0 files, got %d", len(a.pendingLSPFiles))
		}
	})

	t.Run("single file from apply_patch", func(t *testing.T) {
		a := &Agent{}
		a.addPendingLSPFilesFromChange(&tools.FileChange{
			Tool: "apply_patch",
			Details: []tools.FileChangeDetail{
				{FilePath: "/src/main.go", Action: "modified"},
			},
		})
		if len(a.pendingLSPFiles) != 1 || a.pendingLSPFiles[0] != "/src/main.go" {
			t.Errorf("expected [/src/main.go], got %v", a.pendingLSPFiles)
		}
	})

	t.Run("multiple files from apply_patch", func(t *testing.T) {
		a := &Agent{}
		a.addPendingLSPFilesFromChange(&tools.FileChange{
			Tool: "apply_patch",
			Details: []tools.FileChangeDetail{
				{FilePath: "/src/main.go", Action: "modified"},
				{FilePath: "/src/util.go", Action: "created"},
				{FilePath: "/src/old.go", Action: "deleted"},
			},
		})
		if len(a.pendingLSPFiles) != 3 {
			t.Errorf("expected 3 files, got %d", len(a.pendingLSPFiles))
		}
	})

	t.Run("dedup with existing pending files", func(t *testing.T) {
		a := &Agent{}
		a.addPendingLSPFile("/src/main.go") // str_replace で先に追加済み
		a.addPendingLSPFilesFromChange(&tools.FileChange{
			Tool: "apply_patch",
			Details: []tools.FileChangeDetail{
				{FilePath: "/src/main.go", Action: "modified"},
				{FilePath: "/src/new.go", Action: "created"},
			},
		})
		if len(a.pendingLSPFiles) != 2 {
			t.Errorf("expected 2 files (dedup), got %d: %v", len(a.pendingLSPFiles), a.pendingLSPFiles)
		}
	})
}

func TestCheckGitDiffEmpty(t *testing.T) {
	// 非gitディレクトリに移動して git diff が失敗するようにする
	tmpDir := t.TempDir()
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to chdir to tmpdir: %v", err)
	}
	defer func() {
		if err := os.Chdir(origDir); err != nil {
			t.Logf("warning: failed to restore working directory: %v", err)
		}
	}()

	agent := &Agent{} // Dummy agent for test
	needsContinue, feedback := agent.checkGitDiffEmpty()
	if !needsContinue {
		t.Error("expected needsContinue=true when git diff is empty")
	}
	if !strings.Contains(feedback, "NO changes") {
		t.Errorf("expected 'NO changes' in feedback, got %q", feedback)
	}
}
