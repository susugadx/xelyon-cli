package ui

import (
	"fmt"
	"strings"
)

// PlanStep は計画のステップ情報（UI表示用）
type PlanStep struct {
	ID          int
	Description string
	Tools       []string
	Files       []string // 対象ファイル
	Status      string   // "pending", "running", "completed", "failed"
}

// PlanDisplay は計画表示用の構造体
type PlanDisplay struct {
	Title      string
	Summary    string
	Steps      []PlanStep
	FilesTitle string
	Footer     string
}

// NewPlanDisplay は新しい PlanDisplay を作成
func NewPlanDisplay(title string) *PlanDisplay {
	return &PlanDisplay{
		Title:      title,
		Steps:      nil,
		FilesTitle: "修正ファイル",
	}
}

// SetSummary は概要を設定
func (p *PlanDisplay) SetSummary(summary string) *PlanDisplay {
	p.Summary = summary
	return p
}

// SetFilesTitle はファイル一覧の見出しを設定
func (p *PlanDisplay) SetFilesTitle(title string) *PlanDisplay {
	p.FilesTitle = title
	return p
}

// SetFooter は計画表示の下部補足を設定
func (p *PlanDisplay) SetFooter(footer string) *PlanDisplay {
	p.Footer = footer
	return p
}

// AddStep はステップを追加
func (p *PlanDisplay) AddStep(id int, description string, tools []string, files []string) *PlanDisplay {
	p.Steps = append(p.Steps, PlanStep{
		ID:          id,
		Description: description,
		Tools:       tools,
		Files:       files,
		Status:      "pending",
	})
	return p
}

// Render は計画を整形して表示
func (p *PlanDisplay) Render() string {
	var sb strings.Builder

	// 上部罫線
	sb.WriteString(strings.Repeat("╌", 50))
	sb.WriteString("\n")

	// タイトル
	if p.Title != "" {
		fmt.Fprintf(&sb, "📋 Plan: %s\n", p.Title)
		sb.WriteString("\n")
	}

	// 概要
	if p.Summary != "" {
		sb.WriteString("概要\n")
		// 概要を行分割してインデント
		for _, line := range strings.Split(p.Summary, "\n") {
			if line != "" {
				fmt.Fprintf(&sb, "  %s\n", line)
			}
		}
		sb.WriteString("\n")
	}

	// ステップ一覧
	if len(p.Steps) > 0 {
		sb.WriteString("ステップ\n")
		for _, step := range p.Steps {
			icon := getStepStatusIcon(step.Status)
			fmt.Fprintf(&sb, "  %s %d. %s\n", icon, step.ID, step.Description)

			// ツール
			if len(step.Tools) > 0 {
				fmt.Fprintf(&sb, "       Tools: %s\n", strings.Join(step.Tools, ", "))
			}
		}
		sb.WriteString("\n")
	}

	// 修正ファイル一覧（テーブル形式）
	if files := p.collectFiles(); len(files) > 0 {
		sb.WriteString(p.filesTitle())
		sb.WriteString("\n")
		sb.WriteString(p.renderFilesTable(files))
	}

	if p.Footer != "" {
		sb.WriteString("\n確認\n")
		writeIndentedNonEmptyLines(&sb, p.Footer, "  ")
		sb.WriteString("\n")
	}

	// 下部罫線
	sb.WriteString(strings.Repeat("╌", 50))
	sb.WriteString("\n")

	return sb.String()
}

func (p *PlanDisplay) filesTitle() string {
	if p.FilesTitle == "" {
		return "修正ファイル"
	}
	return p.FilesTitle
}

func writeIndentedNonEmptyLines(sb *strings.Builder, text string, indent string) {
	for _, line := range strings.Split(text, "\n") {
		if line == "" {
			continue
		}
		fmt.Fprintf(sb, "%s%s\n", indent, line)
	}
}

// collectFiles はすべてのステップからファイルを収集
func (p *PlanDisplay) collectFiles() []struct {
	File   string
	Action string
} {
	seen := make(map[string]bool)
	var result []struct {
		File   string
		Action string
	}

	for _, step := range p.Steps {
		for _, file := range step.Files {
			if !seen[file] {
				seen[file] = true
				// ステップの説明からアクションを推測
				action := inferActionFromDescription(step.Description)
				result = append(result, struct {
					File   string
					Action string
				}{File: file, Action: action})
			}
		}
	}

	return result
}

// renderFilesTable はファイルテーブルを描画
func (p *PlanDisplay) renderFilesTable(files []struct {
	File   string
	Action string
}) string {
	if len(files) == 0 {
		return ""
	}

	table := NewTable().SetHeaders("ファイル", "変更内容")
	for _, f := range files {
		table.AddRow(f.File, f.Action)
	}

	return table.RenderCompact()
}

// getStepStatusIcon はステータスに応じたアイコンを返す
func getStepStatusIcon(status string) string {
	switch status {
	case "completed":
		return "✅"
	case "running":
		return "⏳"
	case "failed":
		return "❌"
	case "skipped":
		return "⏭️"
	default: // pending
		return "⬜"
	}
}

// inferActionFromDescription は説明文からアクションを推測
func inferActionFromDescription(desc string) string {
	descLower := strings.ToLower(desc)

	// テスト関連（優先度高）
	if strings.Contains(descLower, "test") || strings.Contains(descLower, "テスト") {
		return "テスト追加"
	}
	// リファクタリング
	if strings.Contains(descLower, "refactor") || strings.Contains(descLower, "リファクタ") {
		return "リファクタリング"
	}
	// 削除
	if strings.Contains(descLower, "delete") || strings.Contains(descLower, "remove") ||
		strings.Contains(descLower, "削除") {
		return "削除"
	}
	// 更新・修正
	if strings.Contains(descLower, "update") || strings.Contains(descLower, "modify") ||
		strings.Contains(descLower, "change") || strings.Contains(descLower, "fix") ||
		strings.Contains(descLower, "更新") || strings.Contains(descLower, "修正") {
		return "更新"
	}
	// 新規作成
	if strings.Contains(descLower, "create") || strings.Contains(descLower, "add") ||
		strings.Contains(descLower, "新規") || strings.Contains(descLower, "作成") {
		return "新規作成"
	}

	return "変更"
}

// FormatPlanHeader は計画ヘッダーを表示用に整形
func FormatPlanHeader(title string) string {
	var sb strings.Builder
	sb.WriteString("\n")
	sb.WriteString(strings.Repeat("╌", 50))
	sb.WriteString("\n")
	fmt.Fprintf(&sb, "📋 Plan: %s\n", title)
	sb.WriteString(strings.Repeat("╌", 50))
	sb.WriteString("\n")
	return sb.String()
}

// FormatPlanFooter は計画フッターを表示用に整形
func FormatPlanFooter() string {
	return strings.Repeat("╌", 50) + "\n"
}

// FormatStepProgress はステップ進捗を表示用に整形
func FormatStepProgress(stepID int, total int, description string, status string) string {
	icon := getStepStatusIcon(status)
	return fmt.Sprintf("%s [%d/%d] %s", icon, stepID, total, description)
}
