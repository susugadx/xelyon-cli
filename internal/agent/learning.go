package agent

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/susugadx/xelyon-cli/internal/api"
)

// LearningExtractor extracts coding rules/instructions from conversation history.
// It is interface-based for testability (stub/mocking).
type LearningExtractor interface {
	Extract(ctx context.Context, history []api.Message, model string) ([]string, error)
}

// LLMExtractor uses the current provider to extract rules.
// It requests JSON output for robust parsing.
type LLMExtractor struct {
	Provider     api.Provider
	SystemPrompt string
}

type learnResponse struct {
	Rules []string `json:"rules"`
}

func (e *LLMExtractor) Extract(ctx context.Context, history []api.Message, model string) ([]string, error) {
	if e == nil || e.Provider == nil {
		return nil, errors.New("provider is nil")
	}

	system := "You are an assistant that extracts durable coding rules from a conversation.\n" +
		"Return ONLY valid JSON with the following schema: {\"rules\": [string, ...]}.\n" +
		"Rules must be short, actionable, and written as imperative statements.\n" +
		"Exclude one-off task details, project-specific trivia, or temporary decisions.\n" +
		"If there are no durable rules, return {\"rules\": []}."

	var b strings.Builder
	b.WriteString("Conversation:\n")
	for _, m := range history {
		role := strings.ToLower(m.Role)
		if role != "user" && role != "assistant" {
			continue
		}
		content := strings.TrimSpace(m.Content)
		if content == "" {
			continue
		}
		if strings.HasPrefix(content, "[Tool Result for ") {
			continue
		}
		b.WriteString("- ")
		b.WriteString(role)
		b.WriteString(": ")
		b.WriteString(content)
		b.WriteString("\n")
	}

	learnHistory := []api.Message{{Role: "user", Content: b.String()}}
	resp, err := e.Provider.ChatWithTools(ctx, system, learnHistory, model)
	if err != nil {
		return nil, err
	}

	jsonStr := extractFirstJSONObject(resp)
	if jsonStr == "" {
		jsonStr = strings.TrimSpace(resp)
	}

	var lr learnResponse
	if err := json.Unmarshal([]byte(jsonStr), &lr); err != nil {
		return nil, fmt.Errorf("failed to parse learning JSON: %w", err)
	}

	return normalizeRules(lr.Rules), nil
}

// ProposeAndApplyLearning extracts rules and asks the user whether to append them to XELYON.md.
// It returns (applied, rules, err). If no rules are found, applied=false and err=nil.
func (a *Agent) ProposeAndApplyLearning(ctx context.Context, extractor LearningExtractor, xelyonPath string) (bool, []string, error) {
	if a == nil {
		return false, nil, errors.New("agent is nil")
	}
	if extractor == nil {
		return false, nil, errors.New("extractor is nil")
	}
	if xelyonPath == "" {
		xelyonPath = "XELYON.md"
	}

	learnCtx := ctx
	if learnCtx == nil {
		learnCtx = context.Background()
	}
	learnCtx, cancel := context.WithTimeout(learnCtx, 45*time.Second)
	defer cancel()

	rules, err := extractor.Extract(learnCtx, a.History, a.CurrentModel)
	if err != nil {
		return false, nil, err
	}
	if len(rules) == 0 {
		return false, nil, nil
	}

	contentBytes, err := os.ReadFile(xelyonPath)
	if err != nil {
		return false, rules, fmt.Errorf("failed to read %s: %w", xelyonPath, err)
	}
	content := string(contentBytes)

	appliedRules := filterNewRules(content, rules)
	if len(appliedRules) == 0 {
		return false, rules, nil
	}

	cyan.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	cyan.Println("🧠 会話学習 / Conversation Learning")
	cyan.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	yellow.Println("このセッションで見つかった『今後も守るべきルール』候補があります。")
	yellow.Println("XELYON.md に追加しますか？")
	fmt.Println()
	for i, r := range appliedRules {
		fmt.Printf("  [%d] %s\n", i+1, r)
	}
	fmt.Println()
	yellow.Print("Add to XELYON.md? (y/n): ")

	reader := bufio.NewReader(os.Stdin)
	input, readErr := reader.ReadString('\n')
	if readErr != nil {
		return false, appliedRules, fmt.Errorf("failed to read input: %w", readErr)
	}
	input = strings.TrimSpace(strings.ToLower(input))
	if input != "y" && input != "yes" {
		yellow.Println("Skipped")
		return false, appliedRules, nil
	}

	updated, err := appendRulesToXelyon(content, appliedRules)
	if err != nil {
		return false, appliedRules, err
	}
	if err := os.WriteFile(xelyonPath, []byte(updated), 0600); err != nil {
		return false, appliedRules, fmt.Errorf("failed to write %s: %w", xelyonPath, err)
	}

	green.Printf("✅ XELYON.md に %d 件のルールを追加しました\n", len(appliedRules))
	return true, appliedRules, nil
}

