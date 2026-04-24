package plan

import (
	"regexp"
	"strings"
)

// ツール分類（読み取り専用）
var readTools = map[string]bool{
	"gather_context":  true,
	"read_file":       true,
	"list_dir":        true,
	"git_status":      true,
	"git_diff":        true,
	"git_log":         true,
	"lsp_references":  true,
	"lsp_definition":  true,
	"lsp_hover":       true,
	"lsp_diagnostics": true,
}

// ツール分類（書き込み）
var writeTools = map[string]bool{
	"write_file":  true,
	"str_replace": true,
	"delete_file": true,
	"copy_file":   true,
	"create_dir":  true,
	"lsp_rename":  true, // 実際にはプレビューのみだが、将来の実装に備えてwrite扱い
}

// ファイルパス抽出用の正規表現パターン
var filePathPatterns = []*regexp.Regexp{
	// 引用符で囲まれたパス
	regexp.MustCompile(`["']([^"']+\.[a-zA-Z0-9]+)["']`),
	// パス風文字列（スペースなし）
	regexp.MustCompile(`\b((?:[\w.-]+/)*[\w.-]+\.[a-zA-Z0-9]{1,10})\b`),
	// 絶対パス
	regexp.MustCompile(`(/[^\s]+\.[a-zA-Z0-9]+)`),
}

// ExtractFilesFromStep はステップからファイルパスを抽出する。
func (da *DependencyAnalyzer) ExtractFilesFromStep(step *PlanStep) (readFiles, writeFiles []string) {
	readSet := make(map[string]bool)
	writeSet := make(map[string]bool)

	// Description からファイルパスを抽出
	files := extractFilePaths(step.Description)

	// ツールの種類に基づいて分類
	hasReadTool := false
	hasWriteTool := false
	for _, tool := range step.Tools {
		if readTools[tool] {
			hasReadTool = true
		}
		if writeTools[tool] {
			hasWriteTool = true
		}
	}

	// 抽出されたファイルを分類
	for _, f := range files {
		// 書き込みツールがある場合、そのファイルは書き込み対象
		if hasWriteTool {
			writeSet[f] = true
		}
		// 読み取りツールがある場合、または書き込みツールがない場合は読み取り
		if hasReadTool || !hasWriteTool {
			readSet[f] = true
		}
	}

	// マップからスライスに変換
	for f := range readSet {
		readFiles = append(readFiles, f)
	}
	for f := range writeSet {
		writeFiles = append(writeFiles, f)
	}
	return readFiles, writeFiles
}

// extractFilePaths はテキストからファイルパスを抽出する。
func extractFilePaths(text string) []string {
	pathSet := make(map[string]bool)
	for _, pattern := range filePathPatterns {
		matches := pattern.FindAllStringSubmatch(text, -1)
		for _, match := range matches {
			if len(match) > 1 {
				path := match[1]
				// 明らかに非ファイルを除外
				if isValidFilePath(path) {
					pathSet[path] = true
				}
			}
		}
	}

	var paths []string
	for p := range pathSet {
		paths = append(paths, p)
	}
	return paths
}

// isValidFilePath はパスがファイルパスとして妥当かチェックする。
func isValidFilePath(path string) bool {
	// 空文字列
	if path == "" {
		return false
	}

	// 明らかにURLの場合
	if strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") {
		return false
	}

	// 拡張子のない場合は除外（ディレクトリかもしれない）
	if !strings.Contains(path, ".") {
		return false
	}

	// コモンな非ファイル拡張子を除外
	invalidExts := []string{".com", ".org", ".net", ".io"}
	for _, ext := range invalidExts {
		if strings.HasSuffix(path, ext) && !strings.Contains(path, "/") {
			return false
		}
	}

	return true
}
