package ui

import (
	"fmt"
	"os/exec"
	"strings"
)

// TaskSummary はタスク完了時のサマリー表示用
type TaskSummary struct {
	Changes     []FileChangeSummary // ファイル変更一覧
	TestsPassed *bool               // テスト成功フラグ（nil = 未実行）
	Stats       SummaryStats        // 統計情報
}

// FileChangeSummary はファイル変更の概要
type FileChangeSummary struct {
	FilePath     string
	Action       string // "created", "modified", "deleted"
	LinesAdded   int
	LinesRemoved int
}

// SummaryStats は統計情報
type SummaryStats struct {
	FilesChanged      int
	TotalLinesAdded   int
	TotalLinesRemoved int
	CommitHash        string // 最新コミットハッシュ（あれば）
}

// NewTaskSummary は新しい TaskSummary を作成
func NewTaskSummary() *TaskSummary {
	return &TaskSummary{
		Changes: nil,
	}
}

// AddChange はファイル変更を追加
func (ts *TaskSummary) AddChange(filePath, action string, linesAdded, linesRemoved int) *TaskSummary {
	ts.Changes = append(ts.Changes, FileChangeSummary{
		FilePath:     filePath,
		Action:       action,
		LinesAdded:   linesAdded,
		LinesRemoved: linesRemoved,
	})
	return ts
}

// SetTestResult はテスト結果を設定
func (ts *TaskSummary) SetTestResult(passed bool) *TaskSummary {
	ts.TestsPassed = &passed
	return ts
}

// Calculate は統計情報を計算
func (ts *TaskSummary) Calculate() *TaskSummary {
	// ファイル単位でユニーク化して統計を集計
	unique := make(map[string]*FileChangeSummary)
	for _, c := range ts.Changes {
		if existing, ok := unique[c.FilePath]; ok {
			// action は最も強いものを残す（deleted > created > moved > modified）
			existing.Action = mergeAction(existing.Action, c.Action)
			existing.LinesAdded += c.LinesAdded
			existing.LinesRemoved += c.LinesRemoved
			continue
		}
		copy := c
		unique[c.FilePath] = &copy
	}

	// Changes をユニーク化したものに置き換え（表示も重複しない）
	ts.Changes = make([]FileChangeSummary, 0, len(unique))
	for _, v := range unique {
		ts.Changes = append(ts.Changes, *v)
	}

	ts.Stats.FilesChanged = len(ts.Changes)
	ts.Stats.TotalLinesAdded = 0
	ts.Stats.TotalLinesRemoved = 0

	for _, c := range ts.Changes {
		ts.Stats.TotalLinesAdded += c.LinesAdded
		ts.Stats.TotalLinesRemoved += c.LinesRemoved
	}

	// git commit hash を取得（エラー時は空）
	if hash, err := getGitShortHash(); err == nil {
		ts.Stats.CommitHash = hash
	}

	return ts
}

// mergeAction は2つの action をマージして、より優先度の高いものを返す。
// 優先度: deleted > created > moved > modified
func mergeAction(a, b string) string {
	rank := func(s string) int {
		switch s {
		case "deleted":
			return 4
		case "created":
			return 3
		case "moved":
			return 2
		default:
			return 1 // modified/unknown
		}
	}
	if rank(b) > rank(a) {
		return b
	}
	return a
}

// Render はサマリーを表示用文字列に変換
func (ts *TaskSummary) Render() string {
	if len(ts.Changes) == 0 {
		return ""
	}

	ts.Calculate()

	var sb strings.Builder

	// ヘッダー
	sb.WriteString("\n")
	sb.WriteString("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
	sb.WriteString("✅ Task Completed\n")
	sb.WriteString("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
	sb.WriteString("\n")

	// 変更テーブル
	sb.WriteString("Changes:\n")
	table := NewTable().SetHeaders("File", "Action", "Lines")

	for _, c := range ts.Changes {
		lines := formatLines(c.LinesAdded, c.LinesRemoved)
		table.AddRow(c.FilePath, c.Action, lines)
	}
	sb.WriteString(table.RenderCompact())
	sb.WriteString("\n")

	// テスト結果
	if ts.TestsPassed != nil {
		if *ts.TestsPassed {
			sb.WriteString("✅ All tests passed\n")
		} else {
			sb.WriteString("❌ Tests failed\n")
		}
		sb.WriteString("\n")
	}

	// サマリー行
	summary := fmt.Sprintf("Summary: %d file(s), +%d -%d lines",
		ts.Stats.FilesChanged,
		ts.Stats.TotalLinesAdded,
		ts.Stats.TotalLinesRemoved,
	)
	if ts.Stats.CommitHash != "" {
		summary += fmt.Sprintf(" [%s]", ts.Stats.CommitHash)
	}
	sb.WriteString(summary)
	sb.WriteString("\n")
	sb.WriteString("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")

	return sb.String()
}

// formatLines は行数変更を +N -M 形式でフォーマット
func formatLines(added, removed int) string {
	if added == 0 && removed == 0 {
		return "-"
	}
	parts := []string{}
	if added > 0 {
		parts = append(parts, fmt.Sprintf("+%d", added))
	}
	if removed > 0 {
		parts = append(parts, fmt.Sprintf("-%d", removed))
	}
	return strings.Join(parts, " ")
}

// getGitShortHash は現在の git commit hash の短縮版を取得
// 未コミット変更がある場合は空文字を返す（コミットハッシュに見えて紛らわしいため）
func getGitShortHash() (string, error) {
	// 未コミット変更があるかチェック
	statusCmd := exec.Command("git", "status", "--porcelain")
	statusOutput, err := statusCmd.Output()
	if err != nil {
		return "", err
	}
	if len(strings.TrimSpace(string(statusOutput))) > 0 {
		return "", nil
	}

	cmd := exec.Command("git", "rev-parse", "--short", "HEAD")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

// InferAction はツール名からアクションを推測
func InferAction(tool string) string {
	switch tool {
	case "write_file":
		return "created"
	case "str_replace":
		return "modified"
	case "delete_file":
		return "deleted"
	default:
		return "modified"
	}
}
