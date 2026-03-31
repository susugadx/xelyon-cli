package config

import (
	"testing"

	"gopkg.in/yaml.v3"
)

// --- A. mode ごとの基本ポリシー ---

func TestResolveExecutionPolicy_Balanced(t *testing.T) {
	p := ResolveExecutionPolicy(ExecutionConfig{Mode: "balanced"})

	if !p.AutoApproveRead {
		t.Error("balanced: AutoApproveRead should be true")
	}
	if p.AutoApproveTrustedWrite {
		t.Error("balanced: AutoApproveTrustedWrite should be false")
	}
	if p.AutoApproveFullAuto {
		t.Error("balanced: AutoApproveFullAuto should be false")
	}
	if !p.AutoApproveVerificationBash {
		t.Error("balanced: AutoApproveVerificationBash should be true")
	}
	if p.AllowDiscoveryBash {
		t.Error("balanced: AllowDiscoveryBash should be false")
	}
}

func TestResolveExecutionPolicy_Trusted(t *testing.T) {
	p := ResolveExecutionPolicy(ExecutionConfig{Mode: "trusted"})

	if !p.AutoApproveTrustedWrite {
		t.Error("trusted: AutoApproveTrustedWrite should be true")
	}
	if p.AutoApproveFullAuto {
		t.Error("trusted: AutoApproveFullAuto should be false")
	}
	if p.AllowDiscoveryBash {
		t.Error("trusted: AllowDiscoveryBash should be false")
	}
}

func TestResolveExecutionPolicy_Trusted_OnlyNormalEdits(t *testing.T) {
	// trusted で自動承認される「通常編集」ツール
	for tool := range TrustedWriteTools {
		if !TrustedWriteTools[tool] {
			t.Errorf("TrustedWriteTools[%q] should be true", tool)
		}
	}
	// delete_file は通常編集ではない
	if TrustedWriteTools["delete_file"] {
		t.Error("delete_file should NOT be in TrustedWriteTools")
	}
	// web_search は通常編集ではない
	if TrustedWriteTools["web_search"] {
		t.Error("web_search should NOT be in TrustedWriteTools")
	}
}

func TestResolveExecutionPolicy_FullAuto(t *testing.T) {
	p := ResolveExecutionPolicy(ExecutionConfig{Mode: "full_auto"})

	if !p.AutoApproveFullAuto {
		t.Error("full_auto: AutoApproveFullAuto should be true")
	}
	if !p.AutoApproveTrustedWrite {
		t.Error("full_auto: AutoApproveTrustedWrite should be true")
	}
	if p.AllowDiscoveryBash {
		t.Error("full_auto: AllowDiscoveryBash should be false (always)")
	}
}

func TestResolveExecutionPolicy_DefaultMode(t *testing.T) {
	p := ResolveExecutionPolicy(ExecutionConfig{})
	if p.AutoApproveTrustedWrite {
		t.Error("default (balanced): AutoApproveTrustedWrite should be false")
	}
}

// --- B. always_confirm ---

func TestResolveExecutionPolicy_AlwaysConfirm_FullAuto(t *testing.T) {
	p := ResolveExecutionPolicy(ExecutionConfig{
		Mode:          "full_auto",
		AlwaysConfirm: []string{"delete_file", "bash_destructive", "bash_redirect_write"},
	})

	if !p.ShouldConfirm(ConfirmDeleteFile) {
		t.Error("full_auto + always_confirm: delete_file should require confirm")
	}
	if !p.ShouldConfirm(ConfirmBashDestructive) {
		t.Error("full_auto + always_confirm: bash_destructive should require confirm")
	}
	if !p.ShouldConfirm(ConfirmBashRedirectWrite) {
		t.Error("full_auto + always_confirm: bash_redirect_write should require confirm")
	}
	if p.ShouldConfirm(ConfirmMoveFile) {
		t.Error("move_file should NOT be in always_confirm")
	}
}

func TestResolveExecutionPolicy_AlwaysConfirm_AllCategories(t *testing.T) {
	// 全カテゴリを always_confirm に入れて、全て効くことを確認
	all := []string{
		"delete_file", "move_file", "destructive_write",
		"workspace_outside_write", "bash_destructive",
		"bash_redirect_write", "mass_change",
	}
	p := ResolveExecutionPolicy(ExecutionConfig{
		Mode:          "full_auto",
		AlwaysConfirm: all,
	})

	cats := []ConfirmCategory{
		ConfirmDeleteFile, ConfirmMoveFile, ConfirmDestructiveWrite,
		ConfirmWorkspaceOutside, ConfirmBashDestructive,
		ConfirmBashRedirectWrite, ConfirmMassChange,
	}
	for _, cat := range cats {
		if !p.ShouldConfirm(cat) {
			t.Errorf("ShouldConfirm(%q) = false, want true", cat)
		}
	}
}

