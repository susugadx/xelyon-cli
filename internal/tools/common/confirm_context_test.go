package common

import (
	"testing"

	"github.com/susugadx/xelyon-cli/internal/config"
)

// --- move_file ---

func TestResolveContextCategories_MoveFile(t *testing.T) {
	cats := ResolveContextCategories("apply_patch", ToolConfirmContext{
		MovePath: "/workspace/new.go",
	})
	found := false
	for _, c := range cats {
		if c == config.ConfirmMoveFile {
			found = true
		}
	}
	if !found {
		t.Error("MovePath set → should include ConfirmMoveFile")
	}
}

func TestResolveContextCategories_NoMove(t *testing.T) {
	cats := ResolveContextCategories("write_file", ToolConfirmContext{
		TargetPath: "/workspace/file.go",
	})
	for _, c := range cats {
		if c == config.ConfirmMoveFile {
			t.Error("No MovePath → should NOT include ConfirmMoveFile")
		}
	}
}

// --- workspace_outside_write ---

func TestResolveContextCategories_WorkspaceOutside(t *testing.T) {
	cats := ResolveContextCategories("write_file", ToolConfirmContext{
		TargetPath: "/etc/hosts",
	})
	found := false
	for _, c := range cats {
		if c == config.ConfirmWorkspaceOutside {
			found = true
		}
	}
	if !found {
		t.Error("/etc/hosts → should include ConfirmWorkspaceOutside")
	}
}

func TestResolveContextCategories_WorkspaceInside(t *testing.T) {
	// CWD 配下のパスは workspace_outside にならない
	cats := ResolveContextCategories("write_file", ToolConfirmContext{
		TargetPath: "internal/config/config.go", // 相対パス → CWD 配下
	})
	for _, c := range cats {
		if c == config.ConfirmWorkspaceOutside {
			t.Error("relative path inside workspace → should NOT include ConfirmWorkspaceOutside")
		}
	}
}

func TestResolveContextCategories_EmptyPath(t *testing.T) {
	// パス不明 → workspace 内扱い（安全側に倒さない）
	cats := ResolveContextCategories("write_file", ToolConfirmContext{})
	for _, c := range cats {
		if c == config.ConfirmWorkspaceOutside {
			t.Error("empty path → should NOT include ConfirmWorkspaceOutside")
		}
	}
}

// --- mass_change ---

func TestResolveContextCategories_MassChange(t *testing.T) {
	cats := ResolveContextCategories("apply_patch", ToolConfirmContext{
		AffectedFiles: MassChangeThreshold,
	})
	found := false
	for _, c := range cats {
		if c == config.ConfirmMassChange {
			found = true
		}
	}
	if !found {
		t.Errorf("AffectedFiles=%d → should include ConfirmMassChange", MassChangeThreshold)
	}
}

func TestResolveContextCategories_BelowMassChange(t *testing.T) {
	cats := ResolveContextCategories("apply_patch", ToolConfirmContext{
		AffectedFiles: MassChangeThreshold - 1,
	})
	for _, c := range cats {
		if c == config.ConfirmMassChange {
			t.Errorf("AffectedFiles=%d → should NOT include ConfirmMassChange", MassChangeThreshold-1)
		}
	}
}

// --- trusted + workspace ---

func TestTrustedWrite_WorkspaceInside(t *testing.T) {
	// isInsideWorkspace は CWD 配下のパスで true を返す
	if !isInsideWorkspace("internal/config/config.go") {
		t.Error("relative path should be inside workspace")
	}
}

func TestTrustedWrite_WorkspaceOutside(t *testing.T) {
	if isInsideWorkspace("/etc/hosts") {
		t.Error("/etc/hosts should be outside workspace")
	}
}

func TestTrustedWrite_EmptyPath(t *testing.T) {
	// 空パスは workspace 内扱い
	if !isInsideWorkspace("") {
		t.Error("empty path should be treated as inside workspace")
	}
}

// --- always_confirm が mode より優先される ---

