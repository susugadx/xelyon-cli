// Package plan は計画の永続化を提供する
package plan

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"gopkg.in/yaml.v3"
)

const (
	defaultPlansDir = ".xelyon/plans"
)

// PlanStorage は計画の永続化を管理
type PlanStorage struct {
	baseDir    string
	mu         sync.Mutex
	lastPlanID string
}

// PlanMetadata は計画の概要情報
type PlanMetadata struct {
	ID        string
	Title     string
	Status    PlanStatus
	CreatedAt time.Time
	UpdatedAt time.Time
	Filename  string
}

// NewPlanStorage はカレントディレクトリ基準で Storage を作成
func NewPlanStorage() (*PlanStorage, error) {
	baseDir := defaultPlansDir
	if err := os.MkdirAll(baseDir, 0700); err != nil {
		return nil, fmt.Errorf("failed to create plans dir: %w", err)
	}
	return &PlanStorage{baseDir: baseDir}, nil
}

// Save は計画を Markdown ファイルとして保存
func (s *PlanStorage) Save(p *Plan) (string, error) {
	filename := s.generateFilename(p.Title)
	path := filepath.Join(s.baseDir, filename)

	// 更新時刻を設定
	p.UpdatedAt = time.Now()

	content := s.formatMarkdown(p)

	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		return "", fmt.Errorf("failed to write plan: %w", err)
	}

	return filename, nil
}

// Load は計画を Markdown ファイルから読み込み
func (s *PlanStorage) Load(filename string) (*Plan, error) {
	path := filepath.Join(s.baseDir, filename)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read plan: %w", err)
	}

	return s.parseMarkdown(string(data))
}

// List は全計画を一覧取得（新しい順）
func (s *PlanStorage) List() ([]PlanMetadata, error) {
	entries, err := os.ReadDir(s.baseDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []PlanMetadata{}, nil
		}
		return nil, fmt.Errorf("failed to list plans: %w", err)
	}

	var plans []PlanMetadata
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}

		plan, err := s.Load(entry.Name())
		if err != nil {
			continue // エラーの計画はスキップ
		}

		plans = append(plans, PlanMetadata{
			ID:        plan.ID,
			Title:     plan.Title,
			Status:    plan.Status,
			CreatedAt: plan.CreatedAt,
			UpdatedAt: plan.UpdatedAt,
			Filename:  entry.Name(),
		})
	}

	// 新しい順にソート
	sort.Slice(plans, func(i, j int) bool {
		return plans[i].UpdatedAt.After(plans[j].UpdatedAt)
	})

	return plans, nil
}

// Delete は計画を削除
func (s *PlanStorage) Delete(filename string) error {
	path := filepath.Join(s.baseDir, filename)
	if err := os.Remove(path); err != nil {
		return fmt.Errorf("failed to delete plan: %w", err)
	}
	return nil
}

// LoadByID は ID で計画を検索して読み込み
func (s *PlanStorage) LoadByID(id string) (*Plan, string, error) {
	plans, err := s.List()
	if err != nil {
		return nil, "", err
	}

	for _, meta := range plans {
		if meta.ID == id {
			plan, err := s.Load(meta.Filename)
			return plan, meta.Filename, err
		}
	}

	return nil, "", fmt.Errorf("plan not found: %s", id)
}

// SetLastPlanID は最後に操作した Plan ID を記録
func (s *PlanStorage) SetLastPlanID(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.lastPlanID = id
}

// LastPlanID は最後に操作した Plan ID を返す
func (s *PlanStorage) LastPlanID() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.lastPlanID
}

// ClearLastPlanID は lastPlanID をクリアする
func (s *PlanStorage) ClearLastPlanID() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.lastPlanID = ""
}

// generateFilename はタイトルからファイル名を生成
func (s *PlanStorage) generateFilename(title string) string {
	date := time.Now().Format("20060102")
	slug := slugify(title)
	if slug == "" {
		slug = "plan"
	}
	return fmt.Sprintf("%s_%s.md", date, slug)
}

// slugify はタイトルを URL セーフな形式に変換
func slugify(title string) string {
	// 英数字とハイフンのみ許可
	reg := regexp.MustCompile(`[^a-zA-Z0-9\s-]`)
	slug := reg.ReplaceAllString(title, "")

	// 空白をハイフンに
	slug = strings.ReplaceAll(slug, " ", "-")

	// 連続するハイフンを単一に
	reg = regexp.MustCompile(`-+`)
	slug = reg.ReplaceAllString(slug, "-")

	// 先頭と末尾のハイフンを削除
	slug = strings.Trim(slug, "-")

	// 小文字に
	slug = strings.ToLower(slug)

	// 最大30文字
	if len(slug) > 30 {
		slug = slug[:30]
	}

	return slug
}