// --- C. shell 分類 ---

func TestClassifyShellCommand_Discovery(t *testing.T) {
	tests := []string{
		"grep pattern file.txt", "find . -name '*.go'",
		"cat main.go", "head -20 file.txt", "tail -f log.txt",
		"rg 'func main'", "awk '{print $1}' file", "tree src/",
		"ls", "ls -la", "ls src/", // ls は全て discovery
	}
	for _, cmd := range tests {
		if got := ClassifyShellCommand(cmd); got != ShellDiscovery {
			t.Errorf("ClassifyShellCommand(%q) = %d, want ShellDiscovery", cmd, got)
		}
	}
}

func TestClassifyShellCommand_Verification(t *testing.T) {
	tests := []string{
		"go test ./...", "go build ./...", "make ci-check",
		"npm run build", "git status", "echo hello", "pwd",
	}
	for _, cmd := range tests {
		if got := ClassifyShellCommand(cmd); got != ShellVerification {
			t.Errorf("ClassifyShellCommand(%q) = %d, want ShellVerification", cmd, got)
		}
	}
}

func TestClassifyShellCommand_Destructive(t *testing.T) {
	tests := []string{
		"rm file.txt", "sudo apt install",
		"git push origin main", "mv old.go new.go",
	}
	for _, cmd := range tests {
		if got := ClassifyShellCommand(cmd); got != ShellDestructive {
			t.Errorf("ClassifyShellCommand(%q) = %d, want ShellDestructive", cmd, got)
		}
	}
}

func TestClassifyShellCommand_RedirectWrite(t *testing.T) {
	tests := []string{
		"echo hello > file.txt",
		"go test ./... >> results.txt",
	}
	for _, cmd := range tests {
		if got := ClassifyShellCommand(cmd); got != ShellRedirectWrite {
			t.Errorf("ClassifyShellCommand(%q) = %d, want ShellRedirectWrite", cmd, got)
		}
	}
}

// --- D. ls の扱い ---

func TestClassifyShellCommand_LsIsDiscovery(t *testing.T) {
	tests := []string{"ls", "ls -la", "ls src/", "ls -R .", "ls *.go"}
	for _, cmd := range tests {
		if got := ClassifyShellCommand(cmd); got != ShellDiscovery {
			t.Errorf("ClassifyShellCommand(%q) = %d, want ShellDiscovery (ls should be discovery)", cmd, got)
		}
	}
}

// --- safe_shell_commands 制限 ---

func TestSafeShellCommands_DoNotBypassDiscovery(t *testing.T) {
	cat := ClassifyShellCommand("grep pattern file.go")
	if cat != ShellDiscovery {
		t.Error("grep should be classified as discovery")
	}
	// ClassifyShellCommand は config を見ないので safe_shell_commands に入れても分類は変わらない
}

func TestSafeShellCommands_DoNotBypassDestructive(t *testing.T) {
	cat := ClassifyShellCommand("rm -f temp.txt")
	if cat != ShellDestructive {
		t.Error("rm should be classified as destructive")
	}
}

// --- migration ---

func TestMigrateOldKeys_ToolConfirmToExecution(t *testing.T) {
	tests := []struct {
		name string
		yaml string
		want string
	}{
		{
			name: "auto_approve_medium: true → trusted",
			yaml: `
tool_confirm:
  auto_approve_medium: true
`,
			want: "trusted",
		},
		{
			name: "auto_approve_medium: false → balanced (default)",
			yaml: `
tool_confirm:
  auto_approve_medium: false
`,
			want: "balanced",
		},
		{
			name: "execution あり → execution 優先",
			yaml: `
execution:
  mode: full_auto
tool_confirm:
  auto_approve_medium: true
`,
			want: "full_auto",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := DefaultConfig()
			if err := yaml.Unmarshal([]byte(tt.yaml), cfg); err != nil {
				t.Fatalf("yaml.Unmarshal failed: %v", err)
			}
			migrateOldKeys([]byte(tt.yaml), cfg)
			applyDefaults(cfg)

			if cfg.Execution.Mode != tt.want {
				t.Errorf("Execution.Mode = %q, want %q", cfg.Execution.Mode, tt.want)
			}
		})
	}
}