func TestAlwaysConfirm_OverridesFullAuto(t *testing.T) {
	policy := config.ResolveExecutionPolicy(config.ExecutionConfig{
		Mode:          "full_auto",
		AlwaysConfirm: []string{"delete_file", "move_file", "mass_change", "workspace_outside_write"},
	})

	// full_auto でもこれらは確認される
	if !policy.ShouldConfirm(config.ConfirmDeleteFile) {
		t.Error("full_auto + always_confirm: delete_file should require confirm")
	}
	if !policy.ShouldConfirm(config.ConfirmMoveFile) {
		t.Error("full_auto + always_confirm: move_file should require confirm")
	}
	if !policy.ShouldConfirm(config.ConfirmMassChange) {
		t.Error("full_auto + always_confirm: mass_change should require confirm")
	}
	if !policy.ShouldConfirm(config.ConfirmWorkspaceOutside) {
		t.Error("full_auto + always_confirm: workspace_outside_write should require confirm")
	}
}

// --- 既存カテゴリが壊れていない ---

func TestToolNameToConfirmCategory(t *testing.T) {
	tests := []struct {
		tool string
		want config.ConfirmCategory
	}{
		{"delete_file", config.ConfirmDeleteFile},
		{"git_push", config.ConfirmBashDestructive},
		{"git_force_push", config.ConfirmBashDestructive},
		{"git_reset_hard", config.ConfirmBashDestructive},
		{"write_file", ""},
		{"str_replace", ""},
	}
	for _, tt := range tests {
		got := ToolNameToConfirmCategory(tt.tool)
		if got != tt.want {
			t.Errorf("ToolNameToConfirmCategory(%q) = %q, want %q", tt.tool, got, tt.want)
		}
	}
}

// --- apply_patch 複数パス ---

func TestResolveContextCategories_MultipleTargetPaths_OneOutside(t *testing.T) {
	cats := ResolveContextCategories("apply_patch", ToolConfirmContext{
		TargetPaths: []string{
			"internal/config/config.go", // workspace 内
			"/tmp/outside.go",           // workspace 外
		},
		AffectedFiles: 2,
	})
	found := false
	for _, c := range cats {
		if c == config.ConfirmWorkspaceOutside {
			found = true
		}
	}
	if !found {
		t.Error("1 of 2 paths outside workspace → should include ConfirmWorkspaceOutside")
	}
}

func TestResolveContextCategories_MultipleTargetPaths_AllInside(t *testing.T) {
	cats := ResolveContextCategories("apply_patch", ToolConfirmContext{
		TargetPaths: []string{
			"internal/config/config.go",
			"internal/tools/dev/bash.go",
		},
		AffectedFiles: 2,
	})
	for _, c := range cats {
		if c == config.ConfirmWorkspaceOutside {
			t.Error("all paths inside workspace → should NOT include ConfirmWorkspaceOutside")
		}
	}
}

func TestResolveContextCategories_HasMove(t *testing.T) {
	cats := ResolveContextCategories("apply_patch", ToolConfirmContext{
		HasMove:       true,
		AffectedFiles: 2,
	})
	found := false
	for _, c := range cats {
		if c == config.ConfirmMoveFile {
			found = true
		}
	}
	if !found {
		t.Error("HasMove=true → should include ConfirmMoveFile")
	}
}

func TestAllPathsInsideWorkspace_OneOutside(t *testing.T) {
	tc := ToolConfirmContext{
		TargetPaths: []string{"internal/config/config.go", "/tmp/outside.go"},
	}
	if AllPathsInsideWorkspace(tc) {
		t.Error("one path outside → AllPathsInsideWorkspace should be false")
	}
}

func TestAllPathsInsideWorkspace_AllInside(t *testing.T) {
	tc := ToolConfirmContext{
		TargetPaths: []string{"internal/config/config.go", "internal/tools/dev/bash.go"},
	}
	if !AllPathsInsideWorkspace(tc) {
		t.Error("all paths inside → AllPathsInsideWorkspace should be true")
	}
}

func TestAllPathsInsideWorkspace_Empty(t *testing.T) {
	tc := ToolConfirmContext{}
	if !AllPathsInsideWorkspace(tc) {
		t.Error("empty paths → AllPathsInsideWorkspace should be true")
	}
}