func normalizeRules(rules []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(rules))
	for _, r := range rules {
		r = strings.TrimSpace(r)
		r = strings.Trim(r, "-•*\t ")
		r = strings.Join(strings.Fields(r), " ")
		if r == "" {
			continue
		}
		key := strings.ToLower(r)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, r)
	}
	return out
}

func filterNewRules(xelyonContent string, rules []string) []string {
	out := make([]string, 0, len(rules))
	lower := strings.ToLower(xelyonContent)
	for _, r := range rules {
		if strings.Contains(lower, strings.ToLower(r)) {
			continue
		}
		out = append(out, r)
	}
	return out
}

// extractFirstJSONObject extracts the first JSON object from a string
func extractFirstJSONObject(s string) string {
	start := strings.Index(s, "{")
	if start == -1 {
		return ""
	}

	depth := 0
	inString := false
	escaped := false

	for i := start; i < len(s); i++ {
		ch := s[i]

		if escaped {
			escaped = false
			continue
		}
		if ch == '\\' {
			escaped = true
			continue
		}
		if ch == '"' {
			inString = !inString
			continue
		}
		if inString {
			continue
		}

		switch ch {
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return s[start : i+1]
			}
		}
	}

	return ""
}

func appendRulesToXelyon(xelyonContent string, rules []string) (string, error) {
	sectionHeader := "## 学習したルール"
	if !strings.Contains(xelyonContent, sectionHeader) {
		var b strings.Builder
		b.WriteString(strings.TrimRight(xelyonContent, "\n"))
		b.WriteString("\n\n")
		b.WriteString(sectionHeader)
		b.WriteString("\n\n")
		b.WriteString("このセクションは、セッション終了時に会話から抽出された『今後も守るべきルール』の提案を追記します。\n")
		b.WriteString("（必要に応じて手動で整理してください）\n\n")
		for _, r := range rules {
			b.WriteString("- ")
			b.WriteString(r)
			b.WriteString("\n")
		}
		b.WriteString("\n")
		return b.String(), nil
	}

	lines := strings.Split(xelyonContent, "\n")
	idx := -1
	for i, line := range lines {
		if strings.TrimSpace(line) == sectionHeader {
			idx = i
			break
		}
	}
	if idx == -1 {
		return "", errors.New("learning section header not found")
	}

	insertAt := idx + 1
	if insertAt < len(lines) && strings.TrimSpace(lines[insertAt]) == "" {
		insertAt++
	}

	insertLines := make([]string, 0, len(rules))
	for _, r := range rules {
		insertLines = append(insertLines, "- "+r)
	}

	newLines := make([]string, 0, len(lines)+len(insertLines)+1)
	newLines = append(newLines, lines[:insertAt]...)
	newLines = append(newLines, insertLines...)
	newLines = append(newLines, "")
	newLines = append(newLines, lines[insertAt:]...)

	return strings.Join(newLines, "\n"), nil
}
