package agent

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseXELYONMD(t *testing.T) {
	// テスト用一時ディレクトリ作成
	tmpDir, err := os.MkdirTemp("", "xelyon-sync-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// テスト用XELYON.md作成
	xelyonContent := `# test-project プロジェクト設定

## 概要
This is a test project for testing purposes.

## 技術スタック
- **Go**
- **Docker**

## ディレクトリ構造
` + "```" + `
test-project/
├── cmd/
├── internal/
` + "```" + `

## 開発ルール
Use gofmt for formatting

## コードマップ
### cmd/main.go
- ` + "`func main()`" + ` (line 10)

### internal/agent/agent.go
- ` + "`func NewAgent()`" + ` (line 15)
`

	xelyonPath := filepath.Join(tmpDir, "XELYON.md")
	if err := os.WriteFile(xelyonPath, []byte(xelyonContent), 0644); err != nil {
		t.Fatalf("Failed to write XELYON.md: %v", err)
	}

	// パーステスト
	state, err := parseXELYONMD(xelyonPath)
	if err != nil {
		t.Fatalf("parseXELYONMD() error: %v", err)
	}

	// プロジェクト名
	if state.ProjectName != "test-project" {
		t.Errorf("ProjectName = %s, want test-project", state.ProjectName)
	}

	// 概要
	if !strings.Contains(state.Overview, "test project") {
		t.Errorf("Overview = %s, want to contain 'test project'", state.Overview)
	}

	// 技術スタック
	if len(state.TechStack) != 2 {
		t.Errorf("TechStack count = %d, want 2", len(state.TechStack))
	}
	if state.TechStack[0] != "Go" {
		t.Errorf("TechStack[0] = %s, want Go", state.TechStack[0])
	}
	if state.TechStack[1] != "Docker" {
		t.Errorf("TechStack[1] = %s, want Docker", state.TechStack[1])
	}

	// コードマップファイル
	if len(state.CodeMapFiles) != 2 {
		t.Errorf("CodeMapFiles count = %d, want 2", len(state.CodeMapFiles))
	}
	if state.CodeMapFiles[0] != "cmd/main.go" {
		t.Errorf("CodeMapFiles[0] = %s, want cmd/main.go", state.CodeMapFiles[0])
	}
}

func TestParseXELYONMD_Empty(t *testing.T) {
	// テスト用一時ディレクトリ作成
	tmpDir, err := os.MkdirTemp("", "xelyon-sync-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// 最小限のXELYON.md
	xelyonContent := `# empty-project プロジェクト設定

## 概要
(プロジェクトの説明をここに記載)

## 技術スタック
- (使用している技術をここに記載)
`

	xelyonPath := filepath.Join(tmpDir, "XELYON.md")
	if err := os.WriteFile(xelyonPath, []byte(xelyonContent), 0644); err != nil {
		t.Fatalf("Failed to write XELYON.md: %v", err)
	}

	state, err := parseXELYONMD(xelyonPath)
	if err != nil {
		t.Fatalf("parseXELYONMD() error: %v", err)
	}

	if state.ProjectName != "empty-project" {
		t.Errorf("ProjectName = %s, want empty-project", state.ProjectName)
	}

	// 空の技術スタック（プレースホルダーは検出されない）
	if len(state.TechStack) != 0 {
		t.Errorf("TechStack count = %d, want 0", len(state.TechStack))
	}
}

func TestDetectDiff(t *testing.T) {
	current := &SyncState{
		TechStack:    []string{"Go", "Docker"},
		CodeMapFiles: []string{"cmd/main.go", "internal/old.go"},
	}

	newTech := []string{"Go", "TypeScript"}
	newFiles := []string{"cmd/main.go", "internal/new.go"}
	newSymbolCount := 50

	diff := detectDiff(current, newTech, newFiles, newSymbolCount)

	// 追加された技術
	if len(diff.AddedTech) != 1 || diff.AddedTech[0] != "TypeScript" {
		t.Errorf("AddedTech = %v, want [TypeScript]", diff.AddedTech)
	}

	// 削除された技術
	if len(diff.RemovedTech) != 1 || diff.RemovedTech[0] != "Docker" {
		t.Errorf("RemovedTech = %v, want [Docker]", diff.RemovedTech)
	}

	// 追加されたファイル
	if len(diff.AddedFiles) != 1 || diff.AddedFiles[0] != "internal/new.go" {
		t.Errorf("AddedFiles = %v, want [internal/new.go]", diff.AddedFiles)
	}

	// 削除されたファイル
	if len(diff.RemovedFiles) != 1 || diff.RemovedFiles[0] != "internal/old.go" {
		t.Errorf("RemovedFiles = %v, want [internal/old.go]", diff.RemovedFiles)
	}
}

func TestDetectDiff_NoChanges(t *testing.T) {
	current := &SyncState{
		TechStack:    []string{"Go"},
		CodeMapFiles: []string{"main.go"},
	}

	newTech := []string{"Go"}
	newFiles := []string{"main.go"}
	newSymbolCount := 10

	diff := detectDiff(current, newTech, newFiles, newSymbolCount)

	if diff.HasChanges() {
		t.Error("HasChanges() = true, want false")
	}
}

func TestSyncDiff_HasChanges(t *testing.T) {
	tests := []struct {
		name string
		diff *SyncDiff
		want bool
	}{
		{
			name: "No changes",
			diff: &SyncDiff{},
			want: false,
		},
		{
			name: "Added tech",
			diff: &SyncDiff{AddedTech: []string{"Go"}},
			want: true,
		},
		{
			name: "Removed tech",
			diff: &SyncDiff{RemovedTech: []string{"Python"}},
			want: true,
		},
		{
			name: "Added files",
			diff: &SyncDiff{AddedFiles: []string{"new.go"}},
			want: true,
		},
		{
			name: "Removed files",
			diff: &SyncDiff{RemovedFiles: []string{"old.go"}},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.diff.HasChanges(); got != tt.want {
				t.Errorf("HasChanges() = %v, want %v", got, tt.want)
			}
		})
	}
}
