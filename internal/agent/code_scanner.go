package agent

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

// CodeCheckResult はコードスキャン結果
type CodeCheckResult struct {
	Exists      bool     // 定義が既に存在するか
	Type        string   // "function", "type", "const", "var", "method"
	Name        string   // 定義名
	FilePaths   []string // 定義が存在するファイルパス
	LineNumbers []int    // 各ファイルでの行番号
	Context     string   // 周辺コード（最初の一致のみ）
}

// ScanForDefinition は指定された名前の定義をコードベースからスキャン
func ScanForDefinition(name string, searchPath string) (*CodeCheckResult, error) {
	if searchPath == "" {
		searchPath = "."
	}

	result := &CodeCheckResult{
		Name:        name,
		Exists:      false,
		FilePaths:   []string{},
		LineNumbers: []int{},
	}

	// 検索パターンを定義タイプ別に作成
	patterns := []struct {
		pattern string
		defType string
	}{
		{fmt.Sprintf(`func\s+%s\s*\(`, regexp.QuoteMeta(name)), "function"},
		{fmt.Sprintf(`func\s+\([^)]+\)\s*%s\s*\(`, regexp.QuoteMeta(name)), "method"},
		{fmt.Sprintf(`type\s+%s\s+`, regexp.QuoteMeta(name)), "type"},
		{fmt.Sprintf(`const\s+%s\s*=`, regexp.QuoteMeta(name)), "const"},
		{fmt.Sprintf(`var\s+%s\s+`, regexp.QuoteMeta(name)), "var"},
	}

	for _, p := range patterns {
		files, lines, context, err := grepCodebase(p.pattern, searchPath)
		if err != nil {
			continue
		}
		if len(files) > 0 {
			result.Exists = true
			result.Type = p.defType
			result.FilePaths = files
			result.LineNumbers = lines
			result.Context = context
			return result, nil
		}
	}

	return result, nil
}

// grepCodebase はコードベースを検索
func grepCodebase(pattern, searchPath string) (files []string, lines []int, context string, err error) {
	// ripgrepを試す（高速）
	cmd := exec.Command("rg", "-n", "--no-heading", "-e", pattern, searchPath,
		"--type", "go", "--type", "ts", "--type", "js", "--type", "py", "--type", "rust")
	output, cmdErr := cmd.Output()

	// ripgrepがない場合はgrepを使用
	if cmdErr != nil && strings.Contains(cmdErr.Error(), "executable file not found") {
		cmd = exec.Command("grep", "-rn", "-E", pattern, searchPath,
			"--include=*.go", "--include=*.ts", "--include=*.js", "--include=*.py", "--include=*.rs")
		output, cmdErr = cmd.Output()
	}

	// マッチなしの場合はexit 1を返すので、出力がない場合は空結果
	if len(output) == 0 {
		return nil, nil, "", nil
	}

	// 結果をパース
	// 形式: filepath:line:content
	resultLines := strings.Split(strings.TrimSpace(string(output)), "\n")
	for _, line := range resultLines {
		parts := strings.SplitN(line, ":", 3)
		if len(parts) >= 3 {
			files = append(files, parts[0])
			lineNum := 0
			fmt.Sscanf(parts[1], "%d", &lineNum)
			lines = append(lines, lineNum)
			if context == "" {
				context = parts[2]
			}
		}
	}

	return files, lines, context, nil
}

// ExtractDefinitionNames は入力テキストから関数/型/変数名を抽出
func ExtractDefinitionNames(input string) []string {
	var names []string

	// パターンマッチング
	patterns := []*regexp.Regexp{
		// "handleXxx を追加" / "handleXxx を実装" パターン
		regexp.MustCompile(`(?:追加|実装|作成|定義|add|create|implement|define)\s+(?:する|して)?\s*[「『]?(\w+)[」』]?`),
		// "Xxx を追加" パターン
		regexp.MustCompile(`[「『]?(\w+)[」』]?\s*(?:を|の)?\s*(?:追加|実装|作成|定義)`),
		// "func Xxx" パターン
		regexp.MustCompile(`func\s+(\w+)`),
		// "type Xxx" パターン
		regexp.MustCompile(`type\s+(\w+)`),
		// 関数名っぽいもの (camelCase, PascalCase, snake_case)
		regexp.MustCompile(`\b([a-zA-Z][a-zA-Z0-9_]*(?:Command|Handler|Func|Function|Type|Struct|Interface))\b`),
	}

	for _, pat := range patterns {
		matches := pat.FindAllStringSubmatch(input, -1)
		for _, match := range matches {
			if len(match) > 1 && len(match[1]) > 2 {
				names = append(names, match[1])
			}
		}
	}

	// 重複を除去
	seen := make(map[string]bool)
	var unique []string
	for _, name := range names {
		if !seen[name] {
			seen[name] = true
			unique = append(unique, name)
		}
	}

	return unique
}

// IsCodeModificationRequest はコード変更を伴うリクエストかどうかを判定
func IsCodeModificationRequest(input string) bool {
	modPatterns := []string{
		// 日本語パターン
		"追加", "実装", "作成", "定義", "修正", "変更", "削除",
		"書き換え", "リファクタ", "追加して", "実装して",
		// 英語パターン
		"add", "create", "implement", "define", "modify", "change",
		"delete", "remove", "refactor", "update", "fix", "write",
	}

	lowered := strings.ToLower(input)
	for _, pat := range modPatterns {
		if strings.Contains(lowered, strings.ToLower(pat)) {
			return true
		}
	}

	return false
}

// CheckBeforeImplementation は実装前に既存定義をチェック
// 戻り値: 警告メッセージ（問題なければ空文字列）
func CheckBeforeImplementation(input string) string {
	// コード変更リクエストでない場合はスキップ
	if !IsCodeModificationRequest(input) {
		return ""
	}

	// 定義名を抽出
	names := ExtractDefinitionNames(input)
	if len(names) == 0 {
		return ""
	}

	// 各名前をスキャン
	var warnings []string
	for _, name := range names {
		result, err := ScanForDefinition(name, ".")
		if err != nil {
			continue
		}

		if result.Exists {
			// 既存定義を発見
			filePaths := result.FilePaths
			if len(filePaths) > 3 {
				filePaths = filePaths[:3]
			}
			warning := fmt.Sprintf("⚠️  '%s' (%s) already exists in:\n", name, result.Type)
			for i, fp := range filePaths {
				lineNum := 0
				if i < len(result.LineNumbers) {
					lineNum = result.LineNumbers[i]
				}
				warning += fmt.Sprintf("   - %s:%d\n", fp, lineNum)
			}
			if result.Context != "" {
				warning += fmt.Sprintf("   Context: %s\n", strings.TrimSpace(result.Context))
			}
			warnings = append(warnings, warning)
		}
	}

	if len(warnings) == 0 {
		return ""
	}

	return strings.Join(warnings, "\n") + "\nConsider modifying existing code instead of creating duplicates."
}

// GetFileLanguage はファイル拡張子から言語を判定
func GetFileLanguage(filePath string) string {
	ext := strings.ToLower(filepath.Ext(filePath))
	switch ext {
	case ".go":
		return "go"
	case ".ts", ".tsx":
		return "typescript"
	case ".js", ".jsx":
		return "javascript"
	case ".py":
		return "python"
	case ".rs":
		return "rust"
	case ".java":
		return "java"
	case ".rb":
		return "ruby"
	case ".php":
		return "php"
	default:
		return "unknown"
	}
}