// formatMarkdown は Plan を Markdown 形式にフォーマット
func (s *PlanStorage) formatMarkdown(p *Plan) string {
	var sb strings.Builder

	// YAML フロントマター
	frontmatter := map[string]interface{}{
		"id":         p.ID,
		"title":      p.Title,
		"status":     string(p.Status),
		"created_at": p.CreatedAt.Format(time.RFC3339),
		"updated_at": p.UpdatedAt.Format(time.RFC3339),
	}
	if p.Model != "" {
		frontmatter["model"] = p.Model
	}
	if p.Provider != "" {
		frontmatter["provider"] = p.Provider
	}

	fmBytes, _ := yaml.Marshal(frontmatter)
	sb.WriteString("---\n")
	sb.Write(fmBytes)
	sb.WriteString("---\n\n")

	// タイトル
	sb.WriteString(fmt.Sprintf("# Plan: %s\n\n", p.Title))

	// User Request
	if p.UserRequest != "" {
		sb.WriteString("## User Request\n\n")
		sb.WriteString(p.UserRequest)
		sb.WriteString("\n\n")
	}

	// Clarifications
	if len(p.Clarifications) > 0 {
		sb.WriteString("## Clarifications\n\n")
		for i, c := range p.Clarifications {
			sb.WriteString(fmt.Sprintf("### Q%d: %s\n", i+1, c.Question))
			if c.Answer != "" {
				sb.WriteString(fmt.Sprintf("**Answer**: %s\n", c.Answer))
			} else if len(c.Answers) > 0 {
				sb.WriteString(fmt.Sprintf("**Answers**: %s\n", strings.Join(c.Answers, ", ")))
			}
			sb.WriteString("\n")
		}
	}

	// Summary
	sb.WriteString("## Summary\n\n")
	sb.WriteString(p.Summary)
	sb.WriteString("\n\n")

	// Steps
	sb.WriteString("## Steps\n\n")
	for _, step := range p.Steps {
		var checkbox string
		switch step.Status {
		case "completed":
			checkbox = "[x]"
		case "running":
			checkbox = "[~]"
		case "failed":
			checkbox = "[!]"
		default:
			checkbox = "[ ]"
		}

		sb.WriteString(fmt.Sprintf("- %s **Step %d**: %s\n", checkbox, step.ID, step.Description))

		if len(step.Tools) > 0 {
			sb.WriteString(fmt.Sprintf("  - Tools: %s\n", strings.Join(step.Tools, ", ")))
		}
		if len(step.Files) > 0 {
			sb.WriteString(fmt.Sprintf("  - Files: %s\n", strings.Join(step.Files, ", ")))
		}
		if step.StartedAt != nil {
			sb.WriteString(fmt.Sprintf("  - Started: %s\n", step.StartedAt.Format(time.RFC3339)))
		}
		if step.CompletedAt != nil {
			sb.WriteString(fmt.Sprintf("  - Completed: %s\n", step.CompletedAt.Format(time.RFC3339)))
		}
		sb.WriteString("\n")
	}

	// Execution Log
	if len(p.ExecutionLog) > 0 {
		sb.WriteString("## Execution Log\n\n")
		for _, log := range p.ExecutionLog {
			sb.WriteString(fmt.Sprintf("### Step %d (%s)\n", log.StepID, log.ExecutedAt.Format(time.RFC3339)))
			if log.ToolName != "" {
				sb.WriteString(fmt.Sprintf("- Tool: %s\n", log.ToolName))
			}
			if log.Result != "" {
				// 長い結果は省略
				result := log.Result
				if len(result) > 200 {
					result = result[:200] + "..."
				}
				sb.WriteString(fmt.Sprintf("- Result: %s\n", result))
			}
			if log.Error != "" {
				sb.WriteString(fmt.Sprintf("- Error: %s\n", log.Error))
			}
			sb.WriteString("\n")
		}
	}

	return sb.String()
}

// parseMarkdown は Markdown 形式から Plan を解析
func (s *PlanStorage) parseMarkdown(content string) (*Plan, error) {
	p := &Plan{}

	// フロントマターを解析
	if strings.HasPrefix(content, "---\n") {
		parts := strings.SplitN(content[4:], "\n---\n", 2)
		if len(parts) == 2 {
			var fm map[string]interface{}
			if err := yaml.Unmarshal([]byte(parts[0]), &fm); err == nil {
				if id, ok := fm["id"].(string); ok {
					p.ID = id
				}
				if title, ok := fm["title"].(string); ok {
					p.Title = title
				}
				if status, ok := fm["status"].(string); ok {
					p.Status = PlanStatus(status)
				}
				if createdAt, ok := fm["created_at"].(string); ok {
					if t, err := time.Parse(time.RFC3339, createdAt); err == nil {
						p.CreatedAt = t
					}
				}
				if updatedAt, ok := fm["updated_at"].(string); ok {
					if t, err := time.Parse(time.RFC3339, updatedAt); err == nil {
						p.UpdatedAt = t
					}
				}
				if model, ok := fm["model"].(string); ok {
					p.Model = model
				}
				if provider, ok := fm["provider"].(string); ok {
					p.Provider = provider
				}
			}
			content = parts[1]
		}
	}

	// セクションを解析
	scanner := bufio.NewScanner(strings.NewReader(content))
	currentSection := ""
	var sectionContent strings.Builder

	for scanner.Scan() {
		line := scanner.Text()

		if strings.HasPrefix(line, "## ") {
			// 前のセクションを保存
			if currentSection != "" {
				s.parseSectionContent(p, currentSection, sectionContent.String())
			}
			currentSection = strings.TrimPrefix(line, "## ")
			sectionContent.Reset()
		} else {
			sectionContent.WriteString(line)
			sectionContent.WriteString("\n")
		}
	}

	// 最後のセクションを保存
	if currentSection != "" {
		s.parseSectionContent(p, currentSection, sectionContent.String())
	}

	return p, nil
}

// parseSectionContent はセクション内容を解析
func (s *PlanStorage) parseSectionContent(p *Plan, section, content string) {
	content = strings.TrimSpace(content)

	switch section {
	case "User Request":
		p.UserRequest = content
	case "Summary":
		p.Summary = content
	case "Steps":
		p.Steps = s.parseSteps(content)
	case "Clarifications":
		p.Clarifications = s.parseClarifications(content)
	}
}

// parseSteps はステップを解析
func (s *PlanStorage) parseSteps(content string) []PlanStep {
	var steps []PlanStep
	lines := strings.Split(content, "\n")
	var currentStep *PlanStep

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// ステップ行の解析
		if strings.HasPrefix(line, "- [") {
			if currentStep != nil {
				steps = append(steps, *currentStep)
			}
			currentStep = &PlanStep{Status: "pending"}

			// ステータス
			if strings.HasPrefix(line, "- [x]") {
				currentStep.Status = "completed"
			} else if strings.HasPrefix(line, "- [~]") {
				currentStep.Status = "running"
			} else if strings.HasPrefix(line, "- [!]") {
				currentStep.Status = "failed"
			}

			// ID と説明
			// フォーマット: - [ ] **Step 1**: Description
			stepRegex := regexp.MustCompile(`\*\*Step (\d+)\*\*:\s*(.+)`)
			if matches := stepRegex.FindStringSubmatch(line); len(matches) == 3 {
				if id, err := fmt.Sscanf(matches[1], "%d", &currentStep.ID); id == 1 && err == nil {
					currentStep.Description = matches[2]
				}
			}
		} else if currentStep != nil && strings.HasPrefix(line, "- ") {
			// サブ項目の解析
			subItem := strings.TrimPrefix(line, "- ")
			if strings.HasPrefix(subItem, "Tools: ") {
				tools := strings.TrimPrefix(subItem, "Tools: ")
				currentStep.Tools = strings.Split(tools, ", ")
			} else if strings.HasPrefix(subItem, "Files: ") {
				files := strings.TrimPrefix(subItem, "Files: ")
				currentStep.Files = strings.Split(files, ", ")
			}
		}
	}

	if currentStep != nil {
		steps = append(steps, *currentStep)
	}

	return steps
}

// parseClarifications は Clarifications を解析
func (s *PlanStorage) parseClarifications(content string) []Clarification {
	var clarifications []Clarification
	lines := strings.Split(content, "\n")
	var current *Clarification

	for _, line := range lines {
		line = strings.TrimSpace(line)

		if strings.HasPrefix(line, "### Q") {
			if current != nil {
				clarifications = append(clarifications, *current)
			}
			current = &Clarification{}
			// Q1: Question text
			qRegex := regexp.MustCompile(`### Q\d+:\s*(.+)`)
			if matches := qRegex.FindStringSubmatch(line); len(matches) == 2 {
				current.Question = matches[1]
			}
		} else if current != nil && strings.HasPrefix(line, "**Answer**:") {
			current.Answer = strings.TrimSpace(strings.TrimPrefix(line, "**Answer**:"))
		} else if current != nil && strings.HasPrefix(line, "**Answers**:") {
			answers := strings.TrimSpace(strings.TrimPrefix(line, "**Answers**:"))
			current.Answers = strings.Split(answers, ", ")
		}
	}

	if current != nil {
		clarifications = append(clarifications, *current)
	}

	return clarifications
}
